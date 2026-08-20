package oms_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	appinventory "github.com/adirkuhn/commerceos/application/inventory"
	invDomain "github.com/adirkuhn/commerceos/domain/inventory"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
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
	deleteErr error
	gotDeleteID            string
	byConnection           []omsDomain.SalesOrder
	listByConnectionErr    error
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
	byID    map[string]*pmsDomain.SKU
	findErr error
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
