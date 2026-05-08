package prices

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCoinGeckoProviderFetch(t *testing.T) {
	tokens := []string{
		"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599",
		"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
	}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simple/token_price/ethereum" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		token := r.URL.Query().Get("contract_addresses")
		if token != tokens[0] && token != tokens[1] {
			t.Fatalf("contract_addresses = %s, want one of %#v", token, tokens)
		}
		if strings.Contains(token, ",") {
			t.Fatalf("contract_addresses should contain one token per request, got %s", token)
		}
		if got := r.URL.Query().Get("vs_currencies"); got != "usd" {
			t.Fatalf("vs_currencies = %s, want usd", got)
		}
		if got := r.Header.Get("x-cg-demo-api-key"); got != "test-key" {
			t.Fatalf("api key header = %s, want test-key", got)
		}
		requests++
		_, _ = w.Write([]byte(`{"` + token + `":{"usd":1.23}}`))
	}))
	defer server.Close()

	provider := CoinGeckoProvider{BaseURL: server.URL + "/simple/token_price", APIKey: "test-key"}
	got, err := provider.Fetch(context.Background(), "mainnet", tokens)
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want 2", requests)
	}
	for _, token := range tokens {
		if got[token].PriceUSD != 1.23 {
			t.Fatalf("price[%s] = %#v", token, got[token])
		}
	}
}

func TestCoinGeckoPlatform(t *testing.T) {
	if got := coinGeckoPlatform("arbitrum"); got != "arbitrum-one" {
		t.Fatalf("arbitrum platform = %s", got)
	}
	if got := coinGeckoPlatform("polygon"); got != "polygon-pos" {
		t.Fatalf("polygon platform = %s", got)
	}
}
