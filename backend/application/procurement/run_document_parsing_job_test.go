package procurement_test

import (
	"context"
	"testing"
	"time"

	appprocurement "github.com/adirkuhn/commerceos/application/procurement"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunDocumentParsingJob_structuredDocument_createsLineItems(t *testing.T) {
	db := newMockGormDBN(t, 2)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.csv", FileName: "order.csv", ContentType: "text/csv"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusDraft}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubCountingLineItemRepo{}
	documentRepo := &stubDocumentRepo{findResult: doc}
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"WID-001": {ID: "sku-1", SKUCode: "WID-001"}}}
	aliasRepo := &stubAliasRepo{}
	matchUC := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)
	storage := &stubFileStorage{getData: []byte("SKU,Description,Quantity,Unit Cost\nWID-001,Widget A,10,5.00\nWID-002,Widget B,3,12.50\n")}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, lineItemRepo, documentRepo, matchUC, storage, &stubPDFTextExtractor{}, nil)

	needsReview, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 2)
	assert.Equal(t, "WID-001", lineItemRepo.created[0].RawSKU)
	require.NotNil(t, lineItemRepo.created[0].SKUID)
	assert.Equal(t, "sku-1", *lineItemRepo.created[0].SKUID)
	assert.Nil(t, lineItemRepo.created[1].SKUID, "WID-002 has no matching SKU or alias")
	assert.Equal(t, int64(0), lineItemRepo.created[0].Position, "position must track document order, not be left at the zero value for every row")
	assert.Equal(t, int64(1), lineItemRepo.created[1].Position)
	require.Len(t, poRepo.updated, 1, "PO must move to needs_review since WID-002 is unmatched")
	assert.Equal(t, procurementDomain.PurchaseOrderStatusNeedsReview, poRepo.updated[0].Status)
	assert.True(t, needsReview)
}

func TestRunDocumentParsingJob_flagsAliasMatchedLineItemsAsAutoLinked(t *testing.T) {
	db := newMockGormDBN(t, 2)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.csv", FileName: "order.csv", ContentType: "text/csv"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusDraft}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubCountingLineItemRepo{}
	documentRepo := &stubDocumentRepo{findResult: doc}
	aliasRepo := &stubAliasRepo{byKey: map[string]*procurementDomain.SupplierProductAlias{
		"supplier-1|wid-002": {SupplierID: "supplier-1", RawKey: "wid-002", SKUID: "sku-learned-1"},
	}}
	matchUC := appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, &stubProductRepo{}, &stubVariantRepo{}, aliasRepo)
	storage := &stubFileStorage{getData: []byte("SKU,Description,Quantity,Unit Cost\nWID-002,Widget B,3,12.50\n")}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, lineItemRepo, documentRepo, matchUC, storage, &stubPDFTextExtractor{}, nil)

	_, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 1)
	require.NotNil(t, lineItemRepo.created[0].SKUID)
	assert.Equal(t, "sku-learned-1", *lineItemRepo.created[0].SKUID)
	assert.True(t, lineItemRepo.created[0].SKUAutoLinkedFromAlias, "an alias match is a guess, not a confirmed link")
}

func TestRunDocumentParsingJob_doesNotFlagExactSKUMatchesAsAutoLinked(t *testing.T) {
	db := newMockGormDBN(t, 2)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.csv", FileName: "order.csv", ContentType: "text/csv"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusDraft}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubCountingLineItemRepo{}
	documentRepo := &stubDocumentRepo{findResult: doc}
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"WID-001": {ID: "sku-1", SKUCode: "WID-001"}}}
	matchUC := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{})
	storage := &stubFileStorage{getData: []byte("SKU,Description,Quantity,Unit Cost\nWID-001,Widget A,10,5.00\n")}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, lineItemRepo, documentRepo, matchUC, storage, &stubPDFTextExtractor{}, nil)

	_, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 1)
	assert.False(t, lineItemRepo.created[0].SKUAutoLinkedFromAlias, "exact-SKU matches are unambiguous and supplier-independent -- never revisited")
}

func TestRunDocumentParsingJob_chargeRowNeverTriggersNeedsReview(t *testing.T) {
	db := newMockGormDBN(t, 2)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.pdf", FileName: "order.pdf", ContentType: "application/pdf"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusDraft}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubCountingLineItemRepo{}
	documentRepo := &stubDocumentRepo{findResult: doc}
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"WID-001": {ID: "sku-1", SKUCode: "WID-001"}}}
	matchUC := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{})
	storage := &stubFileStorage{getData: []byte("%PDF-fake-bytes")}
	pdfExtractor := &stubPDFTextExtractor{text: "PO #123\nWidget A x10 @ $5.00\nShipping $15.00"}
	aiParser := &stubAIDocumentParser{out: []procurementDomain.ExtractedLineItem{
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500, LineType: procurementDomain.LineItemTypeProduct},
		{RawDescription: "Shipping", Quantity: 1, UnitCostMinorUnits: 1500, LineType: procurementDomain.LineItemTypeShipping},
	}}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, lineItemRepo, documentRepo, matchUC, storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	needsReview, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 2)
	assert.Equal(t, procurementDomain.LineItemTypeShipping, lineItemRepo.created[1].LineType)
	assert.Nil(t, lineItemRepo.created[1].SKUID)
	assert.False(t, needsReview, "a fully-matched product plus an unmatched charge row must not require review")
	assert.Empty(t, poRepo.updated, "must stay draft -- the shipping row is not an unmatched product")
}

func TestRunDocumentParsingJob_setsNeedsReviewWhenAnyLineItemUnmatched(t *testing.T) {
	db := newMockGormDBN(t, 2)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.csv", FileName: "order.csv", ContentType: "text/csv"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusDraft}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubCountingLineItemRepo{}
	documentRepo := &stubDocumentRepo{findResult: doc}
	matchUC := appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{})
	storage := &stubFileStorage{getData: []byte("SKU,Description,Quantity,Unit Cost\nUNKNOWN,Mystery,1,0\n")}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, lineItemRepo, documentRepo, matchUC, storage, &stubPDFTextExtractor{}, nil)

	needsReview, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	require.NoError(t, err)
	require.Len(t, poRepo.updated, 1)
	assert.Equal(t, procurementDomain.PurchaseOrderStatusNeedsReview, poRepo.updated[0].Status)
	assert.True(t, needsReview)
}

func TestRunDocumentParsingJob_pdfDocument_usesAIWhenTextExtracted(t *testing.T) {
	db := newMockGormDBN(t, 2)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.pdf", FileName: "order.pdf", ContentType: "application/pdf"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusDraft}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubCountingLineItemRepo{}
	documentRepo := &stubDocumentRepo{findResult: doc}
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"WID-001": {ID: "sku-1", SKUCode: "WID-001"}}}
	matchUC := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{})
	storage := &stubFileStorage{getData: []byte("%PDF-fake-bytes")}
	pdfExtractor := &stubPDFTextExtractor{text: "PO #123\nWidget A x10 @ $5.00"}
	aiParser := &stubAIDocumentParser{out: []procurementDomain.ExtractedLineItem{
		{RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500},
	}}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, lineItemRepo, documentRepo, matchUC, storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	needsReview, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 1)
	assert.Equal(t, "application/pdf", aiParser.gotInput.ContentType)
	require.NotNil(t, lineItemRepo.created[0].SKUID)
	assert.Equal(t, "sku-1", *lineItemRepo.created[0].SKUID)
	assert.False(t, needsReview)
}

func TestRunDocumentParsingJob_pdfDocument_carriesBatchCodeAndExpiryDateAsSuggestions(t *testing.T) {
	db := newMockGormDBN(t, 2)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.pdf", FileName: "invoice.pdf", ContentType: "application/pdf"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusDraft}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubCountingLineItemRepo{}
	documentRepo := &stubDocumentRepo{findResult: doc}
	skuRepo := &stubSKURepo{byCode: map[string]*pmsDomain.SKU{"PT26004": {ID: "sku-1", SKUCode: "PT26004"}}}
	matchUC := appprocurement.NewMatchExtractedLineItemsUseCase(skuRepo, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{})
	storage := &stubFileStorage{getData: []byte("%PDF-fake-bytes")}
	pdfExtractor := &stubPDFTextExtractor{text: "PT26004 01-mar-2027 100131 FAROFA PREMIUM"}
	expiry := time.Date(2027, 3, 1, 0, 0, 0, 0, time.UTC)
	aiParser := &stubAIDocumentParser{out: []procurementDomain.ExtractedLineItem{
		{RawSKU: "PT26004", RawDescription: "Farofa Premium", Quantity: 36, UnitCostMinorUnits: 179, BatchCode: "PT26004", ExpiryDate: &expiry},
	}}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, lineItemRepo, documentRepo, matchUC, storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	_, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	require.NoError(t, err)
	require.Len(t, lineItemRepo.created, 1)
	assert.Equal(t, "PT26004", lineItemRepo.created[0].SuggestedLotNumber)
	require.NotNil(t, lineItemRepo.created[0].SuggestedExpiryDate)
	assert.True(t, expiry.Equal(*lineItemRepo.created[0].SuggestedExpiryDate))
}

func TestRunDocumentParsingJob_pdfWithNoExtractableText_failsWithoutCallingAI(t *testing.T) {
	db := newMockGormDBN(t, 1)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.pdf", FileName: "order.pdf", ContentType: "application/pdf"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1")}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	documentRepo := &stubDocumentRepo{findResult: doc}
	storage := &stubFileStorage{getData: []byte("%PDF-scanned-image")}
	pdfExtractor := &stubPDFTextExtractor{text: "   "}
	aiParser := &stubAIDocumentParser{}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, &stubLineItemRepo{}, documentRepo, appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{}), storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	_, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	assert.ErrorIs(t, err, procurementDomain.ErrPDFHasNoExtractableText)
	assert.Empty(t, aiParser.gotInput.FileName, "AI must not be called when there's no text to justify the call")
}

func TestRunDocumentParsingJob_pdfWithoutAIConfigured_failsClearly(t *testing.T) {
	db := newMockGormDBN(t, 1)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.pdf", FileName: "order.pdf", ContentType: "application/pdf"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1")}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	documentRepo := &stubDocumentRepo{findResult: doc}
	storage := &stubFileStorage{getData: []byte("%PDF-with-text")}
	pdfExtractor := &stubPDFTextExtractor{text: "some real text"}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, &stubLineItemRepo{}, documentRepo, appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{}), storage, pdfExtractor, &stubAIParserResolver{})

	_, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	assert.ErrorIs(t, err, procurementDomain.ErrAIParsingNotConfigured)
}

func TestRunDocumentParsingJob_supplierlessPDF_writesSuggestedSupplierAndCurrency(t *testing.T) {
	db := newMockGormDBN(t, 2)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.pdf", FileName: "order.pdf", ContentType: "application/pdf"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: nil, Status: procurementDomain.PurchaseOrderStatusNeedsReview}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubCountingLineItemRepo{}
	documentRepo := &stubDocumentRepo{findResult: doc}
	matchUC := appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{})
	storage := &stubFileStorage{getData: []byte("%PDF-fake-bytes")}
	pdfExtractor := &stubPDFTextExtractor{text: "Confirmação de Encomenda\nMoeda EUR\nPEDAÇOS DO BRASIL, LDA."}
	aiParser := &stubAIDocumentParser{
		out:          []procurementDomain.ExtractedLineItem{{RawSKU: "ANT001", RawDescription: "Guarana Antarctica", Quantity: 42, UnitCostMinorUnits: 136}},
		supplierName: "PEDAÇOS DO BRASIL, LDA.",
	}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, lineItemRepo, documentRepo, matchUC, storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	_, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	require.NoError(t, err)
	require.Len(t, poRepo.suggestionsSet, 1)
	assert.Equal(t, "po-1", poRepo.suggestionsSet[0].id)
	assert.Equal(t, "PEDAÇOS DO BRASIL, LDA.", poRepo.suggestionsSet[0].supplierName)
	assert.Equal(t, "EUR", poRepo.suggestionsSet[0].currencyCode)
}

func TestRunDocumentParsingJob_doesNotWriteSuggestionsWhenSupplierAlreadySet(t *testing.T) {
	db := newMockGormDBN(t, 2)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.pdf", FileName: "order.pdf", ContentType: "application/pdf"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: strPtr("supplier-1"), Status: procurementDomain.PurchaseOrderStatusDraft}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	lineItemRepo := &stubCountingLineItemRepo{}
	documentRepo := &stubDocumentRepo{findResult: doc}
	matchUC := appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{})
	storage := &stubFileStorage{getData: []byte("%PDF-fake-bytes")}
	pdfExtractor := &stubPDFTextExtractor{text: "Moeda EUR"}
	aiParser := &stubAIDocumentParser{
		out:          []procurementDomain.ExtractedLineItem{{RawSKU: "ANT001", RawDescription: "Guarana Antarctica", Quantity: 42, UnitCostMinorUnits: 136}},
		supplierName: "PEDAÇOS DO BRASIL, LDA.",
	}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, lineItemRepo, documentRepo, matchUC, storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	_, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	require.NoError(t, err)
	assert.Empty(t, poRepo.suggestionsSet, "a PO that already has a supplier doesn't need a suggestion")
}

// TestRunDocumentParsingJob_pdfWithEmptyAIResult_failsInsteadOfSilentlySucceeding
// is the actual fix for "supplier gets suggested but line items never
// parse" -- the AI/PDF path used to accept an empty line_items array from
// the model as a successful parse (job marked Completed with a correct
// supplier suggestion and zero line items, no error anywhere), unlike the
// deterministic CSV/XLSX path which already guards this via
// ErrNoLineItemsExtracted.
func TestRunDocumentParsingJob_pdfWithEmptyAIResult_failsInsteadOfSilentlySucceeding(t *testing.T) {
	db := newMockGormDBN(t, 1)
	doc := &procurementDomain.PurchaseOrderDocument{ID: "document-1", FileURL: "https://example.com/doc.pdf", FileName: "order.pdf", ContentType: "application/pdf"}
	po := &procurementDomain.PurchaseOrder{ID: "po-1", SupplierID: nil}
	poRepo := &stubPurchaseOrderRepo{findResult: po}
	documentRepo := &stubDocumentRepo{findResult: doc}
	storage := &stubFileStorage{getData: []byte("%PDF-fake-bytes")}
	pdfExtractor := &stubPDFTextExtractor{text: "Invoice from Acme Corp"}
	aiParser := &stubAIDocumentParser{out: nil, supplierName: "Acme Corp"}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, poRepo, &stubLineItemRepo{}, documentRepo, appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{}), storage, pdfExtractor, &stubAIParserResolver{parser: aiParser})

	_, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", PurchaseOrderID: "po-1", DocumentID: "document-1"})

	assert.ErrorIs(t, err, procurementDomain.ErrNoLineItemsExtracted)
	assert.Empty(t, poRepo.suggestionsSet, "must fail before ever writing the (correct but incomplete) supplier suggestion")
}

func TestRunDocumentParsingJob_propagatesDocumentNotFound(t *testing.T) {
	db := newMockGormDBN(t, 1)
	documentRepo := &stubDocumentRepo{findErr: procurementDomain.ErrDocumentNotFound}
	uc := appprocurement.NewRunDocumentParsingJobUseCase(db, &stubPurchaseOrderRepo{}, &stubLineItemRepo{}, documentRepo, appprocurement.NewMatchExtractedLineItemsUseCase(&stubSKURepo{}, &stubProductRepo{}, &stubVariantRepo{}, &stubAliasRepo{}), &stubFileStorage{}, &stubPDFTextExtractor{}, nil)

	_, err := uc.Execute(context.Background(), procurementDomain.DocumentParsingJob{ID: "job-1", DocumentID: "missing"})

	assert.ErrorIs(t, err, procurementDomain.ErrDocumentNotFound)
}

// stubCountingLineItemRepo is like stubLineItemRepo but its Create tracks
// each created item (not just a bool) -- needed to inspect what
// RunDocumentParsingJobUseCase actually wrote per line item.
type stubCountingLineItemRepo struct {
	created []procurementDomain.PurchaseOrderLineItem
}

func (s *stubCountingLineItemRepo) Create(_ context.Context, li *procurementDomain.PurchaseOrderLineItem) error {
	li.ID = "line-item-generated"
	s.created = append(s.created, *li)
	return nil
}
func (s *stubCountingLineItemRepo) FindByID(_ context.Context, _ string) (*procurementDomain.PurchaseOrderLineItem, error) {
	return nil, nil
}
func (s *stubCountingLineItemRepo) ListByPurchaseOrder(_ context.Context, _ string) ([]procurementDomain.PurchaseOrderLineItem, error) {
	return nil, nil
}
func (s *stubCountingLineItemRepo) Update(_ context.Context, _ *procurementDomain.PurchaseOrderLineItem) error {
	return nil
}
func (s *stubCountingLineItemRepo) Delete(_ context.Context, _ string) error { return nil }
func (s *stubCountingLineItemRepo) SumByPurchaseOrders(_ context.Context, _ []string) (map[string]int64, error) {
	return map[string]int64{}, nil
}
