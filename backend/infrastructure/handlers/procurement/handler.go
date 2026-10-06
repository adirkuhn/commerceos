package procurement

import (
	"context"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	genprocurement "github.com/adirkuhn/commerceos/api/procurement"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	suppliersapp "github.com/adirkuhn/commerceos/application/suppliers"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/repositories"
)

// skuCostUpdaterAdapter bridges application/inventory's local
// skuCostUpdaterForReceiveStock interface to application/pms's real
// ApplyReceiptCostUseCase -- lives in the composition root (not in either
// application package) because application/pms already imports
// application/inventory (see application/pms/recalculate_cost_prices.go's
// own doc comment on that exact cycle), so the reverse import would cycle.
type skuCostUpdaterAdapter struct {
	uc *pmsapp.ApplyReceiptCostUseCase
}

func (a skuCostUpdaterAdapter) Execute(ctx context.Context, in invapp.SKUCostReceiptInput) error {
	return a.uc.Execute(ctx, pmsapp.ApplyReceiptCostInput{
		SKUID: in.SKUID, ReceivedQuantity: in.ReceivedQuantity, LandedUnitCostMinorUnits: in.LandedUnitCostMinorUnits,
		ReferenceType: in.ReferenceType, ReferenceID: in.ReferenceID, OccurredAt: in.OccurredAt,
	})
}

// costLedgerRecorderAdapter bridges application/inventory's local
// costLedgerRecorder interface to application/pms's real
// RecordCostLedgerMovementUseCase -- same cross-domain-avoidance reasoning
// and composition-root placement as skuCostUpdaterAdapter above.
type costLedgerRecorderAdapter struct {
	uc *pmsapp.RecordCostLedgerMovementUseCase
}

func (a costLedgerRecorderAdapter) Execute(ctx context.Context, in invapp.CostLedgerMovementInput) error {
	return a.uc.Execute(ctx, pmsapp.RecordCostLedgerMovementInput{
		SKUID: in.SKUID, QuantityDelta: in.QuantityDelta,
		SourceType:    pmsDomain.SKUCostPriceLedgerSourceType(in.SourceType),
		ReferenceType: in.ReferenceType, ReferenceID: in.ReferenceID, OccurredAt: in.OccurredAt,
	})
}

// ── Use-case interfaces (defined at point of use — ISP) ──────────────────────

type listPurchaseOrdersUC interface {
	Execute(ctx context.Context) ([]procurementDomain.PurchaseOrder, error)
}
type createPurchaseOrderUC interface {
	Execute(ctx context.Context, in appprocurement.CreatePurchaseOrderInput) (*procurementDomain.PurchaseOrder, error)
}
type getPurchaseOrderUC interface {
	Execute(ctx context.Context, id string) (*appprocurement.PurchaseOrderDetail, error)
}
type updatePurchaseOrderUC interface {
	Execute(ctx context.Context, in appprocurement.UpdatePurchaseOrderInput) (*procurementDomain.PurchaseOrder, error)
}
type deletePurchaseOrderUC interface {
	Execute(ctx context.Context, id string) error
}
type addLineItemUC interface {
	Execute(ctx context.Context, in appprocurement.AddPurchaseOrderLineItemInput) (*procurementDomain.PurchaseOrderLineItem, error)
}
type updateLineItemUC interface {
	Execute(ctx context.Context, in appprocurement.UpdatePurchaseOrderLineItemInput) (*procurementDomain.PurchaseOrderLineItem, error)
}
type removeLineItemUC interface {
	Execute(ctx context.Context, id string) error
}
type attachDocumentToPurchaseOrderUC interface {
	Execute(ctx context.Context, in appprocurement.AttachDocumentToPurchaseOrderInput) (*procurementDomain.DocumentParsingJob, error)
}
type deletePurchaseOrderDocumentUC interface {
	Execute(ctx context.Context, id string) error
}
type retryDocumentParsingJobUC interface {
	Execute(ctx context.Context, documentID string) (*procurementDomain.DocumentParsingJob, error)
}
type getDocumentParsingJobUC interface {
	Execute(ctx context.Context, id string) (*procurementDomain.DocumentParsingJob, error)
}
type getReceivingParsingJobUC interface {
	Execute(ctx context.Context, id string) (*procurementDomain.ReceivingParsingJob, error)
}
type createPurchaseOrderFromDocumentUC interface {
	Execute(ctx context.Context, in appprocurement.CreatePurchaseOrderFromDocumentInput) (*procurementDomain.DocumentParsingJob, error)
}
type linkLineItemToSKUUC interface {
	Execute(ctx context.Context, in appprocurement.LinkLineItemToSKUInput) (*procurementDomain.PurchaseOrderLineItem, error)
}
type createProductAndLinkLineItemUC interface {
	Execute(ctx context.Context, in appprocurement.CreateProductAndLinkLineItemInput) (*procurementDomain.PurchaseOrderLineItem, error)
}
type createSupplierAndAssignUC interface {
	Execute(ctx context.Context, in appprocurement.CreateSupplierAndAssignToPurchaseOrderInput) (*procurementDomain.PurchaseOrder, error)
}
type uploadInvoiceUC interface {
	Execute(ctx context.Context, in appprocurement.UploadInvoiceInput) (*procurementDomain.ReceivingParsingJob, error)
}
type listReceivingRecordsUC interface {
	Execute(ctx context.Context, purchaseOrderID string) ([]procurementDomain.ReceivingRecord, error)
}
type getReceivingRecordUC interface {
	Execute(ctx context.Context, id string) (*appprocurement.ReceivingRecordDetail, error)
}
type completeReceivingUC interface {
	Execute(ctx context.Context, receivingRecordID string) (*procurementDomain.ReceivingRecord, error)
}
type startOrResumeBookingUC interface {
	Execute(ctx context.Context, in appprocurement.StartOrResumeBookingInput) (*procurementDomain.Booking, error)
}
type getBookingUC interface {
	Execute(ctx context.Context, id string) (*appprocurement.BookingDetail, error)
}
type scanBookingItemUC interface {
	Execute(ctx context.Context, in appprocurement.ScanBookingItemInput) (*appprocurement.ScanBookingItemResult, error)
}
type resolveBookingScanCodeUC interface {
	Execute(ctx context.Context, bookingID, code string) (*appprocurement.ResolveBookingScanCodeResult, error)
}
type deleteBookingScanUC interface {
	Execute(ctx context.Context, bookingID, scanID string) error
}
type pauseBookingUC interface {
	Execute(ctx context.Context, id string) (*procurementDomain.Booking, error)
}
type completeBookingUC interface {
	Execute(ctx context.Context, id string, completedBy string) (*appprocurement.CompleteBookingResult, error)
}
type listBookingDiscrepanciesUC interface {
	Execute(ctx context.Context) (*appprocurement.BookingDiscrepancyReport, error)
}
type getLandedUnitCostUC interface {
	Execute(ctx context.Context, lineItemID string) (int64, error)
}
type acknowledgeReceivingDiscrepancyUC interface {
	Execute(ctx context.Context, discrepancyID string) (*procurementDomain.ReceivingDiscrepancy, error)
}
type linkReceivingDiscrepanciesUC interface {
	Execute(ctx context.Context, unexpectedDiscrepancyID, missingDiscrepancyID string) error
}
type flagDamagedLineItemUC interface {
	Execute(ctx context.Context, receivingRecordID, lineItemID string, damagedQty int64, note string) (*procurementDomain.ReceivingDiscrepancy, error)
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	listPurchaseOrders              listPurchaseOrdersUC
	createPurchaseOrder             createPurchaseOrderUC
	getPurchaseOrder                getPurchaseOrderUC
	updatePurchaseOrder             updatePurchaseOrderUC
	deletePurchaseOrder             deletePurchaseOrderUC
	addLineItem                     addLineItemUC
	updateLineItem                  updateLineItemUC
	removeLineItem                  removeLineItemUC
	attachDocument                  attachDocumentToPurchaseOrderUC
	getDocumentParsingJob           getDocumentParsingJobUC
	getReceivingParsingJob          getReceivingParsingJobUC
	createPurchaseOrderFromDoc      createPurchaseOrderFromDocumentUC
	linkLineItemToSKU               linkLineItemToSKUUC
	createProductAndLink            createProductAndLinkLineItemUC
	createSupplierAndAssign         createSupplierAndAssignUC
	uploadInvoice                   uploadInvoiceUC
	listReceivingRecords            listReceivingRecordsUC
	getReceivingRecord              getReceivingRecordUC
	completeReceiving               completeReceivingUC
	startOrResumeBooking            startOrResumeBookingUC
	getBooking                      getBookingUC
	scanBookingItem                 scanBookingItemUC
	resolveBookingScanCode          resolveBookingScanCodeUC
	deleteBookingScan               deleteBookingScanUC
	pauseBooking                    pauseBookingUC
	completeBooking                 completeBookingUC
	deletePurchaseOrderDocument     deletePurchaseOrderDocumentUC
	retryDocumentParsingJob         retryDocumentParsingJobUC
	listBookingDiscrepancies        listBookingDiscrepanciesUC
	getLandedUnitCost               getLandedUnitCostUC
	acknowledgeReceivingDiscrepancy acknowledgeReceivingDiscrepancyUC
	linkReceivingDiscrepancies      linkReceivingDiscrepanciesUC
	flagDamagedLineItem             flagDamagedLineItemUC
}

// NewHandler constructs a Handler with injected use cases (useful for testing).
func NewHandler(
	listPurchaseOrders listPurchaseOrdersUC,
	createPurchaseOrder createPurchaseOrderUC,
	getPurchaseOrder getPurchaseOrderUC,
	updatePurchaseOrder updatePurchaseOrderUC,
	deletePurchaseOrder deletePurchaseOrderUC,
	addLineItem addLineItemUC,
	updateLineItem updateLineItemUC,
	removeLineItem removeLineItemUC,
	attachDocument attachDocumentToPurchaseOrderUC,
	getDocumentParsingJob getDocumentParsingJobUC,
	getReceivingParsingJob getReceivingParsingJobUC,
	createPurchaseOrderFromDoc createPurchaseOrderFromDocumentUC,
	linkLineItemToSKU linkLineItemToSKUUC,
	createProductAndLink createProductAndLinkLineItemUC,
	createSupplierAndAssign createSupplierAndAssignUC,
	uploadInvoice uploadInvoiceUC,
	listReceivingRecords listReceivingRecordsUC,
	getReceivingRecord getReceivingRecordUC,
	completeReceiving completeReceivingUC,
	startOrResumeBooking startOrResumeBookingUC,
	getBooking getBookingUC,
	scanBookingItem scanBookingItemUC,
	resolveBookingScanCode resolveBookingScanCodeUC,
	deleteBookingScan deleteBookingScanUC,
	pauseBooking pauseBookingUC,
	completeBooking completeBookingUC,
	deletePurchaseOrderDocument deletePurchaseOrderDocumentUC,
	retryDocumentParsingJob retryDocumentParsingJobUC,
	listBookingDiscrepancies listBookingDiscrepanciesUC,
	getLandedUnitCost getLandedUnitCostUC,
	acknowledgeReceivingDiscrepancy acknowledgeReceivingDiscrepancyUC,
	linkReceivingDiscrepancies linkReceivingDiscrepanciesUC,
	flagDamagedLineItem flagDamagedLineItemUC,
) *Handler {
	return &Handler{
		getLandedUnitCost:               getLandedUnitCost,
		acknowledgeReceivingDiscrepancy: acknowledgeReceivingDiscrepancy,
		linkReceivingDiscrepancies:      linkReceivingDiscrepancies,
		flagDamagedLineItem:             flagDamagedLineItem,
		listPurchaseOrders:              listPurchaseOrders, createPurchaseOrder: createPurchaseOrder,
		getPurchaseOrder: getPurchaseOrder, updatePurchaseOrder: updatePurchaseOrder, deletePurchaseOrder: deletePurchaseOrder,
		addLineItem: addLineItem, updateLineItem: updateLineItem, removeLineItem: removeLineItem,
		attachDocument: attachDocument, getDocumentParsingJob: getDocumentParsingJob, getReceivingParsingJob: getReceivingParsingJob,
		createPurchaseOrderFromDoc: createPurchaseOrderFromDoc,
		linkLineItemToSKU:          linkLineItemToSKU, createProductAndLink: createProductAndLink,
		createSupplierAndAssign:     createSupplierAndAssign,
		uploadInvoice:               uploadInvoice,
		listReceivingRecords:        listReceivingRecords,
		getReceivingRecord:          getReceivingRecord,
		completeReceiving:           completeReceiving,
		startOrResumeBooking:        startOrResumeBooking,
		getBooking:                  getBooking,
		scanBookingItem:             scanBookingItem,
		resolveBookingScanCode:      resolveBookingScanCode,
		deleteBookingScan:           deleteBookingScan,
		pauseBooking:                pauseBooking,
		completeBooking:             completeBooking,
		deletePurchaseOrderDocument: deletePurchaseOrderDocument,
		retryDocumentParsingJob:     retryDocumentParsingJob,
		listBookingDiscrepancies:    listBookingDiscrepancies,
	}
}

var _ genprocurement.ServerInterface = (*Handler)(nil)

// Register is the production wiring: creates repos + use cases, then
// mounts the generated routes (matching the OpenAPI spec exactly).
// fileStorage is the same pms.FileStorage adapter product media uploads use
// (see cmd/api/main.go's buildFileStorage) -- document uploads reuse it
// as-is, no new storage adapter needed.
func Register(rg gin.IRouter, db *gorm.DB, fileStorage pmsDomain.FileStorage) {
	poRepo := repositories.NewPurchaseOrderRepo(db)
	lineItemRepo := repositories.NewPurchaseOrderLineItemRepo(db)
	documentRepo := repositories.NewPurchaseOrderDocumentRepo(db)
	jobRepo := repositories.NewDocumentParsingJobRepo(db)
	aliasRepo := repositories.NewSupplierProductAliasRepo(db)
	productRepo := repositories.NewProductRepo(db)
	variantRepo := repositories.NewVariantRepo(db)
	skuRepo := repositories.NewSKURepo(db)
	supplierRepo := repositories.NewSupplierRepo(db)
	receivingRecordRepo := repositories.NewReceivingRecordRepo(db)
	receivingJobRepo := repositories.NewReceivingParsingJobRepo(db)
	discrepancyRepo := repositories.NewReceivingDiscrepancyRepo(db)
	bookingRepo := repositories.NewBookingRepo(db)
	bookingScanRepo := repositories.NewBookingScanRepo(db)
	barcodeAliasRepo := repositories.NewBarcodeAliasRepo(db)
	scaleCodeAliasRepo := repositories.NewScaleCodeAliasRepo(db)
	batchRepo := repositories.NewBatchRepo(db)
	movementRepo := repositories.NewStockMovementRepo(db)
	discardRepo := repositories.NewDiscardRecordRepo(db)
	outboxRepo := repositories.NewOutboxRepository(db)
	warehouseRepo := repositories.NewWarehouseRepo(db)

	costLedgerRepo := repositories.NewSKUCostPriceLedgerRepo(db)
	batchListerForCost := invapp.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo)
	skuCostUpdater := skuCostUpdaterAdapter{uc: pmsapp.NewApplyReceiptCostUseCase(db, skuRepo, costLedgerRepo, batchListerForCost)}
	costLedgerRecorder := costLedgerRecorderAdapter{uc: pmsapp.NewRecordCostLedgerMovementUseCase(db, skuRepo, costLedgerRepo, batchListerForCost)}

	createPurchaseOrderUC := appprocurement.NewCreatePurchaseOrderUseCase(db, poRepo, outboxRepo)
	attachDocumentUC := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, lineItemRepo, jobRepo, fileStorage, outboxRepo)
	linkLineItemToSKUUC := appprocurement.NewLinkLineItemToSKUUseCase(db, poRepo, lineItemRepo, aliasRepo, outboxRepo)

	h := NewHandler(
		appprocurement.NewListPurchaseOrdersUseCase(db, poRepo, lineItemRepo),
		createPurchaseOrderUC,
		appprocurement.NewGetPurchaseOrderUseCase(db, poRepo, lineItemRepo, documentRepo, skuRepo, variantRepo, productRepo),
		appprocurement.NewUpdatePurchaseOrderUseCase(db, poRepo, lineItemRepo, aliasRepo, outboxRepo),
		appprocurement.NewDeletePurchaseOrderUseCase(db, poRepo, documentRepo, outboxRepo),
		appprocurement.NewAddPurchaseOrderLineItemUseCase(db, lineItemRepo, outboxRepo),
		appprocurement.NewUpdatePurchaseOrderLineItemUseCase(db, lineItemRepo, outboxRepo),
		appprocurement.NewRemovePurchaseOrderLineItemUseCase(db, lineItemRepo, outboxRepo),
		attachDocumentUC,
		appprocurement.NewGetDocumentParsingJobUseCase(db, jobRepo),
		appprocurement.NewGetReceivingParsingJobUseCase(db, receivingJobRepo),
		appprocurement.NewCreatePurchaseOrderFromDocumentUseCase(createPurchaseOrderUC, attachDocumentUC),
		linkLineItemToSKUUC,
		appprocurement.NewCreateProductAndLinkLineItemUseCase(
			pmsapp.NewCreateProductUseCase(db, productRepo, outboxRepo),
			pmsapp.NewCreateVariantUseCase(db, productRepo, variantRepo, outboxRepo),
			pmsapp.NewCreateSKUUseCase(db, variantRepo, skuRepo, repositories.NewSKUPriceHistoryRepo(db), outboxRepo),
			linkLineItemToSKUUC,
		),
		appprocurement.NewCreateSupplierAndAssignToPurchaseOrderUseCase(
			db, supplierRepo, suppliersapp.NewCreateSupplierUseCase(db, supplierRepo, outboxRepo), poRepo, lineItemRepo, aliasRepo, outboxRepo,
		),
		appprocurement.NewUploadInvoiceUseCase(db, poRepo, receivingRecordRepo, receivingJobRepo, fileStorage),
		appprocurement.NewListReceivingRecordsUseCase(db, receivingRecordRepo),
		appprocurement.NewGetReceivingRecordUseCase(db, receivingRecordRepo, discrepancyRepo),
		appprocurement.NewCompleteReceivingUseCase(db, receivingRecordRepo),
		appprocurement.NewStartOrResumeBookingUseCase(db, poRepo, bookingRepo),
		appprocurement.NewGetBookingUseCase(db, bookingRepo, lineItemRepo, bookingScanRepo, skuRepo, variantRepo, productRepo),
		appprocurement.NewScanBookingItemUseCase(db, bookingRepo, poRepo, bookingScanRepo, skuRepo, barcodeAliasRepo, scaleCodeAliasRepo, lineItemRepo),
		appprocurement.NewResolveBookingScanCodeUseCase(db, bookingRepo, poRepo, skuRepo, barcodeAliasRepo, scaleCodeAliasRepo, lineItemRepo, variantRepo, productRepo),
		appprocurement.NewDeleteBookingScanUseCase(db, bookingRepo, bookingScanRepo),
		appprocurement.NewPauseBookingUseCase(db, bookingRepo),
		appprocurement.NewCompleteBookingUseCase(
			db, bookingRepo, poRepo, lineItemRepo, bookingScanRepo,
			invapp.NewReceiveStockUseCase(db, batchRepo, movementRepo, outboxRepo, skuCostUpdater),
			invapp.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo, costLedgerRecorder),
			invapp.NewMarkDamagedOrLostUseCase(db, movementRepo, discardRepo, outboxRepo, costLedgerRecorder),
		),
		appprocurement.NewDeletePurchaseOrderDocumentUseCase(db, documentRepo, fileStorage),
		appprocurement.NewRetryDocumentParsingJobUseCase(db, documentRepo, jobRepo),
		appprocurement.NewListBookingDiscrepanciesUseCase(db, bookingRepo, lineItemRepo, bookingScanRepo, skuRepo, variantRepo, productRepo),
		appprocurement.NewGetLandedUnitCostUseCase(db, lineItemRepo),
		appprocurement.NewAcknowledgeReceivingDiscrepancyUseCase(db, discrepancyRepo),
		appprocurement.NewLinkReceivingDiscrepanciesUseCase(db, discrepancyRepo),
		appprocurement.NewFlagDamagedLineItemUseCase(db, receivingRecordRepo, lineItemRepo, discrepancyRepo),
	)
	genprocurement.RegisterHandlers(rg, h)
}
