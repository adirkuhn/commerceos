package channels_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appchannels "github.com/adirkuhn/commerceos/application/channels"
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

type stubCustomersImportExecutor struct {
	calls []appchannels.ImportWooCustomersInput
	out   *appchannels.ImportCustomersResult
	err   error
}

func (s *stubCustomersImportExecutor) Execute(_ context.Context, in appchannels.ImportWooCustomersInput) (*appchannels.ImportCustomersResult, error) {
	s.calls = append(s.calls, in)
	if s.err != nil {
		return nil, s.err
	}
	if s.out != nil {
		return s.out, nil
	}
	return &appchannels.ImportCustomersResult{}, nil
}

func TestRunManualWooImport_firstImportAlsoRunsCustomersThenOrdersWithSkipStockMovements(t *testing.T) {
	productsImport := &stubProductsImportExecutor{out: &appchannels.ImportResult{Imported: 5}}
	customersImport := &stubCustomersImportExecutor{out: &appchannels.ImportCustomersResult{Created: 7}}
	ordersImport := &stubOrdersImportExecutor{out: &appchannels.ImportOrdersResult{Created: 3}}

	uc := appchannels.NewRunManualWooImportUseCase(productsImport, customersImport, ordersImport)

	result, err := uc.Execute(context.Background(), appchannels.RunManualWooImportInput{
		ConnectionID: "conn-1", DefaultWarehouseID: "wh-1", DefaultCurrency: "EUR", IsFirstImport: true,
	})

	require.NoError(t, err)
	require.Len(t, productsImport.calls, 1)
	assert.Equal(t, "conn-1", productsImport.calls[0].ConnectionID)
	require.Len(t, customersImport.calls, 1)
	assert.Equal(t, "conn-1", customersImport.calls[0].ConnectionID)
	require.Len(t, ordersImport.calls, 1)
	assert.Equal(t, "conn-1", ordersImport.calls[0].ConnectionID)
	assert.Equal(t, "wh-1", ordersImport.calls[0].DefaultWarehouseID)
	assert.True(t, ordersImport.calls[0].SkipStockMovements, "bootstrap orders import must never touch inventory")
	assert.Nil(t, ordersImport.calls[0].ModifiedAfter, "must be a full scan, not incremental")
	require.NotNil(t, result.Products)
	assert.Equal(t, 5, result.Products.Imported)
	require.NotNil(t, result.Customers)
	assert.Equal(t, 7, result.Customers.Created)
	require.NotNil(t, result.Orders)
	assert.Equal(t, 3, result.Orders.Created)
}

func TestRunManualWooImport_notFirstImportSkipsCustomersAndOrdersBootstrap(t *testing.T) {
	productsImport := &stubProductsImportExecutor{}
	customersImport := &stubCustomersImportExecutor{}
	ordersImport := &stubOrdersImportExecutor{}

	uc := appchannels.NewRunManualWooImportUseCase(productsImport, customersImport, ordersImport)

	result, err := uc.Execute(context.Background(), appchannels.RunManualWooImportInput{ConnectionID: "conn-1", IsFirstImport: false})

	require.NoError(t, err)
	require.Len(t, productsImport.calls, 1)
	assert.Empty(t, customersImport.calls, "IsFirstImport false -- customers bootstrap must not run")
	assert.Empty(t, ordersImport.calls, "IsFirstImport false -- orders bootstrap must not run")
	assert.Nil(t, result.Customers)
	assert.Nil(t, result.Orders)
}

func TestRunManualWooImport_productsImportFailurePropagatesAndSkipsCustomersAndOrders(t *testing.T) {
	productsImport := &stubProductsImportExecutor{err: errors.New("woo unreachable")}
	customersImport := &stubCustomersImportExecutor{}
	ordersImport := &stubOrdersImportExecutor{}

	uc := appchannels.NewRunManualWooImportUseCase(productsImport, customersImport, ordersImport)

	_, err := uc.Execute(context.Background(), appchannels.RunManualWooImportInput{ConnectionID: "conn-1", IsFirstImport: true})

	require.Error(t, err)
	assert.Empty(t, customersImport.calls, "customers bootstrap must not run when products import itself failed")
	assert.Empty(t, ordersImport.calls, "orders bootstrap must not run when products import itself failed")
}

func TestRunManualWooImport_customersImportFailureIsBestEffortAndStillRunsOrders(t *testing.T) {
	productsImport := &stubProductsImportExecutor{out: &appchannels.ImportResult{Imported: 5}}
	customersImport := &stubCustomersImportExecutor{err: errors.New("woo unreachable")}
	ordersImport := &stubOrdersImportExecutor{out: &appchannels.ImportOrdersResult{Created: 3}}

	uc := appchannels.NewRunManualWooImportUseCase(productsImport, customersImport, ordersImport)

	result, err := uc.Execute(context.Background(), appchannels.RunManualWooImportInput{ConnectionID: "conn-1", IsFirstImport: true})

	require.NoError(t, err, "the primary products import already succeeded -- a customers-side failure must not fail the whole job")
	require.NotNil(t, result.Customers)
	require.Len(t, result.Customers.Errors, 1)
	assert.Contains(t, result.Customers.Errors[0], "woo unreachable")
	require.Len(t, ordersImport.calls, 1, "orders bootstrap must still run even if customers import failed")
	require.NotNil(t, result.Orders)
	assert.Equal(t, 3, result.Orders.Created)
}

func TestRunManualWooImport_ordersImportFailureIsBestEffortAndDoesNotFailTheJob(t *testing.T) {
	productsImport := &stubProductsImportExecutor{out: &appchannels.ImportResult{Imported: 5}}
	customersImport := &stubCustomersImportExecutor{out: &appchannels.ImportCustomersResult{Created: 7}}
	ordersImport := &stubOrdersImportExecutor{err: errors.New("woo unreachable")}

	uc := appchannels.NewRunManualWooImportUseCase(productsImport, customersImport, ordersImport)

	result, err := uc.Execute(context.Background(), appchannels.RunManualWooImportInput{ConnectionID: "conn-1", IsFirstImport: true})

	require.NoError(t, err, "the primary products import already succeeded -- an orders-side failure must not fail the whole job")
	require.NotNil(t, result.Products)
	require.NotNil(t, result.Orders)
	require.Len(t, result.Orders.Errors, 1)
	assert.Contains(t, result.Orders.Errors[0], "woo unreachable")
}
