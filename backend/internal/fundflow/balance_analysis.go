package fundflow

import (
	"math"
	"math/big"
	"sort"
	"strings"

	"foundry-tx-simulator/backend/internal/model"
)

type TokenPrice struct {
	PriceUSD    float64
	Decimals    int
	HasDecimals bool
	Symbol      string
	LogoURL     string
}

func CountUSDPrices(prices map[string]TokenPrice) int {
	count := 0
	for _, price := range prices {
		if price.PriceUSD > 0 {
			count++
		}
	}
	return count
}

func AnalyzeBalanceChanges(transfers []model.ERC20Transfer, prices map[string]TokenPrice) *model.BalanceAnalysis {
	if len(transfers) == 0 {
		return nil
	}

	rawChanges := make(map[string]*big.Int)
	for _, transfer := range transfers {
		token := normalizeAddressValue(transfer.Token)
		from := normalizeAddressValue(transfer.From)
		to := normalizeAddressValue(transfer.To)
		amount, ok := new(big.Int).SetString(transfer.Amount, 10)
		if !ok {
			continue
		}

		addRawChange(rawChanges, from, token, new(big.Int).Neg(new(big.Int).Set(amount)))
		addRawChange(rawChanges, to, token, amount)
	}

	keys := make([]string, 0, len(rawChanges))
	for key, amount := range rawChanges {
		if amount.Sign() != 0 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	changes := make([]model.TokenBalanceChange, 0, len(keys))
	userTotals := make(map[string]float64)
	for _, key := range keys {
		user, token, _ := strings.Cut(key, "|")
		rawAmount := rawChanges[key]
		change := model.TokenBalanceChange{
			User:      user,
			Token:     token,
			RawAmount: rawAmount.String(),
			Amount:    rawAmount.String(),
		}

		if price, ok := prices[token]; ok {
			change.Symbol = price.Symbol
			change.LogoURL = price.LogoURL
			if price.HasDecimals {
				change.Amount = formatTokenAmount(rawAmount, price.Decimals)
			}
			priceUSD := price.PriceUSD
			if priceUSD <= 0 && IsStablecoinSymbol(price.Symbol) {
				priceUSD = 1
			}
			if price.HasDecimals && priceUSD > 0 {
				amountFloat, ok := tokenAmountFloat(rawAmount, price.Decimals)
				if !ok {
					changes = append(changes, change)
					continue
				}
				usdValue := amountFloat * priceUSD
				change.USDValue = &usdValue
				userTotals[user] += usdValue
			}
		}
		changes = append(changes, change)
	}

	totals := make([]model.UserUSDChange, 0, len(userTotals))
	for user, usdValue := range userTotals {
		totals = append(totals, model.UserUSDChange{
			User:     user,
			USDValue: usdValue,
		})
	}
	sort.Slice(totals, func(i, j int) bool {
		return totals[i].User < totals[j].User
	})

	return &model.BalanceAnalysis{
		Changes:    changes,
		UserTotals: totals,
	}
}

func addRawChange(changes map[string]*big.Int, user string, token string, amount *big.Int) {
	if user == "" || token == "" || amount.Sign() == 0 {
		return
	}
	key := user + "|" + token
	if _, ok := changes[key]; !ok {
		changes[key] = new(big.Int)
	}
	changes[key].Add(changes[key], amount)
}

func formatTokenAmount(rawAmount *big.Int, decimals int) string {
	if decimals <= 0 {
		return rawAmount.String()
	}

	sign := ""
	abs := new(big.Int).Set(rawAmount)
	if abs.Sign() < 0 {
		sign = "-"
		abs.Abs(abs)
	}

	scale := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	integer := new(big.Int).Quo(abs, scale)
	fraction := new(big.Int).Mod(abs, scale).String()
	if fraction == "0" {
		return sign + integer.String()
	}
	if len(fraction) < decimals {
		fraction = strings.Repeat("0", decimals-len(fraction)) + fraction
	}
	fraction = strings.TrimRight(fraction, "0")
	return sign + integer.String() + "." + fraction
}

func tokenAmountFloat(rawAmount *big.Int, decimals int) (float64, bool) {
	if decimals < 0 {
		return 0, false
	}
	denominator := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	value := new(big.Rat).SetFrac(rawAmount, denominator)
	floatValue, _ := value.Float64()
	return floatValue, !math.IsInf(floatValue, 0) && !math.IsNaN(floatValue)
}

func IsStablecoinSymbol(symbol string) bool {
	switch strings.ToUpper(strings.TrimSpace(symbol)) {
	case "USDC", "USDBC", "USDT", "DAI", "USDE", "USDS", "FRAX", "LUSD", "SUSD", "PYUSD", "GHO", "USDP":
		return true
	default:
		return false
	}
}
