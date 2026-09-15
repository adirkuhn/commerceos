package procurement

import (
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"gorm.io/gorm"

	genprocurement "github.com/adirkuhn/commerceos/api/procurement"
	invapp "github.com/adirkuhn/commerceos/application/inventory"
	pmsapp "github.com/adirkuhn/commerceos/application/pms"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	suppliersapp "github.com/adirkuhn/commerceos/application/suppliers"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/adirkuhn/commerceos/infrastructure/middleware"
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

// genDateToTimePtr/timePtrToGenDate convert between openapi_types.Date
// (date-only, no time-of-day) and *time.Time -- same helpers duplicated in
// infrastructure/handlers/inventory and .../pms; kept local rather than
// shared since none of these handler packages otherwise depend on each other.
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

func lineTypeOr(t *genprocurement.LineItemType) procurementDomain.LineItemType {
	if t == nil {
		return ""
	}
	return procurementDomain.LineItemType(*t)
}

func strPtrOr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// strPtrOrNil is strPtrOr's inverse -- empty string becomes nil, for
// mapping a domain string field back onto an omitempty response field.
func strPtrOrNil(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// int64PtrOr unwraps an optional integer request field to 0 -- the
// use-case layer treats 0 as "not specified, use the default" for fields
// like UnitsPerPurchasedItem (see AddPurchaseOrderLineItemUseCase).
func int64PtrOr(v *int64) int64 {
	if v == nil {
		return 0
	}
	return *v
}

// stringPtrOr unwraps an optional string request field to "" -- e.g.
// FlagDamagedLineItemRequest.Note.
func stringPtrOr(v *string) string {
	if v == nil {
		return ""
	}
	return *v
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

	skuCostUpdater := skuCostUpdaterAdapter{uc: pmsapp.NewApplyReceiptCostUseCase(
		db, skuRepo, repositories.NewSKUCostPriceLedgerRepo(db), invapp.NewListBatchesUseCase(db, batchRepo, movementRepo, warehouseRepo),
	)}

	createPurchaseOrderUC := appprocurement.NewCreatePurchaseOrderUseCase(db, poRepo)
	attachDocumentUC := appprocurement.NewAttachDocumentToPurchaseOrderUseCase(db, poRepo, documentRepo, lineItemRepo, jobRepo, fileStorage)
	linkLineItemToSKUUC := appprocurement.NewLinkLineItemToSKUUseCase(db, poRepo, lineItemRepo, aliasRepo)

	h := NewHandler(
		appprocurement.NewListPurchaseOrdersUseCase(db, poRepo, lineItemRepo),
		createPurchaseOrderUC,
		appprocurement.NewGetPurchaseOrderUseCase(db, poRepo, lineItemRepo, documentRepo, skuRepo, variantRepo, productRepo),
		appprocurement.NewUpdatePurchaseOrderUseCase(db, poRepo, lineItemRepo, aliasRepo),
		appprocurement.NewDeletePurchaseOrderUseCase(db, poRepo, documentRepo),
		appprocurement.NewAddPurchaseOrderLineItemUseCase(db, lineItemRepo),
		appprocurement.NewUpdatePurchaseOrderLineItemUseCase(db, lineItemRepo),
		appprocurement.NewRemovePurchaseOrderLineItemUseCase(db, lineItemRepo),
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
			db, supplierRepo, suppliersapp.NewCreateSupplierUseCase(db, supplierRepo), poRepo, lineItemRepo, aliasRepo,
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
			invapp.NewRecordStockMovementUseCase(db, movementRepo, outboxRepo),
			invapp.NewMarkDamagedOrLostUseCase(db, movementRepo, discardRepo, outboxRepo),
		),
		appprocurement.NewDeletePurchaseOrderDocumentUseCase(db, documentRepo, fileStorage),
		appprocurement.NewListBookingDiscrepanciesUseCase(db, bookingRepo, lineItemRepo, bookingScanRepo, skuRepo, variantRepo, productRepo),
		appprocurement.NewGetLandedUnitCostUseCase(db, lineItemRepo),
		appprocurement.NewAcknowledgeReceivingDiscrepancyUseCase(db, discrepancyRepo),
		appprocurement.NewLinkReceivingDiscrepanciesUseCase(db, discrepancyRepo),
		appprocurement.NewFlagDamagedLineItemUseCase(db, receivingRecordRepo, lineItemRepo, discrepancyRepo),
	)
	genprocurement.RegisterHandlers(rg, h)
}

// ── Purchase order route handlers ────────────────────────────────────────────

func (h *Handler) ListPurchaseOrders(c *gin.Context) {
	pos, err := h.listPurchaseOrders.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genprocurement.PurchaseOrder, len(pos))
	for i, po := range pos {
		resp[i] = toGenPurchaseOrder(po)
	}
	c.JSON(http.StatusOK, genprocurement.PurchaseOrderListResponse{Data: &resp})
}

func (h *Handler) CreatePurchaseOrder(c *gin.Context) {
	var req genprocurement.CreatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	po, err := h.createPurchaseOrder.Execute(c.Request.Context(), appprocurement.CreatePurchaseOrderInput{
		SupplierID: &req.SupplierId, WarehouseID: req.WarehouseId, CurrencyCode: req.CurrencyCode,
		ExpectedDate: genDateToTimePtr(req.ExpectedDate), Notes: strPtrOr(req.Notes),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPurchaseOrder(*po)
	c.JSON(http.StatusCreated, genprocurement.PurchaseOrderResponse{Data: &resp})
}

func (h *Handler) GetPurchaseOrder(c *gin.Context, id genprocurement.Id) {
	detail, err := h.getPurchaseOrder.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	lineItems := make([]genprocurement.PurchaseOrderLineItem, len(detail.LineItems))
	for i, li := range detail.LineItems {
		lineItems[i] = toGenLineItem(li)
	}
	documents := make([]genprocurement.PurchaseOrderDocument, len(detail.Documents))
	for i, d := range detail.Documents {
		documents[i] = toGenDocument(d)
	}
	resp := genprocurement.PurchaseOrderDetail{
		PurchaseOrder: toGenPurchaseOrder(detail.PurchaseOrder),
		LineItems:     lineItems,
		Documents:     documents,
	}
	c.JSON(http.StatusOK, genprocurement.PurchaseOrderDetailResponse{Data: &resp})
}

func (h *Handler) UpdatePurchaseOrder(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.UpdatePurchaseOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	po, err := h.updatePurchaseOrder.Execute(c.Request.Context(), appprocurement.UpdatePurchaseOrderInput{
		ID: id, SupplierID: &req.SupplierId, WarehouseID: req.WarehouseId, Status: procurementDomain.PurchaseOrderStatus(req.Status),
		CurrencyCode: req.CurrencyCode, ExpectedDate: genDateToTimePtr(req.ExpectedDate), Notes: strPtrOr(req.Notes),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPurchaseOrder(*po)
	c.JSON(http.StatusOK, genprocurement.PurchaseOrderResponse{Data: &resp})
}

func (h *Handler) DeletePurchaseOrder(c *gin.Context, id genprocurement.Id) {
	err := h.deletePurchaseOrder.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genprocurement.EmptyResponse{})
}

// ── Line item route handlers ─────────────────────────────────────────────────

func (h *Handler) AddPurchaseOrderLineItem(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.AddLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	li, err := h.addLineItem.Execute(c.Request.Context(), appprocurement.AddPurchaseOrderLineItemInput{
		PurchaseOrderID: id, SKUID: req.SkuId, RawSKU: strPtrOr(req.RawSku), RawDescription: req.RawDescription,
		Quantity: req.Quantity, UnitsPerPurchasedItem: int64PtrOr(req.UnitsPerPurchasedItem),
		UnitCostMinorUnits: req.UnitCostMinorUnits, LineType: lineTypeOr(req.LineType),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenLineItem(*li)
	c.JSON(http.StatusCreated, genprocurement.LineItemResponse{Data: &resp})
}

func (h *Handler) UpdatePurchaseOrderLineItem(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.UpdateLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	li, err := h.updateLineItem.Execute(c.Request.Context(), appprocurement.UpdatePurchaseOrderLineItemInput{
		ID: id, SKUID: req.SkuId, RawSKU: strPtrOr(req.RawSku), RawDescription: req.RawDescription,
		Quantity: req.Quantity, UnitsPerPurchasedItem: int64PtrOr(req.UnitsPerPurchasedItem),
		UnitCostMinorUnits: req.UnitCostMinorUnits, LineType: lineTypeOr(req.LineType),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenLineItem(*li)
	c.JSON(http.StatusOK, genprocurement.LineItemResponse{Data: &resp})
}

func (h *Handler) RemovePurchaseOrderLineItem(c *gin.Context, id genprocurement.Id) {
	err := h.removeLineItem.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genprocurement.EmptyResponse{})
}

// ── Document upload route handlers ───────────────────────────────────────────

func (h *Handler) AttachDocumentToPurchaseOrder(c *gin.Context, id genprocurement.Id) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "file is required")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	defer f.Close()

	job, err := h.attachDocument.Execute(c.Request.Context(), appprocurement.AttachDocumentToPurchaseOrderInput{
		PurchaseOrderID: id, FileName: fileHeader.Filename, ContentType: fileHeader.Header.Get("Content-Type"), Data: f,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenDocumentParsingJob(*job)
	c.JSON(http.StatusCreated, genprocurement.DocumentParsingJobResponse{Data: &resp})
}

func (h *Handler) DeletePurchaseOrderDocument(c *gin.Context, id genprocurement.Id) {
	err := h.deletePurchaseOrderDocument.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genprocurement.EmptyResponse{})
}

func (h *Handler) GetDocumentParsingJob(c *gin.Context, id genprocurement.Id) {
	job, err := h.getDocumentParsingJob.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenDocumentParsingJob(*job)
	c.JSON(http.StatusOK, genprocurement.DocumentParsingJobResponse{Data: &resp})
}

func (h *Handler) GetReceivingParsingJob(c *gin.Context, id genprocurement.Id) {
	job, err := h.getReceivingParsingJob.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenReceivingParsingJob(*job)
	c.JSON(http.StatusOK, genprocurement.ReceivingParsingJobResponse{Data: &resp})
}

func (h *Handler) CreatePurchaseOrderFromDocument(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "file is required")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	defer f.Close()

	var supplierID *string
	if v := c.PostForm("supplier_id"); v != "" {
		supplierID = &v
	}
	var warehouseID *string
	if v := c.PostForm("warehouse_id"); v != "" {
		warehouseID = &v
	}
	var expectedDate *time.Time
	if v := c.PostForm("expected_date"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			expectedDate = &parsed
		}
	}

	job, err := h.createPurchaseOrderFromDoc.Execute(c.Request.Context(), appprocurement.CreatePurchaseOrderFromDocumentInput{
		SupplierID: supplierID, WarehouseID: warehouseID, CurrencyCode: c.PostForm("currency_code"),
		ExpectedDate: expectedDate, FileName: fileHeader.Filename, ContentType: fileHeader.Header.Get("Content-Type"), Data: f,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenDocumentParsingJob(*job)
	c.JSON(http.StatusCreated, genprocurement.DocumentParsingJobResponse{Data: &resp})
}

// ── Review UI route handlers ─────────────────────────────────────────────────

func (h *Handler) LinkLineItemToSku(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.LinkLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	li, err := h.linkLineItemToSKU.Execute(c.Request.Context(), appprocurement.LinkLineItemToSKUInput{
		LineItemID: id, SKUID: req.SkuId, UnitsPerPurchasedItem: int64PtrOr(req.UnitsPerPurchasedItem),
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenLineItem(*li)
	c.JSON(http.StatusOK, genprocurement.LineItemResponse{Data: &resp})
}

func (h *Handler) GetLandedUnitCost(c *gin.Context, id genprocurement.Id) {
	cost, err := h.getLandedUnitCost.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genprocurement.LandedUnitCost{UnitCostMinorUnits: cost}
	c.JSON(http.StatusOK, genprocurement.LandedUnitCostResponse{Data: &resp})
}

func (h *Handler) CreateProductAndLinkLineItem(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.CreateProductAndLinkLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	var priceMinorUnits int64
	if req.PriceMinorUnits != nil {
		priceMinorUnits = *req.PriceMinorUnits
	}
	li, err := h.createProductAndLink.Execute(c.Request.Context(), appprocurement.CreateProductAndLinkLineItemInput{
		LineItemID: id, ProductName: req.ProductName, SKUCode: req.SkuCode, Barcode: strPtrOr(req.Barcode),
		PriceMinorUnits: priceMinorUnits, CurrencyCode: req.CurrencyCode,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenLineItem(*li)
	c.JSON(http.StatusOK, genprocurement.LineItemResponse{Data: &resp})
}

func (h *Handler) CreateSupplierAndAssignToPurchaseOrder(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.CreateSupplierAndAssignRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	po, err := h.createSupplierAndAssign.Execute(c.Request.Context(), appprocurement.CreateSupplierAndAssignToPurchaseOrderInput{
		PurchaseOrderID: id, Name: req.Name,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenPurchaseOrder(*po)
	c.JSON(http.StatusOK, genprocurement.PurchaseOrderResponse{Data: &resp})
}

// ── Receiving route handlers ─────────────────────────────────────────────────

func (h *Handler) UploadInvoice(c *gin.Context, id genprocurement.Id) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "file is required")
		return
	}
	f, err := fileHeader.Open()
	if err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	defer f.Close()

	job, err := h.uploadInvoice.Execute(c.Request.Context(), appprocurement.UploadInvoiceInput{
		PurchaseOrderID: id, FileName: fileHeader.Filename, ContentType: fileHeader.Header.Get("Content-Type"), Data: f,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenReceivingParsingJob(*job)
	c.JSON(http.StatusCreated, genprocurement.ReceivingParsingJobResponse{Data: &resp})
}

func (h *Handler) ListReceivingRecords(c *gin.Context, id genprocurement.Id) {
	records, err := h.listReceivingRecords.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := make([]genprocurement.ReceivingRecord, len(records))
	for i, r := range records {
		resp[i] = toGenReceivingRecord(r)
	}
	c.JSON(http.StatusOK, genprocurement.ReceivingRecordListResponse{Data: &resp})
}

func (h *Handler) GetReceivingRecord(c *gin.Context, id genprocurement.Id) {
	detail, err := h.getReceivingRecord.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	discrepancies := make([]genprocurement.ReceivingDiscrepancy, len(detail.Discrepancies))
	for i, d := range detail.Discrepancies {
		discrepancies[i] = toGenReceivingDiscrepancy(d)
	}
	resp := genprocurement.ReceivingRecordDetail{
		ReceivingRecord: toGenReceivingRecord(detail.ReceivingRecord),
		Discrepancies:   discrepancies,
	}
	c.JSON(http.StatusOK, genprocurement.ReceivingRecordDetailResponse{Data: &resp})
}

func (h *Handler) CompleteReceiving(c *gin.Context, id genprocurement.Id) {
	rec, err := h.completeReceiving.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenReceivingRecord(*rec)
	c.JSON(http.StatusOK, genprocurement.ReceivingRecordResponse{Data: &resp})
}

func (h *Handler) AcknowledgeReceivingDiscrepancy(c *gin.Context, id genprocurement.Id) {
	d, err := h.acknowledgeReceivingDiscrepancy.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenReceivingDiscrepancy(*d)
	c.JSON(http.StatusOK, genprocurement.ReceivingDiscrepancyResponse{Data: &resp})
}

func (h *Handler) LinkReceivingDiscrepancies(c *gin.Context) {
	var req genprocurement.LinkReceivingDiscrepanciesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	err := h.linkReceivingDiscrepancies.Execute(c.Request.Context(), req.UnexpectedDiscrepancyId, req.MissingDiscrepancyId)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genprocurement.EmptyResponse{})
}

func (h *Handler) FlagDamagedLineItem(c *gin.Context, id genprocurement.Id, lineItemId genprocurement.LineItemId) {
	var req genprocurement.FlagDamagedLineItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	d, err := h.flagDamagedLineItem.Execute(c.Request.Context(), id, lineItemId, req.DamagedQty, stringPtrOr(req.Note))
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenReceivingDiscrepancy(*d)
	c.JSON(http.StatusCreated, genprocurement.ReceivingDiscrepancyResponse{Data: &resp})
}

// ── Booking (scanner app) route handlers ─────────────────────────────────────

func (h *Handler) StartOrResumeBooking(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.StartOrResumeBookingRequest
	// Body is optional (a resume never needs warehouse_id) -- a missing/
	// empty body is not a binding error, only malformed JSON is.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
			return
		}
	}
	booking, err := h.startOrResumeBooking.Execute(c.Request.Context(), appprocurement.StartOrResumeBookingInput{
		PurchaseOrderID: id, WarehouseID: req.WarehouseId,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenBooking(*booking)
	c.JSON(http.StatusOK, genprocurement.BookingResponse{Data: &resp})
}

func (h *Handler) GetBooking(c *gin.Context, id genprocurement.Id) {
	detail, err := h.getBooking.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	lineItems := make([]genprocurement.PurchaseOrderLineItem, len(detail.LineItems))
	for i, li := range detail.LineItems {
		lineItems[i] = toGenLineItem(li)
	}
	scans := make([]genprocurement.BookingScan, len(detail.Scans))
	for i, s := range detail.Scans {
		scans[i] = toGenBookingScan(s)
	}
	resp := genprocurement.BookingDetail{Booking: toGenBooking(detail.Booking), LineItems: lineItems, Scans: scans}
	c.JSON(http.StatusOK, genprocurement.BookingDetailResponse{Data: &resp})
}

func (h *Handler) ScanBookingItem(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.ScanBookingItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	claims := middleware.ClaimsFromGin(c)
	scannedBy := ""
	if claims != nil {
		scannedBy = claims.UserID
	}
	actionType := procurementDomain.BookingScanActionReceive
	if req.ActionType != nil {
		actionType = procurementDomain.BookingScanActionType(*req.ActionType)
	}
	result, err := h.scanBookingItem.Execute(c.Request.Context(), appprocurement.ScanBookingItemInput{
		BookingID: id, Code: strPtrOr(req.Code), SKUID: strPtrOr(req.SkuId), ActionType: actionType,
		Quantity: req.Quantity, BatchID: req.BatchId,
		LotNumber: strPtrOr(req.LotNumber), ExpiryDate: genDateToTimePtr(req.ExpiryDate), Reason: strPtrOr(req.Reason), ScannedBy: scannedBy,
	})
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genprocurement.ScanBookingItemResult{Scan: toGenBookingScan(result.Scan)}
	c.JSON(http.StatusCreated, genprocurement.ScanBookingItemResultResponse{Data: &resp})
}

func (h *Handler) ResolveBookingScanCode(c *gin.Context, id genprocurement.Id) {
	var req genprocurement.ResolveBookingScanCodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
		return
	}
	result, err := h.resolveBookingScanCode.Execute(c.Request.Context(), id, req.Code)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := genprocurement.ResolveBookingScanCodeResult{SkuId: result.SKUID, ProductName: result.ProductName, UnitsPerScan: result.UnitsPerScan, WeightGrams: result.WeightGrams}
	c.JSON(http.StatusOK, genprocurement.ResolveBookingScanCodeResultResponse{Data: &resp})
}

func (h *Handler) DeleteBookingScan(c *gin.Context, id genprocurement.Id, scanId genprocurement.ScanId) {
	err := h.deleteBookingScan.Execute(c.Request.Context(), id, scanId)
	if err := mapError(c, err); err != nil {
		return
	}
	c.JSON(http.StatusOK, genprocurement.EmptyResponse{})
}

func (h *Handler) PauseBooking(c *gin.Context, id genprocurement.Id) {
	booking, err := h.pauseBooking.Execute(c.Request.Context(), id)
	if err := mapError(c, err); err != nil {
		return
	}
	resp := toGenBooking(*booking)
	c.JSON(http.StatusOK, genprocurement.BookingResponse{Data: &resp})
}

func (h *Handler) CompleteBooking(c *gin.Context, id genprocurement.Id) {
	claims := middleware.ClaimsFromGin(c)
	completedBy := ""
	if claims != nil {
		completedBy = claims.UserID
	}
	result, err := h.completeBooking.Execute(c.Request.Context(), id, completedBy)
	if err := mapError(c, err); err != nil {
		return
	}
	missing := make([]genprocurement.MissingBookingItem, len(result.MissingItems))
	for i, m := range result.MissingItems {
		missing[i] = genprocurement.MissingBookingItem{
			LineItemId: m.LineItemID, RawDescription: m.RawDescription, ExpectedQty: m.ExpectedQty, ScannedQty: m.ScannedQty,
		}
	}
	resp := genprocurement.CompleteBookingResult{Booking: toGenBooking(result.Booking), MissingItems: missing}
	c.JSON(http.StatusOK, genprocurement.CompleteBookingResultResponse{Data: &resp})
}

func (h *Handler) ListBookingDiscrepancies(c *gin.Context) {
	report, err := h.listBookingDiscrepancies.Execute(c.Request.Context())
	if err := mapError(c, err); err != nil {
		return
	}
	missing := make([]genprocurement.MissingBookingItemReport, len(report.MissingItems))
	for i, m := range report.MissingItems {
		missing[i] = genprocurement.MissingBookingItemReport{
			BookingId: m.BookingID, PurchaseOrderId: m.PurchaseOrderID, CompletedAt: m.CompletedAt,
			LineItemId: m.LineItemID, SkuCode: m.SKUCode, ProductName: m.ProductName,
			RawDescription: m.RawDescription, ExpectedQty: m.ExpectedQty, ScannedQty: m.ScannedQty, MissingQty: m.MissingQty,
		}
	}
	damagedLost := make([]genprocurement.DamagedLostBookingItemReport, len(report.DamagedLost))
	for i, d := range report.DamagedLost {
		damagedLost[i] = genprocurement.DamagedLostBookingItemReport{
			BookingId: d.BookingID, PurchaseOrderId: d.PurchaseOrderID, CompletedAt: d.CompletedAt,
			ScanId: d.ScanID, SkuId: d.SKUID, SkuCode: d.SKUCode, ProductName: d.ProductName,
			ActionType: genprocurement.DamagedLostBookingItemReportActionType(d.ActionType),
			Quantity:   d.Quantity, Reason: d.Reason, ScannedAt: d.ScannedAt, ScannedBy: d.ScannedBy,
		}
	}
	resp := genprocurement.BookingDiscrepancyReport{MissingItems: missing, DamagedLost: damagedLost}
	c.JSON(http.StatusOK, genprocurement.BookingDiscrepancyReportResponse{Data: &resp})
}

// ── Mappers + error handling ─────────────────────────────────────────────────

func toGenPurchaseOrder(po procurementDomain.PurchaseOrder) genprocurement.PurchaseOrder {
	return genprocurement.PurchaseOrder{
		Id: po.ID, TenantId: po.TenantID, SupplierId: po.SupplierID, WarehouseId: po.WarehouseID,
		Status: genprocurement.PurchaseOrderStatus(po.Status), CurrencyCode: po.CurrencyCode,
		ExpectedDate: timePtrToGenDate(po.ExpectedDate), Notes: po.Notes,
		SuggestedSupplierName: &po.SuggestedSupplierName, SuggestedCurrencyCode: &po.SuggestedCurrencyCode,
		IsPotentialDuplicate: po.IsPotentialDuplicate, DuplicateOfPurchaseOrderId: po.DuplicateOfPurchaseOrderID,
		TotalMinorUnits: po.TotalMinorUnits,
		CreatedAt:       po.CreatedAt, UpdatedAt: po.UpdatedAt,
	}
}

func toGenLineItem(li procurementDomain.PurchaseOrderLineItem) genprocurement.PurchaseOrderLineItem {
	// LineType's zero value ("") means product (see IsProduct) but isn't
	// itself a valid LineItemType enum value on the wire -- substitute the
	// explicit value so the response always satisfies the schema.
	lineType := li.LineType
	if lineType == "" {
		lineType = procurementDomain.LineItemTypeProduct
	}
	return genprocurement.PurchaseOrderLineItem{
		Id: li.ID, TenantId: li.TenantID, PurchaseOrderId: li.PurchaseOrderID, SkuId: li.SKUID, RawSku: li.RawSKU,
		RawDescription: li.RawDescription, Position: li.Position, ProductName: li.ProductName, SkuCode: li.SKUCode, ProductId: li.ProductID,
		Quantity: li.Quantity, UnitsPerPurchasedItem: li.UnitsPerPurchasedItem, UnitCostMinorUnits: li.UnitCostMinorUnits,
		LineType: genprocurement.LineItemType(lineType), LineTypeConfirmed: li.LineTypeConfirmed,
		SuggestedLotNumber: &li.SuggestedLotNumber, SuggestedExpiryDate: timePtrToGenDate(li.SuggestedExpiryDate),
		CreatedAt: li.CreatedAt, UpdatedAt: li.UpdatedAt,
	}
}

func toGenDocument(d procurementDomain.PurchaseOrderDocument) genprocurement.PurchaseOrderDocument {
	return genprocurement.PurchaseOrderDocument{
		Id: d.ID, PurchaseOrderId: d.PurchaseOrderID, FileUrl: d.FileURL, FileName: d.FileName,
		ContentType: d.ContentType, IsPotentialDuplicate: d.IsPotentialDuplicate, CreatedAt: d.CreatedAt,
	}
}

func toGenDocumentParsingJob(job procurementDomain.DocumentParsingJob) genprocurement.DocumentParsingJob {
	return genprocurement.DocumentParsingJob{
		Id: job.ID, TenantId: job.TenantID, PurchaseOrderId: job.PurchaseOrderID, DocumentId: job.DocumentID,
		Status: genprocurement.DocumentParsingJobStatus(job.Status), ErrorMessage: job.ErrorMessage,
		CreatedAt: job.CreatedAt, StartedAt: job.StartedAt, FinishedAt: job.FinishedAt,
	}
}

func toGenReceivingRecord(r procurementDomain.ReceivingRecord) genprocurement.ReceivingRecord {
	return genprocurement.ReceivingRecord{
		Id: r.ID, TenantId: r.TenantID, PurchaseOrderId: r.PurchaseOrderID,
		InvoiceFileUrl: r.InvoiceFileURL, InvoiceFileName: r.InvoiceFileName, ContentType: r.ContentType,
		Status: genprocurement.ReceivingRecordStatus(r.Status), IsPotentialDuplicate: r.IsPotentialDuplicate,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
	}
}

func toGenReceivingDiscrepancy(d procurementDomain.ReceivingDiscrepancy) genprocurement.ReceivingDiscrepancy {
	return genprocurement.ReceivingDiscrepancy{
		Id: d.ID, TenantId: d.TenantID, ReceivingRecordId: d.ReceivingRecordID, LineItemId: d.LineItemID,
		Type: genprocurement.DiscrepancyType(d.Type), ExpectedQty: d.ExpectedQty, ReceivedQty: d.ReceivedQty,
		Details: d.Details, Status: genprocurement.DiscrepancyStatus(d.Status), AcknowledgedAt: d.AcknowledgedAt,
		CreatedAt: d.CreatedAt,
	}
}

func toGenReceivingParsingJob(job procurementDomain.ReceivingParsingJob) genprocurement.ReceivingParsingJob {
	return genprocurement.ReceivingParsingJob{
		Id: job.ID, TenantId: job.TenantID, ReceivingRecordId: job.ReceivingRecordID,
		Status: genprocurement.ReceivingParsingJobStatus(job.Status), ErrorMessage: job.ErrorMessage,
		CreatedAt: job.CreatedAt, StartedAt: job.StartedAt, FinishedAt: job.FinishedAt,
	}
}

func toGenBooking(b procurementDomain.Booking) genprocurement.Booking {
	return genprocurement.Booking{
		Id: b.ID, TenantId: b.TenantID, PurchaseOrderId: b.PurchaseOrderID, Status: genprocurement.BookingStatus(b.Status),
		StartedAt: b.StartedAt, PausedAt: b.PausedAt, CompletedAt: b.CompletedAt, CompletedBy: b.CompletedBy,
	}
}

func toGenBookingScan(s procurementDomain.BookingScan) genprocurement.BookingScan {
	return genprocurement.BookingScan{
		Id: s.ID, TenantId: s.TenantID, BookingId: s.BookingID, SkuId: s.SKUID, BatchId: s.BatchID,
		LotNumber: strPtrOrNil(s.LotNumber), ExpiryDate: timePtrToGenDate(s.ExpiryDate),
		ActionType: genprocurement.BookingScanActionType(s.ActionType), Quantity: s.Quantity,
		WriteOffQuantity: s.WriteOffQuantity, Reason: strPtrOrNil(s.Reason),
		ScannedAt: s.ScannedAt, ScannedBy: s.ScannedBy,
	}
}

func mapError(c *gin.Context, err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, procurementDomain.ErrPurchaseOrderNotFound), errors.Is(err, procurementDomain.ErrLineItemNotFound),
		errors.Is(err, procurementDomain.ErrDocumentNotFound), errors.Is(err, procurementDomain.ErrReceivingRecordNotFound),
		errors.Is(err, procurementDomain.ErrBookingNotFound), errors.Is(err, procurementDomain.ErrParsingJobNotFound),
		errors.Is(err, procurementDomain.ErrBookingScanNotFound), errors.Is(err, procurementDomain.ErrDiscrepancyNotFound):
		middleware.Fail(c, http.StatusNotFound, "NOT_FOUND", err.Error())
	case errors.Is(err, procurementDomain.ErrInvalidQuantity), errors.Is(err, procurementDomain.ErrUnsupportedFileType),
		errors.Is(err, procurementDomain.ErrMissingRequiredColumns), errors.Is(err, procurementDomain.ErrPDFHasNoExtractableText),
		errors.Is(err, procurementDomain.ErrAIParsingNotConfigured), errors.Is(err, procurementDomain.ErrLineItemAlreadyLinked),
		errors.Is(err, procurementDomain.ErrNoLineItemsExtracted),
		errors.Is(err, procurementDomain.ErrBookingNotActive), errors.Is(err, procurementDomain.ErrWarehouseRequired),
		errors.Is(err, procurementDomain.ErrScanCodeOrSKURequired), errors.Is(err, procurementDomain.ErrInvalidBookingScanActionType),
		errors.Is(err, procurementDomain.ErrPendingBatchNotFound),
		errors.Is(err, procurementDomain.ErrDiscrepancyMismatch), errors.Is(err, procurementDomain.ErrDiscrepancyAlreadyAcknowledged),
		errors.Is(err, procurementDomain.ErrInvalidDamagedQuantity),
		errors.Is(err, invDomain.ErrInvalidQuantity), errors.Is(err, invDomain.ErrInsufficientStock),
		errors.Is(err, pmsDomain.ErrSKUNotFound), errors.Is(err, pmsDomain.ErrBarcodeAliasNotFound),
		errors.Is(err, pmsDomain.ErrDuplicateSlug), errors.Is(err, pmsDomain.ErrDuplicateSKUCode),
		errors.Is(err, pmsDomain.ErrWeightRequiredForPerWeightSKU), errors.Is(err, pmsDomain.ErrInvalidCheckDigit):
		middleware.Fail(c, http.StatusUnprocessableEntity, "VALIDATION_ERROR", err.Error())
	default:
		log.Printf("procurement: unhandled error: %v", err)
		middleware.Fail(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
	}
	return err
}
