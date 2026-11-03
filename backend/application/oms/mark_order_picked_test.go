package oms_test

import (
	"context"
	"errors"
	"github.com/adirkuhn/commerceos/domain/integrations"
	"testing"
	"time"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type markPickedFixture struct {
	origins    *stubOrigins
	orders     *stubSalesOrderRepo
	packing    *stubPackingSessionRepo
	picking    *stubPickingSessionRepo
	scans      *stubPickScanRepo
	packScans  *stubPackScanRepo
	activities *stubSalesOrderActivityRepo
	outbox     *eventrecordtest.Outbox
}

func newMarkPickedFixture(order omsDomain.SalesOrder) markPickedFixture {
	return markPickedFixture{
		origins: &stubOrigins{},
		orders:  &stubSalesOrderRepo{findByIDResult: &order}, packing: &stubPackingSessionRepo{},
		picking: &stubPickingSessionRepo{}, scans: &stubPickScanRepo{}, packScans: &stubPackScanRepo{}, activities: &stubSalesOrderActivityRepo{},
		outbox: &eventrecordtest.Outbox{},
	}
}

func (f markPickedFixture) run(t *testing.T, actor omsDomain.LockActor) error {
	uc := appoms.NewMarkOrderPickedUseCase(newMockGormDB(t), f.orders, f.origins, f.packing, f.packScans, f.picking, f.scans, f.activities, f.outbox)
	return uc.Execute(context.Background(), "order-1", actor)
}

func TestMarkOrderPicked_marksFulfilledAndLogsActivity(t *testing.T) {
	f := newMarkPickedFixture(omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing})

	err := f.run(t, omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	assert.Equal(t, "order-1", f.orders.gotUpdateID)
	assert.Equal(t, true, f.orders.gotUpdateFields["fulfilled"])
	require.Len(t, f.activities.created, 1)
	a := f.activities.created[0]
	assert.Equal(t, "order-1", a.SalesOrderID)
	assert.Equal(t, "user-1", a.UserID)
	assert.Equal(t, omsDomain.ActivityTypeMarkedPicked, a.ActivityType)
	assert.Equal(t, "Marked as picked", a.Summary)

	require.Equal(t, []string{"oms.sales_order.updated", "oms.sales_order_activity.recorded"}, f.outbox.Subjects())
	assert.Equal(t, map[string][2]any{"fulfilled": {false, true}}, f.outbox.Events[0].Changes())
	assert.Equal(t, map[string]any{"reason": "marked_picked"}, f.outbox.Events[0].Data())
}

func TestMarkOrderPicked_rejectsWhatThePickingRuleRejects(t *testing.T) {
	f := newMarkPickedFixture(omsDomain.SalesOrder{ID: "order-1", Fulfilled: true, Status: omsDomain.SalesOrderStatusProcessing})

	err := f.run(t, omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderAlreadyFulfilled)
	assert.Empty(t, f.orders.gotUpdateID)
	assert.Empty(t, f.activities.created)
}

func TestMarkOrderPicked_rejectsPOSOrder(t *testing.T) {
	f := newMarkPickedFixture(omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing})
	f.origins.source = integrations.OriginPOS

	err := f.run(t, omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrPOSOrderNotFulfillable)
	assert.Empty(t, f.orders.gotUpdateID)
}

func TestMarkOrderPicked_rejectsWhilePacking(t *testing.T) {
	f := newMarkPickedFixture(omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing})
	f.packing.activeResult = &omsDomain.PackingSession{ID: "pack-1", Status: omsDomain.PackingSessionStatusInProgress}
	f.packScans.sessionsWithScans = []string{"pack-1"}

	err := f.run(t, omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderPackingInProgress)
	assert.Empty(t, f.orders.gotUpdateID)
}

func TestMarkOrderPicked_closesAnOpenScanPickingSession(t *testing.T) {
	f := newMarkPickedFixture(omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing})
	f.picking.activeResult = &omsDomain.PickingSession{ID: "picking-1", Status: omsDomain.PickingSessionStatusPaused}

	err := f.run(t, omsDomain.LockActor{UserID: "user-1"})

	require.NoError(t, err)
	require.Len(t, f.picking.updated, 1)
	assert.Equal(t, omsDomain.PickingSessionStatusAbandoned, f.picking.updated[0].Status)
	assert.Equal(t, true, f.orders.gotUpdateFields["fulfilled"])
}

func TestMarkOrderPicked_refusesWhileSomeoneElseIsScanPicking(t *testing.T) {
	f := newMarkPickedFixture(omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing})
	picker, now := "picker", time.Now()
	f.picking.activeResult = &omsDomain.PickingSession{ID: "picking-1", Status: omsDomain.PickingSessionStatusInProgress, LockedBy: &picker, LockedAt: &now}

	err := f.run(t, omsDomain.LockActor{UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrOrderPickingInProgress)
	assert.Empty(t, f.orders.gotUpdateID)
}

func TestMarkOrderPicked_propagatesErrors(t *testing.T) {
	boom := errors.New("boom")

	f := newMarkPickedFixture(omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing})
	f.orders.updateErr = boom
	assert.ErrorIs(t, f.run(t, omsDomain.LockActor{UserID: "u"}), boom)

	f = newMarkPickedFixture(omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing})
	f.activities.createErr = boom
	assert.ErrorIs(t, f.run(t, omsDomain.LockActor{UserID: "u"}), boom)

	f = newMarkPickedFixture(omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing})
	f.packing.activeErr = boom
	assert.ErrorIs(t, f.run(t, omsDomain.LockActor{UserID: "u"}), boom)

	f = newMarkPickedFixture(omsDomain.SalesOrder{})
	f.orders.findByIDResult = nil
	assert.ErrorIs(t, f.run(t, omsDomain.LockActor{UserID: "u"}), omsDomain.ErrSalesOrderNotFound)
}
