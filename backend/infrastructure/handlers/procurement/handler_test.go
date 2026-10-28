package procurement_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	genprocurement "github.com/adirkuhn/commerceos/api/procurement"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	procurementhandler "github.com/adirkuhn/commerceos/infrastructure/handlers/procurement"
)

// doMultipartRequest builds a multipart/form-data POST with a "file" part
// plus any extra plain form fields -- used for the document upload endpoints,
// which (unlike every other route in this handler) aren't JSON.
func doMultipartRequest(rg *gin.Engine, path, fileName, fileContent string, extraFields map[string]string) *httptest.ResponseRecorder {
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	fw, _ := mw.CreateFormFile("file", fileName)
	_, _ = fw.Write([]byte(fileContent))
	for k, v := range extraFields {
		_ = mw.WriteField(k, v)
	}
	_ = mw.Close()

	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	rg.ServeHTTP(w, req)
	return w
}

func init() { gin.SetMode(gin.TestMode) }

func strPtr(s string) *string { return &s }

type stubListPurchaseOrders struct {
	out []procurementDomain.PurchaseOrder
	err error
}

func (s *stubListPurchaseOrders) Execute(_ context.Context) ([]procurementDomain.PurchaseOrder, error) {
	return s.out, s.err
}

type stubCreatePurchaseOrder struct {
	out      *procurementDomain.PurchaseOrder
	err      error
	gotInput appprocurement.CreatePurchaseOrderInput
}

func (s *stubCreatePurchaseOrder) Execute(_ context.Context, in appprocurement.CreatePurchaseOrderInput) (*procurementDomain.PurchaseOrder, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: in.SupplierID, WarehouseID: in.WarehouseID, Status: procurementDomain.PurchaseOrderStatusDraft}, nil
}

type stubGetPurchaseOrder struct {
	out *appprocurement.PurchaseOrderDetail
	err error
}

func (s *stubGetPurchaseOrder) Execute(_ context.Context, _ string) (*appprocurement.PurchaseOrderDetail, error) {
	return s.out, s.err
}

type stubUpdatePurchaseOrder struct {
	out      *procurementDomain.PurchaseOrder
	err      error
	gotInput appprocurement.UpdatePurchaseOrderInput
}

func (s *stubUpdatePurchaseOrder) Execute(_ context.Context, in appprocurement.UpdatePurchaseOrderInput) (*procurementDomain.PurchaseOrder, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.PurchaseOrder{ID: in.ID, SupplierID: in.SupplierID, Status: in.Status}, nil
}

type stubDeletePurchaseOrder struct {
	err   error
	calls []string
}

func (s *stubDeletePurchaseOrder) Execute(_ context.Context, id string) error {
	s.calls = append(s.calls, id)
	return s.err
}

type stubAddLineItem struct {
	out      *procurementDomain.PurchaseOrderLineItem
	err      error
	gotInput appprocurement.AddPurchaseOrderLineItemInput
}

func (s *stubAddLineItem) Execute(_ context.Context, in appprocurement.AddPurchaseOrderLineItemInput) (*procurementDomain.PurchaseOrderLineItem, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.PurchaseOrderLineItem{ID: "line-item-1", PurchaseOrderID: in.PurchaseOrderID, Quantity: in.Quantity}, nil
}

type stubUpdateLineItem struct {
	out      *procurementDomain.PurchaseOrderLineItem
	err      error
	gotInput appprocurement.UpdatePurchaseOrderLineItemInput
}

func (s *stubUpdateLineItem) Execute(_ context.Context, in appprocurement.UpdatePurchaseOrderLineItemInput) (*procurementDomain.PurchaseOrderLineItem, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.PurchaseOrderLineItem{ID: in.ID, Quantity: in.Quantity}, nil
}

type stubRemoveLineItem struct {
	err   error
	calls []string
}

func (s *stubRemoveLineItem) Execute(_ context.Context, id string) error {
	s.calls = append(s.calls, id)
	return s.err
}

type stubDeletePurchaseOrderDocument struct {
	err   error
	calls []string
}

func (s *stubDeletePurchaseOrderDocument) Execute(_ context.Context, id string) error {
	s.calls = append(s.calls, id)
	return s.err
}

type stubRetryDocumentParsingJob struct {
	out      *procurementDomain.DocumentParsingJob
	err      error
	gotDocID string
}

func (s *stubRetryDocumentParsingJob) Execute(_ context.Context, documentID string) (*procurementDomain.DocumentParsingJob, error) {
	s.gotDocID = documentID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.DocumentParsingJob{ID: "job-1", DocumentID: documentID, Status: procurementDomain.DocumentParsingJobPending}, nil
}

type stubListBookingDiscrepancies struct {
	out *appprocurement.BookingDiscrepancyReport
	err error
}

func (s *stubListBookingDiscrepancies) Execute(_ context.Context) (*appprocurement.BookingDiscrepancyReport, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appprocurement.BookingDiscrepancyReport{}, nil
}

type stubAttachDocument struct {
	out      *procurementDomain.DocumentParsingJob
	err      error
	gotInput appprocurement.AttachDocumentToPurchaseOrderInput
}

func (s *stubAttachDocument) Execute(_ context.Context, in appprocurement.AttachDocumentToPurchaseOrderInput) (*procurementDomain.DocumentParsingJob, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: in.PurchaseOrderID, Status: procurementDomain.DocumentParsingJobPending}, nil
}

type stubGetDocumentParsingJob struct {
	out *procurementDomain.DocumentParsingJob
	err error
}

func (s *stubGetDocumentParsingJob) Execute(_ context.Context, _ string) (*procurementDomain.DocumentParsingJob, error) {
	return s.out, s.err
}

type stubGetReceivingParsingJob struct {
	out *procurementDomain.ReceivingParsingJob
	err error
}

func (s *stubGetReceivingParsingJob) Execute(_ context.Context, _ string) (*procurementDomain.ReceivingParsingJob, error) {
	return s.out, s.err
}

type stubCreatePurchaseOrderFromDocument struct {
	out      *procurementDomain.DocumentParsingJob
	err      error
	gotInput appprocurement.CreatePurchaseOrderFromDocumentInput
}

func (s *stubCreatePurchaseOrderFromDocument) Execute(_ context.Context, in appprocurement.CreatePurchaseOrderFromDocumentInput) (*procurementDomain.DocumentParsingJob, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-new", Status: procurementDomain.DocumentParsingJobPending}, nil
}

type stubLinkLineItemToSKU struct {
	out      *procurementDomain.PurchaseOrderLineItem
	err      error
	gotInput appprocurement.LinkLineItemToSKUInput
}

func (s *stubLinkLineItemToSKU) Execute(_ context.Context, in appprocurement.LinkLineItemToSKUInput) (*procurementDomain.PurchaseOrderLineItem, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	skuID := in.SKUID
	return &procurementDomain.PurchaseOrderLineItem{ID: in.LineItemID, SKUID: &skuID}, nil
}

type stubCreateProductAndLinkLineItem struct {
	out      *procurementDomain.PurchaseOrderLineItem
	err      error
	gotInput appprocurement.CreateProductAndLinkLineItemInput
}

func (s *stubCreateProductAndLinkLineItem) Execute(_ context.Context, in appprocurement.CreateProductAndLinkLineItemInput) (*procurementDomain.PurchaseOrderLineItem, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	skuID := "new-sku-1"
	return &procurementDomain.PurchaseOrderLineItem{ID: in.LineItemID, SKUID: &skuID}, nil
}

type stubCreateSupplierAndAssign struct {
	out      *procurementDomain.PurchaseOrder
	err      error
	gotInput appprocurement.CreateSupplierAndAssignToPurchaseOrderInput
}

func (s *stubCreateSupplierAndAssign) Execute(_ context.Context, in appprocurement.CreateSupplierAndAssignToPurchaseOrderInput) (*procurementDomain.PurchaseOrder, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.PurchaseOrder{ID: in.PurchaseOrderID, SupplierID: strPtr("new-supplier-1"), Status: procurementDomain.PurchaseOrderStatusDraft}, nil
}

type stubUploadInvoice struct {
	out      *procurementDomain.ReceivingParsingJob
	err      error
	gotInput appprocurement.UploadInvoiceInput
}

func (s *stubUploadInvoice) Execute(_ context.Context, in appprocurement.UploadInvoiceInput) (*procurementDomain.ReceivingParsingJob, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.ReceivingParsingJob{ID: "receiving-job-1", ReceivingRecordID: "receiving-record-1", Status: procurementDomain.ReceivingParsingJobPending}, nil
}

type stubListReceivingRecords struct {
	out []procurementDomain.ReceivingRecord
	err error
}

func (s *stubListReceivingRecords) Execute(_ context.Context, _ string) ([]procurementDomain.ReceivingRecord, error) {
	return s.out, s.err
}

type stubGetReceivingRecord struct {
	out *appprocurement.ReceivingRecordDetail
	err error
}

func (s *stubGetReceivingRecord) Execute(_ context.Context, _ string) (*appprocurement.ReceivingRecordDetail, error) {
	return s.out, s.err
}

type stubCompleteReceiving struct {
	out      *procurementDomain.ReceivingRecord
	err      error
	gotInput string
}

func (s *stubCompleteReceiving) Execute(_ context.Context, receivingRecordID string) (*procurementDomain.ReceivingRecord, error) {
	s.gotInput = receivingRecordID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.ReceivingRecord{ID: receivingRecordID, Status: procurementDomain.ReceivingRecordStatusApplied}, nil
}

type stubAcknowledgeReceivingDiscrepancy struct {
	out      *procurementDomain.ReceivingDiscrepancy
	err      error
	gotInput string
}

func (s *stubAcknowledgeReceivingDiscrepancy) Execute(_ context.Context, discrepancyID string) (*procurementDomain.ReceivingDiscrepancy, error) {
	s.gotInput = discrepancyID
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.ReceivingDiscrepancy{ID: discrepancyID, Status: procurementDomain.DiscrepancyStatusAcknowledged}, nil
}

type stubLinkReceivingDiscrepancies struct {
	err           error
	gotUnexpected string
	gotMissing    string
}

func (s *stubLinkReceivingDiscrepancies) Execute(_ context.Context, unexpectedDiscrepancyID, missingDiscrepancyID string) error {
	s.gotUnexpected, s.gotMissing = unexpectedDiscrepancyID, missingDiscrepancyID
	return s.err
}

type stubFlagDamagedLineItem struct {
	out                  *procurementDomain.ReceivingDiscrepancy
	err                  error
	gotReceivingRecordID string
	gotLineItemID        string
	gotDamagedQty        int64
	gotNote              string
}

func (s *stubFlagDamagedLineItem) Execute(_ context.Context, receivingRecordID, lineItemID string, damagedQty int64, note string) (*procurementDomain.ReceivingDiscrepancy, error) {
	s.gotReceivingRecordID, s.gotLineItemID, s.gotDamagedQty, s.gotNote = receivingRecordID, lineItemID, damagedQty, note
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	id := lineItemID
	return &procurementDomain.ReceivingDiscrepancy{
		ID: "discrepancy-1", ReceivingRecordID: receivingRecordID, LineItemID: &id,
		Type: procurementDomain.DiscrepancyTypeDamagedItem, ReceivedQty: damagedQty, Details: note,
		Status: procurementDomain.DiscrepancyStatusOpen,
	}, nil
}

type stubStartOrResumeBooking struct {
	out      *procurementDomain.Booking
	err      error
	gotInput appprocurement.StartOrResumeBookingInput
}

func (s *stubStartOrResumeBooking) Execute(_ context.Context, in appprocurement.StartOrResumeBookingInput) (*procurementDomain.Booking, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &procurementDomain.Booking{ID: "booking-1", PurchaseOrderID: in.PurchaseOrderID, Status: procurementDomain.BookingStatusInProgress}, nil
}

type stubGetBooking struct {
	out *appprocurement.BookingDetail
	err error
}

func (s *stubGetBooking) Execute(_ context.Context, _ string) (*appprocurement.BookingDetail, error) {
	return s.out, s.err
}

type stubScanBookingItem struct {
	out      *appprocurement.ScanBookingItemResult
	err      error
	gotInput appprocurement.ScanBookingItemInput
}

func (s *stubScanBookingItem) Execute(_ context.Context, in appprocurement.ScanBookingItemInput) (*appprocurement.ScanBookingItemResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appprocurement.ScanBookingItemResult{
		Scan: procurementDomain.BookingScan{ID: "booking-scan-1", BookingID: "booking-1", Quantity: in.Quantity},
	}, nil
}

type stubResolveBookingScanCode struct {
	out                   *appprocurement.ResolveBookingScanCodeResult
	err                   error
	gotBookingID, gotCode string
}

func (s *stubResolveBookingScanCode) Execute(_ context.Context, bookingID, code string) (*appprocurement.ResolveBookingScanCodeResult, error) {
	s.gotBookingID, s.gotCode = bookingID, code
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appprocurement.ResolveBookingScanCodeResult{SKUID: "sku-1", ProductName: "Widget", UnitsPerScan: 1}, nil
}

type stubDeleteBookingScan struct {
	err                     error
	gotBookingID, gotScanID string
}

func (s *stubDeleteBookingScan) Execute(_ context.Context, bookingID, scanID string) error {
	s.gotBookingID, s.gotScanID = bookingID, scanID
	return s.err
}

type stubPauseBooking struct {
	out *procurementDomain.Booking
	err error
}

func (s *stubPauseBooking) Execute(_ context.Context, _ string) (*procurementDomain.Booking, error) {
	return s.out, s.err
}

type stubCompleteBooking struct {
	out *appprocurement.CompleteBookingResult
	err error
}

func (s *stubCompleteBooking) Execute(_ context.Context, _ string, _ string) (*appprocurement.CompleteBookingResult, error) {
	return s.out, s.err
}

type stubGetLandedUnitCost struct {
	out int64
	err error
}

func (s *stubGetLandedUnitCost) Execute(_ context.Context, _ string) (int64, error) {
	return s.out, s.err
}

func newTestHandler(
	list *stubListPurchaseOrders, create *stubCreatePurchaseOrder, get *stubGetPurchaseOrder, update *stubUpdatePurchaseOrder,
	del *stubDeletePurchaseOrder, addLI *stubAddLineItem, updateLI *stubUpdateLineItem, removeLI *stubRemoveLineItem,
) *procurementhandler.Handler {
	return procurementhandler.NewHandler(list, create, get, update, del, addLI, updateLI, removeLI, &stubAttachDocument{},
		&stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{},
		&stubCreatePurchaseOrderFromDocument{}, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{}, &stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
}

// newParsingJobTestHandler builds a Handler for the document/receiving
// parsing-job status routes specifically -- every other use case is a
// harmless zero-value stub, same pattern as newReceivingTestHandler.
func newParsingJobTestHandler(
	getDocumentParsingJob *stubGetDocumentParsingJob, getReceivingParsingJob *stubGetReceivingParsingJob,
) *procurementhandler.Handler {
	return procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{},
		getDocumentParsingJob, getReceivingParsingJob,
		&stubCreatePurchaseOrderFromDocument{}, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
}

// newDeleteDocumentTestHandler builds a Handler for the delete-document
// route specifically -- every other use case is a harmless zero-value stub.
func newDeleteDocumentTestHandler(del *stubDeletePurchaseOrderDocument) *procurementhandler.Handler {
	return procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{},
		&stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{},
		&stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		del, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
}

func newRetryDocumentParsingJobTestHandler(retry *stubRetryDocumentParsingJob) *procurementhandler.Handler {
	return procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{},
		&stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{},
		&stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, retry, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
}

// newBookingTestHandler builds a Handler for the booking routes
// specifically -- every non-booking use case is a harmless zero-value stub.
func newBookingTestHandler(
	startOrResumeBooking *stubStartOrResumeBooking, getBooking *stubGetBooking, scanBookingItem *stubScanBookingItem,
	pauseBooking *stubPauseBooking, completeBooking *stubCompleteBooking,
) *procurementhandler.Handler {
	return procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{},
		&stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{},
		&stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		startOrResumeBooking, getBooking, scanBookingItem, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, pauseBooking, completeBooking,
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
}

// newResolveBookingScanCodeTestHandler builds a Handler for the
// resolve-scan route specifically -- every other use case is a harmless
// zero-value stub.
func newResolveBookingScanCodeTestHandler(resolve *stubResolveBookingScanCode) *procurementhandler.Handler {
	return procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{},
		&stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{},
		&stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, resolve, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
}

// newDeleteBookingScanTestHandler builds a Handler for the delete-scan
// route specifically -- every other use case is a harmless zero-value stub.
func newDeleteBookingScanTestHandler(del *stubDeleteBookingScan) *procurementhandler.Handler {
	return procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{},
		&stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{},
		&stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, del, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
}

// newReceivingTestHandler builds a Handler for the receiving routes
// specifically -- every non-receiving use case is a harmless zero-value stub.
func newReceivingTestHandler(
	uploadInvoice *stubUploadInvoice, listReceivingRecords *stubListReceivingRecords,
	getReceivingRecord *stubGetReceivingRecord, completeReceiving *stubCompleteReceiving,
) *procurementhandler.Handler {
	return procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{},
		&stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{},
		&stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		uploadInvoice, listReceivingRecords, getReceivingRecord, completeReceiving,
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
}

// newReceivingDiscrepancyActionsTestHandler builds a Handler for the
// acknowledge/link/flag-damaged discrepancy routes specifically -- every
// other use case is a harmless zero-value stub.
func newReceivingDiscrepancyActionsTestHandler(
	acknowledge *stubAcknowledgeReceivingDiscrepancy, link *stubLinkReceivingDiscrepancies, flagDamaged *stubFlagDamagedLineItem,
) *procurementhandler.Handler {
	return procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{},
		&stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{},
		&stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, acknowledge, link, flagDamaged)
}

// newBookingDiscrepanciesTestHandler builds a Handler for the discrepancies
// report route specifically -- every other use case is a harmless
// zero-value stub.
func newBookingDiscrepanciesTestHandler(list *stubListBookingDiscrepancies) *procurementhandler.Handler {
	return procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{},
		&stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{},
		&stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, list, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
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

func newTestRouter(h *procurementhandler.Handler) *gin.Engine {
	r := gin.New()
	genprocurement.RegisterHandlers(r.Group("/api/v1"), h)
	return r
}

func TestListPurchaseOrdersHandler_returnsPurchaseOrders(t *testing.T) {
	h := newTestHandler(&stubListPurchaseOrders{out: []procurementDomain.PurchaseOrder{{ID: "po-1", SupplierID: strPtr("supplier-1")}}},
		&stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/purchase-orders", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "po-1")
}

func TestListPurchaseOrdersHandler_includesDuplicateFlag(t *testing.T) {
	h := newTestHandler(&stubListPurchaseOrders{out: []procurementDomain.PurchaseOrder{
		{ID: "po-2", IsPotentialDuplicate: true, DuplicateOfPurchaseOrderID: strPtr("po-1")},
	}}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/purchase-orders", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"is_potential_duplicate":true`)
	assert.Contains(t, w.Body.String(), `"duplicate_of_purchase_order_id":"po-1"`)
}

func TestCreatePurchaseOrderHandler_success(t *testing.T) {
	create := &stubCreatePurchaseOrder{}
	h := newTestHandler(&stubListPurchaseOrders{}, create, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-orders", map[string]any{
		"supplier_id": "supplier-1", "warehouse_id": "wh-1", "currency_code": "USD",
	})

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, create.gotInput.SupplierID)
	assert.Equal(t, "supplier-1", *create.gotInput.SupplierID)
	require.NotNil(t, create.gotInput.WarehouseID)
	assert.Equal(t, "wh-1", *create.gotInput.WarehouseID)
}

func TestCreatePurchaseOrderHandler_allowsOmittingWarehouse(t *testing.T) {
	create := &stubCreatePurchaseOrder{}
	h := newTestHandler(&stubListPurchaseOrders{}, create, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-orders", map[string]any{
		"supplier_id": "supplier-1", "currency_code": "USD",
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Nil(t, create.gotInput.WarehouseID)
}

func TestGetPurchaseOrderHandler_returnsDetail(t *testing.T) {
	get := &stubGetPurchaseOrder{out: &appprocurement.PurchaseOrderDetail{
		PurchaseOrder: procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1")},
		LineItems:     []procurementDomain.PurchaseOrderLineItem{{ID: "line-item-1", Quantity: 10}},
	}}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, get, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/purchase-orders/po-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"supplier_id":"supplier-1"`)
	assert.Contains(t, w.Body.String(), `"quantity":10`)
}

func TestGetPurchaseOrderHandler_includesParsingSuggestions(t *testing.T) {
	get := &stubGetPurchaseOrder{out: &appprocurement.PurchaseOrderDetail{
		PurchaseOrder: procurementDomain.PurchaseOrder{
			ID: "po-1", SuggestedSupplierName: "Pedaços do Brasil, Lda.", SuggestedCurrencyCode: "EUR",
		},
	}}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, get, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/purchase-orders/po-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"suggested_supplier_name":"Pedaços do Brasil, Lda."`)
	assert.Contains(t, w.Body.String(), `"suggested_currency_code":"EUR"`)
}

// TestGetPurchaseOrderHandler_includesDocuments is the fix for "no way to
// view the uploaded document" -- PurchaseOrderDocument.FileURL was always
// stored but never made it into any API response.
func TestGetPurchaseOrderHandler_includesDocuments(t *testing.T) {
	get := &stubGetPurchaseOrder{out: &appprocurement.PurchaseOrderDetail{
		PurchaseOrder: procurementDomain.PurchaseOrder{ID: "po-1"},
		Documents: []procurementDomain.PurchaseOrderDocument{
			{ID: "document-1", PurchaseOrderID: "po-1", FileURL: "https://uploads.example.com/tenant-1/abc.pdf", FileName: "invoice.pdf", ContentType: "application/pdf"},
		},
	}}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, get, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/purchase-orders/po-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"file_url":"https://uploads.example.com/tenant-1/abc.pdf"`)
	assert.Contains(t, w.Body.String(), `"file_name":"invoice.pdf"`)
}

func TestGetPurchaseOrderHandler_notFound(t *testing.T) {
	get := &stubGetPurchaseOrder{err: procurementDomain.ErrPurchaseOrderNotFound}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, get, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/purchase-orders/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUpdatePurchaseOrderHandler_success(t *testing.T) {
	update := &stubUpdatePurchaseOrder{}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, update, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/procurement/purchase-orders/po-1", map[string]any{
		"supplier_id": "supplier-2", "warehouse_id": "wh-2", "status": "sent", "currency_code": "EUR",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusSent, update.gotInput.Status)
}

func TestUpdatePurchaseOrderHandler_notFound(t *testing.T) {
	update := &stubUpdatePurchaseOrder{err: procurementDomain.ErrPurchaseOrderNotFound}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, update, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/procurement/purchase-orders/missing", map[string]any{
		"supplier_id": "s", "warehouse_id": "w", "status": "draft", "currency_code": "USD",
	})

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeletePurchaseOrderHandler_success(t *testing.T) {
	del := &stubDeletePurchaseOrder{}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, del, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/procurement/purchase-orders/po-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"po-1"}, del.calls)
}

func TestAddPurchaseOrderLineItemHandler_success(t *testing.T) {
	add := &stubAddLineItem{}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, add, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-orders/po-1/line-items", map[string]any{
		"raw_description": "Widget", "quantity": 10, "unit_cost_minor_units": 500,
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "po-1", add.gotInput.PurchaseOrderID)
	assert.Equal(t, int64(10), add.gotInput.Quantity)
}

func TestAddPurchaseOrderLineItemHandler_rejectsInvalidQuantity(t *testing.T) {
	add := &stubAddLineItem{err: procurementDomain.ErrInvalidQuantity}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, add, &stubUpdateLineItem{}, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-orders/po-1/line-items", map[string]any{
		"raw_description": "Widget", "quantity": 1, "unit_cost_minor_units": 500,
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestUpdatePurchaseOrderLineItemHandler_success(t *testing.T) {
	update := &stubUpdateLineItem{}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, update, &stubRemoveLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPatch, "/api/v1/procurement/purchase-order-line-items/line-item-1", map[string]any{
		"raw_description": "Widget v2", "quantity": 20, "unit_cost_minor_units": 600,
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "line-item-1", update.gotInput.ID)
	assert.Equal(t, int64(20), update.gotInput.Quantity)
}

func TestRemovePurchaseOrderLineItemHandler_success(t *testing.T) {
	remove := &stubRemoveLineItem{}
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, remove)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/procurement/purchase-order-line-items/line-item-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"line-item-1"}, remove.calls)
}

func TestAttachDocumentToPurchaseOrderHandler_success(t *testing.T) {
	attach := &stubAttachDocument{}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, attach, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{}, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doMultipartRequest(newTestRouter(h), "/api/v1/procurement/purchase-orders/po-1/documents", "order.csv", "sku,qty\nA,1", nil)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "po-1", attach.gotInput.PurchaseOrderID)
	assert.Equal(t, "order.csv", attach.gotInput.FileName)
	assert.Contains(t, w.Body.String(), "job-1")
}

func TestAttachDocumentToPurchaseOrderHandler_missingFileReturns422(t *testing.T) {
	h := newTestHandler(&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{}, &stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{})

	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	_ = mw.Close()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/procurement/purchase-orders/po-1/documents", &buf)
	req.Header.Set("Content-Type", mw.FormDataContentType())
	w := httptest.NewRecorder()
	newTestRouter(h).ServeHTTP(w, req)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestAttachDocumentToPurchaseOrderHandler_notFound(t *testing.T) {
	attach := &stubAttachDocument{err: procurementDomain.ErrPurchaseOrderNotFound}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, attach, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{}, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doMultipartRequest(newTestRouter(h), "/api/v1/procurement/purchase-orders/missing/documents", "order.csv", "data", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeletePurchaseOrderDocumentHandler_success(t *testing.T) {
	del := &stubDeletePurchaseOrderDocument{}
	h := newDeleteDocumentTestHandler(del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/procurement/purchase-order-documents/document-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []string{"document-1"}, del.calls)
}

func TestDeletePurchaseOrderDocumentHandler_notFound(t *testing.T) {
	del := &stubDeletePurchaseOrderDocument{err: procurementDomain.ErrDocumentNotFound}
	h := newDeleteDocumentTestHandler(del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/procurement/purchase-order-documents/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestRetryDocumentParsingJobHandler_success(t *testing.T) {
	retry := &stubRetryDocumentParsingJob{out: &procurementDomain.DocumentParsingJob{
		ID: "job-2", DocumentID: "document-1", Status: procurementDomain.DocumentParsingJobPending,
	}}
	h := newRetryDocumentParsingJobTestHandler(retry)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-order-documents/document-1/retry", nil)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "document-1", retry.gotDocID)
	assert.Contains(t, w.Body.String(), "job-2")
	assert.Contains(t, w.Body.String(), "pending")
}

func TestRetryDocumentParsingJobHandler_notFound(t *testing.T) {
	retry := &stubRetryDocumentParsingJob{err: procurementDomain.ErrDocumentNotFound}
	h := newRetryDocumentParsingJobTestHandler(retry)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-order-documents/missing/retry", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetDocumentParsingJobHandler_returnsJob(t *testing.T) {
	get := &stubGetDocumentParsingJob{out: &procurementDomain.DocumentParsingJob{
		ID: "job-1", Status: procurementDomain.DocumentParsingJobFailed, ErrorMessage: "AI document parsing is not configured",
	}}
	h := newParsingJobTestHandler(get, &stubGetReceivingParsingJob{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/document-parsing-jobs/job-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "failed")
	assert.Contains(t, w.Body.String(), "AI document parsing is not configured")
}

func TestGetDocumentParsingJobHandler_notFound(t *testing.T) {
	get := &stubGetDocumentParsingJob{err: procurementDomain.ErrParsingJobNotFound}
	h := newParsingJobTestHandler(get, &stubGetReceivingParsingJob{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/document-parsing-jobs/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetReceivingParsingJobHandler_returnsJob(t *testing.T) {
	get := &stubGetReceivingParsingJob{out: &procurementDomain.ReceivingParsingJob{
		ID: "receiving-job-1", Status: procurementDomain.ReceivingParsingJobFailed, ErrorMessage: "AI document parsing is not configured",
	}}
	h := newParsingJobTestHandler(&stubGetDocumentParsingJob{}, get)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/receiving-parsing-jobs/receiving-job-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "failed")
	assert.Contains(t, w.Body.String(), "AI document parsing is not configured")
}

func TestGetReceivingParsingJobHandler_notFound(t *testing.T) {
	get := &stubGetReceivingParsingJob{err: procurementDomain.ErrParsingJobNotFound}
	h := newParsingJobTestHandler(&stubGetDocumentParsingJob{}, get)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/receiving-parsing-jobs/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreatePurchaseOrderFromDocumentHandler_success(t *testing.T) {
	create := &stubCreatePurchaseOrderFromDocument{}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, create, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doMultipartRequest(newTestRouter(h), "/api/v1/procurement/purchase-orders/documents", "order.csv", "sku,qty\nA,1", map[string]string{
		"supplier_id": "supplier-1", "warehouse_id": "warehouse-1", "currency_code": "USD",
	})

	require.Equal(t, http.StatusCreated, w.Code)
	require.NotNil(t, create.gotInput.SupplierID)
	assert.Equal(t, "supplier-1", *create.gotInput.SupplierID)
	assert.Equal(t, "order.csv", create.gotInput.FileName)
}

func TestCreatePurchaseOrderFromDocumentHandler_supplierOmittedIsAllowed(t *testing.T) {
	create := &stubCreatePurchaseOrderFromDocument{}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, create, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doMultipartRequest(newTestRouter(h), "/api/v1/procurement/purchase-orders/documents", "order.csv", "sku,qty\nA,1", map[string]string{
		"warehouse_id": "warehouse-1", "currency_code": "USD",
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Nil(t, create.gotInput.SupplierID)
}

func TestCreatePurchaseOrderFromDocumentHandler_propagatesValidationError(t *testing.T) {
	create := &stubCreatePurchaseOrderFromDocument{err: procurementDomain.ErrMissingRequiredColumns}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, create, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doMultipartRequest(newTestRouter(h), "/api/v1/procurement/purchase-orders/documents", "order.csv", "bad data", map[string]string{
		"supplier_id": "supplier-1", "warehouse_id": "warehouse-1", "currency_code": "USD",
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestLinkLineItemToSkuHandler_success(t *testing.T) {
	link := &stubLinkLineItemToSKU{}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{}, link, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-order-line-items/line-item-1/link", map[string]any{
		"sku_id": "sku-1",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "line-item-1", link.gotInput.LineItemID)
	assert.Equal(t, "sku-1", link.gotInput.SKUID)
}

func TestLinkLineItemToSkuHandler_notFound(t *testing.T) {
	link := &stubLinkLineItemToSKU{err: procurementDomain.ErrLineItemNotFound}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{}, link, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-order-line-items/missing/link", map[string]any{
		"sku_id": "sku-1",
	})

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetLandedUnitCostHandler_success(t *testing.T) {
	landedCost := &stubGetLandedUnitCost{out: 900}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{}, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, landedCost, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/purchase-order-line-items/line-item-1/landed-cost", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"unit_cost_minor_units":900`)
}

func TestGetLandedUnitCostHandler_notFound(t *testing.T) {
	landedCost := &stubGetLandedUnitCost{err: procurementDomain.ErrLineItemNotFound}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{}, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, landedCost, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/purchase-order-line-items/missing/landed-cost", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestCreateProductAndLinkLineItemHandler_success(t *testing.T) {
	create := &stubCreateProductAndLinkLineItem{}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{}, &stubLinkLineItemToSKU{}, create, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-order-line-items/line-item-1/create-product-and-link", map[string]any{
		"product_name": "New Widget", "sku_code": "NEW-001", "price_minor_units": 999, "currency_code": "USD",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "line-item-1", create.gotInput.LineItemID)
	assert.Equal(t, "New Widget", create.gotInput.ProductName)
	assert.Equal(t, int64(999), create.gotInput.PriceMinorUnits)
}

func TestCreateProductAndLinkLineItemHandler_duplicateSkuCodeReturns422(t *testing.T) {
	create := &stubCreateProductAndLinkLineItem{err: pmsDomain.ErrDuplicateSKUCode}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{}, &stubLinkLineItemToSKU{}, create, &stubCreateSupplierAndAssign{},
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-order-line-items/line-item-1/create-product-and-link", map[string]any{
		"product_name": "New Widget", "sku_code": "DUP", "currency_code": "USD",
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCreateSupplierAndAssignToPurchaseOrderHandler_success(t *testing.T) {
	create := &stubCreateSupplierAndAssign{}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{}, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, create,
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-orders/po-1/create-supplier-and-assign", map[string]any{
		"name": "New Supplier Co",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "po-1", create.gotInput.PurchaseOrderID)
	assert.Equal(t, "New Supplier Co", create.gotInput.Name)
}

func TestCreateSupplierAndAssignToPurchaseOrderHandler_notFound(t *testing.T) {
	create := &stubCreateSupplierAndAssign{err: procurementDomain.ErrPurchaseOrderNotFound}
	h := procurementhandler.NewHandler(
		&stubListPurchaseOrders{}, &stubCreatePurchaseOrder{}, &stubGetPurchaseOrder{}, &stubUpdatePurchaseOrder{}, &stubDeletePurchaseOrder{},
		&stubAddLineItem{}, &stubUpdateLineItem{}, &stubRemoveLineItem{}, &stubAttachDocument{}, &stubGetDocumentParsingJob{}, &stubGetReceivingParsingJob{}, &stubCreatePurchaseOrderFromDocument{}, &stubLinkLineItemToSKU{}, &stubCreateProductAndLinkLineItem{}, create,
		&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{},
		&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubResolveBookingScanCode{}, &stubDeleteBookingScan{}, &stubPauseBooking{}, &stubCompleteBooking{},
		&stubDeletePurchaseOrderDocument{}, &stubRetryDocumentParsingJob{}, &stubListBookingDiscrepancies{}, &stubGetLandedUnitCost{}, &stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-orders/missing/create-supplier-and-assign", map[string]any{
		"name": "New Supplier Co",
	})

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestUploadInvoiceHandler_success(t *testing.T) {
	upload := &stubUploadInvoice{}
	h := newReceivingTestHandler(upload, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{})
	w := doMultipartRequest(newTestRouter(h), "/api/v1/procurement/purchase-orders/po-1/invoices", "invoice.csv", "sku,qty\nA,1", nil)

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "po-1", upload.gotInput.PurchaseOrderID)
	assert.Equal(t, "invoice.csv", upload.gotInput.FileName)
	assert.Contains(t, w.Body.String(), "receiving-job-1")
}

func TestUploadInvoiceHandler_notFound(t *testing.T) {
	upload := &stubUploadInvoice{err: procurementDomain.ErrPurchaseOrderNotFound}
	h := newReceivingTestHandler(upload, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, &stubCompleteReceiving{})
	w := doMultipartRequest(newTestRouter(h), "/api/v1/procurement/purchase-orders/missing/invoices", "invoice.csv", "data", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListReceivingRecordsHandler_returnsRecords(t *testing.T) {
	list := &stubListReceivingRecords{out: []procurementDomain.ReceivingRecord{
		{ID: "receiving-record-1", PurchaseOrderID: "po-1", Status: procurementDomain.ReceivingRecordStatusPendingReview},
	}}
	h := newReceivingTestHandler(&stubUploadInvoice{}, list, &stubGetReceivingRecord{}, &stubCompleteReceiving{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/purchase-orders/po-1/receiving-records", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "receiving-record-1")
}

func TestGetReceivingRecordHandler_returnsRecordAndDiscrepancies(t *testing.T) {
	lineItemID := "line-item-1"
	get := &stubGetReceivingRecord{out: &appprocurement.ReceivingRecordDetail{
		ReceivingRecord: procurementDomain.ReceivingRecord{ID: "receiving-record-1", Status: procurementDomain.ReceivingRecordStatusPendingReview},
		Discrepancies: []procurementDomain.ReceivingDiscrepancy{
			{ID: "discrepancy-1", ReceivingRecordID: "receiving-record-1", LineItemID: &lineItemID, Type: procurementDomain.DiscrepancyTypeQuantityMismatch, ExpectedQty: 10, ReceivedQty: 8},
		},
	}}
	h := newReceivingTestHandler(&stubUploadInvoice{}, &stubListReceivingRecords{}, get, &stubCompleteReceiving{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/receiving-records/receiving-record-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "discrepancy-1")
	assert.Contains(t, w.Body.String(), "quantity_mismatch")
}

func TestGetReceivingRecordHandler_returnsDiscrepancyExpiryDate(t *testing.T) {
	lineItemID := "line-item-1"
	expiry := time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)
	get := &stubGetReceivingRecord{out: &appprocurement.ReceivingRecordDetail{
		ReceivingRecord: procurementDomain.ReceivingRecord{ID: "receiving-record-1", Status: procurementDomain.ReceivingRecordStatusPendingReview},
		Discrepancies: []procurementDomain.ReceivingDiscrepancy{
			{ID: "discrepancy-1", ReceivingRecordID: "receiving-record-1", LineItemID: &lineItemID, Type: procurementDomain.DiscrepancyTypeExpiryMismatch, ExpectedQty: 10, ReceivedQty: 10, ExpiryDate: &expiry},
			{ID: "discrepancy-2", ReceivingRecordID: "receiving-record-1", Type: procurementDomain.DiscrepancyTypeUnexpectedItem, ReceivedQty: 3},
		},
	}}
	h := newReceivingTestHandler(&stubUploadInvoice{}, &stubListReceivingRecords{}, get, &stubCompleteReceiving{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/receiving-records/receiving-record-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"expiry_date":"2026-10-03"`)
	assert.Equal(t, 1, strings.Count(w.Body.String(), `"expiry_date"`), "only the expiry_mismatch row carries an expiry_date")
}

func TestGetReceivingRecordHandler_notFound(t *testing.T) {
	get := &stubGetReceivingRecord{err: procurementDomain.ErrReceivingRecordNotFound}
	h := newReceivingTestHandler(&stubUploadInvoice{}, &stubListReceivingRecords{}, get, &stubCompleteReceiving{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/receiving-records/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestCompleteReceivingHandler_success(t *testing.T) {
	complete := &stubCompleteReceiving{}
	h := newReceivingTestHandler(&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, complete)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/receiving-records/receiving-record-1/complete", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "receiving-record-1", complete.gotInput)
	assert.Contains(t, w.Body.String(), "applied")
}

func TestCompleteReceivingHandler_notFound(t *testing.T) {
	complete := &stubCompleteReceiving{err: procurementDomain.ErrReceivingRecordNotFound}
	h := newReceivingTestHandler(&stubUploadInvoice{}, &stubListReceivingRecords{}, &stubGetReceivingRecord{}, complete)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/receiving-records/missing/complete", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAcknowledgeReceivingDiscrepancyHandler_success(t *testing.T) {
	ack := &stubAcknowledgeReceivingDiscrepancy{}
	h := newReceivingDiscrepancyActionsTestHandler(ack, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/receiving-discrepancies/discrepancy-1/acknowledge", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "discrepancy-1", ack.gotInput)
	assert.Contains(t, w.Body.String(), "acknowledged")
}

func TestAcknowledgeReceivingDiscrepancyHandler_notFound(t *testing.T) {
	ack := &stubAcknowledgeReceivingDiscrepancy{err: procurementDomain.ErrDiscrepancyNotFound}
	h := newReceivingDiscrepancyActionsTestHandler(ack, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/receiving-discrepancies/missing/acknowledge", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestAcknowledgeReceivingDiscrepancyHandler_alreadyAcknowledged(t *testing.T) {
	ack := &stubAcknowledgeReceivingDiscrepancy{err: procurementDomain.ErrDiscrepancyAlreadyAcknowledged}
	h := newReceivingDiscrepancyActionsTestHandler(ack, &stubLinkReceivingDiscrepancies{}, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/receiving-discrepancies/discrepancy-1/acknowledge", nil)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestLinkReceivingDiscrepanciesHandler_success(t *testing.T) {
	link := &stubLinkReceivingDiscrepancies{}
	h := newReceivingDiscrepancyActionsTestHandler(&stubAcknowledgeReceivingDiscrepancy{}, link, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/receiving-discrepancies/link", map[string]string{
		"unexpected_discrepancy_id": "unexpected-1", "missing_discrepancy_id": "missing-1",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "unexpected-1", link.gotUnexpected)
	assert.Equal(t, "missing-1", link.gotMissing)
}

func TestLinkReceivingDiscrepanciesHandler_mismatch(t *testing.T) {
	link := &stubLinkReceivingDiscrepancies{err: procurementDomain.ErrDiscrepancyMismatch}
	h := newReceivingDiscrepancyActionsTestHandler(&stubAcknowledgeReceivingDiscrepancy{}, link, &stubFlagDamagedLineItem{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/receiving-discrepancies/link", map[string]string{
		"unexpected_discrepancy_id": "unexpected-1", "missing_discrepancy_id": "missing-1",
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestFlagDamagedLineItemHandler_success(t *testing.T) {
	flag := &stubFlagDamagedLineItem{}
	h := newReceivingDiscrepancyActionsTestHandler(&stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, flag)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/receiving-records/receiving-record-1/line-items/line-1/damaged", map[string]any{
		"damaged_qty": 3, "note": "crushed box",
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "receiving-record-1", flag.gotReceivingRecordID)
	assert.Equal(t, "line-1", flag.gotLineItemID)
	assert.Equal(t, int64(3), flag.gotDamagedQty)
	assert.Equal(t, "crushed box", flag.gotNote)
	assert.Contains(t, w.Body.String(), "damaged_item")
}

func TestFlagDamagedLineItemHandler_invalidQuantity(t *testing.T) {
	flag := &stubFlagDamagedLineItem{err: procurementDomain.ErrInvalidDamagedQuantity}
	h := newReceivingDiscrepancyActionsTestHandler(&stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, flag)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/receiving-records/receiving-record-1/line-items/line-1/damaged", map[string]any{
		"damaged_qty": 0,
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestFlagDamagedLineItemHandler_lineItemNotFound(t *testing.T) {
	flag := &stubFlagDamagedLineItem{err: procurementDomain.ErrLineItemNotFound}
	h := newReceivingDiscrepancyActionsTestHandler(&stubAcknowledgeReceivingDiscrepancy{}, &stubLinkReceivingDiscrepancies{}, flag)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/receiving-records/receiving-record-1/line-items/missing/damaged", map[string]any{
		"damaged_qty": 1,
	})

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestStartOrResumeBookingHandler_success(t *testing.T) {
	start := &stubStartOrResumeBooking{}
	h := newBookingTestHandler(start, &stubGetBooking{}, &stubScanBookingItem{}, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-orders/po-1/booking", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "po-1", start.gotInput.PurchaseOrderID)
	assert.Contains(t, w.Body.String(), "in_progress")
}

func TestStartOrResumeBookingHandler_passesWarehouseIdFromBody(t *testing.T) {
	start := &stubStartOrResumeBooking{}
	h := newBookingTestHandler(start, &stubGetBooking{}, &stubScanBookingItem{}, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-orders/po-1/booking", map[string]any{"warehouse_id": "warehouse-1"})

	require.Equal(t, http.StatusOK, w.Code)
	require.NotNil(t, start.gotInput.WarehouseID)
	assert.Equal(t, "warehouse-1", *start.gotInput.WarehouseID)
}

func TestStartOrResumeBookingHandler_warehouseRequiredReturns422(t *testing.T) {
	start := &stubStartOrResumeBooking{err: procurementDomain.ErrWarehouseRequired}
	h := newBookingTestHandler(start, &stubGetBooking{}, &stubScanBookingItem{}, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-orders/po-1/booking", nil)

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestStartOrResumeBookingHandler_notFound(t *testing.T) {
	start := &stubStartOrResumeBooking{err: procurementDomain.ErrPurchaseOrderNotFound}
	h := newBookingTestHandler(start, &stubGetBooking{}, &stubScanBookingItem{}, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/purchase-orders/missing/booking", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetBookingHandler_returnsDetail(t *testing.T) {
	skuID := "sku-1"
	get := &stubGetBooking{out: &appprocurement.BookingDetail{
		Booking:   procurementDomain.Booking{ID: "booking-1", Status: procurementDomain.BookingStatusInProgress},
		LineItems: []procurementDomain.PurchaseOrderLineItem{{ID: "line-1", SKUID: &skuID, Quantity: 10}},
		Scans:     []procurementDomain.BookingScan{{ID: "scan-1", SKUID: "sku-1", Quantity: 4}},
	}}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, get, &stubScanBookingItem{}, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/bookings/booking-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "scan-1")
	assert.Contains(t, w.Body.String(), `"quantity":10`)
}

func TestGetBookingHandler_notFound(t *testing.T) {
	get := &stubGetBooking{err: procurementDomain.ErrBookingNotFound}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, get, &stubScanBookingItem{}, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/bookings/missing", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestScanBookingItemHandler_success(t *testing.T) {
	scan := &stubScanBookingItem{}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, scan, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/scan", map[string]any{
		"code": "BARCODE-1", "quantity": 3,
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "booking-1", scan.gotInput.BookingID)
	assert.Equal(t, "BARCODE-1", scan.gotInput.Code)
	assert.Equal(t, int64(3), scan.gotInput.Quantity)
}

func TestScanBookingItemHandler_correctionActionTypeWithReason(t *testing.T) {
	scan := &stubScanBookingItem{}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, scan, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/scan", map[string]any{
		"sku_id": "sku-1", "action_type": "correction", "quantity": 2, "batch_id": "batch-1", "reason": "counted wrong",
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, procurementDomain.BookingScanActionCorrection, scan.gotInput.ActionType)
	assert.Equal(t, int64(2), scan.gotInput.Quantity)
	assert.Equal(t, "counted wrong", scan.gotInput.Reason)
}

func TestScanBookingItemHandler_defaultsActionTypeToReceive(t *testing.T) {
	scan := &stubScanBookingItem{}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, scan, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/scan", map[string]any{
		"sku_id": "sku-1", "quantity": 3,
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, procurementDomain.BookingScanActionReceive, scan.gotInput.ActionType)
}

func TestScanBookingItemHandler_damagedActionType(t *testing.T) {
	scan := &stubScanBookingItem{}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, scan, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/scan", map[string]any{
		"sku_id": "sku-1", "action_type": "damaged", "quantity": 2, "reason": "arrived crushed",
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, procurementDomain.BookingScanActionDamaged, scan.gotInput.ActionType)
	assert.Equal(t, int64(2), scan.gotInput.Quantity)
}

func TestScanBookingItemHandler_bookDirectlyBySKUID(t *testing.T) {
	scan := &stubScanBookingItem{}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, scan, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/scan", map[string]any{
		"sku_id": "sku-1", "quantity": 4,
	})

	require.Equal(t, http.StatusCreated, w.Code)
	assert.Equal(t, "sku-1", scan.gotInput.SKUID)
	assert.Empty(t, scan.gotInput.Code)
	assert.Equal(t, int64(4), scan.gotInput.Quantity)
}

func TestScanBookingItemHandler_notActiveReturns422(t *testing.T) {
	scan := &stubScanBookingItem{err: procurementDomain.ErrBookingNotActive}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, scan, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/scan", map[string]any{
		"code": "BARCODE-1", "quantity": 3,
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestScanBookingItemHandler_skuNotFoundReturns422(t *testing.T) {
	scan := &stubScanBookingItem{err: pmsDomain.ErrSKUNotFound}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, scan, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/scan", map[string]any{
		"code": "UNKNOWN", "quantity": 1,
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestResolveBookingScanCodeHandler_success(t *testing.T) {
	resolve := &stubResolveBookingScanCode{out: &appprocurement.ResolveBookingScanCodeResult{SKUID: "sku-1", ProductName: "Widget", UnitsPerScan: 12}}
	h := newResolveBookingScanCodeTestHandler(resolve)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/resolve-scan", map[string]any{
		"code": "CASE-1",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "booking-1", resolve.gotBookingID)
	assert.Equal(t, "CASE-1", resolve.gotCode)
	var body struct {
		Data struct {
			SkuId        string `json:"sku_id"`
			ProductName  string `json:"product_name"`
			UnitsPerScan int64  `json:"units_per_scan"`
		} `json:"data"`
	}
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &body))
	assert.Equal(t, "sku-1", body.Data.SkuId)
	assert.Equal(t, "Widget", body.Data.ProductName)
	assert.Equal(t, int64(12), body.Data.UnitsPerScan)
}

func TestResolveBookingScanCodeHandler_includesWeightGramsForAWeightEmbeddedScan(t *testing.T) {
	weightGrams := int64(462)
	resolve := &stubResolveBookingScanCode{out: &appprocurement.ResolveBookingScanCodeResult{SKUID: "sku-1", ProductName: "Dry Meat", UnitsPerScan: 1, WeightGrams: &weightGrams}}
	h := newResolveBookingScanCodeTestHandler(resolve)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/resolve-scan", map[string]any{
		"code": "2994721004622",
	})

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"weight_grams":462`)
}

func TestResolveBookingScanCodeHandler_weightRequiredForPerWeightSKUReturns422(t *testing.T) {
	resolve := &stubResolveBookingScanCode{err: pmsDomain.ErrWeightRequiredForPerWeightSKU}
	h := newResolveBookingScanCodeTestHandler(resolve)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/resolve-scan", map[string]any{
		"code": "DRYMEAT-1",
	})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestResolveBookingScanCodeHandler_bookingNotFoundReturns404(t *testing.T) {
	resolve := &stubResolveBookingScanCode{err: procurementDomain.ErrBookingNotFound}
	h := newResolveBookingScanCodeTestHandler(resolve)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/resolve-scan", map[string]any{
		"code": "CASE-1",
	})

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestResolveBookingScanCodeHandler_skuNotFoundReturns422(t *testing.T) {
	resolve := &stubResolveBookingScanCode{err: pmsDomain.ErrSKUNotFound}
	h := newResolveBookingScanCodeTestHandler(resolve)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/resolve-scan", map[string]any{
		"code": "UNKNOWN",
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestDeleteBookingScanHandler_success(t *testing.T) {
	del := &stubDeleteBookingScan{}
	h := newDeleteBookingScanTestHandler(del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/procurement/bookings/booking-1/scans/scan-1", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "booking-1", del.gotBookingID)
	assert.Equal(t, "scan-1", del.gotScanID)
}

func TestDeleteBookingScanHandler_notFoundReturns404(t *testing.T) {
	del := &stubDeleteBookingScan{err: procurementDomain.ErrBookingScanNotFound}
	h := newDeleteBookingScanTestHandler(del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/procurement/bookings/booking-1/scans/scan-1", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestDeleteBookingScanHandler_notActiveReturns422(t *testing.T) {
	del := &stubDeleteBookingScan{err: procurementDomain.ErrBookingNotActive}
	h := newDeleteBookingScanTestHandler(del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/procurement/bookings/booking-1/scans/scan-1", nil)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestPauseBookingHandler_success(t *testing.T) {
	pause := &stubPauseBooking{out: &procurementDomain.Booking{ID: "booking-1", Status: procurementDomain.BookingStatusPaused}}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, pause, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/pause", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "paused")
}

func TestPauseBookingHandler_notActiveReturns422(t *testing.T) {
	pause := &stubPauseBooking{err: procurementDomain.ErrBookingNotActive}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, pause, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/pause", nil)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestCompleteBookingHandler_success(t *testing.T) {
	complete := &stubCompleteBooking{out: &appprocurement.CompleteBookingResult{
		Booking:      procurementDomain.Booking{ID: "booking-1", Status: procurementDomain.BookingStatusCompleted},
		MissingItems: []appprocurement.MissingBookingItem{{LineItemID: "line-1", RawDescription: "Widget A", ExpectedQty: 10, ScannedQty: 6}},
	}}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubPauseBooking{}, complete)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/complete", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Widget A")
	assert.Contains(t, w.Body.String(), `"expected_qty":10`)
}

func TestCompleteBookingHandler_notFound(t *testing.T) {
	complete := &stubCompleteBooking{err: procurementDomain.ErrBookingNotFound}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubPauseBooking{}, complete)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/missing/complete", nil)

	require.Equal(t, http.StatusNotFound, w.Code)
}

func TestListBookingDiscrepanciesHandler_returnsMissingAndDamagedLostItems(t *testing.T) {
	completedAt := time.Now()
	list := &stubListBookingDiscrepancies{out: &appprocurement.BookingDiscrepancyReport{
		MissingItems: []appprocurement.MissingBookingItemReport{{
			BookingID: "booking-1", PurchaseOrderID: "po-1", CompletedAt: &completedAt,
			LineItemID: "line-1", SKUCode: strPtr("WGT-1"), ProductName: strPtr("Widget"),
			RawDescription: "Widget 12pk", ExpectedQty: 10, ScannedQty: 4, MissingQty: 6,
		}},
		DamagedLost: []appprocurement.DamagedLostBookingItemReport{{
			BookingID: "booking-1", PurchaseOrderID: "po-1", ScanID: "scan-1", SKUID: "sku-1",
			SKUCode: strPtr("WGT-1"), ProductName: strPtr("Widget"),
			ActionType: procurementDomain.BookingScanActionDamaged, Quantity: 2, Reason: "box crushed",
			ScannedAt: completedAt, ScannedBy: "user-1",
		}},
	}}
	h := newBookingDiscrepanciesTestHandler(list)
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/bookings/discrepancies", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"missing_qty":6`)
	assert.Contains(t, w.Body.String(), `"product_name":"Widget"`)
	assert.Contains(t, w.Body.String(), `"action_type":"damaged"`)
	assert.Contains(t, w.Body.String(), `"reason":"box crushed"`)
}

func TestListBookingDiscrepanciesHandler_emptyReport(t *testing.T) {
	h := newBookingDiscrepanciesTestHandler(&stubListBookingDiscrepancies{})
	w := doRequest(newTestRouter(h), http.MethodGet, "/api/v1/procurement/bookings/discrepancies", nil)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"missing_items":[]`)
	assert.Contains(t, w.Body.String(), `"damaged_lost":[]`)
}

func TestDeleteBookingScanHandler_receiveWithDependentsReturns409(t *testing.T) {
	del := &stubDeleteBookingScan{err: procurementDomain.ErrBookingScanHasDependents}
	h := newDeleteBookingScanTestHandler(del)
	w := doRequest(newTestRouter(h), http.MethodDelete, "/api/v1/procurement/bookings/booking-1/scans/scan-1", nil)

	require.Equal(t, http.StatusConflict, w.Code)
	assert.Contains(t, w.Body.String(), "BOOKING_SCAN_HAS_DEPENDENTS")
}

func TestScanBookingItemHandler_pendingLotInsufficientReturns422(t *testing.T) {
	scan := &stubScanBookingItem{err: procurementDomain.ErrPendingLotInsufficient}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, scan, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/scan", map[string]any{
		"sku_id": "sku-1", "quantity": 5, "action_type": "damaged", "lot_number": "L1",
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "BOOKING_PENDING_LOT_INSUFFICIENT")
}

func TestScanBookingItemHandler_insufficientStockReturns422(t *testing.T) {
	scan := &stubScanBookingItem{err: invDomain.ErrInsufficientStock}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, scan, &stubPauseBooking{}, &stubCompleteBooking{})
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/scan", map[string]any{
		"sku_id": "sku-1", "quantity": 1, "action_type": "damaged",
	})

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), "BATCH_INSUFFICIENT_STOCK")
}

func TestCompleteBookingHandler_replayErrorCarriesTheRowAsDetail(t *testing.T) {
	complete := &stubCompleteBooking{err: fmt.Errorf("completeBooking: %w: %s", procurementDomain.ErrPendingBatchNotFound, "Pão de queijo 1kg · 31/10/2027")}
	h := newBookingTestHandler(&stubStartOrResumeBooking{}, &stubGetBooking{}, &stubScanBookingItem{}, &stubPauseBooking{}, complete)
	w := doRequest(newTestRouter(h), http.MethodPost, "/api/v1/procurement/bookings/booking-1/complete", nil)

	require.Equal(t, http.StatusUnprocessableEntity, w.Code)
	assert.Contains(t, w.Body.String(), `"code":"BOOKING_PENDING_BATCH_NOT_FOUND"`)
	assert.Contains(t, w.Body.String(), `"detail":"Pão de queijo 1kg · 31/10/2027"`)
}
