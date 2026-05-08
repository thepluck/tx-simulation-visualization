package prices

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefiLlamaProviderFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/prices/current/ethereum:0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		_, _ = w.Write([]byte(`{"coins":{"ethereum:0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2":{"price":2000.5,"decimals":18,"symbol":"WETH"}}}`))
	}))
	defer server.Close()

	provider := DefiLlamaProvider{BaseURL: server.URL + "/prices/current/"}
	got, err := provider.Fetch(context.Background(), "mainnet", []string{"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"})
	if err != nil {
		t.Fatal(err)
	}

	price, ok := got["0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"]
	if !ok {
		t.Fatalf("missing price: %#v", got)
	}
	if price.PriceUSD != 2000.5 || price.Decimals != 18 || !price.HasDecimals {
		t.Fatalf("unexpected price: %#v", price)
	}
	if price.Symbol != "WETH" {
		t.Fatalf("symbol = %q, want WETH", price.Symbol)
	}
	wantLogo := "https://raw.githubusercontent.com/trustwallet/assets/master/blockchains/ethereum/assets/0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2/logo.png"
	if price.LogoURL != wantLogo {
		t.Fatalf("logo = %q, want %q", price.LogoURL, wantLogo)
	}
}

func TestDefiLlamaCoinIDs(t *testing.T) {
	got := defillamaCoinIDs("arbitrum-one", []string{"0xB", "0xa", "0xB"})
	want := []string{"arbitrum:0xa", "arbitrum:0xb"}
	if len(got) != len(want) {
		t.Fatalf("defillamaCoinIDs length = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("defillamaCoinIDs[%d] = %s, want %s", i, got[i], want[i])
		}
	}
}

func TestChecksumAddress(t *testing.T) {
	got := checksumAddress("0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2")
	want := "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	if got != want {
		t.Fatalf("checksumAddress = %s, want %s", got, want)
	}
}
