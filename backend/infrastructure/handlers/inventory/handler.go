package inventory

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// ── Use-case interfaces (defined at point of use — ISP) ──────────────────────

type listWarehousesUC interface {
	Execute(ctx context.Context) ([]invDomain.Warehouse, error)
}
type createWarehouseUC interface {
	Execute(ctx context.Context, in appinventory.CreateWarehouseInput) (*invDomain.Warehouse, error)
}
type updateWarehouseUC interface {
	Execute(ctx context.Context, in appinventory.UpdateWarehouseInput) (*invDomain.Warehouse, error)
}
type deleteWarehouseUC interface {
	Execute(ctx context.Context, id string) error
}
type createBatchUC interface {
	Execute(ctx context.Context, in appinventory.CreateBatchInput) (*invDomain.Batch, error)
}
type updateBatchUC interface {
	Execute(ctx context.Context, in appinventory.UpdateBatchInput) (*invDomain.Batch, error)
}
type deleteBatchUC interface {
	Execute(ctx context.Context, id string) error
}
type listBatchesUC interface {
	Execute(ctx context.Context, skuID string) ([]invDomain.Batch, error)
}
type getSKUStockLevelUC interface {
	Execute(ctx context.Context, skuID string) (*appinventory.StockLevel, error)
}
type recordStockMovementUC interface {
	Execute(ctx context.Context, in appinventory.RecordStockMovementInput) (*invDomain.StockMovement, error)
}
type listStockMovementsUC interface {
	Execute(ctx context.Context, skuID string) ([]invDomain.StockMovement, error)
}
type listBatchMovementsUC interface {
	Execute(ctx context.Context, batchID string) ([]invDomain.StockMovement, error)
}
type recordRecountUC interface {
	Execute(ctx context.Context, in appinventory.RecordRecountInput) (*invDomain.StockMovement, error)
}
type transferStockUC interface {
	Execute(ctx context.Context, in appinventory.TransferStockInput) (*appinventory.TransferStockResult, error)
}
type splitCaseIntoUnitsUC interface {
	Execute(ctx context.Context, in appinventory.SplitCaseIntoUnitsInput) (*appinventory.SplitCaseIntoUnitsResult, error)
}
type portionFromBulkUC interface {
	Execute(ctx context.Context, in appinventory.PortionFromBulkInput) (*appinventory.PortionFromBulkResult, error)
}
type markDamagedOrLostUC interface {
	Execute(ctx context.Context, in appinventory.MarkDamagedOrLostInput) (*invDomain.StockMovement, error)
}
type receiveStockUC interface {
	Execute(ctx context.Context, in appinventory.ReceiveStockInput) (*appinventory.ReceiveStockResult, error)
}
type listExpiringBatchesUC interface {
	Execute(ctx context.Context, withinDays int) ([]invDomain.ExpiringBatch, error)
}
type listPendingExpiredBatchesUC interface {
	Execute(ctx context.Context) ([]invDomain.ExpiredBatchReview, error)
}
type writeOffExpiredBatchUC interface {
	Execute(ctx context.Context, batchID string) error
}
type extendBatchExpiryUC interface {
	Execute(ctx context.Context, batchID string, newExpiryDate time.Time) error
}
type dismissExpiredBatchUC interface {
	Execute(ctx context.Context, batchID string) error
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listWarehouses      listWarehousesUC
	createWarehouse     createWarehouseUC
	updateWarehouse     updateWarehouseUC
	deleteWarehouse     deleteWarehouseUC
	createBatch         createBatchUC
	updateBatch         updateBatchUC
	deleteBatch         deleteBatchUC
	listBatches         listBatchesUC
	getSKUStockLevel    getSKUStockLevelUC
	recordStockMovement recordStockMovementUC
	recordRecount       recordRecountUC
	transferStock       transferStockUC
	splitCaseIntoUnits  splitCaseIntoUnitsUC
	portionFromBulk     portionFromBulkUC
	markDamagedOrLost   markDamagedOrLostUC
	listExpiringBatches listExpiringBatchesUC
	listStockMovements  listStockMovementsUC
	receiveStock        receiveStockUC

	listPendingExpiredBatches listPendingExpiredBatchesUC
	writeOffExpiredBatch      writeOffExpiredBatchUC
	extendBatchExpiry         extendBatchExpiryUC
	dismissExpiredBatch       dismissExpiredBatchUC
	listBatchMovements        listBatchMovementsUC
}

// NewHandler constructs a Handler with injected use cases (useful for testing).
func NewHandler(
	listWarehouses listWarehousesUC,
	createWarehouse createWarehouseUC,
	updateWarehouse updateWarehouseUC,
	deleteWarehouse deleteWarehouseUC,
	createBatch createBatchUC,
	updateBatch updateBatchUC,
	deleteBatch deleteBatchUC,
	listBatches listBatchesUC,
	getSKUStockLevel getSKUStockLevelUC,
	recordStockMovement recordStockMovementUC,
	recordRecount recordRecountUC,
	transferStock transferStockUC,
	splitCaseIntoUnits splitCaseIntoUnitsUC,
	portionFromBulk portionFromBulkUC,
	markDamagedOrLost markDamagedOrLostUC,
	listExpiringBatches listExpiringBatchesUC,
	listStockMovements listStockMovementsUC,
	receiveStock receiveStockUC,
	listPendingExpiredBatches listPendingExpiredBatchesUC,
	writeOffExpiredBatch writeOffExpiredBatchUC,
	extendBatchExpiry extendBatchExpiryUC,
	dismissExpiredBatch dismissExpiredBatchUC,
	listBatchMovements listBatchMovementsUC,
) *Handler {
	return &Handler{
		listWarehouses:      listWarehouses,
		createWarehouse:     createWarehouse,
		updateWarehouse:     updateWarehouse,
		deleteWarehouse:     deleteWarehouse,
		createBatch:         createBatch,
		updateBatch:         updateBatch,
		deleteBatch:         deleteBatch,
		listBatches:         listBatches,
		getSKUStockLevel:    getSKUStockLevel,
		recordStockMovement: recordStockMovement,
		recordRecount:       recordRecount,
		transferStock:       transferStock,
		splitCaseIntoUnits:  splitCaseIntoUnits,
		portionFromBulk:     portionFromBulk,
		markDamagedOrLost:   markDamagedOrLost,
		listExpiringBatches: listExpiringBatches,
		listStockMovements:  listStockMovements,
		receiveStock:        receiveStock,

		listPendingExpiredBatches: listPendingExpiredBatches,
		writeOffExpiredBatch:      writeOffExpiredBatch,
		extendBatchExpiry:         extendBatchExpiry,
		dismissExpiredBatch:       dismissExpiredBatch,
		listBatchMovements:        listBatchMovements,
	}
}

var _ geninventory.ServerInterface = (*Handler)(nil)

// Register is the production wiring: creates repos + use cases, then
// mounts the generated routes (matching the OpenAPI spec exactly).
func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository) {
	warehouseRepo := repositories.NewWarehouseRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)

	h := NewHandler(
		appinventory.NewListWarehousesUseCase(db, warehouseRepo),
		appinventory.NewCreateWarehouseUseCase(db, warehouseRepo),
		appinventory.NewUpdateWarehouseUseCase(db, warehouseRepo),
		appinventory.NewDeleteWarehouseUseCase(db, warehouseRepo),
		appinventory.NewCreateBatchUseCase(db, batchRepo),
		appinventory.NewUpdateBatchUseCase(db, batchRepo),
		appinventory.NewDeleteBatchUseCase(db, batchRepo, movementRepo),
		appinventory.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo),
		appinventory.NewGetSKUStockLevelUseCase(db, movementRepo, batchRepo, settingsRepo),
		appinventory.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo),
		appinventory.NewRecordRecountUseCase(db, movementRepo, outboxRepo),
		appinventory.NewTransferStockUseCase(db, movementRepo, outboxRepo),
		appinventory.NewSplitCaseIntoUnitsUseCase(db, skuRepo, movementRepo, outboxRepo),
		appinventory.NewPortionFromBulkUseCase(db, skuRepo, movementRepo, outboxRepo),
		appinventory.NewMarkDamagedOrLostUseCase(db, movementRepo, outboxRepo),
		appinventory.NewListExpiringBatchesUseCase(db, batchRepo),
		appinventory.NewListStockMovementsUseCase(db, movementRepo),
		appinventory.NewReceiveStockUseCase(db, batchRepo, movementRepo, outboxRepo),
		appinventory.NewListPendingExpiredBatchesUseCase(db, batchRepo),
		appinventory.NewWriteOffExpiredBatchUseCase(db, batchRepo, movementRepo, appinventory.NewMarkDamagedOrLostUseCase(db, movementRepo, outboxRepo)),
		appinventory.NewExtendBatchExpiryUseCase(db, batchRepo),
		appinventory.NewDismissExpiredBatchUseCase(db, batchRepo),
		appinventory.NewListBatchMovementsUseCase(db, movementRepo),
	)
	geninventory.RegisterHandlers(rg, h)
}

// ── Warehouse route handlers ─────────────────────────────────────────────────

func (h *Handler) ListWarehouses(c *gin.Context) {
	warehouses, err := h.listWarehouses.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.Warehouse, len(warehouses))
	for i, w := range warehouses {
		resp[i] = toGenWarehouse(w)
	}
	c.JSON(http.StatusOK, geninventory.WarehouseListResponse{Data: &resp})
}

func (h *Handler) CreateWarehouse(c *gin.Context) {
	var req geninventory.CreateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Name == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required")
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	warehouse, err := h.createWarehouse.Execute(c.Request.Context(), appinventory.CreateWarehouseInput{
		Name: req.Name, Address1: strPtrOr(req.Address1), Address2: strPtrOr(req.Address2), City: strPtrOr(req.City),
		Province: strPtrOr(req.Province), Zip: strPtrOr(req.Zip), Country: strPtrOr(req.Country), Phone: strPtrOr(req.Phone),
		Active: active,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenWarehouse(*warehouse)
	c.JSON(http.StatusCreated, geninventory.WarehouseResponse{Data: &resp})
}

func (h *Handler) UpdateWarehouse(c *gin.Context, id geninventory.Id) {
	var req geninventory.UpdateWarehouseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	if req.Name == "" {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "name is required")
		return
	}
	active := true
	if req.Active != nil {
		active = *req.Active
	}
	warehouse, err := h.updateWarehouse.Execute(c.Request.Context(), appinventory.UpdateWarehouseInput{
		ID: id, Name: req.Name, Address1: strPtrOr(req.Address1), Address2: strPtrOr(req.Address2), City: strPtrOr(req.City),
		Province: strPtrOr(req.Province), Zip: strPtrOr(req.Zip), Country: strPtrOr(req.Country), Phone: strPtrOr(req.Phone),
		Active: active,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenWarehouse(*warehouse)
	c.JSON(http.StatusOK, geninventory.WarehouseResponse{Data: &resp})
}

func (h *Handler) DeleteWarehouse(c *gin.Context, id geninventory.Id) {
	err := h.deleteWarehouse.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, geninventory.EmptyResponse{})
}

// ── Batch route handlers ─────────────────────────────────────────────────────

func (h *Handler) ListBatches(c *gin.Context, skuId geninventory.SkuId) {
	batches, err := h.listBatches.Execute(c.Request.Context(), skuId)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.Batch, len(batches))
	for i, b := range batches {
		resp[i] = toGenBatch(b)
	}
	c.JSON(http.StatusOK, geninventory.BatchListResponse{Data: &resp})
}

func (h *Handler) CreateBatch(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.CreateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	var receivedAt time.Time
	if req.ReceivedAt != nil {
		receivedAt = *req.ReceivedAt
	}
	batch, err := h.createBatch.Execute(c.Request.Context(), appinventory.CreateBatchInput{
		SKUID: skuId, WarehouseID: req.WarehouseId, LotNumber: strPtrOr(req.LotNumber),
		ExpiryDate: genDateToTimePtr(req.ExpiryDate), ReceivedAt: receivedAt,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenBatch(*batch)
	c.JSON(http.StatusCreated, geninventory.BatchResponse{Data: &resp})
}

func (h *Handler) UpdateBatch(c *gin.Context, id geninventory.Id) {
	var req geninventory.UpdateBatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	batch, err := h.updateBatch.Execute(c.Request.Context(), appinventory.UpdateBatchInput{
		ID: id, WarehouseID: req.WarehouseId, LotNumber: strPtrOr(req.LotNumber),
		ExpiryDate: genDateToTimePtr(req.ExpiryDate),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenBatch(*batch)
	c.JSON(http.StatusOK, geninventory.BatchResponse{Data: &resp})
}

func (h *Handler) DeleteBatch(c *gin.Context, id geninventory.Id) {
	err := h.deleteBatch.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, geninventory.EmptyResponse{})
}

// ── Stock-level / movement route handlers ────────────────────────────────────

func (h *Handler) GetSKUStockLevel(c *gin.Context, skuId geninventory.SkuId) {
	level, err := h.getSKUStockLevel.Execute(c.Request.Context(), skuId)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := geninventory.StockLevel{SkuId: level.SKUID, Total: level.Total, Syncable: level.Syncable}
	c.JSON(http.StatusOK, geninventory.StockLevelResponse{Data: &resp})
}

func (h *Handler) ListExpiringBatches(c *gin.Context, params geninventory.ListExpiringBatchesParams) {
	withinDays := 30
	if params.WithinDays != nil {
		withinDays = *params.WithinDays
	}
	batches, err := h.listExpiringBatches.Execute(c.Request.Context(), withinDays)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.ExpiringBatch, len(batches))
	for i, b := range batches {
		resp[i] = toGenExpiringBatch(b)
	}
	c.JSON(http.StatusOK, geninventory.ExpiringBatchListResponse{Data: &resp})
}

func (h *Handler) ListPendingExpiredBatches(c *gin.Context) {
	batches, err := h.listPendingExpiredBatches.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.ExpiredBatchReview, len(batches))
	for i, b := range batches {
		resp[i] = toGenExpiredBatchReview(b)
	}
	c.JSON(http.StatusOK, geninventory.ExpiredBatchReviewListResponse{Data: &resp})
}

func (h *Handler) WriteOffExpiredBatch(c *gin.Context, id geninventory.Id) {
	err := h.writeOffExpiredBatch.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, geninventory.EmptyResponse{})
}

func (h *Handler) ExtendBatchExpiry(c *gin.Context, id geninventory.Id) {
	var req geninventory.ExtendBatchExpiryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	err := h.extendBatchExpiry.Execute(c.Request.Context(), id, req.NewExpiryDate.Time)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, geninventory.EmptyResponse{})
}

func (h *Handler) DismissExpiredBatch(c *gin.Context, id geninventory.Id) {
	err := h.dismissExpiredBatch.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, geninventory.EmptyResponse{})
}

func (h *Handler) ListBatchMovements(c *gin.Context, id geninventory.Id) {
	movements, err := h.listBatchMovements.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.StockMovement, len(movements))
	for i, m := range movements {
		resp[i] = toGenStockMovement(m)
	}
	c.JSON(http.StatusOK, geninventory.StockMovementListResponse{Data: &resp})
}

func (h *Handler) ListStockMovements(c *gin.Context, skuId geninventory.SkuId) {
	movements, err := h.listStockMovements.Execute(c.Request.Context(), skuId)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]geninventory.StockMovement, len(movements))
	for i, m := range movements {
		resp[i] = toGenStockMovement(m)
	}
	c.JSON(http.StatusOK, geninventory.StockMovementListResponse{Data: &resp})
}

func (h *Handler) RecordStockMovement(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.RecordStockMovementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	movement, err := h.recordStockMovement.Execute(c.Request.Context(), appinventory.RecordStockMovementInput{
		SKUID: skuId, WarehouseID: req.WarehouseId, BatchID: req.BatchId,
		Type: invDomain.MovementType(req.Type), QuantityDelta: req.QuantityDelta,
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenStockMovement(*movement)
	c.JSON(http.StatusCreated, geninventory.StockMovementResponse{Data: &resp})
}

func (h *Handler) RecordRecount(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.RecordRecountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	movement, err := h.recordRecount.Execute(c.Request.Context(), appinventory.RecordRecountInput{
		SKUID: skuId, WarehouseID: req.WarehouseId, BatchID: req.BatchId,
		CountedQuantity: req.CountedQuantity, Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	var resp *geninventory.StockMovement
	if movement != nil {
		g := toGenStockMovement(*movement)
		resp = &g
	}
	c.JSON(http.StatusOK, geninventory.StockMovementResponse{Data: resp})
}

func (h *Handler) TransferStock(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.TransferStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.transferStock.Execute(c.Request.Context(), appinventory.TransferStockInput{
		SKUID: skuId, Quantity: req.Quantity,
		FromWarehouseID: req.FromWarehouseId, FromBatchID: req.FromBatchId,
		ToWarehouseID: req.ToWarehouseId, ToBatchID: req.ToBatchId,
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := geninventory.TransferStockResult{Out: toGenStockMovement(*result.Out), In: toGenStockMovement(*result.In)}
	c.JSON(http.StatusCreated, geninventory.TransferStockResultResponse{Data: &resp})
}

func (h *Handler) SplitCaseIntoUnits(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.SplitCaseIntoUnitsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.splitCaseIntoUnits.Execute(c.Request.Context(), appinventory.SplitCaseIntoUnitsInput{
		CaseSKUID: skuId, WarehouseID: req.WarehouseId, BatchID: req.BatchId, CasesToSplit: req.CasesToSplit,
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := geninventory.SplitCaseIntoUnitsResult{
		CaseOut: toGenStockMovement(result.CaseOutMovement), UnitIn: toGenStockMovement(result.UnitInMovement),
	}
	c.JSON(http.StatusCreated, geninventory.SplitCaseIntoUnitsResultResponse{Data: &resp})
}

func (h *Handler) PortionFromBulk(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.PortionFromBulkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.portionFromBulk.Execute(c.Request.Context(), appinventory.PortionFromBulkInput{
		TargetSKUID: skuId, WarehouseID: req.WarehouseId, BatchID: req.BatchId, WeightGrams: req.WeightGrams,
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := geninventory.PortionFromBulkResult{
		BulkOut: toGenStockMovement(result.BulkOutMovement), UnitIn: toGenStockMovement(result.UnitInMovement),
	}
	c.JSON(http.StatusCreated, geninventory.PortionFromBulkResultResponse{Data: &resp})
}

func (h *Handler) MarkDamagedOrLost(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.MarkDamagedOrLostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	movement, err := h.markDamagedOrLost.Execute(c.Request.Context(), appinventory.MarkDamagedOrLostInput{
		SKUID: skuId, WarehouseID: req.WarehouseId, BatchID: req.BatchId,
		Quantity: req.Quantity, Type: invDomain.MovementType(req.Type),
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenStockMovement(*movement)
	c.JSON(http.StatusCreated, geninventory.StockMovementResponse{Data: &resp})
}

func (h *Handler) ReceiveStock(c *gin.Context, skuId geninventory.SkuId) {
	var req geninventory.ReceiveStockRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.receiveStock.Execute(c.Request.Context(), appinventory.ReceiveStockInput{
		SKUID: skuId, WarehouseID: req.WarehouseId, Quantity: req.Quantity, BatchID: req.BatchId,
		LotNumber: strPtrOr(req.LotNumber), ExpiryDate: genDateToTimePtr(req.ExpiryDate),
		Reason: strPtrOr(req.Reason), OccurredAt: timeOrNow(req.OccurredAt),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	var batch *geninventory.Batch
	if result.Batch != nil {
		b := toGenBatch(*result.Batch)
		batch = &b
	}
	resp := geninventory.ReceiveStockResult{Batch: batch, Movement: toGenStockMovement(*result.Movement)}
	c.JSON(http.StatusCreated, geninventory.ReceiveStockResultResponse{Data: &resp})
}

// ── Domain → generated-type mappers ──────────────────────────────────────────

func toGenWarehouse(w invDomain.Warehouse) geninventory.Warehouse {
	return geninventory.Warehouse{
		Id:        w.ID,
		TenantId:  w.TenantID,
		Name:      w.Name,
		Address1:  w.Address1,
		Address2:  w.Address2,
		City:      w.City,
		Province:  w.Province,
		Zip:       w.Zip,
		Country:   w.Country,
		Phone:     w.Phone,
		Active:    w.Active,
		CreatedAt: w.CreatedAt,
		UpdatedAt: w.UpdatedAt,
	}
}

func toGenBatch(b invDomain.Batch) geninventory.Batch {
	return geninventory.Batch{
		Id: b.ID, TenantId: b.TenantID, SkuId: b.SKUID, WarehouseId: b.WarehouseID,
		LotNumber: b.LotNumber, ExpiryDate: timePtrToGenDate(b.ExpiryDate),
		ReceivedAt: b.ReceivedAt, CreatedAt: b.CreatedAt, UpdatedAt: b.UpdatedAt,
		Quantity: b.Quantity, IsGeneralLedger: b.IsGeneralLedger,
	}
}

func toGenExpiringBatch(b invDomain.ExpiringBatch) geninventory.ExpiringBatch {
	return geninventory.ExpiringBatch{
		BatchId: b.BatchID, SkuId: b.SKUID, SkuCode: b.SKUCode, ProductName: b.ProductName,
		WarehouseId: b.WarehouseID, WarehouseName: b.WarehouseName, LotNumber: b.LotNumber,
		ExpiryDate: b.ExpiryDate, Quantity: b.Quantity,
	}
}

func toGenExpiredBatchReview(b invDomain.ExpiredBatchReview) geninventory.ExpiredBatchReview {
	return geninventory.ExpiredBatchReview{
		BatchId: b.BatchID, SkuId: b.SKUID, SkuCode: b.SKUCode, ProductName: b.ProductName,
		WarehouseId: b.WarehouseID, WarehouseName: b.WarehouseName, LotNumber: b.LotNumber,
		ExpiryDate: b.ExpiryDate, Quantity: b.Quantity,
	}
}

func toGenStockMovement(m invDomain.StockMovement) geninventory.StockMovement {
	return geninventory.StockMovement{
		Id: m.ID, TenantId: m.TenantID, SkuId: m.SKUID, WarehouseId: m.WarehouseID, BatchId: m.BatchID,
		Type: geninventory.MovementType(m.Type), QuantityDelta: m.QuantityDelta,
		Reason: m.Reason, OccurredAt: m.OccurredAt, CreatedAt: m.CreatedAt,
	}
}

func genDateToTimePtr(d *openapi_types.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time
	return &t
}

func timePtrToGenDate(t *time.Time) *openapi_types.Date {
	if t == nil {
		return nil
	}
	return &openapi_types.Date{Time: *t}
}

func timeOrNow(t *time.Time) time.Time {
	if t == nil {
		return time.Now()
	}
	return *t
}

func strPtrOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ── Error mapping ────────────────────────────────────────────────────────────

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, invDomain.ErrWarehouseNotFound), errors.Is(err, invDomain.ErrBatchNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, invDomain.ErrInvalidQuantity), errors.Is(err, invDomain.ErrInvalidMovementType),
		errors.Is(err, invDomain.ErrInsufficientStock), errors.Is(err, invDomain.ErrSKUIsNotACase),
		errors.Is(err, invDomain.ErrSKUIsNotPortioned):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	case errors.Is(err, pmsDomain.ErrSKUNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, invDomain.ErrBatchHasMovements), errors.Is(err, invDomain.ErrDuplicateWarehouseName):
		middleware.Fail(c, http.StatusConflict, "CONFLICT", err.Error())
	default:
		log.Printf("inventory: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
