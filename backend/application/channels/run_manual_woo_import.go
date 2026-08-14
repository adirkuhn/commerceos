package channels

import (
	"context"
)

// productsImportExecutor / ordersImportExecutor / customersImportExecutor
// are ISP interfaces over the real ImportWooProductsUseCase/
// ImportWooOrdersUseCase/ImportWooCustomersUseCase -- narrowed to point of
// use so this orchestration can be unit tested without any of the three
// use cases' own DB/WooClient dependencies.
type productsImportExecutor interface {
	Execute(ctx context.Context, in ImportWooProductsInput) (*ImportResult, error)
}
type ordersImportExecutor interface {
	Execute(ctx context.Context, in ImportWooOrdersInput) (*ImportOrdersResult, error)
}
type customersImportExecutor interface {
	Execute(ctx context.Context, in ImportWooCustomersInput) (*ImportCustomersResult, error)
}

type RunManualWooImportInput struct {
	ConnectionID       string
	DefaultWarehouseID string
	DefaultCurrency    string
	// IsFirstImport is decided once by CreateImportJobUseCase at job
	// creation and persisted on the ImportJob row -- trusted as-is here,
	// never re-derived, so a crash-and-requeue retry (see
	// channels.ImportJobRepository.RecoverOrphanedRunning) makes the same
	// bootstrap decision as the original attempt even if some products
	// already got created before the interruption.
	IsFirstImport bool
}

type RunManualWooImportResult struct {
	Products *ImportResult
	// Customers/Orders are nil unless in.IsFirstImport was true -- see
	// RunManualWooImportUseCase's doc comment.
	Customers *ImportCustomersResult
	Orders    *ImportOrdersResult
}

// RunManualWooImportUseCase wraps the manual "Import" button's products
// import with a one-time customers + orders bootstrap: the very first time
// a connection ever imports products, it also imports every Woo customer
// account and then runs a full orders import, so a newly connected store's
// customers and order history show up in CommerceOS without separate
// manual steps. Customers run before orders so order import's existing
// FindByWooCustomerID lookup (see ImportWooOrdersUseCase) can link each
// order to its customer immediately instead of leaving it unlinked.
//
// The bootstrap orders import never touches inventory
// (ImportWooOrdersInput.SkipStockMovements) -- the products import that
// just ran already set every SKU's stock to Woo's current,
// already-net-of-those-orders quantity, so decrementing again would
// double-count. Customer passwords are never imported: Woo's REST API
// never exposes password hashes, and this codebase has no local
// password-checking code for one to be verified against even if it did --
// CommerceOS auth is delegated entirely to lm2 (see ImportWooCustomersUseCase).
//
// Every import after the first is products only, exactly as before;
// orders keep syncing on their own schedule from then on.
type RunManualWooImportUseCase struct {
	productsImport  productsImportExecutor
	customersImport customersImportExecutor
	ordersImport    ordersImportExecutor
}

func NewRunManualWooImportUseCase(
	productsImport productsImportExecutor,
	customersImport customersImportExecutor,
	ordersImport ordersImportExecutor,
) *RunManualWooImportUseCase {
	return &RunManualWooImportUseCase{productsImport: productsImport, customersImport: customersImport, ordersImport: ordersImport}
}

func (uc *RunManualWooImportUseCase) Execute(ctx context.Context, in RunManualWooImportInput) (*RunManualWooImportResult, error) {
	productsResult, err := uc.productsImport.Execute(ctx, ImportWooProductsInput{
		ConnectionID: in.ConnectionID, DefaultWarehouseID: in.DefaultWarehouseID, DefaultCurrency: in.DefaultCurrency,
	})
	if err != nil {
		return nil, err
	}

	result := &RunManualWooImportResult{Products: productsResult}
	if !in.IsFirstImport {
		return result, nil
	}

	customersResult, err := uc.customersImport.Execute(ctx, ImportWooCustomersInput{ConnectionID: in.ConnectionID})
	if err != nil {
		// Best-effort, same reasoning as the orders failure below: the
		// products import already succeeded and is the job's primary
		// purpose.
		customersResult = &ImportCustomersResult{Errors: []string{err.Error()}}
	}
	result.Customers = customersResult

	ordersResult, err := uc.ordersImport.Execute(ctx, ImportWooOrdersInput{
		ConnectionID: in.ConnectionID, DefaultWarehouseID: in.DefaultWarehouseID, SkipStockMovements: true,
	})
	if err != nil {
		// Best-effort: the products import already succeeded and is the
		// job's primary purpose -- an orders-side failure (e.g. Woo
		// unreachable mid-bootstrap) shouldn't fail the whole job or block
		// retrying orders later via the connection's normal sync schedule.
		result.Orders = &ImportOrdersResult{Errors: []string{err.Error()}}
		return result, nil
	}
	result.Orders = ordersResult
	return result, nil
}
