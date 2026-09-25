package oms

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genoms "github.com/adirkuhn/commerceos/api/oms"
	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	appoms "github.com/adirkuhn/commerceos/application/oms"
	apppms "github.com/adirkuhn/commerceos/application/pms"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
	"github.com/adirkuhn/commerceos/infrastructure/revolut"
	"github.com/adirkuhn/commerceos/infrastructure/stripe"
)

// ── Use-case interfaces (defined at point of use — ISP) ───────────────────────

type listSalesOrdersUC interface {
	Execute(ctx context.Context, in appoms.ListSalesOrdersInput) (*appoms.ListSalesOrdersResult, error)
}
type getSalesOrderUC interface {
	Execute(ctx context.Context, id string) (*appoms.GetSalesOrderResult, error)
}
type createSalesOrderUC interface {
	Execute(ctx context.Context, in appoms.CreateSalesOrderInput) (*appoms.CreateSalesOrderResult, error)
}
type updateSalesOrderUC interface {
	Execute(ctx context.Context, id string, in appoms.UpdateSalesOrderInput) (*omsDomain.SalesOrder, error)
}
type deleteSalesOrderUC interface {
	Execute(ctx context.Context, id string) error
}
type bulkUpdateSalesOrderStatusUC interface {
	Execute(ctx context.Context, in appoms.BulkUpdateSalesOrderStatusInput) (*appoms.BulkUpdateSalesOrderStatusResult, error)
}
type linkLineItemSKUUC interface {
	Execute(ctx context.Context, in appoms.LinkLineItemSKUInput) (*omsDomain.SalesOrder, error)
}
type retryStockDecrementUC interface {
	Execute(ctx context.Context, salesOrderID string) (*appoms.RetryStockDecrementResult, error)
}
type forceAcceptStockUC interface {
	Execute(ctx context.Context, salesOrderID string) (*appoms.ForceAcceptStockResult, error)
}
type addSalesOrderLineItemUC interface {
	Execute(ctx context.Context, in appoms.AddSalesOrderLineItemInput) (*omsDomain.SalesOrderLineItem, error)
}
type startOrResumePickingUC interface {
	Execute(ctx context.Context, salesOrderID string, actor omsDomain.LockActor) (*omsDomain.PickingSession, error)
}
type getPickingSessionUC interface {
	Execute(ctx context.Context, id string) (*appoms.PickingSessionDetail, error)
}
type touchPickingSessionUC interface {
	Execute(ctx context.Context, sessionID, userID string) error
}
type resetPickingSessionUC interface {
	Execute(ctx context.Context, sessionID string) (*omsDomain.PickingSession, error)
}
type scanPickItemUC interface {
	Execute(ctx context.Context, in appoms.ScanPickItemInput) (*appoms.ScanPickItemResult, error)
}
type substituteLineItemUC interface {
	Execute(ctx context.Context, in appoms.SubstituteLineItemInput) (*appoms.SubstituteLineItemResult, error)
}
type refundLineItemUC interface {
	Execute(ctx context.Context, in appoms.RefundLineItemInput) (*appoms.RefundLineItemResult, error)
}
type resolvePickScanCodeUC interface {
	Execute(ctx context.Context, sessionID, code string) (*appoms.ResolvedScanCode, error)
}
type deletePickScanUC interface {
	Execute(ctx context.Context, sessionID, scanID string) error
}
type pausePickingUC interface {
	Execute(ctx context.Context, id string) (*omsDomain.PickingSession, error)
}
type completePickingUC interface {
	Execute(ctx context.Context, in appoms.CompletePickingInput) (*appoms.CompletePickingResult, error)
}
type listSalesOrderActivitiesUC interface {
	Execute(ctx context.Context, salesOrderID string) ([]omsDomain.SalesOrderActivity, error)
}

// Packing use-case interfaces live in packing_handler.go alongside the
// handler methods that use them (see that file's doc comment for why).

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listSalesOrders            listSalesOrdersUC
	getSalesOrder              getSalesOrderUC
	createSalesOrder           createSalesOrderUC
	updateSalesOrder           updateSalesOrderUC
	deleteSalesOrder           deleteSalesOrderUC
	linkLineItemSKU            linkLineItemSKUUC
	retryStockDecrement        retryStockDecrementUC
	forceAcceptStock           forceAcceptStockUC
	addSalesOrderLineItem      addSalesOrderLineItemUC
	startOrResumePicking       startOrResumePickingUC
	getPickingSession          getPickingSessionUC
	touchPickingSession        touchPickingSessionUC
	resetPickingSession        resetPickingSessionUC
	scanPickItem               scanPickItemUC
	substituteLineItem         substituteLineItemUC
	refundLineItem             refundLineItemUC
	resolvePickScanCode        resolvePickScanCodeUC
	deletePickScan             deletePickScanUC
	pausePicking               pausePickingUC
	completePicking            completePickingUC
	bulkUpdateSalesOrderStatus bulkUpdateSalesOrderStatusUC
	listSalesOrderActivities   listSalesOrderActivitiesUC

	startOrResumePacking startOrResumePackingUC
	getPackingSession    getPackingSessionUC
	touchPackingSession  touchPackingSessionUC
	resetPackingSession  resetPackingSessionUC
	scanPackItem         scanPackItemUC
	deletePackScan       deletePackScanUC
	completePacking      completePackingUC

	buildPickList    buildPickListUC
	listPackingQueue listPackingQueueUC
	markOrderPicked  markOrderPickedUC

	resolvePackScanCode resolvePickScanCodeUC
	locks               SessionLockUseCases
	listPickingQueue    listPackingQueueUC
}

func NewHandler(
	listSalesOrders listSalesOrdersUC, getSalesOrder getSalesOrderUC, createSalesOrder createSalesOrderUC,
	updateSalesOrder updateSalesOrderUC, deleteSalesOrder deleteSalesOrderUC, linkLineItemSKU linkLineItemSKUUC,
	retryStockDecrement retryStockDecrementUC, forceAcceptStock forceAcceptStockUC,
	addSalesOrderLineItem addSalesOrderLineItemUC, startOrResumePicking startOrResumePickingUC,
	getPickingSession getPickingSessionUC, touchPickingSession touchPickingSessionUC, resetPickingSession resetPickingSessionUC,
	scanPickItem scanPickItemUC, substituteLineItem substituteLineItemUC,
	refundLineItem refundLineItemUC,
	resolvePickScanCode resolvePickScanCodeUC,
	deletePickScan deletePickScanUC, pausePicking pausePickingUC, completePicking completePickingUC,
	bulkUpdateSalesOrderStatus bulkUpdateSalesOrderStatusUC,
	listSalesOrderActivities listSalesOrderActivitiesUC,
	startOrResumePacking startOrResumePackingUC, getPackingSession getPackingSessionUC,
	touchPackingSession touchPackingSessionUC, resetPackingSession resetPackingSessionUC,
	scanPackItem scanPackItemUC, deletePackScan deletePackScanUC, completePacking completePackingUC,
	buildPickList buildPickListUC, listPackingQueue listPackingQueueUC, markOrderPicked markOrderPickedUC,
	resolvePackScanCode resolvePickScanCodeUC, locks SessionLockUseCases, listPickingQueue listPackingQueueUC,
) *Handler {
	return &Handler{
		listSalesOrders: listSalesOrders, getSalesOrder: getSalesOrder,
		createSalesOrder: createSalesOrder, updateSalesOrder: updateSalesOrder, deleteSalesOrder: deleteSalesOrder,
		linkLineItemSKU: linkLineItemSKU, retryStockDecrement: retryStockDecrement, forceAcceptStock: forceAcceptStock,
		addSalesOrderLineItem: addSalesOrderLineItem, startOrResumePicking: startOrResumePicking,
		getPickingSession: getPickingSession, touchPickingSession: touchPickingSession, resetPickingSession: resetPickingSession,
		scanPickItem: scanPickItem, substituteLineItem: substituteLineItem,
		refundLineItem:      refundLineItem,
		resolvePickScanCode: resolvePickScanCode,
		deletePickScan:      deletePickScan, pausePicking: pausePicking, completePicking: completePicking,
		bulkUpdateSalesOrderStatus: bulkUpdateSalesOrderStatus,
		listSalesOrderActivities:   listSalesOrderActivities,
		startOrResumePacking:       startOrResumePacking, getPackingSession: getPackingSession,
		touchPackingSession: touchPackingSession, resetPackingSession: resetPackingSession,
		scanPackItem: scanPackItem, deletePackScan: deletePackScan, completePacking: completePacking,
		buildPickList: buildPickList, listPackingQueue: listPackingQueue, markOrderPicked: markOrderPicked,
		resolvePackScanCode: resolvePackScanCode, locks: locks, listPickingQueue: listPickingQueue,
	}
}

var _ genoms.ServerInterface = (*Handler)(nil)

// costLedgerRecorderAdapter bridges application/inventory's local
// costLedgerRecorder interface to application/pms's real
// RecordCostLedgerMovementUseCase -- lives in the composition root because
// application/pms already imports application/inventory (see
// application/pms/recalculate_cost_prices.go's doc comment), so the
// reverse import would cycle. Duplicated per handler package, same
// reasoning as infrastructure/handlers/inventory's own copy.
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

// Register is the production wiring. Callers mount rg under the normal
// human-JWT-authenticated /api/v1 group -- this is staff creating/editing
// sales orders from the admin UI. The POS's own push path
// (ReceivePOSSalesOrderUseCase) is separate, under api/pos, authenticated
// by the POS's lm2 license instead -- see api/oms/generate.go's doc comment.
func Register(rg gin.IRouter, db *gorm.DB, encryptionSecret string) {
	salesOrderRepo := repositories.NewSalesOrderRepo(db)
	lineItemRepo := repositories.NewSalesOrderLineItemRepo(db)
	linkRepo := repositories.NewSalesOrderLineItemStockMovementRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	skuRepo := repositories.NewSKURepo(db)
	warehouseRepo := repositories.NewWarehouseRepo(db)
	costLedgerRecorder := costLedgerRecorderAdapter{uc: apppms.NewRecordCostLedgerMovementUseCase(
		db, skuRepo, repositories.NewSKUCostPriceLedgerRepo(db), appinventory.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo),
	)}
	recordMovement := appinventory.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo, costLedgerRecorder)
	allocateStock := appinventory.NewAllocateSaleStockUseCase(db, movementRepo, batchRepo, outboxRepo)

	pickingSessionRepo := repositories.NewPickingSessionRepo(db)
	pickScanRepo := repositories.NewPickScanRepo(db)
	barcodeAliasRepo := repositories.NewBarcodeAliasRepo(db)
	scaleCodeAliasRepo := repositories.NewScaleCodeAliasRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	productRepo := repositories.NewProductRepo(db)
	addLineItem := appoms.NewAddSalesOrderLineItemUseCase(db, salesOrderRepo, lineItemRepo, skuRepo)
	updateSalesOrder := appoms.NewUpdateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, skuRepo, allocateStock, recordMovement, outboxRepo)

	packingSessionRepo := repositories.NewPackingSessionRepo(db)
	packScanRepo := repositories.NewPackScanRepo(db)

	permissionRepo := repositories.NewGroupPermissionRepo(db)

	connRepo := repositories.NewConnectionRepo(db)
	refundClientResolver := appoms.NewPaymentRefundClientResolver(db, connRepo, stripe.NewClientFactory(), revolut.NewClientFactory(), encryptionSecret)
	// allocateStock/recordMovement's roles swap here vs. their names --
	// allocateStock (AllocateSaleStockUseCase) is the recordSaleMovementUC
	// param, recordMovement (RecordStockMovementUseCase) the movementReverser.
	orderChangeApplier := appoms.NewOrderChangeApplier(db, salesOrderRepo, lineItemRepo, linkRepo, addLineItem, allocateStock, recordMovement, allocateStock, skuRepo, outboxRepo, refundClientResolver)

	h := NewHandler(
		appoms.NewListSalesOrdersUseCase(db, salesOrderRepo),
		appoms.NewGetSalesOrderUseCase(db, salesOrderRepo, lineItemRepo),
		appoms.NewCreateSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, skuRepo, allocateStock),
		updateSalesOrder,
		appoms.NewDeleteSalesOrderUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, recordMovement),
		appoms.NewLinkLineItemSKUUseCase(db, salesOrderRepo, lineItemRepo, skuRepo),
		appoms.NewRetryStockDecrementUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, skuRepo, allocateStock),
		appoms.NewForceAcceptStockUseCase(db, salesOrderRepo, lineItemRepo, linkRepo, skuRepo, allocateStock),
		addLineItem,
		appoms.NewStartOrResumePickingUseCase(db, salesOrderRepo, pickingSessionRepo, packingSessionRepo, packScanRepo),
		appoms.NewGetPickingSessionUseCase(db, pickingSessionRepo, lineItemRepo, pickScanRepo, skuRepo, variantRepo),
		appoms.NewTouchPickingSessionUseCase(db, pickingSessionRepo),
		appoms.NewResetPickingSessionUseCase(db, pickingSessionRepo, pickScanRepo),
		appoms.NewScanPickItemUseCase(db, pickingSessionRepo, pickScanRepo, lineItemRepo, salesOrderRepo, skuRepo, barcodeAliasRepo, scaleCodeAliasRepo, variantRepo, productRepo),
		appoms.NewSubstituteLineItemUseCase(db, packingSessionRepo, packScanRepo, lineItemRepo),
		appoms.NewRefundLineItemUseCase(db, packingSessionRepo, packScanRepo, lineItemRepo, permissionRepo),
		appoms.NewResolvePickScanCodeUseCase(db, pickingSessionRepo, lineItemRepo, skuRepo, barcodeAliasRepo, scaleCodeAliasRepo, variantRepo, productRepo),
		appoms.NewDeletePickScanUseCase(db, pickingSessionRepo, pickScanRepo),
		appoms.NewPausePickingUseCase(db, pickingSessionRepo),
		// allocateStock/recordMovement's roles swap here vs. their names --
		// allocateStock (AllocateSaleStockUseCase) is the recordSaleMovementUC
		// param, recordMovement (RecordStockMovementUseCase) is the
		// movementReverser param. Same swap UpdateSalesOrderUseCase's own
		// wiring above already has -- see CompletePickingUseCase's doc.
		appoms.NewCompletePickingUseCase(db, pickingSessionRepo, salesOrderRepo, lineItemRepo, linkRepo, pickScanRepo, addLineItem, allocateStock, recordMovement, allocateStock, skuRepo, outboxRepo, refundClientResolver),
		appoms.NewBulkUpdateSalesOrderStatusUseCase(updateSalesOrder),
		appoms.NewListSalesOrderActivitiesUseCase(db, repositories.NewSalesOrderActivityRepo(db)),
		appoms.NewStartOrResumePackingUseCase(db, salesOrderRepo, packingSessionRepo, pickingSessionRepo, pickScanRepo),
		appoms.NewGetPackingSessionUseCase(db, packingSessionRepo, lineItemRepo, packScanRepo, skuRepo, variantRepo),
		appoms.NewTouchPackingSessionUseCase(db, packingSessionRepo),
		appoms.NewResetPackingSessionUseCase(db, packingSessionRepo, packScanRepo),
		appoms.NewScanPackItemUseCase(db, packingSessionRepo, packScanRepo, lineItemRepo, skuRepo, barcodeAliasRepo, scaleCodeAliasRepo, variantRepo, productRepo),
		appoms.NewDeletePackScanUseCase(db, packingSessionRepo, packScanRepo),
		appoms.NewCompletePackingUseCase(db, packingSessionRepo, salesOrderRepo, lineItemRepo, packScanRepo, orderChangeApplier),
		appoms.NewBuildPickListUseCase(db, salesOrderRepo, lineItemRepo),
		appoms.NewListPackingQueueUseCase(db, salesOrderRepo, pickingSessionRepo, packingSessionRepo, packScanRepo),
		appoms.NewMarkOrderPickedUseCase(db, salesOrderRepo, packingSessionRepo, packScanRepo, pickingSessionRepo, pickScanRepo, repositories.NewSalesOrderActivityRepo(db)),
		appoms.NewResolvePackScanCodeUseCase(db, packingSessionRepo, lineItemRepo, skuRepo, barcodeAliasRepo, scaleCodeAliasRepo, variantRepo, productRepo),
		SessionLockUseCases{
			ReleasePacking:  appoms.NewReleasePackingSessionLockUseCase(db, packingSessionRepo),
			ReleasePicking:  appoms.NewReleasePickingSessionLockUseCase(db, pickingSessionRepo),
			TakeOverPacking: appoms.NewTakeOverPackingSessionUseCase(db, packingSessionRepo),
			TakeOverPicking: appoms.NewTakeOverPickingSessionUseCase(db, pickingSessionRepo),
		},
		appoms.NewListPickingQueueUseCase(db, salesOrderRepo, pickingSessionRepo, packingSessionRepo, packScanRepo),
	)
	genoms.RegisterHandlers(rg, h)
}
