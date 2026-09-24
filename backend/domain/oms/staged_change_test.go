package oms_test

import (
	"testing"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPickScan_Change(t *testing.T) {
	li, covered, price, now := "li-1", int64(1), int64(250), time.Now()
	scan := omsDomain.PickScan{
		ID: "s1", LineItemID: &li, SKUID: "sku-2", ActionType: omsDomain.PickScanActionSubstitute, Quantity: 3,
		OriginalQtyCovered: &covered, ReturnToStock: true, PriceMinorUnits: &price, Reason: "out", ScannedAt: now,
	}

	c, ok := scan.Change()

	require.True(t, ok)
	assert.Equal(t, omsDomain.StagedChange{
		ID: "s1", LineItemID: &li, SKUID: "sku-2", Action: omsDomain.ChangeSubstitute, Quantity: 3,
		OriginalQtyCovered: &covered, ReturnToStock: true, PriceMinorUnits: &price, Reason: "out", ScannedAt: now,
	}, c)

	_, ok = omsDomain.PickScan{ActionType: omsDomain.PickScanActionPick}.Change()
	assert.False(t, ok, "a plain pick changes nothing")
}

func TestPickScan_ChangeCoversEveryChangeAction(t *testing.T) {
	for _, a := range []omsDomain.PickScanActionType{
		omsDomain.PickScanActionSubstitute, omsDomain.PickScanActionFreebie, omsDomain.PickScanActionNewItem, omsDomain.PickScanActionRefund,
	} {
		c, ok := omsDomain.PickScan{ActionType: a}.Change()
		assert.True(t, ok, a)
		assert.Equal(t, string(a), string(c.Action))
	}
}

func TestPackScan_Change(t *testing.T) {
	li, price := "li-1", int64(500)
	c, ok := omsDomain.PackScan{ID: "p1", LineItemID: nil, SKUID: "sku-9", ActionType: omsDomain.PackScanActionNewItem, Quantity: 1, PriceMinorUnits: &price, Reason: "Extra jar"}.Change()
	require.True(t, ok)
	assert.Equal(t, omsDomain.ChangeNewItem, c.Action)
	assert.Equal(t, "p1", c.ID)
	assert.Equal(t, &price, c.PriceMinorUnits)
	assert.Equal(t, "Extra jar", c.Reason)

	_, ok = omsDomain.PackScan{LineItemID: &li, ActionType: omsDomain.PackScanActionPack}.Change()
	assert.False(t, ok, "a plain pack changes nothing")
}

func TestPackScan_Covers(t *testing.T) {
	li, covered := "li-1", int64(2)
	cases := []struct {
		name string
		scan omsDomain.PackScan
		qty  int64
	}{
		{"plain pack", omsDomain.PackScan{LineItemID: &li, ActionType: omsDomain.PackScanActionPack, Quantity: 3}, 3},
		{"substitute covers the original quantity, not its own", omsDomain.PackScan{LineItemID: &li, ActionType: omsDomain.PackScanActionSubstitute, Quantity: 8, OriginalQtyCovered: &covered}, 2},
		{"refund resolves its quantity", omsDomain.PackScan{LineItemID: &li, ActionType: omsDomain.PackScanActionRefund, Quantity: 1}, 1},
		{"freebie covers no line", omsDomain.PackScan{ActionType: omsDomain.PackScanActionFreebie, Quantity: 1}, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			id, qty := tc.scan.Covers()
			assert.Equal(t, tc.qty, qty)
			if tc.qty > 0 {
				assert.Equal(t, "li-1", id)
			}
		})
	}
}
