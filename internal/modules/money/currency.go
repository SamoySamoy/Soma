// Package money keeps accounts, transactions, categories, budgets and exchange
// rates, and turns them into net worth and monthly totals (FIN-01..08).
//
// Amounts are integers in minor units (ADR-006). Exchange rates are integers
// scaled by 10^8 (ADR-016). No floating point is used anywhere in this package.
package money

import (
	"math/big"
	"sort"

	"github.com/SamoySamoy/Soma/internal/platform/apperr"
)

// digits is the number of minor-unit digits per currency (ISO 4217).
var digits = map[string]int{
	"AUD": 2, "CAD": 2, "CHF": 2, "CNY": 2, "EUR": 2, "GBP": 2, "HKD": 2, "JPY": 0,
	"KRW": 0, "KWD": 3, "MYR": 2, "SGD": 2, "THB": 2, "USD": 2, "VND": 0,
}

// rateScale is the factor of rate_e8: a stored rate of 25,000 x 10^8 means one
// unit of the currency is 25,000 units of the base currency.
const rateScale = 100_000_000

// CurrencyInfo describes one supported currency.
type CurrencyInfo struct {
	Code   string
	Digits int
}

// Currencies returns the supported currencies, sorted by code.
func Currencies() []CurrencyInfo {
	out := make([]CurrencyInfo, 0, len(digits))
	for code, d := range digits {
		out = append(out, CurrencyInfo{Code: code, Digits: d})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })
	return out
}

// Supported reports whether code is a currency this build knows the digits of.
func Supported(code string) bool {
	_, ok := digits[code]
	return ok
}

// Convert returns amount, in minor units of from, expressed in minor units of
// to, given rateE8 (one unit of from is rateE8/10^8 units of to, in major units).
// The result is rounded half to even exactly once.
func Convert(amount int64, from, to string, rateE8 int64) (int64, error) {
	if from == to {
		return amount, nil
	}
	dFrom, okFrom := digits[from]
	dTo, okTo := digits[to]
	if !okFrom || !okTo {
		return 0, apperr.Invalid("money.unknown_currency", "Unknown currency.", nil)
	}
	// value_to = amount * rate * 10^dTo / (10^dFrom * 10^8)
	num := new(big.Int).Mul(big.NewInt(amount), big.NewInt(rateE8))
	num.Mul(num, pow10(dTo))
	den := new(big.Int).Mul(pow10(dFrom), big.NewInt(rateScale))
	return roundHalfEven(num, den).Int64(), nil
}

func pow10(n int) *big.Int {
	return new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(n)), nil)
}

// roundHalfEven divides num by den, rounding to the nearest integer and, on a
// tie, to the even one. It is exact for any sign.
func roundHalfEven(num, den *big.Int) *big.Int {
	q, r := new(big.Int).QuoRem(num, den, new(big.Int))
	twiceRem := new(big.Int).Mul(new(big.Int).Abs(r), big.NewInt(2))
	switch twiceRem.Cmp(new(big.Int).Abs(den)) {
	case 1:
		q = step(q, num, den)
	case 0:
		if new(big.Int).Abs(q).Bit(0) == 1 {
			q = step(q, num, den)
		}
	}
	return q
}

// step moves q one away from zero in the direction of the true quotient.
func step(q, num, den *big.Int) *big.Int {
	if num.Sign()*den.Sign() < 0 {
		return q.Sub(q, big.NewInt(1))
	}
	return q.Add(q, big.NewInt(1))
}
