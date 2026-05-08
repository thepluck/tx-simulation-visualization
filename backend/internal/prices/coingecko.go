package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"tx-simulation-visualization/backend/internal/fundflow"
)

const coinGeckoBaseURL = "https://api.coingecko.com/api/v3/simple/token_price/"

type CoinGeckoProvider struct {
	Client  *http.Client
	BaseURL string
	APIKey  string
}

func (p CoinGeckoProvider) Fetch(ctx context.Context, chain string, tokens []string) (map[string]fundflow.TokenPrice, error) {
	platform := coinGeckoPlatform(chain)
	if platform == "" {
		return nil, nil
	}
	tokens = normalizeTokens(tokens)
	if len(tokens) == 0 {
		return nil, nil
	}

	endpoint := trimBaseURL(p.BaseURL, coinGeckoBaseURL) + "/" + url.PathEscape(platform)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	query := req.URL.Query()
	query.Set("contract_addresses", strings.Join(tokens, ","))
	query.Set("vs_currencies", "usd")
	req.URL.RawQuery = query.Encode()
	if strings.TrimSpace(p.APIKey) != "" {
		req.Header.Set("x-cg-demo-api-key", strings.TrimSpace(p.APIKey))
	}

	resp, err := defaultHTTPClient(p.Client).Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("coingecko price request failed: %s", resp.Status)
	}

	var payload map[string]struct {
		USD float64 `json:"usd"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}

	out := make(map[string]fundflow.TokenPrice)
	for token, price := range payload {
		if price.USD <= 0 {
			continue
		}
		out[normalizeAddress(token)] = fundflow.TokenPrice{PriceUSD: price.USD}
	}
	return out, nil
}

func coinGeckoPlatform(chain string) string {
	switch strings.ToLower(strings.TrimSpace(chain)) {
	case "mainnet", "eth", "ethereum":
		return "ethereum"
	case "arbitrum", "arbitrum-one", "arbitrum_one":
		return "arbitrum-one"
	case "optimism", "optimistic-ethereum", "op":
		return "optimistic-ethereum"
	case "polygon", "polygon-pos", "matic":
		return "polygon-pos"
	case "bsc", "bnb", "binance-smart-chain":
		return "binance-smart-chain"
	case "avalanche", "avax":
		return "avalanche"
	case "gnosis", "xdai":
		return "xdai"
	default:
		return strings.ToLower(strings.TrimSpace(chain))
	}
}
