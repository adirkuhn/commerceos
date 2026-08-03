package main

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	channelsapp "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

type stubProductsImporter struct {
	calls []channelsapp.ImportWooProductsInput
	err   error
}

func (s *stubProductsImporter) Execute(_ context.Context, in channelsapp.ImportWooProductsInput) (*channelsapp.ImportResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.ImportResult{}, s.err
}

type stubProductsExporter struct {
	calls []channelsapp.RunScheduledProductExportInput
	err   error
}

func (s *stubProductsExporter) Execute(_ context.Context, in channelsapp.RunScheduledProductExportInput) (*channelsapp.ProductExportResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.ProductExportResult{}, s.err
}

type stubStockImporter struct {
	calls []channelsapp.ImportWooStockLevelsInput
	err   error
}

func (s *stubStockImporter) Execute(_ context.Context, in channelsapp.ImportWooStockLevelsInput) (*channelsapp.StockSyncResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.StockSyncResult{}, s.err
}

type stubStockExporter struct {
	calls []channelsapp.ExportWooStockLevelsInput
	err   error
}

func (s *stubStockExporter) Execute(_ context.Context, in channelsapp.ExportWooStockLevelsInput) (*channelsapp.StockSyncResult, error) {
	s.calls = append(s.calls, in)
	return &channelsapp.StockSyncResult{}, s.err
}

func TestDispatch_routesProductsImportWithWarehouseAndCurrency(t *testing.T) {
	pi := &stubProductsImporter{}
	rs := buildRunners(pi, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{})
	warehouseID, currency := "wh-1", "USD"

	err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
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
	rs := buildRunners(&stubProductsImporter{}, pe, &stubStockImporter{}, &stubStockExporter{})

	err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts, Direction: channelsDomain.SyncDirectionExport,
	})

	require.NoError(t, err)
	require.Len(t, pe.calls, 1)
	require.Equal(t, "conn-1", pe.calls[0].ConnectionID)
}

func TestDispatch_routesStockImportWithWarehouse(t *testing.T) {
	si := &stubStockImporter{}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, si, &stubStockExporter{})
	warehouseID := "wh-1"

	err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock, Direction: channelsDomain.SyncDirectionImport,
		DefaultWarehouseID: &warehouseID,
	})

	require.NoError(t, err)
	require.Len(t, si.calls, 1)
	require.Equal(t, "wh-1", si.calls[0].DefaultWarehouseID)
}

func TestDispatch_routesStockExport(t *testing.T) {
	se := &stubStockExporter{}
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, se)

	err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityStock, Direction: channelsDomain.SyncDirectionExport,
	})

	require.NoError(t, err)
	require.Len(t, se.calls, 1)
}

func TestDispatch_handlesNilWarehouseAndCurrencyGracefully(t *testing.T) {
	pi := &stubProductsImporter{}
	rs := buildRunners(pi, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{})

	err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts, Direction: channelsDomain.SyncDirectionImport,
	})

	require.NoError(t, err)
	require.Equal(t, "", pi.calls[0].DefaultWarehouseID)
	require.Equal(t, "", pi.calls[0].DefaultCurrency)
}

func TestDispatch_returnsErrorForUnrunnableEntity(t *testing.T) {
	rs := buildRunners(&stubProductsImporter{}, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{})

	err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityOrders, Direction: channelsDomain.SyncDirectionImport,
	})

	require.Error(t, err)
}

func TestDispatch_propagatesRunnerError(t *testing.T) {
	pi := &stubProductsImporter{err: errors.New("woo down")}
	rs := buildRunners(pi, &stubProductsExporter{}, &stubStockImporter{}, &stubStockExporter{})

	err := rs.dispatch(context.Background(), channelsDomain.SyncSchedule{
		ConnectionID: "conn-1", EntityType: channelsDomain.SyncEntityProducts, Direction: channelsDomain.SyncDirectionImport,
	})

	require.Error(t, err)
	require.ErrorContains(t, err, "woo down")
}
