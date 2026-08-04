package inventory_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	inventoryhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/inventory"
)

func init() { gin.SetMode(gin.TestMode) }

type stubListWarehouses struct {
	out []invDomain.Warehouse
	err error
}

func (s *stubListWarehouses) Execute(_ context.Context) ([]invDomain.Warehouse, error) {
	return s.out, s.err
}

type stubCreateWarehouse struct {
	out      *invDomain.Warehouse
	err      error
	gotInput appinventory.CreateWarehouseInput
}

func (s *stubCreateWarehouse) Execute(_ context.Context, in appinventory.CreateWarehouseInput) (*invDomain.Warehouse, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &invDomain.Warehouse{ID: "w1", Name: in.Name}, nil
}

type stubUpdateWarehouse struct {
	out      *invDomain.Warehouse
	err      error
	gotInput appinventory.UpdateWarehouseInput
}

func (s *stubUpdateWarehouse) Execute(_ context.Context, in appinventory.UpdateWarehouseInput) (*invDomain.Warehouse, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &invDomain.Warehouse{ID: in.ID, Name: in.Name}, nil
}

type stubDeleteWarehouse struct{ err error }

func (s *stubDeleteWarehouse) Execute(_ context.Context, _ string) error { return s.err }

type stubCreateBatch struct {
	out      *invDomain.Batch
	err      error
	gotInput appinventory.CreateBatchInput
}

func (s *stubCreateBatch) Execute(_ context.Context, in appinventory.CreateBatchInput) (*invDomain.Batch, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &invDomain.Batch{ID: "batch-1", SKUID: in.SKUID, WarehouseID: in.WarehouseID}, nil
}

type stubUpdateBatch struct {
	out      *invDomain.Batch
	err      error
	gotInput appinventory.UpdateBatchInput
}

func (s *stubUpdateBatch) Execute(_ context.Context, in appinventory.UpdateBatchInput) (*invDomain.Batch, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &invDomain.Batch{ID: in.ID, WarehouseID: in.WarehouseID}, nil
}

type stubDeleteBatch struct{ err error }

func (s *stubDeleteBatch) Execute(_ context.Context, _ string) error { return s.err }

type stubListBatches struct {
	out []invDomain.Batch
	err error
}

func (s *stubListBatches) Execute(_ context.Context, _ string) ([]invDomain.Batch, error) {
	return s.out, s.err
}

type stubGetSKUStockLevel struct {
	out *appinventory.StockLevel
	err error
}

func (s *stubGetSKUStockLevel) Execute(_ context.Context, skuID string) (*appinventory.StockLevel, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appinventory.StockLevel{SKUID: skuID}, nil
}

type stubRecordStockMovement struct {
	out      *invDomain.StockMovement
	err      error
	gotInput appinventory.RecordStockMovementInput
}

func (s *stubRecordStockMovement) Execute(_ context.Context, in appinventory.RecordStockMovementInput) (*invDomain.StockMovement, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &invDomain.StockMovement{ID: "movement-1", SKUID: in.SKUID, Type: in.Type, QuantityDelta: in.QuantityDelta}, nil
}

type stubRecordRecount struct {
	out      *invDomain.StockMovement
	err      error
	gotInput appinventory.RecordRecountInput
}

func (s *stubRecordRecount) Execute(_ context.Context, in appinventory.RecordRecountInput) (*invDomain.StockMovement, error) {
	s.gotInput = in
	return s.out, s.err
}

type stubTransferStock struct {
	out      *appinventory.TransferStockResult
	err      error
	gotInput appinventory.TransferStockInput
}

func (s *stubTransferStock) Execute(_ context.Context, in appinventory.TransferStockInput) (*appinventory.TransferStockResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appinventory.TransferStockResult{
		Out: &invDomain.StockMovement{ID: "movement-out", Type: invDomain.MovementTypeTransferOut, QuantityDelta: -in.Quantity},
		In:  &invDomain.StockMovement{ID: "movement-in", Type: invDomain.MovementTypeTransferIn, QuantityDelta: in.Quantity},
	}, nil
}

type stubMarkDamagedOrLost struct {
	out      *invDomain.StockMovement
	err      error
	gotInput appinventory.MarkDamagedOrLostInput
}

func (s *stubMarkDamagedOrLost) Execute(_ context.Context, in appinventory.MarkDamagedOrLostInput) (*invDomain.StockMovement, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &invDomain.StockMovement{ID: "movement-1", Type: in.Type, QuantityDelta: -in.Quantity}, nil
}

func newTestHandler(list *stubListWarehouses, create *stubCreateWarehouse, update *stubUpdateWarehouse, del *stubDeleteWarehouse) *inventoryhandler.Handler {
	return inventoryhandler.NewHandler(
		list, create, update, del,
		&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{},
	)
}

// newBatchTestHandler builds a Handler for the batch/movement routes
// specifically — every other use case is a harmless zero-value stub.
func newBatchTestHandler(
	createBatch *stubCreateBatch, updateBatch *stubUpdateBatch, deleteBatch *stubDeleteBatch, listBatches *stubListBatches,
	stockLevel *stubGetSKUStockLevel, recordMovement *stubRecordStockMovement, recordRecount *stubRecordRecount,
	transfer *stubTransferStock, markDamagedOrLost *stubMarkDamagedOrLost,
) *inventoryhandler.Handler {
	return inventoryhandler.NewHandler(
		&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{},
		createBatch, updateBatch, deleteBatch, listBatches,
		stockLevel, recordMovement, recordRecount, transfer, markDamagedOrLost,
	)
}

func doRequest(rg *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func newTestRouter(h *inventoryhandler.Handler) *gin.Engine {
	r := gin.New()
	geninventory.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func TestListWarehousesHandler_returnsWarehouses(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{out: []invDomain.Warehouse{{ID: "w1", Name: "Main DC"}}}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/inventory/warehouses", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Main DC")
}

func TestCreateWarehouseHandler_success(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/warehouses", map[string]string{"name": "Main DC"})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "Main DC")
}

func TestCreateWarehouseHandler_passesAddressFieldsAndDefaultsActiveTrue(t *testing.T) {
	create := &stubCreateWarehouse{}
	h := newTestHandler(&stubListWarehouses{}, create, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	body := map[string]any{"name": "Main DC", "address1": "1 Main St", "city": "Springfield", "country": "US"}
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/warehouses", body)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "1 Main St", create.gotInput.Address1)
	require.Equal(t, "Springfield", create.gotInput.City)
	require.Equal(t, "US", create.gotInput.Country)
	require.True(t, create.gotInput.Active)
}

func TestCreateWarehouseHandler_rejectsMissingName(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/warehouses", map[string]string{})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdateWarehouseHandler_success(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/inventory/warehouses/w1", map[string]string{"name": "Renamed DC"})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Renamed DC")
}

func TestUpdateWarehouseHandler_passesAddressFieldsAndActive(t *testing.T) {
	update := &stubUpdateWarehouse{}
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, update, &stubDeleteWarehouse{})
	body := map[string]any{"name": "Main DC", "province": "IL", "zip": "62701", "phone": "+1 555-0100", "active": false}
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/inventory/warehouses/w1", body)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "IL", update.gotInput.Province)
	require.Equal(t, "62701", update.gotInput.Zip)
	require.Equal(t, "+1 555-0100", update.gotInput.Phone)
	require.False(t, update.gotInput.Active)
}

func TestUpdateWarehouseHandler_notFound(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{err: invDomain.ErrWarehouseNotFound}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/inventory/warehouses/missing", map[string]string{"name": "X"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteWarehouseHandler_success(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/inventory/warehouses/w1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteWarehouseHandler_notFound(t *testing.T) {
	h := newTestHandler(&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{err: invDomain.ErrWarehouseNotFound})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/inventory/warehouses/missing", nil)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestListBatchesHandler_returnsBatches(t *testing.T) {
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{},
		&stubListBatches{out: []invDomain.Batch{{ID: "batch-1", LotNumber: "LOT-1"}}},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/inventory/skus/sku-1/batches", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "LOT-1")
}

func TestCreateBatchHandler_success(t *testing.T) {
	create := &stubCreateBatch{}
	h := newBatchTestHandler(create, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/batches", map[string]string{"warehouse_id": "wh-1", "lot_number": "LOT-9"})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "sku-1", create.gotInput.SKUID)
	require.Equal(t, "wh-1", create.gotInput.WarehouseID)
	require.Equal(t, "LOT-9", create.gotInput.LotNumber)
}

func TestCreateBatchHandler_withExpiryDate(t *testing.T) {
	create := &stubCreateBatch{}
	h := newBatchTestHandler(create, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/batches", map[string]string{"warehouse_id": "wh-1", "expiry_date": "2027-01-15"})

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, create.gotInput.ExpiryDate)
	require.Equal(t, 2027, create.gotInput.ExpiryDate.Year())
}

func TestUpdateBatchHandler_success(t *testing.T) {
	update := &stubUpdateBatch{}
	h := newBatchTestHandler(&stubCreateBatch{}, update, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/inventory/batches/batch-1", map[string]string{"warehouse_id": "wh-2", "lot_number": "LOT-2"})

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "batch-1", update.gotInput.ID)
	require.Equal(t, "wh-2", update.gotInput.WarehouseID)
}

func TestUpdateBatchHandler_notFound(t *testing.T) {
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{err: invDomain.ErrBatchNotFound}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/inventory/batches/missing", map[string]string{"warehouse_id": "wh-1"})

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteBatchHandler_success(t *testing.T) {
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/inventory/batches/batch-1", nil)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteBatchHandler_conflictWhenHasMovements(t *testing.T) {
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{err: invDomain.ErrBatchHasMovements}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/inventory/batches/batch-1", nil)

	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestGetSKUStockLevelHandler_returnsLevel(t *testing.T) {
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{out: &appinventory.StockLevel{SKUID: "sku-1", Total: 20, Syncable: 15}},
		&stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/inventory/skus/sku-1/stock-level", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"total":20`)
	assert.Contains(t, w.Body.String(), `"syncable":15`)
}

func TestRecordStockMovementHandler_success(t *testing.T) {
	record := &stubRecordStockMovement{}
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, record, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/movements",
		map[string]any{"warehouse_id": "wh-1", "type": "adjustment", "quantity_delta": -3})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "sku-1", record.gotInput.SKUID)
	require.Equal(t, invDomain.MovementTypeAdjustment, record.gotInput.Type)
	require.Equal(t, int64(-3), record.gotInput.QuantityDelta)
}

func TestRecordStockMovementHandler_rejectsInvalidQuantity(t *testing.T) {
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{err: invDomain.ErrInvalidQuantity}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/movements",
		map[string]any{"warehouse_id": "wh-1", "type": "receipt", "quantity_delta": -3})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestRecordRecountHandler_returnsNullDataWhenNothingChanged(t *testing.T) {
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{out: nil}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/recount",
		map[string]any{"warehouse_id": "wh-1", "counted_quantity": 10})

	require.Equal(t, http.StatusOK, w.Code)
	assert.NotContains(t, w.Body.String(), "movement-1", "no movement was created, so nothing should be echoed back")
}

func TestRecordRecountHandler_returnsMovementWhenAdjusted(t *testing.T) {
	recount := &stubRecordRecount{out: &invDomain.StockMovement{ID: "movement-1", Type: invDomain.MovementTypeAdjustment, QuantityDelta: 3}}
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, recount, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/recount",
		map[string]any{"warehouse_id": "wh-1", "counted_quantity": 13})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "movement-1")
}

func TestTransferStockHandler_success(t *testing.T) {
	transfer := &stubTransferStock{}
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, transfer, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/transfer",
		map[string]any{"quantity": 5, "from_warehouse_id": "wh-a", "to_warehouse_id": "wh-b"})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "sku-1", transfer.gotInput.SKUID)
	require.Equal(t, int64(5), transfer.gotInput.Quantity)
	assert.Contains(t, w.Body.String(), "movement-out")
	assert.Contains(t, w.Body.String(), "movement-in")
}

func TestTransferStockHandler_general_ledger_toBatch(t *testing.T) {
	transfer := &stubTransferStock{}
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, transfer, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/transfer",
		map[string]any{"quantity": 5, "from_warehouse_id": "wh-a", "to_warehouse_id": "wh-a", "to_batch_id": "batch-1"})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Nil(t, transfer.gotInput.FromBatchID, "nil FromBatchID means moving out of the general ledger")
	require.NotNil(t, transfer.gotInput.ToBatchID)
	require.Equal(t, "batch-1", *transfer.gotInput.ToBatchID)
}

func TestMarkDamagedOrLostHandler_success(t *testing.T) {
	mark := &stubMarkDamagedOrLost{}
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, mark)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/mark-damaged-lost",
		map[string]any{"warehouse_id": "wh-1", "quantity": 2, "type": "damaged"})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, int64(2), mark.gotInput.Quantity)
	require.Equal(t, invDomain.MovementTypeDamaged, mark.gotInput.Type)
}

func TestMarkDamagedOrLostHandler_rejectsInvalidType(t *testing.T) {
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{},
		&stubMarkDamagedOrLost{err: invDomain.ErrInvalidMovementType})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/mark-damaged-lost",
		map[string]any{"warehouse_id": "wh-1", "quantity": 2, "type": "damaged"})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}
