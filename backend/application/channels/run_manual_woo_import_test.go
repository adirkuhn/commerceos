package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
)

type stubProductsImportExecutor struct {
	calls []appchannels.ImportWooProductsInput
	out   *appchannels.ImportResult
	err   error
}

func (s *stubProductsImportExecutor) Execute(_ context.Context, in appchannels.ImportWooProductsInput) (*appchannels.ImportResult, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appchannels.ImportResult{}, nil
}

type stubOrdersImportExecutor struct {
	calls []appchannels.ImportWooOrdersInput
	out   *appchannels.ImportOrdersResult
	err   error
}

func (s *stubOrdersImportExecutor) Execute(_ context.Context, in appchannels.ImportWooOrdersInput) (*appchannels.ImportOrdersResult, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appchannels.ImportOrdersResult{}, nil
}

func TestRunManualWooImport_firstImportAlsoRunsOrdersWithSkipStockMovements(t *testing.T) {
	listingRepo := &stubListingRepo{} // byConnection empty -> first-ever import
	productsImport := &stubProductsImportExecutor{out: &appchannels.ImportResult{Imported: 5}}
	ordersImport := &stubOrdersImportExecutor{out: &appchannels.ImportOrdersResult{Created: 3}}

	uc := appchannels.NewRunManualWooImportUseCase(newMockGormDB(t, 1), listingRepo, productsImport, ordersImport)

	result, err := uc.Execute(context.Background(), appchannels.RunManualWooImportInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR",
	})

	require.NoError(t, err)
	require.Len(t, productsImport.calls, 1)
	assert.Equal(t, "conn-1", productsImport.calls[0].ConnectionID)
	require.Len(t, ordersImport.calls, 1)
	assert.Equal(t, "conn-1", ordersImport.calls[0].ConnectionID)
	assert.Equal(t, "wh-1", ordersImport.calls[0].DefaultWarehouseID)
	assert.True(t, ordersImport.calls[0].SkipStockMovements, "bootstrap orders import must never touch inventory")
	assert.Nil(t, ordersImport.calls[0].ModifiedAfter, "must be a full scan, not incremental")
	require.NotNil(t, result.Products)
	assert.Equal(t, 5, result.Products.Imported)
	require.NotNil(t, result.Orders)
	assert.Equal(t, 3, result.Orders.Created)
}

func TestRunManualWooImport_secondImportSkipsOrdersBootstrap(t *testing.T) {
	listingRepo := &stubListingRepo{byConnection: []channelsDomain.ProductChannelListing{{ID: "listing-1"}}}
	productsImport := &stubProductsImportExecutor{}
	ordersImport := &stubOrdersImportExecutor{}

	uc := appchannels.NewRunManualWooImportUseCase(newMockGormDB(t, 1), listingRepo, productsImport, ordersImport)

	result, err := uc.Execute(context.Background(), appchannels.RunManualWooImportInput{ConnectionID: "conn-1"})

	require.NoError(t, err)
	require.Len(t, productsImport.calls, 1)
	assert.Empty(t, ordersImport.calls, "not the connection's first import, orders bootstrap must not run")
	assert.Nil(t, result.Orders)
}

func TestRunManualWooImport_productsImportFailurePropagatesAndSkipsOrders(t *testing.T) {
	listingRepo := &stubListingRepo{}
	productsImport := &stubProductsImportExecutor{err: errors.New("woo unreachable")}
	ordersImport := &stubOrdersImportExecutor{}

	uc := appchannels.NewRunManualWooImportUseCase(newMockGormDB(t, 1), listingRepo, productsImport, ordersImport)

	_, err := uc.Execute(context.Background(), appchannels.RunManualWooImportInput{ConnectionID: "conn-1"})

	require.Error(t, err)
	assert.Empty(t, ordersImport.calls, "orders bootstrap must not run when products import itself failed")
}

func TestRunManualWooImport_ordersImportFailureIsBestEffortAndDoesNotFailTheJob(t *testing.T) {
	listingRepo := &stubListingRepo{}
	productsImport := &stubProductsImportExecutor{out: &appchannels.ImportResult{Imported: 5}}
	ordersImport := &stubOrdersImportExecutor{err: errors.New("woo unreachable")}

	uc := appchannels.NewRunManualWooImportUseCase(newMockGormDB(t, 1), listingRepo, productsImport, ordersImport)

	result, err := uc.Execute(context.Background(), appchannels.RunManualWooImportInput{ConnectionID: "conn-1"})

	require.NoError(t, err, "the primary products import already succeeded -- an orders-side failure must not fail the whole job")
	require.NotNil(t, result.Products)
	require.NotNil(t, result.Orders)
	require.Len(t, result.Orders.Errors, 1)
	assert.Contains(t, result.Orders.Errors[0], "woo unreachable")
}
