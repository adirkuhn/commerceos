package inventory_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appinv "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
)

func onlyEvent(t *testing.T, o *eventrecordtest.Outbox) eventrecordtest.Event {
	t.Helper()
	e, err := o.Only()
	require.NoError(t, err)
	return e
}

func TestCreateWarehouse_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appinv.NewCreateWarehouseUseCase(newMockGormDB(t), &stubWarehouseRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appinv.CreateWarehouseInput{Name: "Main", Country: "NL", Active: true})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "inventory.warehouse.created", e.Subject)
	assert.Equal(t, "warehouse-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"name": "Main", "country": "NL", "active": true}, e.Data())
}

func TestUpdateWarehouse_RecordsChangedFields(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubWarehouseRepo{findResult: &invDomain.Warehouse{ID: "warehouse-1", Name: "Main", City: "Utrecht", Active: true}}
	uc := appinv.NewUpdateWarehouseUseCase(newMockGormDB(t), repo, outbox)

	_, err := uc.Execute(context.Background(), appinv.UpdateWarehouseInput{ID: "warehouse-1", Name: "Main", City: "Amsterdam", Active: false})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "inventory.warehouse.updated", e.Subject)
	assert.Equal(t, map[string][2]any{"city": {"Utrecht", "Amsterdam"}, "active": {true, false}}, e.Changes())
}

func TestDeleteWarehouse_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubWarehouseRepo{findResult: &invDomain.Warehouse{ID: "warehouse-1", Name: "Main"}}
	uc := appinv.NewDeleteWarehouseUseCase(newMockGormDB(t), repo, outbox)

	require.NoError(t, uc.Execute(context.Background(), "warehouse-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "inventory.warehouse.deleted", e.Subject)
	assert.Equal(t, map[string]any{"name": "Main"}, e.Data())
}

func TestCreateBatch_RecordsCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	expiry := time.Date(2027, 1, 31, 0, 0, 0, 0, time.UTC)
	uc := appinv.NewCreateBatchUseCase(newMockGormDB(t), &stubBatchRepo{}, outbox)

	_, err := uc.Execute(context.Background(), appinv.CreateBatchInput{SKUID: "sku-1", WarehouseID: "wh-1", LotNumber: "L1", ExpiryDate: &expiry})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "inventory.batch.created", e.Subject)
	assert.Equal(t, "batch-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"sku_id": "sku-1", "warehouse_id": "wh-1", "lot_number": "L1", "expiry_date": "2027-01-31T00:00:00Z"}, e.Data())
}

func TestDeleteBatch_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-1", LotNumber: "L1"}}
	uc := appinv.NewDeleteBatchUseCase(newMockGormDB(t), repo, &stubStockMovementRepo{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "batch-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "inventory.batch.deleted", e.Subject)
	assert.Equal(t, map[string]any{"sku_id": "sku-1", "warehouse_id": "wh-1", "lot_number": "L1"}, e.Data())
}

func TestDismissExpiredBatch_RecordsReviewed(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-1"}}
	uc := appinv.NewDismissExpiredBatchUseCase(newMockGormDB(t), repo, outbox)

	require.NoError(t, uc.Execute(context.Background(), "batch-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "inventory.batch.reviewed", e.Subject)
	assert.Equal(t, "batch-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"sku_id": "sku-1", "warehouse_id": "wh-1", "outcome": "dismissed"}, e.Data())
}

func TestWriteOffExpiredBatch_RecordsReviewed(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-1"}}
	movements := &stubStockMovementRepo{byBatch: []invDomain.StockMovement{{QuantityDelta: 4}}}
	uc := appinv.NewWriteOffExpiredBatchUseCase(newMockGormDBN(t, 2), repo, movements, &stubDamagedOrLostRecorder{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "batch-1", "user-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "inventory.batch.reviewed", e.Subject)
	assert.Equal(t, "written_off", e.Data()["outcome"])
}

func TestDiscardBatch_RecordsReviewed(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	repo := &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-1"}}
	uc := appinv.NewDiscardBatchUseCase(newMockGormDB(t), repo, &stubDamagedOrLostRecorder{}, &stubFileStorage{}, outbox)
	batchID := "batch-1"

	_, err := uc.Execute(context.Background(), appinv.DiscardBatchInput{SKUID: "sku-1", WarehouseID: "wh-1", BatchID: &batchID, Quantity: 2})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "inventory.batch.reviewed", e.Subject)
	assert.Equal(t, "discarded", e.Data()["outcome"])
}

func TestInventoryUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	ctx := context.Background()
	wh := func() *stubWarehouseRepo {
		return &stubWarehouseRepo{findResult: &invDomain.Warehouse{ID: "warehouse-1"}}
	}
	batch := func() *stubBatchRepo { return &stubBatchRepo{findResult: &invDomain.Batch{ID: "batch-1"}} }

	_, err := appinv.NewCreateWarehouseUseCase(newMockGormDB(t), wh(), failing).Execute(ctx, appinv.CreateWarehouseInput{Name: "x"})
	assert.ErrorIs(t, err, boom, "create warehouse")
	_, err = appinv.NewUpdateWarehouseUseCase(newMockGormDB(t), wh(), failing).Execute(ctx, appinv.UpdateWarehouseInput{ID: "warehouse-1", Name: "y"})
	assert.ErrorIs(t, err, boom, "update warehouse")
	assert.ErrorIs(t, appinv.NewDeleteWarehouseUseCase(newMockGormDB(t), wh(), failing).Execute(ctx, "warehouse-1"), boom, "delete warehouse")
	_, err = appinv.NewCreateBatchUseCase(newMockGormDB(t), batch(), failing).Execute(ctx, appinv.CreateBatchInput{SKUID: "s"})
	assert.ErrorIs(t, err, boom, "create batch")
	assert.ErrorIs(t, appinv.NewDeleteBatchUseCase(newMockGormDB(t), batch(), &stubStockMovementRepo{}, failing).Execute(ctx, "batch-1"), boom, "delete batch")
	assert.ErrorIs(t, appinv.NewDismissExpiredBatchUseCase(newMockGormDB(t), batch(), failing).Execute(ctx, "batch-1"), boom, "dismiss")
}
