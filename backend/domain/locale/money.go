package locale

import (
	"strconv"
	"strings"
)

var currencySymbols = map[string]string{"EUR": "€", "USD": "$", "GBP": "£", "BRL": "R$"}

// FormatMoney renders an amount in minor units (cents) for display in this
// language, e.g. €1,234.50 (en) or € 1.234,50 (nl, pt-BR). Currencies
// without a known symbol show their ISO code.
func (l Locale) FormatMoney(minorUnits int64, currencyCode string) string {
	sign := ""
	if minorUnits < 0 {
		sign = "-"
		minorUnits = -minorUnits
	}
	symbol, ok := currencySymbols[currencyCode]
	if !ok {
		symbol = currencyCode
	}

	thousands, decimal, gap := ",", ".", ""
	if l.OrDefault() != English {
		thousands, decimal, gap = ".", ",", " "
	}
	if !ok {
		gap = " "
	}

	whole := strconv.FormatInt(minorUnits/100, 10)
	cents := minorUnits % 100
	return sign + symbol + gap + groupThousands(whole, thousands) + decimal + twoDigits(cents)
}

func groupThousands(digits, sep string) string {
	var b strings.Builder
	for i, d := range digits {
		if i > 0 && (len(digits)-i)%3 == 0 {
			b.WriteString(sep)
		}
		b.WriteRune(d)
	}
	return b.String()
}

func twoDigits(n int64) string {
	if n < 10 {
		return "0" + strconv.FormatInt(n, 10)
	}
	return strconv.FormatInt(n, 10)
}
