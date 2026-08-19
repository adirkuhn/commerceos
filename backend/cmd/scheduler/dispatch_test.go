package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

type stubProductsImporter struct {
	calls []channelsapp.ImportWooProductsInput
	err   error
	errs  []string
}

func (s *stubProductsImporter) Execute(_ context.Context, in channelsapp.ImportWooProductsInput) (*channelsapp.ImportResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.ImportResult{Errors: s.errs}, s.err
}

type stubProductsExporter struct {
	calls []channelsapp.RunScheduledProductExportInput
	err   error
	errs  []string
}

func (s *stubProductsExporter) Execute(_ context.Context, in channelsapp.RunScheduledProductExportInput) (*channelsapp.ProductExportResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.ProductExportResult{Errors: s.errs}, s.err
}

type stubStockImporter struct {
	calls []channelsapp.ImportWooStockLevelsInput
	err   error
	errs  []string
}

func (s *stubStockImporter) Execute(_ context.Context, in channelsapp.ImportWooStockLevelsInput) (*channelsapp.StockSyncResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.StockSyncResult{Errors: s.errs}, s.err
}

type stubStockExporter struct {
	calls []channelsapp.ExportWooStockLevelsInput
	err   error
	errs  []string
}

func (s *stubStockExporter) Execute(_ context.Context, in channelsapp.ExportWooStockLevelsInput) (*channelsapp.StockSyncResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.StockSyncResult{Errors: s.errs}, s.err
}

type stubOrdersImporter struct {
	calls          []channelsapp.ImportWooOrdersInput
	err            error
	errs           []string
	lastModifiedAt *time.Time
}

func (s *stubOrdersImporter) Execute(_ context.Context, in channelsapp.ImportWooOrdersInput) (*channelsapp.ImportOrdersResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.ImportOrdersResult{Errors: s.errs, LastModifiedAt: s.lastModifiedAt}, s.err
}

func TestDispatch_routesProductsImportWithWarehouseAndCurrency(t *testing.T) {
	pi := &stubProductsImporter{}
	rs := buildRunners(pi, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, &stubOrdersImporter{})
	warehouseID, currency := "wh-1", "USD"

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts, Direction: channelsDomain.SyncDirectionImport,
		DefaultWarehouseID: &warehouseID, DefaultCurrency: &currency,
	})

	require.NoError(t, err)
	require.Len(t, pi.calls, 1)
	require.Equal(t, "conn-1", pi.calls[0].ConnectionID)
	require.Equal(t, "wh-1", pi.calls[0].DefaultWarehouseID)
	require.Equal(t, "USD", pi.calls[0].DefaultCurrency)
}

func TestDispatch_routesProductsExport(t *testing.T) {
	pe := &stubProductsExporter{}
	rs := buildRunners(&stubProductsImporter{}, pe, &stubStockImporter{}, &stubStockExporter{}, &stubOrdersImporter{})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts, Direction: channelsDomain.SyncDirectionExport,
	})

	require.NoError(t, err)
	require.Len(t, pe.calls, 1)
	require.Equal(t, "conn-1", pe.calls[0].ConnectionID)
}

func TestDispatch_routesStockImportWithWarehouse(t *testing.T) {
	si := &stubStockImporter{}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, si, &stubStockExporter{}, &stubOrdersImporter{})
	warehouseID := "wh-1"

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock, Direction: channelsDomain.SyncDirectionImport,
		DefaultWarehouseID: &warehouseID,
	})

	require.NoError(t, err)
	require.Len(t, si.calls, 1)
	require.Equal(t, "wh-1", si.calls[0].DefaultWarehouseID)
}

func TestDispatch_routesStockExport(t *testing.T) {
	se := &stubStockExporter{}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, se, &stubOrdersImporter{})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock, Direction: channelsDomain.SyncDirectionExport,
	})

	require.NoError(t, err)
	require.Len(t, se.calls, 1)
}

func TestDispatch_handlesNilWarehouseAndCurrencyGracefully(t *testing.T) {
	pi := &stubProductsImporter{}
	rs := buildRunners(pi, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, &stubOrdersImporter{})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts, Direction: channelsDomain.SyncDirectionImport,
	})

	require.NoError(t, err)
	require.Equal(t, "", pi.calls[0].DefaultWarehouseID)
	require.Equal(t, "", pi.calls[0].DefaultCurrency)
}

func TestDispatch_returnsErrorForUnrunnableEntity(t *testing.T) {
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, &stubOrdersImporter{})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityCategories, Direction: channelsDomain.SyncDirectionImport,
	})

	require.Error(t, err)
}

func TestDispatch_returnsErrorForOrdersExport(t *testing.T) {
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, &stubOrdersImporter{})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionExport,
	})

	require.Error(t, err, "only import is supported for orders -- CommerceOS never pushes orders back to Woo")
}

func TestDispatch_routesOrdersImportAsFullOnFirstRun(t *testing.T) {
	oi := &stubOrdersImporter{}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, oi)
	warehouseID := "wh-1"

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
		DefaultWarehouseID: &warehouseID, LastRunAt: nil,
	})

	require.NoError(t, err)
	require.Len(t, oi.calls, 1)
	require.Equal(t, "wh-1", oi.calls[0].DefaultWarehouseID)
	require.Nil(t, oi.calls[0].ModifiedAfter, "never run before -- no cursor, must be a full import")
}

func TestDispatch_routesOrdersImportIncrementallyWithOverlapBufferAfterFirstRun(t *testing.T) {
	oi := &stubOrdersImporter{}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, oi)
	lastRun := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
		LastRunAt: &lastRun,
	})

	require.NoError(t, err)
	require.Len(t, oi.calls, 1)
	require.NotNil(t, oi.calls[0].ModifiedAfter)
	require.True(t, oi.calls[0].ModifiedAfter.Before(lastRun), "must overlap slightly before the last run, not exactly at it")
	require.Equal(t, lastRun.Add(-ordersModifiedAfterOverlap), *oi.calls[0].ModifiedAfter)
}

// TestDispatch_ordersImportReturnsWooModifiedTimeAsTheCursorNotWallClock
// proves the orders runner's returned cursor is
// ImportOrdersResult.LastModifiedAt (Woo's own clock), never this
// process's wall-clock time -- the whole point of tying the cursor to
// Woo's own data instead of the poller's clock (see runner's doc comment).
func TestDispatch_ordersImportReturnsWooModifiedTimeAsTheCursorNotWallClock(t *testing.T) {
	wooModifiedAt := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	oi := &stubOrdersImporter{lastModifiedAt: &wooModifiedAt}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, oi)

	before := time.Now()
	cursorAt, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
	})

	require.NoError(t, err)
	require.NotNil(t, cursorAt)
	assert.True(t, cursorAt.Before(before), "cursor must be Woo's modified time, not this process's wall clock")
	assert.Equal(t, wooModifiedAt, *cursorAt)
}

// TestDispatch_ordersImportCursorIsNilWhenNoOrdersWereSeen proves an empty
// run (nothing returned by Woo at all) reports no cursor override --
// runOne falls back to wall-clock time.Now() only in that case, same as
// every non-orders runner always does.
func TestDispatch_ordersImportCursorIsNilWhenNoOrdersWereSeen(t *testing.T) {
	oi := &stubOrdersImporter{}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, oi)

	cursorAt, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
	})

	require.NoError(t, err)
	assert.Nil(t, cursorAt)
}

// TestDispatch_routesOrdersReconciliationWithAFixedRollingWindow proves the
// reconciliation runner never looks at s.LastRunAt as a cursor -- unlike
// the incremental "orders" schedule, ModifiedAfter is always "now minus
// reconciliationWindow", regardless of whether/when this schedule last ran.
func TestDispatch_routesOrdersReconciliationWithAFixedRollingWindow(t *testing.T) {
	oi := &stubOrdersImporter{}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, oi)
	warehouseID := "wh-1"
	lastRun := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC) // long ago -- must not be used as the cursor

	before := time.Now()
	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrdersReconciliation, Direction: channelsDomain.SyncDirectionImport,
		DefaultWarehouseID: &warehouseID, LastRunAt: &lastRun,
	})
	after := time.Now()

	require.NoError(t, err)
	require.Len(t, oi.calls, 1)
	require.Equal(t, "wh-1", oi.calls[0].DefaultWarehouseID)
	require.NotNil(t, oi.calls[0].ModifiedAfter)
	require.False(t, oi.calls[0].ModifiedAfter.Equal(lastRun), "must not use LastRunAt as the cursor")
	require.True(t, !oi.calls[0].ModifiedAfter.Before(before.Add(-reconciliationWindow)) && !oi.calls[0].ModifiedAfter.After(after.Add(-reconciliationWindow)))
}

func TestDispatch_returnsErrorForOrdersReconciliationExport(t *testing.T) {
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, &stubOrdersImporter{})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrdersReconciliation, Direction: channelsDomain.SyncDirectionExport,
	})

	require.Error(t, err, "only import is supported for orders reconciliation -- same as orders")
}

func TestDispatch_propagatesRunnerError(t *testing.T) {
	pi := &stubProductsImporter{err: errors.New("woo down")}
	rs := buildRunners(pi, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, &stubOrdersImporter{})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts, Direction: channelsDomain.SyncDirectionImport,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "woo down")
}

// A per-item failure (e.g. one order's customer lookup errored) is
// reported via the result's Errors slice, not the use case's own returned
// error -- see ImportOrdersResult/ImportResult/StockSyncResult/
// ProductExportResult's doc comments ("partial-failure tolerant"). Before
// this fix, dispatch discarded that slice entirely (`_, err := ...`), so a
// run where every single item failed still recorded LastRunStatus
// "success" -- this is what made "orders aren't syncing" invisible.
func TestDispatch_reportsAnErrorWhenOrdersImportHasPerOrderFailuresEvenWithNoTopLevelError(t *testing.T) {
	oi := &stubOrdersImporter{errs: []string{"order #123: customer lookup failed", "order #456: sku not linked"}}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, oi)
	warehouseID := "wh-1"

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
		DefaultWarehouseID: &warehouseID,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "2")
	require.ErrorContains(t, err, "customer lookup failed")
}

func TestDispatch_reportsAnErrorWhenProductsImportHasPerItemFailures(t *testing.T) {
	pi := &stubProductsImporter{errs: []string{"product X: duplicate slug"}}
	rs := buildRunners(pi, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{}, &stubOrdersImporter{})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts, Direction: channelsDomain.SyncDirectionImport,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "duplicate slug")
}

func TestDispatch_reportsAnErrorWhenProductsExportHasPerItemFailures(t *testing.T) {
	pe := &stubProductsExporter{errs: []string{"product Y: push failed"}}
	rs := buildRunners(&stubProductsImporter{}, pe, &stubStockImporter{}, &stubStockExporter{}, &stubOrdersImporter{})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts, Direction: channelsDomain.SyncDirectionExport,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "push failed")
}

func TestDispatch_reportsAnErrorWhenStockImportHasPerItemFailures(t *testing.T) {
	si := &stubStockImporter{errs: []string{"sku Z: batch not found"}}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, si, &stubStockExporter{}, &stubOrdersImporter{})
	warehouseID := "wh-1"

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock, Direction: channelsDomain.SyncDirectionImport,
		DefaultWarehouseID: &warehouseID,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "batch not found")
}

func TestDispatch_reportsAnErrorWhenStockExportHasPerItemFailures(t *testing.T) {
	se := &stubStockExporter{errs: []string{"sku Z: push failed"}}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, se, &stubOrdersImporter{})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock, Direction: channelsDomain.SyncDirectionExport,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "push failed")
}
