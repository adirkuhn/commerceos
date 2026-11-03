package shipping_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appoms "github.com/adirkuhn/commerceos/application/oms"
	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	"github.com/adirkuhn/commerceos/domain/events"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

func TestApplyShipmentFulfillmentSignal_carrierConfirmed_flipsFulfilledAndMovesToShipped(t *testing.T) {
	// Reproduces the real bug: an order fulfilled purely by creating a
	// MyParcel label (no internal picking session ever run, e.g. many Bol
	// orders) has Fulfilled == false forever -- MarkOrderShippedOnFulfillment
	// used to just no-op on it, leaving the order stuck on Processing no
	// matter how many carrier webhooks arrived. "handed_to_carrier" is real
	// evidence the carrier has the parcel -- that must be enough on its own.
	order := &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: false}
	salesOrderRepo := &fakeSalesOrderRepo{findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil }}
	updater := &fakeSalesOrderStatusUpdater{executeFn: func(_ context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error) {
		// Mirrors what the real UpdateSalesOrderUseCase would persist --
		// MarkOrderShippedOnFulfillment's own FindByID (right after this)
		// must see the just-set Fulfilled, not a stale copy.
		if in.Fulfilled != nil {
			order.Fulfilled = *in.Fulfilled
		}
		return order, nil
	}}

	uc := appshipping.NewApplyShipmentFulfillmentSignalUseCase(newMockGormDB(t, 2), salesOrderRepo, updater, nil)

	err := uc.Execute(context.Background(), "order-1", events.ShipmentFulfillmentSignalKindShipmentStatus, "handed_to_carrier", 3)

	require.NoError(t, err)
	require.Len(t, updater.calls, 2)
	require.NotNil(t, updater.calls[0].fulfilled, "must explicitly set Fulfilled -- the manual-override path, since no picking session set it")
	assert.True(t, *updater.calls[0].fulfilled)
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[1].status)
}

func TestApplyShipmentFulfillmentSignal_registeredOnly_doesNotFlipFulfilledOrMoveOrder(t *testing.T) {
	// "registered" alone is just a label existing -- not proof the carrier
	// has the parcel -- so it must not flip Fulfilled or move Status.
	order := &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: false}
	salesOrderRepo := &fakeSalesOrderRepo{findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil }}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewApplyShipmentFulfillmentSignalUseCase(newMockGormDB(t, 1), salesOrderRepo, updater, nil)

	err := uc.Execute(context.Background(), "order-1", events.ShipmentFulfillmentSignalKindShipmentStatus, "registered", 1)

	require.NoError(t, err)
	assert.Empty(t, updater.calls, "registered alone is not carrier-confirmed -- must not touch the order")
}

func TestApplyShipmentFulfillmentSignal_alreadyFulfilled_doesNotReMarkFulfilled(t *testing.T) {
	order := &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusProcessing, Fulfilled: true}
	salesOrderRepo := &fakeSalesOrderRepo{findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil }}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewApplyShipmentFulfillmentSignalUseCase(newMockGormDB(t, 2), salesOrderRepo, updater, nil)

	err := uc.Execute(context.Background(), "order-1", events.ShipmentFulfillmentSignalKindShipmentStatus, "handed_to_carrier", 3)

	require.NoError(t, err)
	require.Len(t, updater.calls, 1, "already Fulfilled -- only the status transition call, no redundant Fulfilled write")
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[0].status)
}

func TestApplyShipmentFulfillmentSignal_alreadyPastProcessing_doesNotReapply(t *testing.T) {
	order := &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusOnHold, Fulfilled: true}
	salesOrderRepo := &fakeSalesOrderRepo{findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil }}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewApplyShipmentFulfillmentSignalUseCase(newMockGormDB(t, 2), salesOrderRepo, updater, nil)

	err := uc.Execute(context.Background(), "order-1", events.ShipmentFulfillmentSignalKindShipmentStatus, "handed_to_carrier", 3)

	require.NoError(t, err)
	assert.Empty(t, updater.calls, "already past Pending/Processing -- must not move status")
}

func TestApplyShipmentFulfillmentSignal_delivered_completesOrder(t *testing.T) {
	order := &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusShipped, Fulfilled: true}
	salesOrderRepo := &fakeSalesOrderRepo{findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil }}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewApplyShipmentFulfillmentSignalUseCase(newMockGormDB(t, 2), salesOrderRepo, updater, nil)

	err := uc.Execute(context.Background(), "order-1", events.ShipmentFulfillmentSignalKindShipmentStatus, "delivered", 7)

	require.NoError(t, err)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, omsDomain.SalesOrderStatusCompleted, updater.calls[0].status)
}

func TestApplyShipmentFulfillmentSignal_orderProcessed_movesToShippedRegardlessOfLocalFulfilled(t *testing.T) {
	// MyParcel's own Orders/Fulfilment API "processed" event is its own
	// unambiguous fulfilled signal -- must not be gated on local Fulfilled.
	order := &omsDomain.SalesOrder{ID: "order-1", Status: omsDomain.SalesOrderStatusPending, Fulfilled: false}
	salesOrderRepo := &fakeSalesOrderRepo{findByIDFn: func(context.Context, string) (*omsDomain.SalesOrder, error) { return order, nil }}
	updater := &fakeSalesOrderStatusUpdater{}

	uc := appshipping.NewApplyShipmentFulfillmentSignalUseCase(newMockGormDB(t, 1), salesOrderRepo, updater, nil)

	err := uc.Execute(context.Background(), "order-1", events.ShipmentFulfillmentSignalKindOrderProcessed, "", 0)

	require.NoError(t, err)
	require.Len(t, updater.calls, 1)
	assert.Equal(t, omsDomain.SalesOrderStatusShipped, updater.calls[0].status)
	assert.Nil(t, updater.calls[0].fulfilled, "order_processed never touches Fulfilled directly -- MarkOrderShippedOnFulfillment's own status write is the only call")
}

func TestApplyShipmentFulfillmentSignal_unknownKind_returnsError(t *testing.T) {
	uc := appshipping.NewApplyShipmentFulfillmentSignalUseCase(newMockGormDB(t, 0), &fakeSalesOrderRepo{}, &fakeSalesOrderStatusUpdater{}, nil)

	err := uc.Execute(context.Background(), "order-1", "some_unknown_kind", "", 0)

	require.Error(t, err)
}
