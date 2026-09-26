package inventory

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	geninventory "github.com/adirkuhn/commerceos/api/inventory"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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
	Execute(ctx context.Context, in appinventory.MarkDamagedOrLostInput) (*invDomain.StockMovement, *invDomain.DiscardRecord, error)
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
	Execute(ctx context.Context, batchID string, performedBy string) error
}
type extendBatchExpiryUC interface {
	Execute(ctx context.Context, batchID string, newExpiryDate time.Time) error
}
type dismissExpiredBatchUC interface {
	Execute(ctx context.Context, batchID string) error
}
type discardBatchUC interface {
	Execute(ctx context.Context, in appinventory.DiscardBatchInput) (*invDomain.DiscardRecord, error)
}
type listDiscardsUC interface {
	Execute(ctx context.Context, in appinventory.ListDiscardsInput) (*appinventory.ListDiscardsResult, error)
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
	discardBatch              discardBatchUC
	listDiscards              listDiscardsUC
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
	discardBatch discardBatchUC,
	listDiscards listDiscardsUC,
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
		discardBatch:              discardBatch,
		listDiscards:              listDiscards,
	}
}

var _ geninventory.ServerInterface = (*Handler)(nil)

// skuCostUpdaterAdapter bridges application/inventory's local
// skuCostUpdaterForReceiveStock interface to application/pms's real
// ApplyReceiptCostUseCase -- lives in the composition root (not in either
// application package) because application/pms already imports
// application/inventory (see application/pms/recalculate_cost_prices.go's
// own doc comment on that exact cycle), so the reverse import would cycle.
// Duplicated (not shared) with infrastructure/handlers/procurement's own
// copy -- each handler package builds its own wiring, same reasoning as
// this file's genDateToTimePtr-style local helpers elsewhere in the repo.
type skuCostUpdaterAdapter struct {
	uc *apppms.ApplyReceiptCostUseCase
}

func (a skuCostUpdaterAdapter) Execute(ctx context.Context, in appinventory.SKUCostReceiptInput) error {
	return a.uc.Execute(ctx, apppms.ApplyReceiptCostInput{
		SKUID: in.SKUID, ReceivedQuantity: in.ReceivedQuantity, LandedUnitCostMinorUnits: in.LandedUnitCostMinorUnits,
		ReferenceType: in.ReferenceType, ReferenceID: in.ReferenceID, OccurredAt: in.OccurredAt,
	})
}

// costLedgerRecorderAdapter bridges application/inventory's local
// costLedgerRecorder interface to application/pms's real
// RecordCostLedgerMovementUseCase -- same cross-domain-avoidance reasoning
// and composition-root placement as skuCostUpdaterAdapter above.
type costLedgerRecorderAdapter struct {
	uc *apppms.RecordCostLedgerMovementUseCase
}

func (a costLedgerRecorderAdapter) Execute(ctx context.Context, in appinventory.CostLedgerMovementInput) error {
	return a.uc.Execute(ctx, apppms.RecordCostLedgerMovementInput{
		SKUID: in.SKUID, QuantityDelta: in.QuantityDelta,
		SourceType:    pmsDomain.SKUCostPriceLedgerSourceType(in.SourceType),
		ReferenceType: in.ReferenceType, ReferenceID: in.ReferenceID, OccurredAt: in.OccurredAt,
	})
}

// Register is the production wiring: creates repos + use cases, then
// mounts the generated routes (matching the OpenAPI spec exactly).
// fileStorage backs the Discard app's optional photo upload -- the same
// adapter cmd/api already builds for pms media (S3Storage/LocalStorage
// satisfy invDomain.FileStorage structurally, no wrapper needed).
func Register(rg gin.IRouter, db *gorm.DB, outboxRepo outbox.Repository, fileStorage invDomain.FileStorage) {
	warehouseRepo := repositories.NewWarehouseRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	settingsRepo := repositories.NewTenantSettingsRepo(db)
	discardRepo := repositories.NewDiscardRecordRepo(db)
	costLedgerRepo := repositories.NewSKUCostPriceLedgerRepo(db)
	batchListerForCost := appinventory.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo)
	skuCostUpdater := skuCostUpdaterAdapter{uc: apppms.NewApplyReceiptCostUseCase(db, skuRepo, costLedgerRepo, batchListerForCost)}
	costLedgerRecorder := costLedgerRecorderAdapter{uc: apppms.NewRecordCostLedgerMovementUseCase(db, skuRepo, costLedgerRepo, batchListerForCost)}
	markDamagedOrLost := appinventory.NewMarkDamagedOrLostUseCase(db, movementRepo, discardRepo, outboxRepo, costLedgerRecorder)

	h := NewHandler(
		appinventory.NewListWarehousesUseCase(db, warehouseRepo),
		appinventory.NewCreateWarehouseUseCase(db, warehouseRepo),
		appinventory.NewUpdateWarehouseUseCase(db, warehouseRepo),
		appinventory.NewDeleteWarehouseUseCase(db, warehouseRepo),
		appinventory.NewCreateBatchUseCase(db, batchRepo),
		appinventory.NewUpdateBatchUseCase(db, batchRepo, outboxRepo),
		appinventory.NewDeleteBatchUseCase(db, batchRepo, movementRepo),
		appinventory.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo),
		appinventory.NewGetSKUStockLevelUseCase(db, movementRepo, batchRepo, settingsRepo),
		appinventory.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo, costLedgerRecorder),
		appinventory.NewRecordRecountUseCase(db, movementRepo, outboxRepo),
		appinventory.NewTransferStockUseCase(db, movementRepo, outboxRepo),
		appinventory.NewSplitCaseIntoUnitsUseCase(db, skuRepo, batchRepo, movementRepo, outboxRepo),
		appinventory.NewPortionFromBulkUseCase(db, skuRepo, movementRepo, outboxRepo),
		markDamagedOrLost,
		appinventory.NewListExpiringBatchesUseCase(db, batchRepo),
		appinventory.NewListStockMovementsUseCase(db, movementRepo),
		appinventory.NewReceiveStockUseCase(db, batchRepo, movementRepo, outboxRepo, skuCostUpdater),
		appinventory.NewListPendingExpiredBatchesUseCase(db, batchRepo),
		appinventory.NewWriteOffExpiredBatchUseCase(db, batchRepo, movementRepo, markDamagedOrLost),
		appinventory.NewExtendBatchExpiryUseCase(db, batchRepo),
		appinventory.NewDismissExpiredBatchUseCase(db, batchRepo),
		appinventory.NewListBatchMovementsUseCase(db, movementRepo),
		appinventory.NewDiscardBatchUseCase(db, batchRepo, markDamagedOrLost, fileStorage),
		appinventory.NewListDiscardsUseCase(db, discardRepo),
	)
	geninventory.RegisterHandlers(rg, h)
}
