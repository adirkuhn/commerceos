package shipping_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appshipping "github.com/adirkuhn/commerceos/application/shipping"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func TestListShipmentTrackingEventsUseCase_listsNewestFirst(t *testing.T) {
	events := []shippingDomain.ShipmentStatusEvent{
		{ID: "evt-2", ShipmentID: "s-1", Status: 5, StatusMain: "enroute", Source: shippingDomain.StatusEventSourceWebhook, OccurredAt: time.Now()},
		{ID: "evt-1", ShipmentID: "s-1", Status: 2, StatusMain: "registered", Source: shippingDomain.StatusEventSourceWebhook, OccurredAt: time.Now().Add(-time.Hour)},
	}
	repo := &fakeShipmentStatusEventRepo{listByShipmentIDFn: func(_ context.Context, shipmentID string) ([]shippingDomain.ShipmentStatusEvent, error) {
		require.Equal(t, "s-1", shipmentID)
		return events, nil
	}}
	uc := appshipping.NewListShipmentTrackingEventsUseCase(newMockGormDB(t, 1), repo)

	result, err := uc.Execute(context.Background(), "s-1")

	require.NoError(t, err)
	require.Len(t, result, 2)
	assert.Equal(t, "evt-2", result[0].ID)
}

func TestListShipmentTrackingEventsUseCase_propagatesRepoError(t *testing.T) {
	repo := &fakeShipmentStatusEventRepo{listByShipmentIDFn: func(context.Context, string) ([]shippingDomain.ShipmentStatusEvent, error) {
		return nil, errors.New("db down")
	}}
	uc := appshipping.NewListShipmentTrackingEventsUseCase(newMockGormDB(t, 1), repo)

	_, err := uc.Execute(context.Background(), "s-1")

	require.Error(t, err)
}
