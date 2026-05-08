package fundflow

import (
	"fmt"
	"testing"

	"tx-simulation-visualization/backend/internal/model"
)

func TestAnalyzeBalanceChanges(t *testing.T) {
	analysis := AnalyzeBalanceChanges(
		[]model.ERC20Transfer{
			{
				Token:  "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
				From:   "0x0000000000000000000000000000000000000001",
				To:     "0x0000000000000000000000000000000000000003",
				Amount: "1000000000000000000",
			},
		},
		map[string]TokenPrice{
			"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
				PriceUSD:    2000,
				Decimals:    18,
				HasDecimals: true,
				Symbol:      "WETH",
				LogoURL:     "https://example.com/weth.png",
			},
		},
	)

	if analysis == nil {
		t.Fatal("expected analysis")
	}
	if len(analysis.Changes) != 2 {
		t.Fatalf("changes = %#v, want 2", analysis.Changes)
	}

	wantByUser := map[string]struct {
		raw string
		amt string
		usd string
	}{
		"0x0000000000000000000000000000000000000001": {raw: "-1000000000000000000", amt: "-1", usd: "-2000"},
		"0x0000000000000000000000000000000000000003": {raw: "1000000000000000000", amt: "1", usd: "2000"},
	}
	for _, change := range analysis.Changes {
		want, ok := wantByUser[change.User]
		if !ok {
			t.Fatalf("unexpected user change: %#v", change)
		}
		if change.RawAmount != want.raw || change.Amount != want.amt {
			t.Fatalf("change = %#v, want raw %s amount %s", change, want.raw, want.amt)
		}
		if change.Symbol != "WETH" || change.LogoURL != "https://example.com/weth.png" {
			t.Fatalf("token metadata = %q %q", change.Symbol, change.LogoURL)
		}
		if change.USDValue == nil || fmt.Sprintf("%.0f", *change.USDValue) != want.usd {
			t.Fatalf("usd = %#v, want %s", change.USDValue, want.usd)
		}
	}

	if len(analysis.UserTotals) != 2 {
		t.Fatalf("user totals = %#v, want 2", analysis.UserTotals)
	}
}

func TestAnalyzeBalanceChangesWithoutPrice(t *testing.T) {
	analysis := AnalyzeBalanceChanges(
		[]model.ERC20Transfer{
			{
				Token:  "0xToken",
				From:   "0xFrom",
				To:     "0xTo",
				Amount: "123",
			},
		},
		nil,
	)

	if analysis == nil || len(analysis.Changes) != 2 {
		t.Fatalf("analysis = %#v", analysis)
	}
	if analysis.Changes[0].USDValue != nil || len(analysis.UserTotals) != 0 {
		t.Fatalf("unexpected priced analysis: %#v", analysis)
	}
}

func TestAnalyzeBalanceChangesRequiresDecimalsForUSD(t *testing.T) {
	analysis := AnalyzeBalanceChanges(
		[]model.ERC20Transfer{
			{
				Token:  "0xToken",
				From:   "0xFrom",
				To:     "0xTo",
				Amount: "1000000",
			},
		},
		map[string]TokenPrice{
			"0xtoken": {
				PriceUSD: 1,
				Symbol:   "USDC",
			},
		},
	)

	if analysis == nil || len(analysis.Changes) != 2 {
		t.Fatalf("analysis = %#v", analysis)
	}
	for _, change := range analysis.Changes {
		if change.Amount != change.RawAmount {
			t.Fatalf("amount should stay raw without decimals: %#v", change)
		}
		if change.USDValue != nil {
			t.Fatalf("usd value should be omitted without decimals: %#v", change)
		}
	}
	if len(analysis.UserTotals) != 0 {
		t.Fatalf("user totals = %#v, want empty", analysis.UserTotals)
	}
}

func TestAnalyzeBalanceChangesUsesNonExactDecimalAmountForUSD(t *testing.T) {
	analysis := AnalyzeBalanceChanges(
		[]model.ERC20Transfer{
			{
				Token:  "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
				From:   "0x0000000000000000000000000000000000000001",
				To:     "0x0000000000000000000000000000000000000002",
				Amount: "1161185423",
			},
		},
		map[string]TokenPrice{
			"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913": {
				PriceUSD:    0.9998,
				Decimals:    6,
				HasDecimals: true,
				Symbol:      "USDC",
			},
		},
	)

	if analysis == nil || len(analysis.Changes) != 2 {
		t.Fatalf("analysis = %#v", analysis)
	}
	for _, change := range analysis.Changes {
		if change.USDValue == nil {
			t.Fatalf("expected usd value for non-exact decimal amount: %#v", change)
		}
	}
}

func TestAnalyzeBalanceChangesUsesStablecoinFallback(t *testing.T) {
	analysis := AnalyzeBalanceChanges(
		[]model.ERC20Transfer{
			{
				Token:  "0x833589fcd6edb6e08f4c7c32d4f71b54bda02913",
				From:   "0x0000000000000000000000000000000000000001",
				To:     "0x0000000000000000000000000000000000000002",
				Amount: "1161185423",
			},
		},
		map[string]TokenPrice{
			"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913": {
				Decimals:    6,
				HasDecimals: true,
				Symbol:      "USDC",
			},
		},
	)

	if analysis == nil || len(analysis.Changes) != 2 {
		t.Fatalf("analysis = %#v", analysis)
	}
	for _, change := range analysis.Changes {
		if change.USDValue == nil {
			t.Fatalf("expected stablecoin fallback usd value: %#v", change)
		}
	}
	if len(analysis.UserTotals) != 2 {
		t.Fatalf("user totals = %#v, want 2", analysis.UserTotals)
	}
}

func TestEnrichERC20Transfers(t *testing.T) {
	transfers := EnrichERC20Transfers(
		[]model.ERC20Transfer{
			{
				Token:  "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
				From:   "0x0000000000000000000000000000000000000001",
				To:     "0x0000000000000000000000000000000000000003",
				Amount: "1000000000000000000",
			},
		},
		map[string]TokenPrice{
			"0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": {
				Decimals:    18,
				HasDecimals: true,
				Symbol:      "WETH",
				LogoURL:     "https://example.com/weth.png",
			},
		},
	)

	got := transfers[0]
	if got.NormalizedAmount != "1" || got.Symbol != "WETH" || got.LogoURL != "https://example.com/weth.png" {
		t.Fatalf("enriched transfer = %#v", got)
	}
}
