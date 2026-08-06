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
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, pdfExtractor, nil)

	_, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	assert.ErrorIs(t, err, procurementDomain.ErrAIParsingNotConfigured)
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
	uc := appprocurement.NewRunReceivingParsingJobUseCase(db, recRepo, lineItemRepo, discrepancyRepo, storage, pdfExtractor, aiParser)

	count, err := uc.Execute(context.Background(), procurementDomain.ReceivingParsingJob{ID: "job-1", ReceivingRecordID: "receiving-record-1"})

	require.NoError(t, err)
	assert.Equal(t, 0, count)
}
