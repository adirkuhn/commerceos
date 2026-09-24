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
