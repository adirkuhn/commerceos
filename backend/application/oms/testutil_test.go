package oms_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	iamDomain "github.com/adirkuhn/commerceos/domain/iam"
	"github.com/adirkuhn/commerceos/domain/integrations"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
)

func strPtr(s string) *string { return &s }

// stubOutboxRepo is a no-op outbox -- the tests that need
// UpdateSalesOrderUseCase to raise an oms.sales_order.status_changed event
// only care that Enqueue was called (via `calls`), not that it actually
// persists anything.
type stubOutboxRepo struct {
	calls    []string // subjects enqueued
	payloads [][]byte // one per call, same order as calls
	err      error
}

func (s *stubOutboxRepo) Enqueue(_ context.Context, subject, _ string, payload []byte, _ time.Time) error {
	s.calls = append(s.calls, subject)
	s.payloads = append(s.payloads, payload)
	return s.err
}
func (s *stubOutboxRepo) FetchPending(_ context.Context, _ int) ([]outbox.Message, error) {
	return nil, nil
}
func (s *stubOutboxRepo) MarkPublished(_ context.Context, _ uint) error { return nil }
func (s *stubOutboxRepo) MarkFailed(_ context.Context, _ uint) error    { return nil }

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
// one tenant.WithTenant transaction -- e.g. CreateSalesOrderUseCase opens
// one for the order+line-items write, then one more per line item to link
// its stock movement (recordMovement itself is stubbed, so it doesn't
// count against a real transaction).
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

type stubSalesOrderRepo struct {
	findByExternalIDResult *omsDomain.SalesOrder
	findByExternalIDErr    error
	findByIDResult         *omsDomain.SalesOrder
	findByIDErr            error
	createErr              error
	created                []omsDomain.SalesOrder
	listResult             []omsDomain.SalesOrder
	listTotal              int64
	listErr                error
	gotListFilter          omsDomain.ListSalesOrdersFilter
	updateErr              error
	gotUpdateID            string
	gotUpdateFields        map[string]any
	updateCalls            []struct {
		ID     string
		Fields map[string]any
	}
	deleteErr           error
	gotDeleteID         string
	byConnection        []omsDomain.SalesOrder
	listByConnectionErr error
	// trySwapResult/trySwapErr drive TrySetStockDecremented -- default
	// (zero value) is "swap succeeded, no error", the common case. Set
	// trySwapResult=false to simulate losing the race (another caller
	// already flipped the flag first).
	trySwapResult    bool
	trySwapResultSet bool
	trySwapErr       error
	gotTrySwapID     string
	gotTrySwapFrom   bool
	gotTrySwapTo     bool
	trySwapCalls     int
}

func (s *stubSalesOrderRepo) Create(_ context.Context, o *omsDomain.SalesOrder) error {
	if s.createErr != nil {
		return s.createErr
	}
	o.ID = "sales-order-1"
	s.created = append(s.created, *o)
	return nil
}
func (s *stubSalesOrderRepo) FindByID(_ context.Context, _ string) (*omsDomain.SalesOrder, error) {
	if s.findByIDResult == nil && s.findByIDErr == nil {
		return nil, omsDomain.ErrSalesOrderNotFound
	}
	return s.findByIDResult, s.findByIDErr
}
func (s *stubSalesOrderRepo) FindByExternalID(_ context.Context, _ omsDomain.SalesOrderSource, _ string) (*omsDomain.SalesOrder, error) {
	return s.findByExternalIDResult, s.findByExternalIDErr
}
func (s *stubSalesOrderRepo) FindByNumber(_ context.Context, _ string) (*omsDomain.SalesOrder, error) {
	return nil, nil
}
func (s *stubSalesOrderRepo) Search(_ context.Context, _ string, _ int) ([]omsDomain.SalesOrder, error) {
	return nil, nil
}
func (s *stubSalesOrderRepo) List(_ context.Context, filter omsDomain.ListSalesOrdersFilter) ([]omsDomain.SalesOrder, int64, error) {
	s.gotListFilter = filter
	if s.listErr != nil {
		return nil, 0, s.listErr
	}
	return s.listResult, s.listTotal, nil
}
func (s *stubSalesOrderRepo) ListByConnection(_ context.Context, _ string) ([]omsDomain.SalesOrder, error) {
	return s.byConnection, s.listByConnectionErr
}
func (s *stubSalesOrderRepo) Update(_ context.Context, id string, fields map[string]any) error {
	s.gotUpdateID = id
	s.gotUpdateFields = fields
	s.updateCalls = append(s.updateCalls, struct {
		ID     string
		Fields map[string]any
	}{id, fields})
	return s.updateErr
}
func (s *stubSalesOrderRepo) Delete(_ context.Context, id string) error {
	s.gotDeleteID = id
	return s.deleteErr
}
func (s *stubSalesOrderRepo) TrySetStockDecremented(_ context.Context, id string, from, to bool) (bool, error) {
	s.trySwapCalls++
	s.gotTrySwapID, s.gotTrySwapFrom, s.gotTrySwapTo = id, from, to
	if s.trySwapErr != nil {
		return false, s.trySwapErr
	}
	if !s.trySwapResultSet {
		return true, nil
	}
	return s.trySwapResult, nil
}

// stubSalesOrderPaymentRepo records CreateForSalesOrder calls and returns
// byOrder from ListBySalesOrder.
type stubSalesOrderPaymentRepo struct {
	created   []omsDomain.SalesOrderPayments
	createErr error
	byOrder   omsDomain.SalesOrderPayments
	listErr   error
}

func (s *stubSalesOrderPaymentRepo) CreateForSalesOrder(_ context.Context, o *omsDomain.SalesOrder) error {
	if s.createErr != nil {
		return s.createErr
	}
	s.created = append(s.created, o.Payments)
	return nil
}
func (s *stubSalesOrderPaymentRepo) ListBySalesOrder(_ context.Context, _ string) (omsDomain.SalesOrderPayments, error) {
	return s.byOrder, s.listErr
}

type stubSalesOrderLineItemRepo struct {
	createErr             error
	created               []omsDomain.SalesOrderLineItem
	listBySalesOrder      []omsDomain.SalesOrderLineItem
	listErr               error
	deleteBySalesOrderErr error
	gotDeleteSalesOrderID string
	findByIDResult        *omsDomain.SalesOrderLineItem
	findByIDErr           error
	updateErr             error
	gotUpdateID           string
	gotUpdateFields       map[string]any
	updateCalls           []struct {
		ID     string
		Fields map[string]any
	}
}

func (s *stubSalesOrderLineItemRepo) Create(_ context.Context, li *omsDomain.SalesOrderLineItem) error {
	if s.createErr != nil {
		return s.createErr
	}
	li.ID = "sales-order-line-1"
	s.created = append(s.created, *li)
	return nil
}
func (s *stubSalesOrderLineItemRepo) FindByID(_ context.Context, id string) (*omsDomain.SalesOrderLineItem, error) {
	if s.findByIDResult == nil && s.findByIDErr == nil {
		return nil, omsDomain.ErrSalesOrderLineItemNotFound
	}
	return s.findByIDResult, s.findByIDErr
}
func (s *stubSalesOrderLineItemRepo) ListBySalesOrder(_ context.Context, _ string) ([]omsDomain.SalesOrderLineItem, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listBySalesOrder, nil
}
func (s *stubSalesOrderLineItemRepo) DeleteBySalesOrder(_ context.Context, salesOrderID string) error {
	s.gotDeleteSalesOrderID = salesOrderID
	return s.deleteBySalesOrderErr
}
func (s *stubSalesOrderLineItemRepo) Update(_ context.Context, id string, fields map[string]any) error {
	s.gotUpdateID = id
	s.gotUpdateFields = fields
	s.updateCalls = append(s.updateCalls, struct {
		ID     string
		Fields map[string]any
	}{id, fields})
	return s.updateErr
}

type stubSalesOrderLineItemStockMovementRepo struct {
	createErr     error
	created       []omsDomain.SalesOrderLineItemStockMovement
	existsResult  bool
	existsErr     error
	byLineItem    map[string][]omsDomain.SalesOrderLineItemStockMovement
	listByLineErr error
}

func (s *stubSalesOrderLineItemStockMovementRepo) Create(_ context.Context, l *omsDomain.SalesOrderLineItemStockMovement) error {
	if s.createErr != nil {
		return s.createErr
	}
	l.ID = fmt.Sprintf("link-%d", len(s.created)+1)
	s.created = append(s.created, *l)
	return nil
}
func (s *stubSalesOrderLineItemStockMovementRepo) ExistsForLineItem(_ context.Context, _ string) (bool, error) {
	return s.existsResult, s.existsErr
}
func (s *stubSalesOrderLineItemStockMovementRepo) ListByLineItem(_ context.Context, lineItemID string) ([]omsDomain.SalesOrderLineItemStockMovement, error) {
	if s.listByLineErr != nil {
		return nil, s.listByLineErr
	}
	return s.byLineItem[lineItemID], nil
}

// stubSKURepo defaults to returning a SKU whose SKUCode/Barcode are derived
// from the requested ID (sku-1 -> SKU-1/BARCODE-1) so tests that don't care
// about the exact snapshotted values don't need to configure it.
type stubSKURepo struct {
	byID        map[string]*pmsDomain.SKU
	byCode      map[string]*pmsDomain.SKU
	byBarcode   map[string]*pmsDomain.SKU
	byScaleCode map[string]*pmsDomain.SKU
	findErr     error
	codeErr     error
	barcodeErr  error
}

func (s *stubSKURepo) FindByCode(_ context.Context, code string) (*pmsDomain.SKU, error) {
	if s.codeErr != nil {
		return nil, s.codeErr
	}
	if sku, ok := s.byCode[code]; ok {
		return sku, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubSKURepo) FindByScaleCode(_ context.Context, scaleCode string) (*pmsDomain.SKU, error) {
	if sku, ok := s.byScaleCode[scaleCode]; ok {
		return sku, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}
func (s *stubSKURepo) FindByBarcode(_ context.Context, barcode string) (*pmsDomain.SKU, error) {
	if s.barcodeErr != nil {
		return nil, s.barcodeErr
	}
	if sku, ok := s.byBarcode[barcode]; ok {
		return sku, nil
	}
	return nil, pmsDomain.ErrSKUNotFound
}

func (s *stubSKURepo) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if sku, ok := s.byID[id]; ok {
		return sku, nil
	}
	return &pmsDomain.SKU{ID: id, SKUCode: "SKU-" + id, Barcode: "BARCODE-" + id}, nil
}

type stubBarcodeAliasRepo struct {
	byCode  map[string]*pmsDomain.BarcodeAlias
	findErr error
}

func (s *stubBarcodeAliasRepo) FindByCode(_ context.Context, code string) (*pmsDomain.BarcodeAlias, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if a, ok := s.byCode[code]; ok {
		return a, nil
	}
	return nil, pmsDomain.ErrBarcodeAliasNotFound
}

type stubScaleCodeAliasRepo struct {
	byCode  map[string]*pmsDomain.ScaleCodeAlias
	findErr error
}

func (s *stubScaleCodeAliasRepo) FindByCode(_ context.Context, code string) (*pmsDomain.ScaleCodeAlias, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if a, ok := s.byCode[code]; ok {
		return a, nil
	}
	return nil, pmsDomain.ErrScaleCodeAliasNotFound
}

type stubVariantRepoForPicking struct {
	byID    map[string]*pmsDomain.Variant
	findErr error
}

func (s *stubVariantRepoForPicking) FindByID(_ context.Context, id string) (*pmsDomain.Variant, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if v, ok := s.byID[id]; ok {
		return v, nil
	}
	return &pmsDomain.Variant{ID: id, ProductID: "product-" + id}, nil
}

type stubProductRepoForPicking struct {
	byID    map[string]*pmsDomain.Product
	findErr error
}

func (s *stubProductRepoForPicking) FindByID(_ context.Context, id string) (*pmsDomain.Product, error) {
	if s.findErr != nil {
		return nil, s.findErr
	}
	if p, ok := s.byID[id]; ok {
		return p, nil
	}
	return &pmsDomain.Product{ID: id, Name: "Product " + id}, nil
}

type stubRecordMovement struct {
	calls []appinventory.AllocateSaleStockInput
	err   error
	// errBySKU overrides err for a specific SKU -- lets a test simulate one
	// line item succeeding while another fails differently, within the
	// same Execute call.
	errBySKU map[string]error
	// batchIDs, if set for a SKU, splits that call's returned movements one
	// per batch ID listed (nil entries mean the general ledger) instead of
	// the single-movement default -- lets a test simulate an allocation
	// that spanned more than one batch.
	batchIDs map[string][]*string
}

func (s *stubRecordMovement) Execute(_ context.Context, in appinventory.AllocateSaleStockInput) ([]invDomain.StockMovement, error) {
	s.calls = append(s.calls, in)
	if err, ok := s.errBySKU[in.SKUID]; ok {
		return nil, err
	}
	if s.err != nil {
		return nil, s.err
	}
	batchIDs, ok := s.batchIDs[in.SKUID]
	if !ok {
		batchIDs = []*string{nil}
	}
	out := make([]invDomain.StockMovement, len(batchIDs))
	remaining := in.Quantity
	for i, b := range batchIDs {
		qty := remaining
		if i < len(batchIDs)-1 {
			qty = remaining / int64(len(batchIDs)-i)
		}
		remaining -= qty
		out[i] = invDomain.StockMovement{
			ID: fmt.Sprintf("movement-%d", i+1), SKUID: in.SKUID, WarehouseID: in.WarehouseID,
			BatchID: b, Type: in.Type, QuantityDelta: -qty,
		}
	}
	return out, nil
}

type stubPickingSessionRepo struct {
	createErr      error
	created        []omsDomain.PickingSession
	findByIDResult *omsDomain.PickingSession
	findByIDErr    error
	activeResult   *omsDomain.PickingSession
	activeErr      error
	updateErr      error
	updated        []omsDomain.PickingSession
	acquireLockErr error
	acquiredLocks  []string
	releaseLockErr error
	released       []string

	openBySalesOrders []omsDomain.PickingSession
	openErr           error
	gotOpenIDs        []string
}

func (s *stubPickingSessionRepo) ListOpenBySalesOrders(_ context.Context, ids []string) ([]omsDomain.PickingSession, error) {
	s.gotOpenIDs = ids
	return s.openBySalesOrders, s.openErr
}

func (s *stubPickingSessionRepo) Create(_ context.Context, session *omsDomain.PickingSession) error {
	if s.createErr != nil {
		return s.createErr
	}
	session.ID = "picking-session-1"
	s.created = append(s.created, *session)
	return nil
}
func (s *stubPickingSessionRepo) FindByID(_ context.Context, _ string) (*omsDomain.PickingSession, error) {
	if s.findByIDResult == nil && s.findByIDErr == nil {
		return nil, omsDomain.ErrPickingSessionNotFound
	}
	return s.findByIDResult, s.findByIDErr
}
func (s *stubPickingSessionRepo) FindActiveBySalesOrder(_ context.Context, _ string) (*omsDomain.PickingSession, error) {
	return s.activeResult, s.activeErr
}
func (s *stubPickingSessionRepo) Update(_ context.Context, session *omsDomain.PickingSession) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *session)
	return nil
}
func (s *stubPickingSessionRepo) AcquireLock(_ context.Context, id, _ string, _ time.Time) error {
	if s.acquireLockErr != nil {
		return s.acquireLockErr
	}
	s.acquiredLocks = append(s.acquiredLocks, id)
	return nil
}
func (s *stubPickingSessionRepo) ReleaseLock(_ context.Context, id string) error {
	if s.releaseLockErr != nil {
		return s.releaseLockErr
	}
	s.released = append(s.released, id)
	return nil
}

type stubPickScanRepo struct {
	createErr       error
	created         []omsDomain.PickScan
	listBySession   []omsDomain.PickScan
	listErr         error
	markReplayed    []string
	markReplayedErr error
	deleteErr       error
	deleted         []string
}

func (s *stubPickScanRepo) Create(_ context.Context, scan *omsDomain.PickScan) error {
	if s.createErr != nil {
		return s.createErr
	}
	scan.ID = fmt.Sprintf("pick-scan-%d", len(s.created)+1)
	s.created = append(s.created, *scan)
	return nil
}
func (s *stubPickScanRepo) ListBySession(_ context.Context, _ string) ([]omsDomain.PickScan, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listBySession, nil
}
func (s *stubPickScanRepo) MarkReplayed(_ context.Context, id string, _ time.Time) error {
	if s.markReplayedErr != nil {
		return s.markReplayedErr
	}
	s.markReplayed = append(s.markReplayed, id)
	return nil
}
func (s *stubPickScanRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}

type stubPackingSessionRepo struct {
	createErr      error
	created        []omsDomain.PackingSession
	findByIDResult *omsDomain.PackingSession
	findByIDErr    error
	activeResult   *omsDomain.PackingSession
	activeErr      error
	updateErr      error
	updated        []omsDomain.PackingSession
	acquireLockErr error
	acquiredLocks  []string
	releaseLockErr error
	released       []string

	activeBySalesOrders []omsDomain.PackingSession
	listActiveErr       error
	gotActiveIDs        []string
}

func (s *stubPackingSessionRepo) ListActiveBySalesOrders(_ context.Context, ids []string) ([]omsDomain.PackingSession, error) {
	s.gotActiveIDs = ids
	return s.activeBySalesOrders, s.listActiveErr
}

func (s *stubPackingSessionRepo) Create(_ context.Context, session *omsDomain.PackingSession) error {
	if s.createErr != nil {
		return s.createErr
	}
	session.ID = "packing-session-1"
	s.created = append(s.created, *session)
	return nil
}
func (s *stubPackingSessionRepo) FindByID(_ context.Context, _ string) (*omsDomain.PackingSession, error) {
	if s.findByIDResult == nil && s.findByIDErr == nil {
		return nil, omsDomain.ErrPackingSessionNotFound
	}
	return s.findByIDResult, s.findByIDErr
}
func (s *stubPackingSessionRepo) FindActiveBySalesOrder(_ context.Context, _ string) (*omsDomain.PackingSession, error) {
	return s.activeResult, s.activeErr
}
func (s *stubPackingSessionRepo) Update(_ context.Context, session *omsDomain.PackingSession) error {
	if s.updateErr != nil {
		return s.updateErr
	}
	s.updated = append(s.updated, *session)
	return nil
}
func (s *stubPackingSessionRepo) AcquireLock(_ context.Context, id, _ string, _ time.Time) error {
	if s.acquireLockErr != nil {
		return s.acquireLockErr
	}
	s.acquiredLocks = append(s.acquiredLocks, id)
	return nil
}
func (s *stubPackingSessionRepo) ReleaseLock(_ context.Context, id string) error {
	if s.releaseLockErr != nil {
		return s.releaseLockErr
	}
	s.released = append(s.released, id)
	return nil
}

type stubPackScanRepo struct {
	createErr     error
	created       []omsDomain.PackScan
	listBySession []omsDomain.PackScan
	listErr       error
	deleteErr     error
	deleted       []string
	markReplayed  []string
	markErr       error

	sessionsWithScans []string
	withScansErr      error
	gotSessionIDs     []string
}

func (s *stubPackScanRepo) SessionIDsWithScans(_ context.Context, ids []string) ([]string, error) {
	s.gotSessionIDs = ids
	if s.withScansErr != nil {
		return nil, s.withScansErr
	}
	return s.sessionsWithScans, nil
}

func (s *stubPackScanRepo) MarkReplayed(_ context.Context, id string, _ time.Time) error {
	if s.markErr != nil {
		return s.markErr
	}
	s.markReplayed = append(s.markReplayed, id)
	return nil
}

func (s *stubPackScanRepo) Create(_ context.Context, scan *omsDomain.PackScan) error {
	if s.createErr != nil {
		return s.createErr
	}
	scan.ID = fmt.Sprintf("pack-scan-%d", len(s.created)+1)
	s.created = append(s.created, *scan)
	return nil
}
func (s *stubPackScanRepo) ListBySession(_ context.Context, _ string) ([]omsDomain.PackScan, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.listBySession, nil
}
func (s *stubPackScanRepo) Delete(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = append(s.deleted, id)
	return nil
}

// stubReverseMovement backs the second, narrower interface UpdateSalesOrderUseCase/
// DeleteSalesOrderUseCase use to write a restock's per-batch reversal movements --
// same shape as the old single-movement RecordStockMovementUseCase.
type stubReverseMovement struct {
	calls []appinventory.RecordStockMovementInput
	err   error
}

func (s *stubReverseMovement) Execute(_ context.Context, in appinventory.RecordStockMovementInput) (*invDomain.StockMovement, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	return &invDomain.StockMovement{ID: "movement-1", SKUID: in.SKUID, BatchID: in.BatchID, Type: in.Type, QuantityDelta: in.QuantityDelta}, nil
}

// stubStockAvailabilityChecker's zero value reports no shortfall for every
// SKU (short: 0) -- opens no transaction of its own, unlike the real
// *inventory.AllocateSaleStockUseCase, so it doesn't affect a test's
// newMockGormDBN count unless a test explicitly wants a shortfall.
type stubStockAvailabilityChecker struct {
	calls      []string // SKUIDs checked, in order
	shortBySKU map[string]int64
	err        error
}

func (s *stubStockAvailabilityChecker) CheckAvailability(_ context.Context, skuID, _ string, _ int64) (int64, error) {
	s.calls = append(s.calls, skuID)
	if s.err != nil {
		return 0, s.err
	}
	return s.shortBySKU[skuID], nil
}

// stubSKUFinderForShortfall's zero value returns a SKU whose SKUCode
// mirrors the id -- CompletePickingUseCase only calls this to build a
// human-readable StockShortfall.Name for a substitute (Freebie/NewItem use
// the scan's own Reason instead, see PickScan.Reason's doc comment).
type stubSKUFinderForShortfall struct {
	err error
}

func (s *stubSKUFinderForShortfall) FindByID(_ context.Context, id string) (*pmsDomain.SKU, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &pmsDomain.SKU{ID: id, SKUCode: id}, nil
}

type stubSalesOrderActivityRepo struct {
	createErr        error
	created          []omsDomain.SalesOrderActivity
	listBySalesOrder []omsDomain.SalesOrderActivity
	listBySOErr      error
	listByUser       []omsDomain.SalesOrderActivity
	listByUserErr    error
}

func (s *stubSalesOrderActivityRepo) Create(_ context.Context, a *omsDomain.SalesOrderActivity) error {
	if s.createErr != nil {
		return s.createErr
	}
	a.ID = "activity-1"
	s.created = append(s.created, *a)
	return nil
}
func (s *stubSalesOrderActivityRepo) ListBySalesOrder(_ context.Context, _ string) ([]omsDomain.SalesOrderActivity, error) {
	return s.listBySalesOrder, s.listBySOErr
}
func (s *stubSalesOrderActivityRepo) ListByUser(_ context.Context, _ string) ([]omsDomain.SalesOrderActivity, error) {
	return s.listByUser, s.listByUserErr
}

// stubGroupPermissionRepo backs RefundLineItemUseCase's fine-grained RBAC
// check -- only HasPermission is ever called by that use case, but the
// stub implements the full iamDomain.GroupPermissionRepository interface.
type stubGroupPermissionRepo struct {
	hasPermissionResult bool
	hasPermissionErr    error
	gotUserID           string
	gotRule             iamDomain.Rule
}

func (s *stubGroupPermissionRepo) Assign(_ context.Context, _ string, _ iamDomain.Rule) error {
	return nil
}
func (s *stubGroupPermissionRepo) Revoke(_ context.Context, _ string, _ iamDomain.Rule) error {
	return nil
}
func (s *stubGroupPermissionRepo) ListByGroup(_ context.Context, _ string) ([]iamDomain.Rule, error) {
	return nil, nil
}
func (s *stubGroupPermissionRepo) ListByUser(_ context.Context, _ string) ([]iamDomain.Rule, error) {
	return nil, nil
}
func (s *stubGroupPermissionRepo) HasPermission(_ context.Context, userID string, rule iamDomain.Rule) (bool, error) {
	s.gotUserID, s.gotRule = userID, rule
	return s.hasPermissionResult, s.hasPermissionErr
}

// stubPaymentLinkClient backs CompletePickingUseCase's refund replay path --
// only Refund is exercised by those tests, CreatePaymentLink is never
// called from picking.
type stubPaymentLinkClient struct {
	refundRef            string
	refundErr            error
	gotExternalPaymentID string
	gotAmountMinorUnits  int64
	gotReason            string
	refundCalls          int
}

func (s *stubPaymentLinkClient) CreatePaymentLink(_ context.Context, _ integrations.PaymentLinkRequest) (integrations.PaymentLink, error) {
	return integrations.PaymentLink{}, nil
}
func (s *stubPaymentLinkClient) Refund(_ context.Context, externalPaymentID string, amountMinorUnits int64, reason string) (string, error) {
	s.refundCalls++
	s.gotExternalPaymentID, s.gotAmountMinorUnits, s.gotReason = externalPaymentID, amountMinorUnits, reason
	return s.refundRef, s.refundErr
}

// stubPaymentRefundClientResolver backs CompletePickingUseCase's refund
// replay path -- resolveResult/resolveOK/resolveErr drive Resolve's return
// directly, standing in for the real PaymentRefundClientResolver's
// Connection/factory/decrypt plumbing (not exercised by these unit tests).
type stubPaymentRefundClientResolver struct {
	resolveResult integrations.PaymentLinkClient
	resolveOK     bool
	resolveErr    error
	gotProvider   string
}

func (s *stubPaymentRefundClientResolver) Resolve(_ context.Context, provider string) (integrations.PaymentLinkClient, bool, error) {
	s.gotProvider = provider
	return s.resolveResult, s.resolveOK, s.resolveErr
}

func strPtrLI(s string) *string { return &s }
