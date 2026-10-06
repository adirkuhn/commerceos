package oms_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordSalesOrderActivity_createsEntryWithMetadata(t *testing.T) {
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "so-1"}}
	activityRepo := &stubSalesOrderActivityRepo{}
	outbox := &eventrecordtest.Outbox{}
	uc := appoms.NewRecordSalesOrderActivityUseCase(newMockGormDB(t), salesOrderRepo, activityRepo, outbox)

	activity, err := uc.Execute(context.Background(), appoms.RecordSalesOrderActivityInput{
		SalesOrderID: "so-1", UserID: "user-1", ActivityType: "myparcel_shipment_created",
		Summary: "Shipment created via MyParcel", Metadata: map[string]any{"barcode": "3SMYPA1"},
	})

	require.NoError(t, err)
	require.Len(t, activityRepo.created, 1)
	assert.Equal(t, "so-1", activity.SalesOrderID)
	assert.Equal(t, "user-1", activity.UserID)
	assert.Equal(t, "myparcel_shipment_created", activity.ActivityType)

	var metadata map[string]any
	require.NoError(t, json.Unmarshal(activity.Metadata, &metadata))
	assert.Equal(t, "3SMYPA1", metadata["barcode"])

	assert.Contains(t, outbox.Subjects(), "oms.sales_order_activity.recorded")

}

func TestRecordSalesOrderActivity_nilMetadataRecordsEmptyObjectNotNull(t *testing.T) {
	salesOrderRepo := &stubSalesOrderRepo{findByIDResult: &omsDomain.SalesOrder{ID: "so-1"}}
	activityRepo := &stubSalesOrderActivityRepo{}
	uc := appoms.NewRecordSalesOrderActivityUseCase(newMockGormDB(t), salesOrderRepo, activityRepo, &eventrecordtest.Outbox{})

	activity, err := uc.Execute(context.Background(), appoms.RecordSalesOrderActivityInput{
		SalesOrderID: "so-1", UserID: "user-1", ActivityType: "myparcel_error", Summary: "failed",
	})

	require.NoError(t, err)
	assert.Equal(t, "{}", string(activity.Metadata))
}

func TestRecordSalesOrderActivity_propagatesSalesOrderNotFound(t *testing.T) {
	uc := appoms.NewRecordSalesOrderActivityUseCase(newMockGormDB(t), &stubSalesOrderRepo{}, &stubSalesOrderActivityRepo{}, &eventrecordtest.Outbox{})

	_, err := uc.Execute(context.Background(), appoms.RecordSalesOrderActivityInput{SalesOrderID: "missing", UserID: "user-1"})

	assert.ErrorIs(t, err, omsDomain.ErrSalesOrderNotFound)
}
