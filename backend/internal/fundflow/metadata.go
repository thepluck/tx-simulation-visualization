package fundflow

import (
	"math/big"

	"foundry-tx-simulator/backend/internal/model"
)

func EnrichERC20Transfers(transfers []model.ERC20Transfer, prices map[string]TokenPrice) []model.ERC20Transfer {
	enriched := make([]model.ERC20Transfer, len(transfers))
	for i, transfer := range transfers {
		enriched[i] = transfer
		price, ok := prices[normalizeAddressValue(transfer.Token)]
		if !ok {
			continue
		}

		enriched[i].Symbol = price.Symbol
		enriched[i].LogoURL = price.LogoURL
		amount, ok := new(big.Int).SetString(transfer.Amount, 10)
		if ok && price.HasDecimals {
			enriched[i].NormalizedAmount = formatTokenAmount(amount, price.Decimals)
		}
	}
	return enriched
}
