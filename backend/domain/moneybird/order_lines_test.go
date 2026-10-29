package moneybird_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/moneybird"
	"github.com/adirkuhn/commerceos/domain/oms"
)

// Woo-style line: 2 x 3.49 incl. 9% VAT, stored excl. VAT (6.40 + 0.58).
func wooLine(name string) oms.SalesOrderLineItem {
	return oms.SalesOrderLineItem{Name: name, Quantity: 2, TotalMinorUnits: 640, TotalTaxMinorUnits: 58}
}

func TestOrderLines_productLinesInclVAT(t *testing.T) {
	order := oms.SalesOrder{TotalMinorUnits: 698, TaxTotalMinorUnits: 58}

	got, err := moneybird.OrderLines(order, []oms.SalesOrderLineItem{wooLine("Farofa")})

	require.NoError(t, err)
	assert.Equal(t, []moneybird.BookedLine{{Description: "Farofa", Amount: "2", PriceMinorUnits: 349, RateBasisPoints: 906}}, got)
}

// 3 x 1/3 of 10.00 doesn't divide into whole cents: one line for the total.
func TestOrderLines_undivisibleTotalBookedAsOneLine(t *testing.T) {
	line := oms.SalesOrderLineItem{Name: "Carne Seca", Quantity: 3, TotalMinorUnits: 1000, TotalTaxMinorUnits: 210, TotalIncludesTax: true}
	order := oms.SalesOrder{TotalMinorUnits: 1000, TaxTotalMinorUnits: 210}

	got, err := moneybird.OrderLines(order, []oms.SalesOrderLineItem{line})

	require.NoError(t, err)
	assert.Equal(t, []moneybird.BookedLine{{Description: "3 x Carne Seca", Amount: "1", PriceMinorUnits: 1000, RateBasisPoints: 2658}}, got)
}

// Woo: shipping_total is excl. VAT; its VAT is in the order's tax total.
func TestOrderLines_shippingExclVATGetsItsShareOfTheRemainingVAT(t *testing.T) {
	order := oms.SalesOrder{ShippingTotalMinorUnits: 495, ShippingMethodTitle: "PostNL", TotalMinorUnits: 698 + 599, TaxTotalMinorUnits: 58 + 104}

	got, err := moneybird.OrderLines(order, []oms.SalesOrderLineItem{wooLine("Farofa")})

	require.NoError(t, err)
	require.Len(t, got, 2)
	assert.Equal(t, moneybird.BookedLine{Description: "Shipping: PostNL", Amount: "1", PriceMinorUnits: 599, RateBasisPoints: 2101}, got[1])
}

// Manual order: shipping and fee typed incl. VAT; the VAT is already in the totals.
func TestOrderLines_shippingAndFeeInclVAT(t *testing.T) {
	order := oms.SalesOrder{
		ShippingTotalMinorUnits: 605, FeeLabel: "Packaging", FeeTotalMinorUnits: 121,
		TotalMinorUnits: 698 + 605 + 121, TaxTotalMinorUnits: 58 + 105 + 21,
	}

	got, err := moneybird.OrderLines(order, []oms.SalesOrderLineItem{wooLine("Farofa")})

	require.NoError(t, err)
	require.Len(t, got, 3)
	assert.Equal(t, int64(605), got[1].PriceMinorUnits)
	assert.Equal(t, "Packaging", got[2].Description)
	assert.Equal(t, int64(121), got[2].PriceMinorUnits)
	assert.InDelta(t, 2100, got[1].RateBasisPoints, 10)
	assert.InDelta(t, 2100, got[2].RateBasisPoints, 30)
}

// The booked lines must add up to exactly what the customer was charged.
func TestOrderLines_refusesWhenLinesDontAddUpToTheOrderTotal(t *testing.T) {
	order := oms.SalesOrder{TotalMinorUnits: 900, TaxTotalMinorUnits: 58}

	_, err := moneybird.OrderLines(order, []oms.SalesOrderLineItem{wooLine("Farofa")})

	assert.True(t, errors.Is(err, moneybird.ErrTotalsDontAddUp))
}

func TestOrderLines_nothingToBook(t *testing.T) {
	_, err := moneybird.OrderLines(oms.SalesOrder{}, nil)
	assert.True(t, errors.Is(err, moneybird.ErrNothingToBook))
}

func TestQuoteLine(t *testing.T) {
	assert.Equal(t, moneybird.BookedLine{Description: "Farofa", Amount: "4", PriceMinorUnits: 349, RateBasisPoints: 900},
		moneybird.QuoteLine("Farofa", 4, 349, 900))
}
