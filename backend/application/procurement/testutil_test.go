package procurement_test

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
)

func strPtr(s string) *string { return &s }

func newMockGormDB(t *testing.T) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.ExpectBegin()
	mock.MatchExpectationsInOrder(false)
	mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectCommit()

	return gormDB
}

// newMockGormDBN is like newMockGormDB but for use cases that open more than
// one tenant.WithTenant transaction (e.g. AttachDocumentToPurchaseOrderUseCase:
// one to validate the PO exists, one to record the document+job rows, with
// the storage write in between -- same shape as pms.UploadMediaUseCase).
func newMockGormDBN(t *testing.T, n int) *gorm.DB {
	t.Helper()
	mockDB, mock, err := sqlmock.New()
	require.NoError(t, err)
	t.Cleanup(func() { mockDB.Close() })

	gormDB, err := gorm.Open(postgres.New(postgres.Config{Conn: mockDB}), &gorm.Config{})
	require.NoError(t, err)

	mock.MatchExpectationsInOrder(false)
	for i := 0; i < n; i++ {
		mock.ExpectBegin()
		mock.ExpectExec(`SELECT set_config`).WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()
	}
	return gormDB
}

// ── Stub repos ────────────────────────────────────────────────────────────────

type stubPurchaseOrderRepo struct {
	purchaseOrders []procurementDomain.PurchaseOrder
	findResult     *procurementDomain.PurchaseOrder
	findErr        error
	// byID, when set, answers FindByID per-id -- lets a single stub stand
	// in for more than one PO (e.g. "this PO" vs. "the one it duplicates"),
	// unlike findResult which answers every call the same way.
	byID            map[string]*procurementDomain.PurchaseOrder
	createErr       error
	createCalled    bool
	updateErr       error
	updated         []procurementDomain.PurchaseOrder
	deleteErr       error
	deleted         []string
	suggestionsErr  error
	suggestionsSet  []suggestionUpdate
	duplicateErr    error
	duplicatesSet   []duplicateMark
	clearErr        error
	cleared         []string
	duplicatesOf    map[string][]procurementDomain.PurchaseOrder
	duplicatesOfErr error
}

type duplicateMark struct {
	id                         string
	duplicateOfPurchaseOrderID string
}

type suggestionUpdate struct {
	id           string
	supplierName string
	currencyCode string
}

func (s *stubPurchaseOrderRepo) Create(_ context.Context, po *procurementDomain.PurchaseOrder) error {
	s.createCalled = true
	if s.createErr != nil {
		return s.createErr
	}
	po.ID = "po-1"
	return nil
}
func (s *stubPurchaseOrderRepo) FindByID(_ context.Context, id string) (*procurementDomain.PurchaseOrder, error) {
	if s.byID != nil {
		if po, ok := s.byID[id]; ok {
			return po, nil
		}
	}
	return s.findResult, s.findErr
}
func (s *stubPurchaseOrderRepo) List(_ context.Context) ([]procurementDomain.PurchaseOrder, error) {
	return s.purchaseOrders, nil
}
func (s *stubPurchaseOrderRepo) Update(_ context.Context, po *procurementDomain.PurchaseOrder) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *po)
	return nil
}
func (s *stubPurchaseOrderRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}
func (s *stubPurchaseOrderRepo) UpdateSuggestions(_ context.Context, id string, supplierName, currencyCode string) error {
	if s.suggestionsErr != nil {
		return s.suggestionsErr
	}
	s.suggestionsSet = append(s.suggestionsSet, suggestionUpdate{id: id, supplierName: supplierName, currencyCode: currencyCode})
	return nil
}
func (s *stubPurchaseOrderRepo) MarkPotentialDuplicate(_ context.Context, id, duplicateOfPurchaseOrderID string) error {
	if s.duplicateErr != nil {
		return s.duplicateErr
	}
	s.duplicatesSet = append(s.duplicatesSet, duplicateMark{id: id, duplicateOfPurchaseOrderID: duplicateOfPurchaseOrderID})
	return nil
}
func (s *stubPurchaseOrderRepo) ClearPotentialDuplicate(_ context.Context, id string) error {
	if s.clearErr != nil {
		return s.clearErr
	}
	s.cleared = append(s.cleared, id)
	return nil
}
func (s *stubPurchaseOrderRepo) FindDuplicatesOf(_ context.Context, id string) ([]procurementDomain.PurchaseOrder, error) {
	if s.duplicatesOfErr != nil {
		return nil, s.duplicatesOfErr
	}
	return s.duplicatesOf[id], nil
}

type stubLineItemRepo struct {
	byPO         []procurementDomain.PurchaseOrderLineItem
	findResult   *procurementDomain.PurchaseOrderLineItem
	findErr      error
	createErr    error
	createCalled bool
	created      []procurementDomain.PurchaseOrderLineItem
	updateErr    error
	updated      []procurementDomain.PurchaseOrderLineItem
	deleteErr    error
	deleted      []string
	sumByPO      map[string]int64
	sumErr       error
}

func (s *stubLineItemRepo) Create(_ context.Context, li *procurementDomain.PurchaseOrderLineItem) error {
	s.createCalled = true
	if s.createErr != nil {
		return s.createErr
	}
	li.ID = "line-item-1"
	s.created = append(s.created, *li)
	return nil
}
func (s *stubLineItemRepo) FindByID(_ context.Context, _ string) (*procurementDomain.PurchaseOrderLineItem, error) {
	return s.findResult, s.findErr
}
func (s *stubLineItemRepo) ListByPurchaseOrder(_ context.Context, _ string) ([]procurementDomain.PurchaseOrderLineItem, error) {
	return s.byPO, nil
}
func (s *stubLineItemRepo) Update(_ context.Context, li *procurementDomain.PurchaseOrderLineItem) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *li)
	// Mirrors a real repo: a subsequent ListByPurchaseOrder in the same
	// use case (e.g. the needs_review revert check) must see this change.
	for i := range s.byPO {
		if s.byPO[i].ID == li.ID {
			s.byPO[i] = *li
		}
	}
	return nil
}
func (s *stubLineItemRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}
func (s *stubLineItemRepo) SumByPurchaseOrders(_ context.Context, _ []string) (map[string]int64, error) {
	if s.sumErr != nil {
		return nil, s.sumErr
	}
	if s.sumByPO != nil {
		return s.sumByPO, nil
	}
	return map[string]int64{}, nil
}

// stubSKURepo implements pms.SKURepository -- FindByCode is exercised by the
// document-matching pipeline, FindByBarcode by the scanner booking flow,
// the rest are unused no-ops.
type stubSKURepo struct {
	byCode    map[string]*pmsDomain.SKU
	byBarcode map[string]*pmsDomain.SKU
	findByIDs map[string]*pmsDomain.SKU
	byVariant map[string][]pmsDomain.SKU
}

func (s *stubSKURepo) Create(_ context.Context, _ *pmsDomain.SKU) error { return nil }
func (s *stubSKURepo) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	if sku, ok := s.findByIDs[id]; ok {
		return sku, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubSKURepo) FindByCode(_ context.Context, code string) (*pmsDomain.SKU, error) {
	if sku, ok := s.byCode[code]; ok {
		return sku, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubSKURepo) FindByBarcode(_ context.Context, barcode string) (*pmsDomain.SKU, error) {
	if sku, ok := s.byBarcode[barcode]; ok {
		return sku, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubSKURepo) ListByVariant(_ context.Context, variantID string) ([]pmsDomain.SKU, error) {
	return s.byVariant[variantID], nil
}
func (s *stubSKURepo) Update(_ context.Context, _ *pmsDomain.SKU) error { return nil }
func (s *stubSKURepo) Delete(_ context.Context, _ string) error         { return nil }
func (s *stubSKURepo) ListWithExpiredSale(_ context.Context) ([]pmsDomain.SKU, error) {
	return nil, nil
}

// stubProductRepo implements pms.ProductRepository -- only List (the
// name-search matching step uses) and FindByID are exercised; the rest are
// unused no-ops.
type stubProductRepo struct {
	listResult  []pmsDomain.Product
	listErr     error
	byID        map[string]*pmsDomain.Product
	gotFilter   pmsDomain.ProductFilter
	tokenResult []pmsDomain.Product
	tokenErr    error
	gotTokens   []string
}

func (s *stubProductRepo) Create(_ context.Context, _ *pmsDomain.Product) error { return nil }
func (s *stubProductRepo) FindByID(_ context.Context, id string) (*pmsDomain.Product, error) {
	if p, ok := s.byID[id]; ok {
		return p, nil
	}
	return nil, pmsDomain.ErrProductNotFound
}
func (s *stubProductRepo) FindBySlug(_ context.Context, _ string) (*pmsDomain.Product, error) {
	return nil, pmsDomain.ErrProductNotFound
}
func (s *stubProductRepo) List(_ context.Context, filter pmsDomain.ProductFilter) ([]pmsDomain.Product, int64, error) {
	s.gotFilter = filter
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.listResult, int64(len(s.listResult)), nil
}
func (s *stubProductRepo) Update(_ context.Context, _ *pmsDomain.Product) error { return nil }
func (s *stubProductRepo) Delete(_ context.Context, _ string) error             { return nil }
func (s *stubProductRepo) SearchByAnyNameToken(_ context.Context, tokens []string, _ int) ([]pmsDomain.Product, error) {
	s.gotTokens = tokens
	if s.tokenErr != nil {
		return nil, s.tokenErr
	}
	return s.tokenResult, nil
}

// stubVariantRepo implements pms.VariantRepository -- only ListByProduct
// (the name-search matching step uses) is exercised; the rest are unused
// no-ops.
type stubVariantRepo struct {
	byProduct map[string][]pmsDomain.Variant
	byID      map[string]*pmsDomain.Variant
	listErr   error
}

func (s *stubVariantRepo) Create(_ context.Context, _ *pmsDomain.Variant) error { return nil }
func (s *stubVariantRepo) FindByID(_ context.Context, id string) (*pmsDomain.Variant, error) {
	if v, ok := s.byID[id]; ok {
		return v, nil
	}
	return nil, pmsDomain.ErrVariantNotFound
}
func (s *stubVariantRepo) ListByProduct(_ context.Context, productID string) ([]pmsDomain.Variant, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.byProduct[productID], nil
}
func (s *stubVariantRepo) Update(_ context.Context, _ *pmsDomain.Variant) error { return nil }
func (s *stubVariantRepo) Delete(_ context.Context, _ string) error             { return nil }

type stubAliasRepo struct {
	byKey      map[string]*procurementDomain.SupplierProductAlias
	upserted   []procurementDomain.SupplierProductAlias
	upsertErr  error
	findErr    error
	findCalled bool
}

func (s *stubAliasRepo) Upsert(_ context.Context, alias *procurementDomain.SupplierProductAlias) error {
	if s.upsertErr != nil {
		return s.upsertErr
	}
	s.upserted = append(s.upserted, *alias)
	return nil
}
func (s *stubAliasRepo) FindBySupplierAndRawKey(_ context.Context, supplierID, rawKey string) (*procurementDomain.SupplierProductAlias, error) {
	s.findCalled = true
	if s.findErr != nil {
		return nil, s.findErr
	}
	if alias, ok := s.byKey[supplierID+"|"+rawKey]; ok {
		return alias, nil
	}
	return nil, nil
}

type stubDocumentRepo struct {
	findResult          *procurementDomain.PurchaseOrderDocument
	findErr             error
	createErr           error
	created             []procurementDomain.PurchaseOrderDocument
	byPO                []procurementDomain.PurchaseOrderDocument
	listErr             error
	deleted             []string
	deleteErr           error
	duplicateCandidates []procurementDomain.PurchaseOrderDocument
	duplicateErr        error
	gotDuplicateHash    string
	gotDuplicatePOID    string
}

func (s *stubDocumentRepo) Create(_ context.Context, doc *procurementDomain.PurchaseOrderDocument) error {
	if s.createErr != nil {
		return s.createErr
	}
	doc.ID = "document-1"
	s.created = append(s.created, *doc)
	return nil
}
func (s *stubDocumentRepo) FindByID(_ context.Context, _ string) (*procurementDomain.PurchaseOrderDocument, error) {
	return s.findResult, s.findErr
}
func (s *stubDocumentRepo) ListByPurchaseOrder(_ context.Context, _ string) ([]procurementDomain.PurchaseOrderDocument, error) {
	return s.byPO, s.listErr
}
func (s *stubDocumentRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}
func (s *stubDocumentRepo) FindDuplicateCandidates(_ context.Context, contentHash, purchaseOrderID string) ([]procurementDomain.PurchaseOrderDocument, error) {
	s.gotDuplicateHash = contentHash
	s.gotDuplicatePOID = purchaseOrderID
	return s.duplicateCandidates, s.duplicateErr
}

type stubParsingJobRepo struct {
	createErr  error
	created    []procurementDomain.DocumentParsingJob
	findResult *procurementDomain.DocumentParsingJob
	findErr    error
}

func (s *stubParsingJobRepo) Create(_ context.Context, job *procurementDomain.DocumentParsingJob) error {
	if s.createErr != nil {
		return s.createErr
	}
	job.ID = "job-1"
	s.created = append(s.created, *job)
	return nil
}
func (s *stubParsingJobRepo) FindByID(_ context.Context, _ string) (*procurementDomain.DocumentParsingJob, error) {
	return s.findResult, s.findErr
}
func (s *stubParsingJobRepo) ListPending(_ context.Context) ([]procurementDomain.DocumentParsingJob, error) {
	return nil, nil
}
func (s *stubParsingJobRepo) MarkRunning(_ context.Context, _ string, _ time.Time) error { return nil }
func (s *stubParsingJobRepo) MarkCompleted(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (s *stubParsingJobRepo) MarkFailed(_ context.Context, _ string, _ string, _ time.Time) error {
	return nil
}
func (s *stubParsingJobRepo) MarkOrphanedRunningAsFailed(_ context.Context, _ string, _ time.Time) (int64, error) {
	return 0, nil
}

type stubFileStorage struct {
	savedData  []byte
	savedURL   string
	saveErr    error
	getData    []byte
	getErr     error
	deletedURL string
	deleteErr  error
}

func (s *stubFileStorage) Save(_ context.Context, _, _ string, data io.Reader) (string, error) {
	if s.saveErr != nil {
		return "", s.saveErr
	}
	b, _ := io.ReadAll(data)
	s.savedData = b
	if s.savedURL != "" {
		return s.savedURL, nil
	}
	return "https://example.com/tenant-1/doc.csv", nil
}
func (s *stubFileStorage) Delete(_ context.Context, url string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedURL = url
	return nil
}
func (s *stubFileStorage) Get(_ context.Context, _ string) (io.ReadCloser, error) {
	if s.getErr != nil {
		return nil, s.getErr
	}
	return io.NopCloser(bytes.NewReader(s.getData)), nil
}

type stubReceivingRecordRepo struct {
	findResult             *procurementDomain.ReceivingRecord
	findErr                error
	byPO                   []procurementDomain.ReceivingRecord
	createErr              error
	created                []procurementDomain.ReceivingRecord
	updateErr              error
	updated                []procurementDomain.ReceivingRecord
	duplicateCandidates    []procurementDomain.ReceivingRecord
	duplicateErr           error
	gotDuplicateHash       string
	gotDuplicatePOID       string
	gotDuplicateSupplierID *string
}

func (s *stubReceivingRecordRepo) Create(_ context.Context, r *procurementDomain.ReceivingRecord) error {
	if s.createErr != nil {
		return s.createErr
	}
	r.ID = "receiving-record-1"
	s.created = append(s.created, *r)
	return nil
}
func (s *stubReceivingRecordRepo) FindByID(_ context.Context, _ string) (*procurementDomain.ReceivingRecord, error) {
	return s.findResult, s.findErr
}
func (s *stubReceivingRecordRepo) ListByPurchaseOrder(_ context.Context, _ string) ([]procurementDomain.ReceivingRecord, error) {
	return s.byPO, nil
}
func (s *stubReceivingRecordRepo) FindDuplicateCandidates(_ context.Context, contentHash, purchaseOrderID string, supplierID *string) ([]procurementDomain.ReceivingRecord, error) {
	s.gotDuplicateHash, s.gotDuplicatePOID, s.gotDuplicateSupplierID = contentHash, purchaseOrderID, supplierID
	return s.duplicateCandidates, s.duplicateErr
}
func (s *stubReceivingRecordRepo) Update(_ context.Context, r *procurementDomain.ReceivingRecord) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *r)
	return nil
}

type stubReceivingDiscrepancyRepo struct {
	byRecord  []procurementDomain.ReceivingDiscrepancy
	createErr error
	created   []procurementDomain.ReceivingDiscrepancy
}

func (s *stubReceivingDiscrepancyRepo) Create(_ context.Context, d *procurementDomain.ReceivingDiscrepancy) error {
	if s.createErr != nil {
		return s.createErr
	}
	d.ID = "discrepancy-1"
	s.created = append(s.created, *d)
	return nil
}
func (s *stubReceivingDiscrepancyRepo) ListByReceivingRecord(_ context.Context, _ string) ([]procurementDomain.ReceivingDiscrepancy, error) {
	return s.byRecord, nil
}

type stubReceivingParsingJobRepo struct {
	createErr  error
	created    []procurementDomain.ReceivingParsingJob
	findResult *procurementDomain.ReceivingParsingJob
	findErr    error
}

func (s *stubReceivingParsingJobRepo) Create(_ context.Context, job *procurementDomain.ReceivingParsingJob) error {
	if s.createErr != nil {
		return s.createErr
	}
	job.ID = "receiving-job-1"
	s.created = append(s.created, *job)
	return nil
}
func (s *stubReceivingParsingJobRepo) FindByID(_ context.Context, _ string) (*procurementDomain.ReceivingParsingJob, error) {
	return s.findResult, s.findErr
}
func (s *stubReceivingParsingJobRepo) ListPending(_ context.Context) ([]procurementDomain.ReceivingParsingJob, error) {
	return nil, nil
}
func (s *stubReceivingParsingJobRepo) MarkRunning(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (s *stubReceivingParsingJobRepo) MarkCompleted(_ context.Context, _ string, _ time.Time) error {
	return nil
}
func (s *stubReceivingParsingJobRepo) MarkFailed(_ context.Context, _ string, _ string, _ time.Time) error {
	return nil
}
func (s *stubReceivingParsingJobRepo) MarkOrphanedRunningAsFailed(_ context.Context, _ string, _ time.Time) (int64, error) {
	return 0, nil
}

type stubPDFTextExtractor struct {
	text string
	err  error
}

func (s *stubPDFTextExtractor) ExtractText(_ []byte) (string, error) {
	return s.text, s.err
}

type stubAIDocumentParser struct {
	out          []procurementDomain.ExtractedLineItem
	supplierName string
	err          error
	gotInput     procurementDomain.DocumentInput
}

func (s *stubAIDocumentParser) ExtractLineItems(_ context.Context, doc procurementDomain.DocumentInput) (procurementDomain.DocumentExtractionResult, error) {
	s.gotInput = doc
	if s.err != nil {
		return procurementDomain.DocumentExtractionResult{}, s.err
	}
	return procurementDomain.DocumentExtractionResult{LineItems: s.out, SupplierName: s.supplierName}, nil
}

// stubAIParserResolver wraps a fixed AIDocumentParser (nil for "AI not
// configured") -- RunDocumentParsingJobUseCase/RunReceivingParsingJobUseCase
// resolve per job instead of taking a parser directly, so tests wrap
// whatever stubAIDocumentParser they want in one of these.
type stubAIParserResolver struct {
	parser procurementDomain.AIDocumentParser
	err    error
}

func (s *stubAIParserResolver) Resolve(_ context.Context) (procurementDomain.AIDocumentParser, error) {
	return s.parser, s.err
}

type stubBookingRepo struct {
	findResult   *procurementDomain.Booking
	findErr      error
	activeResult *procurementDomain.Booking
	activeErr    error
	createErr    error
	created      []procurementDomain.Booking
	updateErr    error
	updated      []procurementDomain.Booking
}

func (s *stubBookingRepo) Create(_ context.Context, b *procurementDomain.Booking) error {
	if s.createErr != nil {
		return s.createErr
	}
	b.ID = "booking-1"
	s.created = append(s.created, *b)
	return nil
}
func (s *stubBookingRepo) FindByID(_ context.Context, _ string) (*procurementDomain.Booking, error) {
	return s.findResult, s.findErr
}
func (s *stubBookingRepo) FindActiveByPurchaseOrder(_ context.Context, _ string) (*procurementDomain.Booking, error) {
	return s.activeResult, s.activeErr
}
func (s *stubBookingRepo) Update(_ context.Context, b *procurementDomain.Booking) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *b)
	return nil
}

type stubBookingScanRepo struct {
	byBooking []procurementDomain.BookingScan
	createErr error
	created   []procurementDomain.BookingScan
}

func (s *stubBookingScanRepo) Create(_ context.Context, sc *procurementDomain.BookingScan) error {
	if s.createErr != nil {
		return s.createErr
	}
	sc.ID = "booking-scan-1"
	s.created = append(s.created, *sc)
	return nil
}
func (s *stubBookingScanRepo) ListByBooking(_ context.Context, _ string) ([]procurementDomain.BookingScan, error) {
	return s.byBooking, nil
}

type stubBarcodeAliasRepo struct {
	byCode    map[string]*pmsDomain.BarcodeAlias
	findErr   error
	listBySKU []pmsDomain.BarcodeAlias
}

func (s *stubBarcodeAliasRepo) Create(_ context.Context, _ *pmsDomain.BarcodeAlias) error { return nil }
func (s *stubBarcodeAliasRepo) FindByCode(_ context.Context, code string) (*pmsDomain.BarcodeAlias, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if a, ok := s.byCode[code]; ok {
		return a, nil
	}
	return nil, pmsDomain.ErrBarcodeAliasNotFound
}
func (s *stubBarcodeAliasRepo) ListBySKU(_ context.Context, _ string) ([]pmsDomain.BarcodeAlias, error) {
	return s.listBySKU, nil
}
func (s *stubBarcodeAliasRepo) Delete(_ context.Context, _ string) error { return nil }

type stubReceiveStockForBooking struct {
	out      *appinventory.ReceiveStockResult
	err      error
	gotInput appinventory.ReceiveStockInput
}

func (s *stubReceiveStockForBooking) Execute(_ context.Context, in appinventory.ReceiveStockInput) (*appinventory.ReceiveStockResult, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appinventory.ReceiveStockResult{Movement: &invDomain.StockMovement{ID: "movement-1", SKUID: in.SKUID, Type: invDomain.MovementTypeReceipt, QuantityDelta: in.Quantity}}, nil
}

type stubRecordAdjustmentForBooking struct {
	out      *invDomain.StockMovement
	err      error
	gotInput appinventory.RecordStockMovementInput
}

func (s *stubRecordAdjustmentForBooking) Execute(_ context.Context, in appinventory.RecordStockMovementInput) (*invDomain.StockMovement, error) {
	s.gotInput = in
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &invDomain.StockMovement{ID: "movement-adj-1", SKUID: in.SKUID, Type: in.Type, QuantityDelta: in.QuantityDelta}, nil
}
