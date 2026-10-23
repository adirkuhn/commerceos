package pms_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func TestSKUActiveSalePrice(t *testing.T) {
	today := time.Date(2026, 10, 12, 15, 0, 0, 0, time.UTC)
	sale := int64(399)
	endsToday := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	ended := time.Date(2026, 10, 11, 0, 0, 0, 0, time.UTC)

	assert.Equal(t, &sale, pmsDomain.SKU{SalePriceMinorUnits: &sale}.ActiveSalePrice(today), "open-ended sale")
	assert.Equal(t, &sale, pmsDomain.SKU{SalePriceMinorUnits: &sale, SaleEndDate: &endsToday}.ActiveSalePrice(today), "end date is inclusive")
	assert.Nil(t, pmsDomain.SKU{SalePriceMinorUnits: &sale, SaleEndDate: &ended}.ActiveSalePrice(today), "sale already over")
	assert.Nil(t, pmsDomain.SKU{}.ActiveSalePrice(today), "no sale")
}
