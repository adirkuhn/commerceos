package oms_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func TestLineItem_totalsWhenVATComesOnTop(t *testing.T) {
	// A Woo line: 3 x 4.58 excl. VAT, 1.24 VAT on top.
	li := omsDomain.SalesOrderLineItem{Quantity: 3, TotalMinorUnits: 1373, TotalTaxMinorUnits: 124}
	assert.Equal(t, int64(1373), li.TotalExclTax())
	assert.Equal(t, int64(1497), li.TotalInclTax())
}

func TestLineItem_totalsWhenVATIsInside(t *testing.T) {
	// A manual line: 2 x 1.99 incl. 9% VAT, of which 0.33 VAT.
	li := omsDomain.SalesOrderLineItem{Quantity: 2, TotalMinorUnits: 398, TotalTaxMinorUnits: 33, TotalIncludesTax: true}
	assert.Equal(t, int64(365), li.TotalExclTax())
	assert.Equal(t, int64(398), li.TotalInclTax())
}

func TestLineItem_refundAmountIsItsShareOfTheTotalInclVAT(t *testing.T) {
	li := omsDomain.SalesOrderLineItem{Quantity: 3, TotalMinorUnits: 1373, TotalTaxMinorUnits: 124}
	assert.Equal(t, int64(1497), li.RefundAmount(3))
	assert.Equal(t, int64(499), li.RefundAmount(1), "14.97 / 3")
	assert.Equal(t, int64(998), li.RefundAmount(2), "29.94 / 3 rounded")
}

func TestLineItem_refundAmountOfAnEmptyLineIsZero(t *testing.T) {
	assert.Zero(t, omsDomain.SalesOrderLineItem{}.RefundAmount(1))
}
