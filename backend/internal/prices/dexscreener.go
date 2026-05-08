package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"tx-simulation-visualization/backend/internal/fundflow"
)

const dexScreenerBaseURL = "https://api.dexscreener.com/tokens/v1/"

type DexScreenerProvider struct {
	Client  *http.Client
	BaseURL string
}

func (p DexScreenerProvider) Fetch(ctx context.Context, chain string, tokens []string) (map[string]fundflow.TokenPrice, error) {
	chainID := dexScreenerChain(chain)
	if chainID == "" {
		return nil, nil
	}
	tokens = normalizeTokens(tokens)
	if len(tokens) == 0 {
		return nil, nil
	}

	wanted := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		wanted[token] = struct{}{}
	}

	endpoint := trimBaseURL(p.BaseURL, dexScreenerBaseURL) + "/" + url.PathEscape(chainID) + "/" + strings.Join(tokens, ",")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := defaultHTTPClient(p.Client).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("dexscreener price request failed: %s", resp.Status)
	}

	var pairs []struct {
		BaseToken struct {
			Address string `json:"address"`
			Symbol  string `json:"symbol"`
		} `json:"baseToken"`
		PriceUSD  string `json:"priceUsd"`
		Liquidity *struct {
			USD float64 `json:"usd"`
		} `json:"liquidity"`
		Info *struct {
			ImageURL string `json:"imageUrl"`
		} `json:"info"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&pairs); err != nil {
		return nil, err
	}

	best := make(map[string]dexScreenerCandidate)
	for _, pair := range pairs {
		token := normalizeAddress(pair.BaseToken.Address)
		if _, ok := wanted[token]; !ok {
			continue
		}
		priceUSD, err := strconv.ParseFloat(strings.TrimSpace(pair.PriceUSD), 64)
		if err != nil || priceUSD <= 0 {
			continue
		}
		liquidityUSD := 0.0
		if pair.Liquidity != nil {
			liquidityUSD = pair.Liquidity.USD
		}
		candidate := dexScreenerCandidate{
			price:     priceUSD,
			liquidity: liquidityUSD,
			symbol:    strings.TrimSpace(pair.BaseToken.Symbol),
		}
		if pair.Info != nil {
			candidate.logoURL = strings.TrimSpace(pair.Info.ImageURL)
		}
		if current, ok := best[token]; !ok || candidate.liquidity > current.liquidity {
			best[token] = candidate
		}
	}

	out := make(map[string]fundflow.TokenPrice)
	for token, candidate := range best {
		out[token] = fundflow.TokenPrice{
			PriceUSD: candidate.price,
			Symbol:   candidate.symbol,
			LogoURL:  candidate.logoURL,
		}
	}
	return out, nil
}

type dexScreenerCandidate struct {
	price     float64
	liquidity float64
	symbol    string
	logoURL   string
}

func dexScreenerChain(chain string) string {
	switch strings.ToLower(strings.TrimSpace(chain)) {
	case "mainnet", "eth", "ethereum":
		return "ethereum"
	case "arbitrum-one", "arbitrum_one":
		return "arbitrum"
	case "optimism", "optimistic-ethereum", "op":
		return "optimism"
	case "polygon-pos", "matic":
		return "polygon"
	case "binance-smart-chain", "bnb":
		return "bsc"
	case "avalanche", "avax":
		return "avalanche"
	default:
		return strings.ToLower(strings.TrimSpace(chain))
	}
}
