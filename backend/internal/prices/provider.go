package prices

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"tx-simulation-visualization/backend/internal/fundflow"
)

type Provider interface {
	Fetch(ctx context.Context, chain string, tokens []string) (map[string]fundflow.TokenPrice, error)
}

type MultiProvider struct {
	Providers []Provider
}

func DefaultProvider(rpcURLs map[string]string) MultiProvider {
	client := &http.Client{Timeout: 10 * time.Second}
	return MultiProvider{
		Providers: []Provider{
			RPCMetadataProvider{Client: client, RPCURLs: rpcURLs},
			DefiLlamaProvider{Client: client},
			CoinGeckoProvider{Client: client, APIKey: os.Getenv("COINGECKO_API_KEY")},
			DexScreenerProvider{Client: client},
		},
	}
}

func (p MultiProvider) Fetch(ctx context.Context, chain string, tokens []string) (map[string]fundflow.TokenPrice, error) {
	normalizedTokens := normalizeTokens(tokens)
	if len(normalizedTokens) == 0 {
		return nil, nil
	}

	out := make(map[string]fundflow.TokenPrice)
	for _, provider := range p.Providers {
		if provider == nil {
			continue
		}
		fetched, err := provider.Fetch(ctx, chain, normalizedTokens)
		if err != nil {
			slog.Warn("token price provider failed", "provider", priceProviderName(provider), "chain", chain, "token_count", len(normalizedTokens), "error", err)
			continue
		}
		slog.Info(
			"token price provider completed",
			"provider", priceProviderName(provider),
			"chain", chain,
			"token_count", len(normalizedTokens),
			"returned_tokens", len(fetched),
			"usd_priced_tokens", fundflow.CountUSDPrices(fetched),
		)
		for token, next := range fetched {
			token = normalizeAddress(token)
			if token == "" {
				continue
			}
			out[token] = mergeTokenPrice(out[token], next)
		}
	}
	applyStablecoinFallback(out)
	return out, nil
}

func mergeTokenPrice(existing fundflow.TokenPrice, next fundflow.TokenPrice) fundflow.TokenPrice {
	if existing.PriceUSD <= 0 && next.PriceUSD > 0 {
		existing.PriceUSD = next.PriceUSD
	}
	if !existing.HasDecimals && next.HasDecimals {
		existing.Decimals = next.Decimals
		existing.HasDecimals = true
	}
	if existing.Symbol == "" && next.Symbol != "" {
		existing.Symbol = next.Symbol
	}
	if next.LogoURL != "" && (existing.LogoURL == "" || isTrustWalletLogoURL(existing.LogoURL)) {
		existing.LogoURL = next.LogoURL
	}
	return existing
}

func applyStablecoinFallback(prices map[string]fundflow.TokenPrice) {
	for token, price := range prices {
		if price.PriceUSD > 0 || !fundflow.IsStablecoinSymbol(price.Symbol) {
			continue
		}
		price.PriceUSD = 1
		prices[token] = price
		slog.Info("stablecoin price fallback applied", "token", token, "symbol", price.Symbol, "price_usd", price.PriceUSD)
	}
}

func priceProviderName(provider Provider) string {
	name := fmt.Sprintf("%T", provider)
	name = strings.TrimPrefix(name, "prices.")
	name = strings.TrimPrefix(name, "*prices.")
	return name
}

func isTrustWalletLogoURL(value string) bool {
	return strings.Contains(value, "raw.githubusercontent.com/trustwallet/assets/")
}

func normalizeTokens(tokens []string) []string {
	seen := make(map[string]struct{}, len(tokens))
	out := make([]string, 0, len(tokens))
	for _, token := range tokens {
		token = normalizeAddress(token)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		out = append(out, token)
	}
	sort.Strings(out)
	return out
}

func normalizeAddress(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func defaultHTTPClient(client *http.Client) *http.Client {
	if client != nil {
		return client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func trimBaseURL(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		value = fallback
	}
	return strings.TrimRight(value, "/")
}
