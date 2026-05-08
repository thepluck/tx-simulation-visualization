package prices

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDexScreenerProviderFetchUsesMostLiquidBasePair(t *testing.T) {
	token := "0x4200000000000000000000000000000000000006"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tokens/v1/base/"+token {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[
			{
				"baseToken":{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH"},
				"liquidity":{"usd":10},
				"info":{"imageUrl":"https://example.com/low.png"}
			},
			{
				"baseToken":{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH"},
				"liquidity":{"usd":1000},
				"info":{"imageUrl":"https://example.com/high.png"}
			}
		]`))
	}))
	defer server.Close()

	provider := DexScreenerProvider{BaseURL: server.URL + "/tokens/v1"}
	got, err := provider.Fetch(context.Background(), "base", []string{token})
	if err != nil {
		t.Fatal(err)
	}
	price := got[token]
	if price.PriceUSD != 0 || price.Symbol != "WETH" || price.LogoURL != "https://example.com/high.png" {
		t.Fatalf("metadata = %#v", price)
	}
}

func TestDexScreenerProviderFetchIgnoresOtherBaseToken(t *testing.T) {
	token := "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tokens/v1/base/"+token {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`[
			{
				"baseToken":{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH"},
				"liquidity":{"usd":1000000},
				"info":{"imageUrl":"https://example.com/weth.png"}
			}
		]`))
	}))
	defer server.Close()

	provider := DexScreenerProvider{BaseURL: server.URL + "/tokens/v1"}
	got, err := provider.Fetch(context.Background(), "base", []string{token})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := got[token]; ok {
		t.Fatalf("unrequested base token metadata should be ignored: %#v", got[token])
	}
}

func TestDexScreenerChain(t *testing.T) {
	if got := dexScreenerChain("mainnet"); got != "ethereum" {
		t.Fatalf("mainnet chain = %s", got)
	}
	if got := dexScreenerChain("arbitrum-one"); got != "arbitrum" {
		t.Fatalf("arbitrum chain = %s", got)
	}
}
