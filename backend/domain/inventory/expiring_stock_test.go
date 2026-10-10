package inventory_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/inventory"
)

func day(s string) time.Time {
	d, _ := time.Parse("2006-01-02", s)
	return d
}

func TestExpiringStockLine_DaysLeft(t *testing.T) {
	l := inventory.ExpiringStockLine{ExpiryDate: day("2026-10-20")}
	assert.Equal(t, 10, l.DaysLeft(day("2026-10-10")))
	assert.Equal(t, 0, inventory.ExpiringStockLine{ExpiryDate: day("2026-10-10")}.DaysLeft(day("2026-10-10")))
}

func TestExpiringStockLine_ActiveSalePrice(t *testing.T) {
	sale := int64(399)
	today := day("2026-10-10")

	assert.Nil(t, inventory.ExpiringStockLine{}.ActiveSalePrice(today), "no sale")
	assert.Equal(t, &sale, inventory.ExpiringStockLine{SalePriceMinorUnits: &sale}.ActiveSalePrice(today), "open-ended sale")

	ends := day("2026-10-10")
	assert.Equal(t, &sale, inventory.ExpiringStockLine{SalePriceMinorUnits: &sale, SaleEndDate: &ends}.ActiveSalePrice(today), "runs through its end date")

	ended := day("2026-10-09")
	assert.Nil(t, inventory.ExpiringStockLine{SalePriceMinorUnits: &sale, SaleEndDate: &ended}.ActiveSalePrice(today), "ended yesterday")
}
