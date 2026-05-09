package prices

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"foundry-tx-simulator/backend/internal/fundflow"
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

	out := make(map[string]fundflow.TokenPrice)
	for _, token := range tokens {
		price, err := p.fetchOne(ctx, platform, token)
		if err != nil {
			slog.Warn("coingecko price fetch failed", "chain", chain, "platform", platform, "token", token, "error", err)
			continue
		}
		if price.PriceUSD <= 0 {
			slog.Warn("coingecko price missing token", "chain", chain, "platform", platform, "token", token)
			continue
		}
		slog.Info("coingecko price fetched", "chain", chain, "platform", platform, "token", token, "price_usd", price.PriceUSD)
		out[token] = price
	}
	return out, nil
}

func (p CoinGeckoProvider) fetchOne(ctx context.Context, platform string, token string) (fundflow.TokenPrice, error) {
	endpoint := trimBaseURL(p.BaseURL, coinGeckoBaseURL) + "/" + url.PathEscape(platform)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return fundflow.TokenPrice{}, err
	}
	query := req.URL.Query()
	query.Set("contract_addresses", token)
	query.Set("vs_currencies", "usd")
	req.URL.RawQuery = query.Encode()
	if strings.TrimSpace(p.APIKey) != "" {
		req.Header.Set("x-cg-demo-api-key", strings.TrimSpace(p.APIKey))
	}

	resp, err := defaultHTTPClient(p.Client).Do(req)
	if err != nil {
		return fundflow.TokenPrice{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fundflow.TokenPrice{}, fmt.Errorf("coingecko price request failed: %s", resp.Status)
	}

	var payload map[string]struct {
		USD float64 `json:"usd"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fundflow.TokenPrice{}, err
	}

	for _, price := range payload {
		if price.USD <= 0 {
			continue
		}
		return fundflow.TokenPrice{PriceUSD: price.USD}, nil
	}
	return fundflow.TokenPrice{}, nil
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
