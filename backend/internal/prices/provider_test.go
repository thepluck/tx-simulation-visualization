package prices

import (
	"context"
	"testing"

	"tx-simulation-visualization/backend/internal/fundflow"
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

type fakeProvider map[string]fundflow.TokenPrice

func (p fakeProvider) Fetch(_ context.Context, _ string, _ []string) (map[string]fundflow.TokenPrice, error) {
	return map[string]fundflow.TokenPrice(p), nil
}
