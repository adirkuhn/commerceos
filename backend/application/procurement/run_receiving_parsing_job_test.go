package procurement_test

import (
	"context"
	"errors"
	"testing"
	"time"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newReceivingJobFixture() (*stubReceivingRecordRepo, *stubLineItemRepo, *stubReceivingDiscrepancyRepo, *stubFileStorage) {
	recRepo := &stubReceivingRecordRepo{findResult: &procurementDomain.ReceivingRecord{
		ID: "receiving-record-1", PurchaseOrderID: "po-1", InvoiceFileURL: "https://example.com/invoice.csv",
		InvoiceFileName: "invoice.csv", ContentType: "text/csv",
	}}
	lineItemRepo := &stubLineItemRepo{byPO: []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10},
	}}
	discrepancyRepo := &stubReceivingDiscrepancyRepo{}
	storage := &stubFileStorage{getData: []byte("SKU,Description,Quantity\nWID-001,Widget A,10\n")}
	return recRepo, lineItemRepo, discrepancyRepo, storage
}

func TestRunReceivingParsingJob_noDiscrepanciesWhenInvoiceMatchesPO(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, &stubPDFTextExtractor{}, nil)

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, discrepancyRepo.created)
}

func TestRunReceivingParsingJob_quantityMismatch(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	storage.getData = []byte("SKU,Description,Quantity\nWID-001,Widget A,8\n")
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, &stubPDFTextExtractor{}, nil)

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, discrepancyRepo.created, 1)
	d := discrepancyRepo.created[0]
	assert.Equal(t, procurementDomain.DiscrepancyTypeQuantityMismatch, d.Type)
	assert.Equal(t, int64(10), d.ExpectedQty)
	assert.Equal(t, int64(8), d.ReceivedQty)
	require.NotNil(t, d.LineItemID)
	assert.Equal(t, "line-1", *d.LineItemID)
	assert.Equal(t, "receiving-record-1", d.ReceivingRecordID)
}

func TestRunReceivingParsingJob_missingItemWhenPOLineNotOnInvoice(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	storage.getData = []byte("SKU,Description,Quantity\nOTHER-SKU,Other thing,5\n")
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, &stubPDFTextExtractor{}, nil)

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 2, count)
	types := map[procurementDomain.DiscrepancyType]bool{}
	for _, d := range discrepancyRepo.created {
		types[d.Type] = true
	}
	assert.True(t, types[procurementDomain.DiscrepancyTypeMissingItem])
	assert.True(t, types[procurementDomain.DiscrepancyTypeUnexpectedItem])
}

func TestRunReceivingParsingJob_expiryMismatchWhenExpiringSoon(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	soon := time.Now().AddDate(0, 0, 5).Format("2006-01-02")
	storage.getData = []byte("SKU,Description,Quantity,Expiry\nWID-001,Widget A,10," + soon + "\n")
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, &stubPDFTextExtractor{}, nil)

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, procurementDomain.DiscrepancyTypeExpiryMismatch, discrepancyRepo.created[0].Type)
}

func TestRunReceivingParsingJob_noExpiryMismatchWhenExpiryFarInFuture(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	storage.getData = []byte("SKU,Description,Quantity,Expiry\nWID-001,Widget A,10,2030-01-01\n")
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, &stubPDFTextExtractor{}, nil)

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}

func TestRunReceivingParsingJob_propagatesReceivingRecordNotFound(t *testing.T) {
	db := newMockGormDBN(t, 1)
	recRepo := &stubReceivingRecordRepo{findErr: procurementDomain.ErrReceivingRecordNotFound}
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, &stubLineItemRepo{}, &stubReceivingDiscrepancyRepo{}, &stubFileStorage{}, &stubPDFTextExtractor{}, nil)

	_, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "missing"})

	assert.ErrorIs(t, err, procurementDomain.ErrReceivingRecordNotFound)
}

func TestRunReceivingParsingJob_propagatesStorageGetError(t *testing.T) {
	db := newMockGormDBN(t, 1)
	recRepo, lineItemRepo, discrepancyRepo, _ := newReceivingJobFixture()
	storage := &stubFileStorage{getErr: errors.New("not found")}
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, &stubPDFTextExtractor{}, nil)

	_, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.Error(t, err)
}

func TestRunReceivingParsingJob_pdfWithNoAIParserConfiguredReturnsError(t *testing.T) {
	db := newMockGormDBN(t, 1)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	recRepo.findResult.ContentType = "application/pdf"
	recRepo.findResult.InvoiceFileName = "invoice.pdf"
	storage.getData = []byte("not extractable as csv")
	pdfExtractor := &stubPDFTextExtractor{text: "Widget A x10"}
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, pdfExtractor, &stubAIParserResolver{})

	_, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	assert.ErrorIs(t, err, procurementDomain.ErrAIParsingNotConfigured)
}

// TestRunReceivingParsingJob_fuzzyDescriptionMatchWhenRawSKUMissingOnOneSide
// is the reported bug's exact repro: PO and invoice come from two
// independent extraction passes of the same document, so RawSKU can be
// present on one side and blank on the other, with RawDescription differing
// by whitespace/case/accents. The two must still resolve to the same line
// item instead of showing up as both missing_item and unexpected_item.
func TestRunReceivingParsingJob_fuzzyDescriptionMatchWhenRawSKUMissingOnOneSide(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	recRepo.findResult.ContentType = "application/pdf"
	recRepo.findResult.InvoiceFileName = "invoice.pdf"
	lineItemRepo.byPO = []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", RawSKU: "7897122604130", RawDescription: "Mandioquinha Cozida no Vapor Vapza 12 x 500g", Quantity: 108},
	}
	storage.getData = []byte("pdf bytes")
	pdfExtractor := &stubPDFTextExtractor{text: "invoice text"}
	aiParser := &stubAIDocumentParser{out: []procurementDomain.ExtractedLineItem{
		// no RawSKU captured this pass, description differs by case/spacing only
		{RawSKU: "", RawDescription: "MANDIOQUINHA  COZIDA NO VAPOR VAPZA 12 X 500G", Quantity: 108},
	}}
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, discrepancyRepo.created)
}

// TestRunReceivingParsingJob_duplicateRawSKUAcrossTwoPOLinesBothTracked
// guards against the old map[key]*LineItem indexing that silently dropped a
// line item whenever two PO lines shared a RawSKU.
func TestRunReceivingParsingJob_duplicateRawSKUAcrossTwoPOLinesBothTracked(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	lineItemRepo.byPO = []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 5},
		{ID: "line-2", RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 3},
	}
	storage.getData = []byte("SKU,Description,Quantity\nWID-001,Widget A,5\n")
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, &stubPDFTextExtractor{}, nil)

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, discrepancyRepo.created, 1)
	d := discrepancyRepo.created[0]
	assert.Equal(t, procurementDomain.DiscrepancyTypeMissingItem, d.Type)
	require.NotNil(t, d.LineItemID)
	assert.Equal(t, "line-2", *d.LineItemID)
}

// TestRunReceivingParsingJob_batchSplitInvoiceLinesAggregateAgainstOnePOLine
// covers a supplier splitting one PO line across multiple invoice lines by
// batch/expiry -- these must sum before comparing to the ordered quantity,
// not each produce their own quantity_mismatch.
func TestRunReceivingParsingJob_batchSplitInvoiceLinesAggregateAgainstOnePOLine(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	recRepo.findResult.ContentType = "application/pdf"
	recRepo.findResult.InvoiceFileName = "invoice.pdf"
	lineItemRepo.byPO = []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 100},
	}
	storage.getData = []byte("pdf bytes")
	farFuture := time.Now().AddDate(1, 0, 0)
	pdfExtractor := &stubPDFTextExtractor{text: "invoice text"}
	aiParser := &stubAIDocumentParser{out: []procurementDomain.ExtractedLineItem{
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 50, BatchCode: "B1", ExpiryDate: &farFuture},
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 50, BatchCode: "B2", ExpiryDate: &farFuture},
	}}
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, discrepancyRepo.created)
}

func TestRunReceivingParsingJob_batchSplitInvoiceLinesAggregateToSingleQuantityMismatch(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	recRepo.findResult.ContentType = "application/pdf"
	recRepo.findResult.InvoiceFileName = "invoice.pdf"
	lineItemRepo.byPO = []procurementDomain.PurchaseOrderLineItem{
		{ID: "line-1", RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 100},
	}
	storage.getData = []byte("pdf bytes")
	farFuture := time.Now().AddDate(1, 0, 0)
	pdfExtractor := &stubPDFTextExtractor{text: "invoice text"}
	aiParser := &stubAIDocumentParser{out: []procurementDomain.ExtractedLineItem{
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 50, BatchCode: "B1", ExpiryDate: &farFuture},
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 40, BatchCode: "B2", ExpiryDate: &farFuture},
	}}
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, discrepancyRepo.created, 1)
	d := discrepancyRepo.created[0]
	assert.Equal(t, procurementDomain.DiscrepancyTypeQuantityMismatch, d.Type)
	assert.Equal(t, int64(100), d.ExpectedQty)
	assert.Equal(t, int64(90), d.ReceivedQty)
}

// TestRunReceivingParsingJob_noPlausibleMatchStaysMissingAndUnexpected is a
// regression guard: two genuinely unrelated items must NOT be fuzzy-matched
// to each other, and stay as an ordinary missing_item/unexpected_item pair.
func TestRunReceivingParsingJob_noPlausibleMatchStaysMissingAndUnexpected(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	storage.getData = []byte("SKU,Description,Quantity\nOTHER-SKU,Other thing,5\n")
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, &stubPDFTextExtractor{}, nil)

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 2, count)
	types := map[procurementDomain.DiscrepancyType]bool{}
	for _, d := range discrepancyRepo.created {
		types[d.Type] = true
	}
	assert.True(t, types[procurementDomain.DiscrepancyTypeMissingItem])
	assert.True(t, types[procurementDomain.DiscrepancyTypeUnexpectedItem])
}

func TestRunReceivingParsingJob_pdfUsesAIParserWhenConfigured(t *testing.T) {
	db := newMockGormDBN(t, 2)
	recRepo, lineItemRepo, discrepancyRepo, storage := newReceivingJobFixture()
	recRepo.findResult.ContentType = "application/pdf"
	recRepo.findResult.InvoiceFileName = "invoice.pdf"
	storage.getData = []byte("pdf bytes")
	pdfExtractor := &stubPDFTextExtractor{text: "Widget A x10"}
	aiParser := &stubAIDocumentParser{out: []procurementDomain.ExtractedLineItem{
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10},
	}}
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
