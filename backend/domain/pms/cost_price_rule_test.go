package pms_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func ptr(v int64) *int64 { return &v }

func TestCostPriceAfterReceipt(t *testing.T) {
	cases := []struct {
		name        string
		sku         pmsDomain.SKU
		landed      int64
		fromPO      bool
		wantCost    int64
		wantChanged bool
	}{
		{"purchase order receipt sets the last landed cost",
			pmsDomain.SKU{CostPriceMinorUnits: ptr(440)}, 520, true, 520, true},
		{"purchase order receipt at the same cost changes nothing",
			pmsDomain.SKU{CostPriceMinorUnits: ptr(520)}, 520, true, 0, false},
		{"first purchase order receipt sets an unknown cost",
			pmsDomain.SKU{CostPriceSource: pmsDomain.SKUCostPriceSourceUnknown}, 520, true, 520, true},
		{"a manual cost always wins",
			pmsDomain.SKU{CostPriceMinorUnits: ptr(400), CostPriceSource: pmsDomain.SKUCostPriceSourceManual}, 520, true, 0, false},
		{"a zero-cost receipt never becomes the last cost",
			pmsDomain.SKU{CostPriceMinorUnits: ptr(440)}, 0, true, 0, false},
		{"a zero-cost receipt does not set an unknown cost either",
			pmsDomain.SKU{}, 0, true, 0, false},
		{"a costed receipt without a purchase order leaves a known cost alone",
			pmsDomain.SKU{CostPriceMinorUnits: ptr(440)}, 520, false, 0, false},
		{"a costed receipt without a purchase order fills an unknown cost",
			pmsDomain.SKU{}, 520, false, 520, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, changed := pmsDomain.CostPriceAfterReceipt(tc.sku, tc.landed, tc.fromPO)
			assert.Equal(t, tc.wantChanged, changed)
			if tc.wantChanged {
				assert.Equal(t, tc.wantCost, got)
			}
		})
	}
}

func TestSKUActualCost(t *testing.T) {
	assert.Equal(t, ptr(300), pmsDomain.SKU{InventoryCostMinorUnits: ptr(300), CostPriceMinorUnits: ptr(500)}.ActualCostMinorUnits(),
		"the moving average of stock on hand is the actual cost")
	assert.Equal(t, ptr(500), pmsDomain.SKU{CostPriceMinorUnits: ptr(500)}.ActualCostMinorUnits(),
		"without an average yet, the cost price is the best known cost")
	assert.Nil(t, pmsDomain.SKU{}.ActualCostMinorUnits())
}
