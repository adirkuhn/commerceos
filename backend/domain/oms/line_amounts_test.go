package oms_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func TestPriceLine_taxInsidePrices(t *testing.T) {
	// 10 x 12.10 incl. 21% VAT = 121.00, of which 21.00 VAT.
	got := omsDomain.PriceLine(10, 1210, 2100, true)
	assert.Equal(t, omsDomain.LineAmounts{Subtotal: 12100, SubtotalTax: 2100, Total: 12100, TotalTax: 2100}, got)
}

func TestPriceLine_taxOnTopOfPrices(t *testing.T) {
	// 3 x 9.99 excl. 9% = 29.97 + 2.70 VAT (2.6973 rounded).
	got := omsDomain.PriceLine(3, 999, 900, false)
	assert.Equal(t, omsDomain.LineAmounts{Subtotal: 2997, SubtotalTax: 270, Total: 2997, TotalTax: 270}, got)
}

func TestPriceLine_noRate(t *testing.T) {
	assert.Equal(t, omsDomain.LineAmounts{Subtotal: 500, Total: 500}, omsDomain.PriceLine(1, 500, 0, true))
}

func TestOrderTotals(t *testing.T) {
	lines := []omsDomain.LineAmounts{omsDomain.PriceLine(10, 1210, 2100, true), omsDomain.PriceLine(1, 500, 0, true)}
	total, tax := omsDomain.OrderTotals(lines, true)
	assert.Equal(t, int64(12600), total, "inclusive prices: the lines already carry the tax")
	assert.Equal(t, int64(2100), tax)

	lines = []omsDomain.LineAmounts{omsDomain.PriceLine(3, 999, 900, false)}
	total, tax = omsDomain.OrderTotals(lines, false)
	assert.Equal(t, int64(3267), total, "exclusive prices: tax is added on top")
	assert.Equal(t, int64(270), tax)
}
