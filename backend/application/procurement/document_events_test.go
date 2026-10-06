package procurement_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/application/eventrecord/eventrecordtest"
	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func TestDeletePurchaseOrderDocument_RecordsDeleted(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	docs := &stubDocumentRepo{findResult: &procurementDomain.PurchaseOrderDocument{ID: "doc-1", PurchaseOrderID: "po-1", FileName: "order.pdf", FileURL: "u"}}
	uc := appprocurement.NewDeletePurchaseOrderDocumentUseCase(newMockGormDB(t), docs, &stubFileStorage{}, outbox)

	require.NoError(t, uc.Execute(context.Background(), "doc-1"))

	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.purchase_order_document.deleted", e.Subject)
	assert.Equal(t, "doc-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1", "file_name": "order.pdf"}, e.Data())
}

func TestRetryDocumentParsingJob_RecordsRetried(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	docs := &stubDocumentRepo{findResult: &procurementDomain.PurchaseOrderDocument{ID: "doc-1", PurchaseOrderID: "po-1"}}
	uc := appprocurement.NewRetryDocumentParsingJobUseCase(newMockGormDB(t), docs, &stubParsingJobRepo{}, outbox)

	_, err := uc.Execute(context.Background(), "doc-1")

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.purchase_order_document.parsing_retried", e.Subject)
	assert.Equal(t, map[string]any{"purchase_order_id": "po-1", "parsing_job_id": "job-1"}, e.Data())
}

// Parsing adds each line (source document), flips the PO to needs_review
// when a product line is unmatched, and summarizes the document.
func TestRunDocumentParsingJob_RecordsLinesReviewAndParsed(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	doc := &procurementDomain.PurchaseOrderDocument{ID: "doc-1", FileURL: "u", FileName: "order.csv", ContentType: "text/csv"}
	poRepo := &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("sup-1"), Status: procurementDomain.PurchaseOrderStatusDraft}}
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"WID-001": {ID: "sku-1", SKUCode: "WID-001"}}}
	matchUC := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{})
	storage := &stubFileStorage{getData: []byte("SKU,Description,Quantity,Unit Cost\nWID-001,Widget A,10,5.00\nWID-002,Widget B,3,12.50\n")}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(newMockGormDBN(t, 2), poRepo, &stubCountingLineItemRepo{}, &stubDocumentRepo{findResult: doc},
		matchUC, storage, &stubPDFTextExtractor{}, nil, outbox)

	_, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "doc-1"})

	require.NoError(t, err)
	require.Equal(t, []string{
		"procurement.purchase_order_line_item.added", "procurement.purchase_order_line_item.added",
		"procurement.purchase_order.updated", "procurement.purchase_order_document.parsed",
	}, outbox.Subjects())
	assert.Equal(t, "document", outbox.Events[0].Data()["source"])
	assert.Equal(t, map[string][2]any{"status": {"draft", "needs_review"}}, outbox.Events[2].Changes())
	assert.Equal(t, map[string]any{"reason": "line_items_unmatched"}, outbox.Events[2].Data())
	parsed := outbox.Events[3]
	assert.Equal(t, "doc-1", parsed.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"purchase_order_id": "po-1", "parsing_job_id": "job-1", "line_count": float64(2), "unmatched_count": float64(1),
		"suggested_supplier_name": "", "suggested_currency_code": "",
	}, parsed.Data())
}

func TestUploadInvoice_RecordsReceivingRecordCreated(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	uc := appprocurement.NewUploadInvoiceUseCase(newMockGormDBN(t, 2), &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}},
		&stubReceivingRecordRepo{}, &stubReceivingParsingJobRepo{}, &stubFileStorage{}, outbox)

	_, err := uc.Execute(context.Background(), appprocurement.UploadInvoiceInput{
		PurchaseOrderID: "po-1", FileName: "invoice.csv", ContentType: "text/csv", Data: strings.NewReader("x"),
	})

	require.NoError(t, err)
	e := onlyEvent(t, outbox)
	assert.Equal(t, "procurement.receiving_record.created", e.Subject)
	assert.Equal(t, "receiving-record-1", e.Payload["entity_id"])
	assert.Equal(t, map[string]any{
		"purchase_order_id": "po-1", "file_name": "invoice.csv", "content_type": "text/csv",
		"parsing_job_id": "receiving-job-1", "potential_duplicate": false,
	}, e.Data())
}

func TestRunReceivingParsingJob_RecordsDiscrepanciesAndParsed(t *testing.T) {
	outbox := &eventrecordtest.Outbox{}
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	storage.getData = []byte("SKU,Description,Quantity\nWID-001,Widget A,8\n")
	uc := appprocurement.NewRunReceivingParsingJobUseCase(newMockGormDBN(t, 2), recRepo, lineItemRepo, discrepancyRepo, storage, &stubPDFTextExtractor{}, nil, outbox)

	_, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	require.Equal(t, []string{"procurement.receiving_discrepancy.detected", "procurement.receiving_record.parsed"}, outbox.Subjects())
	assert.Equal(t, map[string]any{
		"receiving_record_id": "receiving-record-1", "type": "quantity_mismatch", "line_item_id": "line-1",
		"expected_qty": float64(10), "received_qty": float64(8), "source": "invoice",
	}, outbox.Events[0].Data())
	assert.Equal(t, "receiving-record-1", outbox.Events[1].Payload["entity_id"])
	assert.Equal(t, float64(1), outbox.Events[1].Data()["discrepancy_count"])
}

func TestDocumentUseCases_FailWhenTheEventCantBeRecorded(t *testing.T) {
	boom := errors.New("outbox down")
	failing := &eventrecordtest.Outbox{Err: boom}
	docs := func() *stubDocumentRepo {
		return &stubDocumentRepo{findResult: &procurementDomain.PurchaseOrderDocument{ID: "doc-1", PurchaseOrderID: "po-1"}}
	}

	assert.ErrorIs(t, appprocurement.NewDeletePurchaseOrderDocumentUseCase(newMockGormDB(t), docs(), &stubFileStorage{}, failing).Execute(context.Background(), "doc-1"), boom)
	_, err := appprocurement.NewRetryDocumentParsingJobUseCase(newMockGormDB(t), docs(), &stubParsingJobRepo{}, failing).Execute(context.Background(), "doc-1")
	assert.ErrorIs(t, err, boom)
	_, err = appprocurement.NewUploadInvoiceUseCase(newMockGormDBN(t, 2), &stubPurchaseOrderRepo{findResult: &procurementDomain.PurchaseOrder{ID: "po-1"}},
		&stubReceivingRecordRepo{}, &stubReceivingParsingJobRepo{}, &stubFileStorage{}, failing).
		Execute(context.Background(), appprocurement.UploadInvoiceInput{PurchaseOrderID: "po-1", Data: strings.NewReader("x")})
	assert.ErrorIs(t, err, boom)
}
