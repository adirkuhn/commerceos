package main

import (
	"context"
	"fmt"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

type scheduleKey struct {
	Entity    channelsDomain.SyncEntityType
	Direction channelsDomain.SyncDirection
}

// runner executes one due schedule.
type runner func(ctx context.Context, s channelsDomain.SyncSchedule) error

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

// buildRunners maps each runnable (entity, direction) pair to the use case
// that actually performs it. orders/categories are schema-legal
// SyncSchedule values (see domain/channels/sync_schedule.go) but
// deliberately have no entry here — dispatch returns an error for them,
// which the caller logs and skips rather than crashing the loop.
func buildRunners(
	productsImport productsImporter,
	productsExport productsExporter,
	stockImport stockImporter,
	stockExport stockExporter,
) runners {
	return runners{
		{channelsDomain.SyncEntityProducts, channelsDomain.SyncDirectionImport}: func(ctx context.Context, s channelsDomain.SyncSchedule) error {
			_, err := productsImport.Execute(ctx, channelsapp.ImportWooProductsInput{
				ConnectionID:       s.ConnectionID,
				DefaultWarehouseID: strOrEmpty(s.DefaultWarehouseID),
				DefaultCurrency:    strOrEmpty(s.DefaultCurrency),
			})
			return err
		},
		{channelsDomain.SyncEntityProducts, channelsDomain.SyncDirectionExport}: func(ctx context.Context, s channelsDomain.SyncSchedule) error {
			_, err := productsExport.Execute(ctx, channelsapp.RunScheduledProductExportInput{ConnectionID: s.ConnectionID})
			return err
		},
		{channelsDomain.SyncEntityStock, channelsDomain.SyncDirectionImport}: func(ctx context.Context, s channelsDomain.SyncSchedule) error {
			_, err := stockImport.Execute(ctx, channelsapp.ImportWooStockLevelsInput{
				ConnectionID:       s.ConnectionID,
				DefaultWarehouseID: strOrEmpty(s.DefaultWarehouseID),
			})
			return err
		},
		{channelsDomain.SyncEntityStock, channelsDomain.SyncDirectionExport}: func(ctx context.Context, s channelsDomain.SyncSchedule) error {
			_, err := stockExport.Execute(ctx, channelsapp.ExportWooStockLevelsInput{ConnectionID: s.ConnectionID})
			return err
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
func (rs runners) dispatch(ctx context.Context, s channelsDomain.SyncSchedule) error {
	r, ok := rs[scheduleKey{s.EntityType, s.Direction}]
	if !ok {
		return fmt.Errorf("no runner for entity=%s direction=%s", s.EntityType, s.Direction)
	}
	return r(ctx, s)
}
