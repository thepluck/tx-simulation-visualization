package prices

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"foundry-tx-simulator/backend/internal/fundflow"
)

func TestMultiProviderMergesPriceMetadata(t *testing.T) {
	token := "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"
	provider := MultiProvider{
		Providers: []Provider{
			fakeProvider{token: {Decimals: 6, HasDecimals: true, Symbol: "USDC", LogoURL: "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48/logo.png"}},
			fakeProvider{token: {PriceUSD: 1.01}},
			fakeProvider{token: {LogoURL: "https://example.com/usdc.png"}},
		},
	}

	got, err := provider.Fetch(context.Background(), "mainnet", []string{token})
	if err != nil {
		t.Fatal(err)
	}
	price := got[token]
	if price.PriceUSD != 1.01 || price.Decimals != 6 || !price.HasDecimals || price.Symbol != "USDC" || price.LogoURL != "https://example.com/usdc.png" {
		t.Fatalf("merged price = %#v", price)
	}
}

func TestMultiProviderAppliesStablecoinFallback(t *testing.T) {
	token := "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	provider := MultiProvider{
		Providers: []Provider{
			fakeProvider{token: {Decimals: 6, HasDecimals: true, Symbol: "USDC"}},
		},
	}

	got, err := provider.Fetch(context.Background(), "base", []string{token})
	if err != nil {
		t.Fatal(err)
	}
	price := got[token]
	if price.PriceUSD != 1 || price.Decimals != 6 || !price.HasDecimals || price.Symbol != "USDC" {
		t.Fatalf("stablecoin fallback price = %#v", price)
	}
}

func TestFetchJSONConfiguresAndDecodesRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("token"); got != "0xabc" {
			t.Fatalf("token query = %s, want 0xabc", got)
		}
		if got := r.Header.Get("x-test-key"); got != "test-key" {
			t.Fatalf("x-test-key header = %s, want test-key", got)
		}
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	var payload struct {
		OK bool `json:"ok"`
	}
	err := fetchJSON(context.Background(), nil, server.URL, "test price", &payload, func(req *http.Request) {
		query := req.URL.Query()
		query.Set("token", "0xabc")
		req.URL.RawQuery = query.Encode()
		req.Header.Set("x-test-key", "test-key")
	})
	if err != nil {
		t.Fatal(err)
	}
	if !payload.OK {
		t.Fatalf("payload.OK = false, want true")
	}
}

func TestFetchJSONReturnsStatusError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
	}))
	defer server.Close()

	var payload struct{}
	err := fetchJSON(context.Background(), nil, server.URL, "test price", &payload, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "test price request failed: 429 Too Many Requests") {
		t.Fatalf("unexpected error: %v", err)
	}
}

type fakeProvider map[string]fundflow.TokenPrice

func (p fakeProvider) Fetch(_ context.Context, _ string, _ []string) (map[string]fundflow.TokenPrice, error) {
	return map[string]fundflow.TokenPrice(p), nil
}
