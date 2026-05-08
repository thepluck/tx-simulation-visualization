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
				"priceUsd":"2000.1",
				"liquidity":{"usd":10},
				"info":{"imageUrl":"https://example.com/low.png"}
			},
			{
				"baseToken":{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH"},
				"priceUsd":"2000.2",
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
	if price.PriceUSD != 2000.2 || price.Symbol != "WETH" || price.LogoURL != "https://example.com/high.png" {
		t.Fatalf("price = %#v", price)
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
