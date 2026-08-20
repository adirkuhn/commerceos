package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

// resultErr turns a "partial-failure tolerant" use case result (see
// ImportResult/StockSyncResult/ProductExportResult/ImportOrdersResult's own
// doc comments) into dispatch's single returned error. Those use cases
// return a non-nil err only for a whole-run failure (auth, page fetch) --
// a per-item failure (one bad order, one bad product) is appended to the
// result's own Errors slice instead, so the rest of the batch can still go
// through. Before this, every runner below discarded that slice
// (`_, err := ...Execute(...)`), so a run where every single item failed
// still recorded the schedule's LastRunStatus as "success" -- this is what
// made "sync isn't working" invisible in the UI.
func resultErr(err error, itemErrors []string) error {
	if err != nil {
		return err
	}
	if len(itemErrors) > 0 {
		return fmt.Errorf("%d item(s) failed: %s", len(itemErrors), strings.Join(itemErrors, "; "))
	}
	return nil
}

// ordersModifiedAfterOverlap is subtracted from a schedule's LastRunAt to
// build the next run's modified_after cutoff -- a small safety margin
// against clock skew between this process and Woo's server, and against
// an order modified in the same second a previous run's window closed.
// Cheap insurance: re-fetching a few extra orders every run is far less
// costly than silently missing one.
const ordersModifiedAfterOverlap = 10 * time.Hour

// reconciliationWindow is how far back SyncEntityOrdersReconciliation
// re-scans on every run, regardless of its own LastRunAt -- unlike the
// incremental "orders" schedule, this is never used as a moving cursor.
// ImportWooOrdersUseCase already dedupes by ExternalID, so re-fetching the
// same recent orders every day is a cheap no-op for anything already
// imported; it only does real work for whatever the incremental schedule
// missed.
const reconciliationWindow = 5 * 24 * time.Hour

type scheduleKey struct {
	Entity    channelsDomain.SyncEntityType
	Direction channelsDomain.SyncDirection
}

// runner executes one due schedule, returning the timestamp its own
// cursor (LastRunAt) should advance to on success. nil means "use the
// wall-clock time the poll happened" -- the previous, universal behavior,
// still correct for every runner except orders import (see its own entry
// below for why that one ties its cursor to Woo's own clock instead).
type runner func(ctx context.Context, s channelsDomain.SyncSchedule) (cursorAt *time.Time, err error)

type runners map[scheduleKey]runner

// The four interfaces below are each satisfied by the real use case
// (application/channels) — narrowed to just Execute so dispatch logic can
// be unit tested with stubs instead of the real use cases and their own
// DB/WooClient dependencies.
type productsImporter interface {
	Execute(ctx context.Context, in channelsapp.ImportWooProductsInput) (*channelsapp.ImportResult, error)
}
type productsExporter interface {
	Execute(ctx context.Context, in channelsapp.RunScheduledProductExportInput) (*channelsapp.ProductExportResult, error)
}
type stockImporter interface {
	Execute(ctx context.Context, in channelsapp.ImportWooStockLevelsInput) (*channelsapp.StockSyncResult, error)
}
type stockExporter interface {
	Execute(ctx context.Context, in channelsapp.ExportWooStockLevelsInput) (*channelsapp.StockSyncResult, error)
}
type ordersImporter interface {
	Execute(ctx context.Context, in channelsapp.ImportWooOrdersInput) (*channelsapp.ImportOrdersResult, error)
}

// buildRunners maps each runnable (entity, direction) pair to the use case
// that actually performs it. categories is a schema-legal SyncSchedule
// value (see domain/channels/sync_schedule.go) but deliberately has no
// entry here — dispatch returns an error for it, which the caller logs and
// skips rather than crashing the loop. orders/export has no entry either:
// this codebase only imports Woo orders, never pushes CommerceOS-created
// ones back to Woo.
func buildRunners(
	productsImport productsImporter,
	productsExport productsExporter,
	stockImport stockImporter,
	stockExport stockExporter,
	ordersImport ordersImporter,
) runners {
	return runners{
		{channelsDomain.SyncEntityProducts, channelsDomain.SyncDirectionImport}: func(ctx context.Context, s channelsDomain.SyncSchedule) (*time.Time, error) {
			result, err := productsImport.Execute(ctx, channelsapp.ImportWooProductsInput{
				ConnectionID:       s.ConnectionID,
				DefaultWarehouseID: strOrEmpty(s.DefaultWarehouseID),
				DefaultCurrency:    strOrEmpty(s.DefaultCurrency),
			})
			if result == nil {
				return nil, err
			}
			return nil, resultErr(err, result.Errors)
		},
		{channelsDomain.SyncEntityProducts, channelsDomain.SyncDirectionExport}: func(ctx context.Context, s channelsDomain.SyncSchedule) (*time.Time, error) {
			result, err := productsExport.Execute(ctx, channelsapp.RunScheduledProductExportInput{ConnectionID: s.ConnectionID})
			if result == nil {
				return nil, err
			}
			return nil, resultErr(err, result.Errors)
		},
		{channelsDomain.SyncEntityStock, channelsDomain.SyncDirectionImport}: func(ctx context.Context, s channelsDomain.SyncSchedule) (*time.Time, error) {
			result, err := stockImport.Execute(ctx, channelsapp.ImportWooStockLevelsInput{
				ConnectionID:       s.ConnectionID,
				DefaultWarehouseID: strOrEmpty(s.DefaultWarehouseID),
			})
			if result == nil {
				return nil, err
			}
			return nil, resultErr(err, result.Errors)
		},
		{channelsDomain.SyncEntityStock, channelsDomain.SyncDirectionExport}: func(ctx context.Context, s channelsDomain.SyncSchedule) (*time.Time, error) {
			result, err := stockExport.Execute(ctx, channelsapp.ExportWooStockLevelsInput{ConnectionID: s.ConnectionID})
			if result == nil {
				return nil, err
			}
			return nil, resultErr(err, result.Errors)
		},
		// LastRunAt nil (this schedule has never successfully run before)
		// means a full import -- no cursor exists yet to be incremental
		// from. Every subsequent run is incremental off that cursor, with
		// a small overlap buffer -- see ordersModifiedAfterOverlap. The
		// cursor itself is ImportOrdersResult.LastModifiedAt (Woo's own
		// date_modified_gmt of the latest order this run actually saw),
		// not this poll's wall-clock time -- see that field's doc comment
		// for why a wall-clock cursor is the wrong choice here.
		{channelsDomain.SyncEntityOrders, channelsDomain.SyncDirectionImport}: func(ctx context.Context, s channelsDomain.SyncSchedule) (*time.Time, error) {
			var modifiedAfter *time.Time
			if s.LastRunAt != nil {
				t := s.LastRunAt.Add(-ordersModifiedAfterOverlap)
				modifiedAfter = &t
			}
			result, err := ordersImport.Execute(ctx, channelsapp.ImportWooOrdersInput{
				ConnectionID: s.ConnectionID, DefaultWarehouseID: strOrEmpty(s.DefaultWarehouseID),
				ModifiedAfter: modifiedAfter,
			})
			if result == nil {
				return nil, err
			}
			return result.LastModifiedAt, resultErr(err, result.Errors)
		},
		// See reconciliationWindow's doc comment -- this always re-scans the
		// last reconciliationWindow, never advancing off s.LastRunAt like the
		// incremental "orders" schedule above -- so its own LastRunAt is
		// pure "did I run" bookkeeping, wall-clock is fine (nil cursorAt).
		{channelsDomain.SyncEntityOrdersReconciliation, channelsDomain.SyncDirectionImport}: func(ctx context.Context, s channelsDomain.SyncSchedule) (*time.Time, error) {
			modifiedAfter := time.Now().Add(-reconciliationWindow)
			result, err := ordersImport.Execute(ctx, channelsapp.ImportWooOrdersInput{
				ConnectionID: s.ConnectionID, DefaultWarehouseID: strOrEmpty(s.DefaultWarehouseID),
				ModifiedAfter: &modifiedAfter,
			})
			if result == nil {
				return nil, err
			}
			return nil, resultErr(err, result.Errors)
		},
	}
}

func strOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// dispatch runs the one runner matching s.EntityType/s.Direction.
func (rs runners) dispatch(ctx context.Context, s channelsDomain.SyncSchedule) (*time.Time, error) {
	r, ok := rs[scheduleKey{s.EntityType, s.Direction}]
	if !ok {
		return nil, fmt.Errorf("no runner for entity=%s direction=%s", s.EntityType, s.Direction)
	}
	return r(ctx, s)
}
