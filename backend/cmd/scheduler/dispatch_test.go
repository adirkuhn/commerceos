package main

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	shippingapp "github.com/adirkuhn/commerceos/application/shipping"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

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

type stubShipmentReconciler struct {
	calls []string
	err   error
	errs  []string
}

func (s *stubShipmentReconciler) Execute(_ context.Context, connectionID string) (*shippingapp.ReconcileShipmentsResult, error) {
	s.calls = append(s.calls, connectionID)
	return &shippingapp.ReconcileShipmentsResult{Errors: s.errs}, s.err
}

type stubBolOrdersImporter struct {
	calls []channelsapp.ImportBolOrdersInput
	err   error
	errs  []string
}

func (s *stubBolOrdersImporter) Execute(_ context.Context, in channelsapp.ImportBolOrdersInput) (*channelsapp.ImportOrdersResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.ImportOrdersResult{Errors: s.errs}, s.err
}

type stubBolExporter struct {
	calls []channelsapp.RunScheduledBolExportInput
	err   error
	errs  []string
}

func (s *stubBolExporter) Execute(_ context.Context, in channelsapp.RunScheduledBolExportInput) (*channelsapp.BolExportResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.BolExportResult{Errors: s.errs}, s.err
}






func TestDispatch_returnsErrorForUnrunnableEntity(t *testing.T) {
	rs := buildRunners(runnerDeps{OrdersImport: &stubOrdersImporter{}, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityCategories, Direction: channelsDomain.SyncDirectionImport,
	})

	require.Error(t, err)
}

func TestDispatch_returnsErrorForOrdersExport(t *testing.T) {
	rs := buildRunners(runnerDeps{OrdersImport: &stubOrdersImporter{}, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionExport,
	})

	require.Error(t, err, "only import is supported for orders -- CommerceOS never pushes orders back to Woo")
}

func TestDispatch_routesOrdersImportAsFullOnFirstRun(t *testing.T) {
	oi := &stubOrdersImporter{}
	rs := buildRunners(runnerDeps{OrdersImport: oi, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})
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
	rs := buildRunners(runnerDeps{OrdersImport: oi, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})
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
	rs := buildRunners(runnerDeps{OrdersImport: oi, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})

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
	rs := buildRunners(runnerDeps{OrdersImport: oi, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})

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
	rs := buildRunners(runnerDeps{OrdersImport: oi, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})
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
	rs := buildRunners(runnerDeps{OrdersImport: &stubOrdersImporter{}, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrdersReconciliation, Direction: channelsDomain.SyncDirectionExport,
	})

	require.Error(t, err, "only import is supported for orders reconciliation -- same as orders")
}

func TestDispatch_routesShipmentReconciliationWithConnectionID(t *testing.T) {
	sr := &stubShipmentReconciler{}
	rs := buildRunners(runnerDeps{OrdersImport: &stubOrdersImporter{}, ShipmentReconcile: sr, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityShipmentReconciliation, Direction: channelsDomain.SyncDirectionImport,
	})

	require.NoError(t, err)
	require.Len(t, sr.calls, 1)
	require.Equal(t, "conn-1", sr.calls[0])
}

func TestDispatch_returnsErrorForShipmentReconciliationExport(t *testing.T) {
	rs := buildRunners(runnerDeps{OrdersImport: &stubOrdersImporter{}, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityShipmentReconciliation, Direction: channelsDomain.SyncDirectionExport,
	})

	require.Error(t, err, "only import is supported for shipment reconciliation")
}

func TestDispatch_reportsAnErrorWhenShipmentReconciliationHasPerShipmentFailures(t *testing.T) {
	sr := &stubShipmentReconciler{errs: []string{"shipment s-1: boom"}}
	rs := buildRunners(runnerDeps{OrdersImport: &stubOrdersImporter{}, ShipmentReconcile: sr, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityShipmentReconciliation, Direction: channelsDomain.SyncDirectionImport,
	})

	require.Error(t, err, "per-shipment failures must surface as a run error, not silently record success")
}

func TestDispatch_propagatesRunnerError(t *testing.T) {
	oi := &stubOrdersImporter{err: errors.New("woo down")}
	rs := buildRunners(runnerDeps{OrdersImport: oi, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
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
	rs := buildRunners(runnerDeps{OrdersImport: oi, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})
	warehouseID := "wh-1"

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
		DefaultWarehouseID: &warehouseID,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "2")
	require.ErrorContains(t, err, "customer lookup failed")
}





func TestDispatch_routesBolOrdersImportWithWarehouse(t *testing.T) {
	boi := &stubBolOrdersImporter{}
	rs := buildRunners(runnerDeps{OrdersImport: &stubOrdersImporter{}, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: boi, BolExport: &stubBolExporter{}})
	warehouseID := "wh-1"

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolOrders, Direction: channelsDomain.SyncDirectionImport,
		DefaultWarehouseID: &warehouseID,
	})

	require.NoError(t, err)
	require.Len(t, boi.calls, 1)
	require.Equal(t, "conn-1", boi.calls[0].ConnectionID)
	require.Equal(t, "wh-1", boi.calls[0].DefaultWarehouseID)
}

func TestDispatch_routesBolProductsAndBolStockExportToTheSameRunner(t *testing.T) {
	be := &stubBolExporter{}
	rs := buildRunners(runnerDeps{OrdersImport: &stubOrdersImporter{}, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: be})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolProducts, Direction: channelsDomain.SyncDirectionExport,
	})
	require.NoError(t, err)

	_, err = rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolStock, Direction: channelsDomain.SyncDirectionExport,
	})
	require.NoError(t, err)

	require.Len(t, be.calls, 2, "both bol_products and bol_stock schedule rows dispatch to the same runner")
}

func TestDispatch_returnsErrorForBolOrdersExport(t *testing.T) {
	rs := buildRunners(runnerDeps{OrdersImport: &stubOrdersImporter{}, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: &stubBolOrdersImporter{}, BolExport: &stubBolExporter{}})

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolOrders, Direction: channelsDomain.SyncDirectionExport,
	})

	require.Error(t, err, "no runner registered for bol_orders/export -- Bol never gets a pushed-back order")
}

func TestDispatch_reportsAnErrorWhenBolOrdersImportHasPerItemFailures(t *testing.T) {
	boi := &stubBolOrdersImporter{errs: []string{"bol order X: fetch detail failed"}}
	rs := buildRunners(runnerDeps{OrdersImport: &stubOrdersImporter{}, ShipmentReconcile: &stubShipmentReconciler{}, BolOrdersImport: boi, BolExport: &stubBolExporter{}})
	warehouseID := "wh-1"

	_, err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityBolOrders, Direction: channelsDomain.SyncDirectionImport,
		DefaultWarehouseID: &warehouseID,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "fetch detail failed")
}
