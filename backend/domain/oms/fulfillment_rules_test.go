package oms_test

import (
	"testing"
	"time"

	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
)

func TestSalesOrderStatus_IsPickable(t *testing.T) {
	for _, s := range []omsDomain.SalesOrderStatus{
		omsDomain.SalesOrderStatusProcessing, omsDomain.SalesOrderStatusOnHold,
	} {
		assert.True(t, s.IsPickable(), s)
	}
	// Pending = payment not received yet (WooCommerce's "Pending payment"):
	// never offered for picking or packing.
	for _, s := range []omsDomain.SalesOrderStatus{
		omsDomain.SalesOrderStatusPending, omsDomain.SalesOrderStatusShipped, omsDomain.SalesOrderStatusCompleted, omsDomain.SalesOrderStatusCancelled,
		omsDomain.SalesOrderStatusRefunded, omsDomain.SalesOrderStatusFailed, omsDomain.SalesOrderStatusPartiallyRefunded,
	} {
		assert.False(t, s.IsPickable(), s)
	}
}

func TestPickableStatuses_matchesIsPickable(t *testing.T) {
	for _, s := range omsDomain.PickableStatuses() {
		assert.True(t, s.IsPickable(), s)
	}
	assert.Len(t, omsDomain.PickableStatuses(), 2)
}

func TestCanStartPacking(t *testing.T) {
	cases := []struct {
		name  string
		order omsDomain.SalesOrder
		want  error
	}{
		{"picked order", omsDomain.SalesOrder{Fulfilled: true, Status: omsDomain.SalesOrderStatusProcessing}, nil},
		{"unpicked order in pickable status skips picking", omsDomain.SalesOrder{Status: omsDomain.SalesOrderStatusProcessing}, nil},
		{"unpaid (pending) order", omsDomain.SalesOrder{Status: omsDomain.SalesOrderStatusPending}, omsDomain.ErrOrderNotPackable},
		{"picked but unpaid (pending) order", omsDomain.SalesOrder{Fulfilled: true, Status: omsDomain.SalesOrderStatusPending}, omsDomain.ErrOrderNotPackable},
		{"already packed", omsDomain.SalesOrder{Fulfilled: true, Packed: true, Status: omsDomain.SalesOrderStatusProcessing}, omsDomain.ErrOrderAlreadyPacked},
		{"unpicked terminal-status order", omsDomain.SalesOrder{Status: omsDomain.SalesOrderStatusCancelled}, omsDomain.ErrOrderNotPackable},
		{"picked but completed", omsDomain.SalesOrder{Fulfilled: true, Status: omsDomain.SalesOrderStatusCompleted}, omsDomain.ErrOrderNotPackable},
		{"picked but shipped", omsDomain.SalesOrder{Fulfilled: true, Status: omsDomain.SalesOrderStatusShipped}, omsDomain.ErrOrderNotPackable},
		{"picked on-hold order", omsDomain.SalesOrder{Fulfilled: true, Status: omsDomain.SalesOrderStatusOnHold}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.order.CanStartPacking(omsDomain.SalesOrderSourceWooCommerce)
			if tc.want == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.want)
			}
		})
	}
}

func TestCanStartPicking(t *testing.T) {
	cases := []struct {
		name              string
		order             omsDomain.SalesOrder
		packingInProgress bool
		want              error
	}{
		{"unfulfilled order", omsDomain.SalesOrder{Status: omsDomain.SalesOrderStatusProcessing}, false, nil},
		{"already fulfilled", omsDomain.SalesOrder{Fulfilled: true}, false, omsDomain.ErrOrderAlreadyFulfilled},
		{"already packed", omsDomain.SalesOrder{Packed: true}, false, omsDomain.ErrOrderAlreadyPacked},
		{"packing session in progress", omsDomain.SalesOrder{Status: omsDomain.SalesOrderStatusProcessing}, true, omsDomain.ErrOrderPackingInProgress},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.order.CanStartPicking(omsDomain.SalesOrderSourceWooCommerce, tc.packingInProgress)
			if tc.want == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.want)
			}
		})
	}
}

func TestPickingSession_HasPendingChanges(t *testing.T) {
	now := time.Now()
	scans := []omsDomain.PickScan{
		{ActionType: omsDomain.PickScanActionPick},
		{ActionType: omsDomain.PickScanActionSubstitute, ReplayedAt: &now},
	}
	assert.False(t, omsDomain.HasPendingPickChanges(scans), "plain picks and already-replayed changes are not pending")

	scans = append(scans, omsDomain.PickScan{ActionType: omsDomain.PickScanActionRefund})
	assert.True(t, omsDomain.HasPendingPickChanges(scans))
}

func TestPickingSession_IsOpen(t *testing.T) {
	assert.True(t, omsDomain.PickingSession{Status: omsDomain.PickingSessionStatusInProgress}.IsOpen())
	assert.True(t, omsDomain.PickingSession{Status: omsDomain.PickingSessionStatusPaused}.IsOpen())
	assert.False(t, omsDomain.PickingSession{Status: omsDomain.PickingSessionStatusCompleted}.IsOpen())
	assert.False(t, omsDomain.PickingSession{Status: omsDomain.PickingSessionStatusAbandoned}.IsOpen())
}

func TestCanMarkPicked(t *testing.T) {
	cases := []struct {
		name              string
		order             omsDomain.SalesOrder
		packingInProgress bool
		want              error
	}{
		{"unpicked processing order", omsDomain.SalesOrder{Status: omsDomain.SalesOrderStatusProcessing}, false, nil},
		{"already picked", omsDomain.SalesOrder{Fulfilled: true, Status: omsDomain.SalesOrderStatusProcessing}, false, omsDomain.ErrOrderAlreadyFulfilled},
		{"packing in progress", omsDomain.SalesOrder{Status: omsDomain.SalesOrderStatusProcessing}, true, omsDomain.ErrOrderPackingInProgress},
		{"cancelled order", omsDomain.SalesOrder{Status: omsDomain.SalesOrderStatusCancelled}, false, omsDomain.ErrOrderNotPickable},
		{"unpaid (pending) order", omsDomain.SalesOrder{Status: omsDomain.SalesOrderStatusPending}, false, omsDomain.ErrOrderNotPickable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.order.CanMarkPicked(omsDomain.SalesOrderSourceWooCommerce, tc.packingInProgress)
			if tc.want == nil {
				assert.NoError(t, err)
			} else {
				assert.ErrorIs(t, err, tc.want)
			}
		})
	}
}

func TestSalesOrderSource_IsFulfillable(t *testing.T) {
	assert.False(t, omsDomain.SalesOrderSourcePOS.IsFulfillable(), "POS sales are handed over at the counter")
	for _, src := range []omsDomain.SalesOrderSource{
		omsDomain.SalesOrderSourceManual, omsDomain.SalesOrderSourceWooCommerce, omsDomain.SalesOrderSourceBol,
		omsDomain.SalesOrderSourceWhatsApp, omsDomain.SalesOrderSourceInternal,
	} {
		assert.True(t, src.IsFulfillable(), src)
	}
}

// A POS sale was handed over at the counter: never picked or packed.
func TestFulfillmentRules_aPOSSaleIsNeverPickedOrPacked(t *testing.T) {
	order := omsDomain.SalesOrder{Status: omsDomain.SalesOrderStatusProcessing}
	assert.ErrorIs(t, order.CanStartPacking(omsDomain.SalesOrderSourcePOS), omsDomain.ErrPOSOrderNotFulfillable)
	assert.ErrorIs(t, order.CanStartPicking(omsDomain.SalesOrderSourcePOS, false), omsDomain.ErrPOSOrderNotFulfillable)
	assert.ErrorIs(t, order.CanMarkPicked(omsDomain.SalesOrderSourcePOS, false), omsDomain.ErrPOSOrderNotFulfillable)
}
