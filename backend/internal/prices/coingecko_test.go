package prices

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCoinGeckoProviderFetch(t *testing.T) {
	token := "0x2260fac5e5542a773aa44fbcfedf7c193bc2c599"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/simple/token_price/ethereum" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("contract_addresses"); got != token {
			t.Fatalf("contract_addresses = %s, want %s", got, token)
		}
		if got := r.URL.Query().Get("vs_currencies"); got != "usd" {
			t.Fatalf("vs_currencies = %s, want usd", got)
		}
		if got := r.Header.Get("x-cg-demo-api-key"); got != "test-key" {
			t.Fatalf("api key header = %s, want test-key", got)
		}
		_, _ = w.Write([]byte(`{"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599":{"usd":67187.33}}`))
	}))
	defer server.Close()

	provider := CoinGeckoProvider{BaseURL: server.URL + "/simple/token_price", APIKey: "test-key"}
	got, err := provider.Fetch(context.Background(), "mainnet", []string{token})
	if err != nil {
		t.Fatal(err)
	}
	if got[token].PriceUSD != 67187.33 {
		t.Fatalf("price = %#v", got[token])
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
