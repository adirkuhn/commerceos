package inventory_test

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

type stubSplitCaseIntoUnits struct {
	out      *appinventory.SplitCaseIntoUnitsResult
	err      error
	gotInput appinventory.SplitCaseIntoUnitsInput
}

func (s *stubSplitCaseIntoUnits) Execute(_ context.Context, in appinventory.SplitCaseIntoUnitsInput) (*appinventory.SplitCaseIntoUnitsResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appinventory.SplitCaseIntoUnitsResult{
		CaseOutMovement: invDomain.StockMovement{ID: "movement-case-out", Type: invDomain.MovementTypeSplitOut, QuantityDelta: -in.CasesToSplit},
		UnitInMovement:  invDomain.StockMovement{ID: "movement-unit-in", Type: invDomain.MovementTypeSplitIn, QuantityDelta: in.CasesToSplit},
	}, nil
}

type stubPortionFromBulk struct {
	out      *appinventory.PortionFromBulkResult
	err      error
	gotInput appinventory.PortionFromBulkInput
}

func (s *stubPortionFromBulk) Execute(_ context.Context, in appinventory.PortionFromBulkInput) (*appinventory.PortionFromBulkResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	weight := in.WeightGrams
	if weight == nil {
		w := int64(1)
		weight = &w
	}
	return &appinventory.PortionFromBulkResult{
		BulkOutMovement: invDomain.StockMovement{ID: "movement-bulk-out", Type: invDomain.MovementTypeSplitOut, QuantityDelta: -*weight},
		UnitInMovement:  invDomain.StockMovement{ID: "movement-unit-in", Type: invDomain.MovementTypeSplitIn, QuantityDelta: 1},
	}, nil
}

type stubMarkDamagedOrLost struct {
	out      *invDomain.StockMovement
	err      error
	gotInput appinventory.MarkDamagedOrLostInput
}

func (s *stubMarkDamagedOrLost) Execute(_ context.Context, in appinventory.MarkDamagedOrLostInput) (*invDomain.StockMovement, *invDomain.DiscardRecord, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, nil, s.err
	}
	movement := s.out
	if movement == nil {
		movement = &invDomain.StockMovement{ID: "movement-1", Type: in.Type, QuantityDelta: -in.Quantity}
	}
	return movement, &invDomain.DiscardRecord{ID: "discard-1", MovementID: movement.ID}, nil
}

type stubListExpiringBatches struct {
	out []invDomain.ExpiringBatch
	err error
}

func (s *stubListExpiringBatches) Execute(_ context.Context, _ int) ([]invDomain.ExpiringBatch, error) {
	return s.out, s.err
}

type stubListStockMovements struct {
	out []invDomain.StockMovement
	err error
}

func (s *stubListStockMovements) Execute(_ context.Context, _ string) ([]invDomain.StockMovement, error) {
	return s.out, s.err
}

type stubReceiveStock struct {
	out      *appinventory.ReceiveStockResult
	err      error
	gotInput appinventory.ReceiveStockInput
}

func (s *stubReceiveStock) Execute(_ context.Context, in appinventory.ReceiveStockInput) (*appinventory.ReceiveStockResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appinventory.ReceiveStockResult{Movement: &invDomain.StockMovement{ID: "movement-1", SKUID: in.SKUID, Type: invDomain.MovementTypeReceipt, QuantityDelta: in.Quantity}}, nil
}

type stubListPendingExpiredBatches struct {
	out []invDomain.ExpiredBatchReview
	err error
}

func (s *stubListPendingExpiredBatches) Execute(_ context.Context) ([]invDomain.ExpiredBatchReview, error) {
	return s.out, s.err
}

type stubWriteOffExpiredBatch struct {
	err            error
	calls          []string
	performedByArg []string
}

func (s *stubWriteOffExpiredBatch) Execute(_ context.Context, batchID string, performedBy string) error {
	s.calls = append(s.calls, batchID)
	s.performedByArg = append(s.performedByArg, performedBy)
	return s.err
}

type stubExtendBatchExpiry struct {
	err          error
	gotBatchID   string
	gotNewExpiry time.Time
}

func (s *stubExtendBatchExpiry) Execute(_ context.Context, batchID string, newExpiryDate time.Time) error {
	s.gotBatchID, s.gotNewExpiry = batchID, newExpiryDate
	return s.err
}

type stubDismissExpiredBatch struct {
	err   error
	calls []string
}

func (s *stubDismissExpiredBatch) Execute(_ context.Context, batchID string) error {
	s.calls = append(s.calls, batchID)
	return s.err
}

type stubListBatchMovements struct {
	out []invDomain.StockMovement
	err error
}

func (s *stubListBatchMovements) Execute(_ context.Context, _ string) ([]invDomain.StockMovement, error) {
	return s.out, s.err
}

type stubDiscardBatch struct {
	out      *invDomain.DiscardRecord
	err      error
	gotInput appinventory.DiscardBatchInput
}

func (s *stubDiscardBatch) Execute(_ context.Context, in appinventory.DiscardBatchInput) (*invDomain.DiscardRecord, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &invDomain.DiscardRecord{ID: "discard-1"}, nil
}

type stubListDiscards struct {
	out *appinventory.ListDiscardsResult
	err error
}

func (s *stubListDiscards) Execute(_ context.Context, _ appinventory.ListDiscardsInput) (*appinventory.ListDiscardsResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appinventory.ListDiscardsResult{}, nil
}

func newTestHandler(list *stubListWarehouses, create *stubCreateWarehouse, update *stubUpdateWarehouse, del *stubDeleteWarehouse) *inventoryhandler.Handler {
	return inventoryhandler.NewHandler(
		list, create, update, del,
		&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubSplitCaseIntoUnits{}, &stubPortionFromBulk{}, &stubMarkDamagedOrLost{},
		&stubListExpiringBatches{}, &stubListStockMovements{}, &stubReceiveStock{},
		&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{},
		&stubListBatchMovements{},
		&stubDiscardBatch{}, &stubListDiscards{},
	)
}

// newBatchTestHandler builds a Handler for the batch/movement routes
// specifically — every other use case is a harmless zero-value stub.
func newBatchTestHandler(
	createBatch *stubCreateBatch, updateBatch *stubUpdateBatch, deleteBatch *stubDeleteBatch, listBatches *stubListBatches,
	stockLevel *stubGetSKUStockLevel, recordMovement *stubRecordStockMovement, recordRecount *stubRecordRecount,
	transfer *stubTransferStock, markDamagedOrLost *stubMarkDamagedOrLost,
) *inventoryhandler.Handler {
	return newBatchTestHandlerWithReceive(createBatch, updateBatch, deleteBatch, listBatches,
		stockLevel, recordMovement, recordRecount, transfer, markDamagedOrLost, &stubReceiveStock{})
}

func newBatchTestHandlerWithReceive(
	createBatch *stubCreateBatch, updateBatch *stubUpdateBatch, deleteBatch *stubDeleteBatch, listBatches *stubListBatches,
	stockLevel *stubGetSKUStockLevel, recordMovement *stubRecordStockMovement, recordRecount *stubRecordRecount,
	transfer *stubTransferStock, markDamagedOrLost *stubMarkDamagedOrLost, receiveStock *stubReceiveStock,
) *inventoryhandler.Handler {
	return inventoryhandler.NewHandler(
		&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{},
		createBatch, updateBatch, deleteBatch, listBatches,
		stockLevel, recordMovement, recordRecount, transfer, &stubSplitCaseIntoUnits{}, &stubPortionFromBulk{}, markDamagedOrLost,
		&stubListExpiringBatches{}, &stubListStockMovements{}, receiveStock,
		&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{},
		&stubListBatchMovements{},
		&stubDiscardBatch{}, &stubListDiscards{},
	)
}

func newSplitCaseTestHandler(split *stubSplitCaseIntoUnits) *inventoryhandler.Handler {
	return inventoryhandler.NewHandler(
		&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{},
		&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, split, &stubPortionFromBulk{}, &stubMarkDamagedOrLost{},
		&stubListExpiringBatches{}, &stubListStockMovements{}, &stubReceiveStock{},
		&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{},
		&stubListBatchMovements{},
		&stubDiscardBatch{}, &stubListDiscards{},
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

func TestListBatchesHandler_includesQuantityAndGeneralLedgerRow(t *testing.T) {
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{},
		&stubListBatches{out: []invDomain.Batch{
			{WarehouseID: "wh-1", IsGeneralLedger: true, Quantity: 0},
			{ID: "batch-1", LotNumber: "LOT-1", Quantity: 82},
		}},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/inventory/skus/sku-1/batches", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"is_general_ledger":true`)
	assert.Contains(t, w.Body.String(), `"quantity":82`)
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

func TestListExpiringBatchesHandler_returnsBatches(t *testing.T) {
	expiry := time.Now().Add(48 * time.Hour)
	h := inventoryhandler.NewHandler(
		&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{},
		&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubSplitCaseIntoUnits{}, &stubPortionFromBulk{}, &stubMarkDamagedOrLost{},
		&stubListExpiringBatches{out: []invDomain.ExpiringBatch{
			{BatchID: "batch-1", SKUID: "sku-1", SKUCode: "SKU-1", ProductName: "Widget", ExpiryDate: expiry, Quantity: 5},
		}},
		&stubListStockMovements{}, &stubReceiveStock{},
		&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{},
		&stubListBatchMovements{},
		&stubDiscardBatch{}, &stubListDiscards{},
	)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/inventory/batches/expiring-soon?within_days=10", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_code":"SKU-1"`)
	assert.Contains(t, w.Body.String(), `"product_name":"Widget"`)
}

func TestListStockMovementsHandler_returnsMovements(t *testing.T) {
	h := inventoryhandler.NewHandler(
		&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{},
		&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubSplitCaseIntoUnits{}, &stubPortionFromBulk{}, &stubMarkDamagedOrLost{},
		&stubListExpiringBatches{},
		&stubListStockMovements{out: []invDomain.StockMovement{
			{ID: "movement-1", SKUID: "sku-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 10},
		}},
		&stubReceiveStock{},
		&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{},
		&stubListBatchMovements{},
		&stubDiscardBatch{}, &stubListDiscards{},
	)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/inventory/skus/sku-1/movements", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"movement-1"`)
	assert.Contains(t, w.Body.String(), `"quantity_delta":10`)
}

func TestListBatchMovementsHandler_returnsMovementsForOneBatch(t *testing.T) {
	batchID := "batch-1"
	h := inventoryhandler.NewHandler(
		&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{},
		&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubSplitCaseIntoUnits{}, &stubPortionFromBulk{}, &stubMarkDamagedOrLost{},
		&stubListExpiringBatches{}, &stubListStockMovements{}, &stubReceiveStock{},
		&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{},
		&stubListBatchMovements{out: []invDomain.StockMovement{
			{ID: "movement-1", SKUID: "sku-1", BatchID: &batchID, Type: invDomain.MovementTypeReceipt, QuantityDelta: 6},
		}},
		&stubDiscardBatch{}, &stubListDiscards{},
	)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/inventory/batches/batch-1/movements", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"id":"movement-1"`)
	assert.Contains(t, w.Body.String(), `"quantity_delta":6`)
}

func newExpiredReviewTestHandler(list *stubListPendingExpiredBatches, writeOff *stubWriteOffExpiredBatch, extend *stubExtendBatchExpiry, dismiss *stubDismissExpiredBatch) *inventoryhandler.Handler {
	return inventoryhandler.NewHandler(
		&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{},
		&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubSplitCaseIntoUnits{}, &stubPortionFromBulk{}, &stubMarkDamagedOrLost{},
		&stubListExpiringBatches{}, &stubListStockMovements{}, &stubReceiveStock{},
		list, writeOff, extend, dismiss,
		&stubListBatchMovements{},
		&stubDiscardBatch{}, &stubListDiscards{},
	)
}

func TestListPendingExpiredBatchesHandler_returnsBatches(t *testing.T) {
	expiry := time.Now().Add(-10 * 24 * time.Hour)
	h := newExpiredReviewTestHandler(&stubListPendingExpiredBatches{out: []invDomain.ExpiredBatchReview{
		{BatchID: "batch-1", SKUID: "sku-1", SKUCode: "SKU-1", ProductName: "Widget", ExpiryDate: expiry, Quantity: 4},
	}}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{})

	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/inventory/batches/expired-review", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"sku_code":"SKU-1"`)
	assert.Contains(t, w.Body.String(), `"batch_id":"batch-1"`)
}

func TestWriteOffExpiredBatchHandler_success(t *testing.T) {
	writeOff := &stubWriteOffExpiredBatch{}
	h := newExpiredReviewTestHandler(&stubListPendingExpiredBatches{}, writeOff, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{})

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/batches/batch-1/write-off", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []string{"batch-1"}, writeOff.calls)
}

func TestWriteOffExpiredBatchHandler_propagatesNotFound(t *testing.T) {
	h := newExpiredReviewTestHandler(&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{err: invDomain.ErrBatchNotFound}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{})

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/batches/batch-1/write-off", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestExtendBatchExpiryHandler_success(t *testing.T) {
	extend := &stubExtendBatchExpiry{}
	h := newExpiredReviewTestHandler(&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, extend, &stubDismissExpiredBatch{})

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/batches/batch-1/extend-expiry",
		map[string]any{"new_expiry_date": "2026-12-01"})

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "batch-1", extend.gotBatchID)
	require.Equal(t, 2026, extend.gotNewExpiry.Year())
	require.Equal(t, time.Month(12), extend.gotNewExpiry.Month())
	require.Equal(t, 1, extend.gotNewExpiry.Day())
}

func TestDismissExpiredBatchHandler_success(t *testing.T) {
	dismiss := &stubDismissExpiredBatch{}
	h := newExpiredReviewTestHandler(&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, dismiss)

	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/batches/batch-1/dismiss", nil)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, []string{"batch-1"}, dismiss.calls)
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

func TestRecordStockMovementHandler_rejectsInsufficientStock(t *testing.T) {
	h := newBatchTestHandler(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{err: invDomain.ErrInsufficientStock}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/movements",
		map[string]any{"warehouse_id": "wh-1", "type": "sale", "quantity_delta": -18})

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

func TestSplitCaseIntoUnitsHandler_success(t *testing.T) {
	split := &stubSplitCaseIntoUnits{}
	h := newSplitCaseTestHandler(split)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-case-1/split-case",
		map[string]any{"warehouse_id": "wh-1", "cases_to_split": 2})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "sku-case-1", split.gotInput.CaseSKUID)
	require.Equal(t, int64(2), split.gotInput.CasesToSplit)
	assert.Contains(t, w.Body.String(), "movement-case-out")
	assert.Contains(t, w.Body.String(), "movement-unit-in")
}

func TestSplitCaseIntoUnitsHandler_notACaseReturns422(t *testing.T) {
	split := &stubSplitCaseIntoUnits{err: invDomain.ErrSKUIsNotACase}
	h := newSplitCaseTestHandler(split)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/split-case",
		map[string]any{"warehouse_id": "wh-1", "cases_to_split": 1})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestPortionFromBulkHandler_success(t *testing.T) {
	portion := &stubPortionFromBulk{}
	h := inventoryhandler.NewHandler(
		&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{},
		&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{},
		&stubSplitCaseIntoUnits{}, portion, &stubMarkDamagedOrLost{},
		&stubListExpiringBatches{}, &stubListStockMovements{}, &stubReceiveStock{},
		&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{},
		&stubListBatchMovements{},
		&stubDiscardBatch{}, &stubListDiscards{},
	)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-378g/portion-from-bulk",
		map[string]any{"warehouse_id": "wh-1", "weight_grams": 378})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "sku-378g", portion.gotInput.TargetSKUID)
	require.NotNil(t, portion.gotInput.WeightGrams)
	require.Equal(t, int64(378), *portion.gotInput.WeightGrams)
	assert.Contains(t, w.Body.String(), "movement-bulk-out")
	assert.Contains(t, w.Body.String(), "movement-unit-in")
}

func TestPortionFromBulkHandler_notPortionedReturns422(t *testing.T) {
	portion := &stubPortionFromBulk{err: invDomain.ErrSKUIsNotPortioned}
	h := inventoryhandler.NewHandler(
		&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{},
		&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{},
		&stubSplitCaseIntoUnits{}, portion, &stubMarkDamagedOrLost{},
		&stubListExpiringBatches{}, &stubListStockMovements{}, &stubReceiveStock{},
		&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{},
		&stubListBatchMovements{},
		&stubDiscardBatch{}, &stubListDiscards{},
	)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/portion-from-bulk",
		map[string]any{"warehouse_id": "wh-1"})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
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

func TestReceiveStockHandler_success(t *testing.T) {
	receive := &stubReceiveStock{out: &appinventory.ReceiveStockResult{
		Batch:    &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-1"},
		Movement: &invDomain.StockMovement{ID: "movement-1", SKUID: "sku-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 10},
	}}
	h := newBatchTestHandlerWithReceive(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{}, receive)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/receive",
		map[string]any{"warehouse_id": "wh-1", "quantity": 10, "lot_number": "LOT-1"})

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "sku-1", receive.gotInput.SKUID)
	require.Equal(t, int64(10), receive.gotInput.Quantity)
	assert.Contains(t, w.Body.String(), "batch-1")
	assert.Contains(t, w.Body.String(), "movement-1")
}

func TestReceiveStockHandler_passesThroughPurchaseOrderLineItemAndUnitCost(t *testing.T) {
	receive := &stubReceiveStock{out: &appinventory.ReceiveStockResult{
		Batch:    &invDomain.Batch{ID: "batch-1", SKUID: "sku-1", WarehouseID: "wh-1"},
		Movement: &invDomain.StockMovement{ID: "movement-1", SKUID: "sku-1", Type: invDomain.MovementTypeReceipt, QuantityDelta: 10},
	}}
	h := newBatchTestHandlerWithReceive(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{}, receive)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/receive",
		map[string]any{"warehouse_id": "wh-1", "quantity": 10, "purchase_order_line_item_id": "line-1", "unit_cost_minor_units": 900})

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, receive.gotInput.PurchaseOrderLineItemID)
	assert.Equal(t, "line-1", *receive.gotInput.PurchaseOrderLineItemID)
	require.NotNil(t, receive.gotInput.UnitCostMinorUnits)
	assert.Equal(t, int64(900), *receive.gotInput.UnitCostMinorUnits)
}

func TestReceiveStockHandler_rejectsInvalidQuantity(t *testing.T) {
	h := newBatchTestHandlerWithReceive(&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubMarkDamagedOrLost{},
		&stubReceiveStock{err: invDomain.ErrInvalidQuantity})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/inventory/skus/sku-1/receive",
		map[string]any{"warehouse_id": "wh-1", "quantity": 1})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func newDiscardTestHandler(discard *stubDiscardBatch, list *stubListDiscards) *inventoryhandler.Handler {
	return inventoryhandler.NewHandler(
		&stubListWarehouses{}, &stubCreateWarehouse{}, &stubUpdateWarehouse{}, &stubDeleteWarehouse{},
		&stubCreateBatch{}, &stubUpdateBatch{}, &stubDeleteBatch{}, &stubListBatches{},
		&stubGetSKUStockLevel{}, &stubRecordStockMovement{}, &stubRecordRecount{}, &stubTransferStock{}, &stubSplitCaseIntoUnits{}, &stubPortionFromBulk{}, &stubMarkDamagedOrLost{},
		&stubListExpiringBatches{}, &stubListStockMovements{}, &stubReceiveStock{},
		&stubListPendingExpiredBatches{}, &stubWriteOffExpiredBatch{}, &stubExtendBatchExpiry{}, &stubDismissExpiredBatch{},
		&stubListBatchMovements{},
		discard, list,
	)
}

func TestDiscardBatchHandler_withAPhoto(t *testing.T) {
	discard := &stubDiscardBatch{out: &invDomain.DiscardRecord{ID: "discard-1", SKUID: "sku-1", Quantity: 3}}
	h := newDiscardTestHandler(discard, &stubListDiscards{})
	router := newTestRouter(h)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("warehouse_id", "wh-1"))
	require.NoError(t, mw.WriteField("batch_id", "batch-1"))
	require.NoError(t, mw.WriteField("quantity", "3"))
	require.NoError(t, mw.WriteField("reason", "expired"))
	fw, err := mw.CreateFormFile("photo", "photo.jpg")
	require.NoError(t, err)
	_, err = fw.Write([]byte("fake-jpeg-bytes"))
	require.NoError(t, err)
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/skus/sku-1/discard", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Equal(t, "sku-1", discard.gotInput.SKUID)
	require.Equal(t, "wh-1", discard.gotInput.WarehouseID)
	require.Equal(t, "batch-1", *discard.gotInput.BatchID)
	require.Equal(t, int64(3), discard.gotInput.Quantity)
	require.Equal(t, "expired", discard.gotInput.Reason)
	require.NotNil(t, discard.gotInput.PhotoData)
	require.Equal(t, "photo.jpg", discard.gotInput.PhotoFilename)
	assert.Contains(t, w.Body.String(), "discard-1")
}

func TestDiscardBatchHandler_withoutAPhotoOrBatchIsGeneralLedger(t *testing.T) {
	discard := &stubDiscardBatch{}
	h := newDiscardTestHandler(discard, &stubListDiscards{})
	router := newTestRouter(h)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("warehouse_id", "wh-1"))
	require.NoError(t, mw.WriteField("quantity", "5"))
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/skus/sku-1/discard", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
	require.Nil(t, discard.gotInput.BatchID)
	require.Nil(t, discard.gotInput.PhotoData)
}

func TestDiscardBatchHandler_rejectsInsufficientStock(t *testing.T) {
	discard := &stubDiscardBatch{err: invDomain.ErrInsufficientStock}
	h := newDiscardTestHandler(discard, &stubListDiscards{})
	router := newTestRouter(h)

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	require.NoError(t, mw.WriteField("warehouse_id", "wh-1"))
	require.NoError(t, mw.WriteField("quantity", "5"))
	require.NoError(t, mw.Close())

	req := httptest.NewRequest(http.MethodPost, "/api/v1/inventory/skus/sku-1/discard", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestListDiscardsHandler_returnsReportAndTotal(t *testing.T) {
	list := &stubListDiscards{out: &appinventory.ListDiscardsResult{
		Discards: []invDomain.DiscardRecordReport{{DiscardRecord: invDomain.DiscardRecord{ID: "d1"}, ProductName: "Widget"}},
		Total:    1,
	}}
	h := newDiscardTestHandler(&stubDiscardBatch{}, list)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/inventory/discards?warehouse_id=wh-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Widget")
	assert.Contains(t, w.Body.String(), `"total":1`)
}
