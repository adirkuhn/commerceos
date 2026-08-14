package channels

import (
	"context"
)

// productsImportExecutor / ordersImportExecutor are ISP interfaces over the
// real ImportWooProductsUseCase/ImportWooOrdersUseCase -- narrowed to point
// of use so this orchestration can be unit tested without either use
// case's own DB/WooClient dependencies.
type productsImportExecutor interface {
	Execute(ctx context.Context, in ImportWooProductsInput) (*ImportResult, error)
}
type ordersImportExecutor interface {
	Execute(ctx context.Context, in ImportWooOrdersInput) (*ImportOrdersResult, error)
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
	// Orders is nil unless in.IsFirstImport was true -- see
	// RunManualWooImportUseCase's doc comment.
	Orders *ImportOrdersResult
}

// RunManualWooImportUseCase wraps the manual "Import" button's products
// import with a one-time orders bootstrap: the very first time a
// connection ever imports products, it also runs a full orders import
// right after, so a newly connected store's order history shows up in
// CommerceOS without a separate manual step. That bootstrap orders import
// never touches inventory (ImportWooOrdersInput.SkipStockMovements) -- the
// products import that just ran already set every SKU's stock to Woo's
// current, already-net-of-those-orders quantity, so decrementing again
// would double-count. Every import after the first is products only,
// exactly as before; orders keep syncing on their own schedule from then
// on.
type RunManualWooImportUseCase struct {
	productsImport productsImportExecutor
	ordersImport   ordersImportExecutor
}

func NewRunManualWooImportUseCase(productsImport productsImportExecutor, ordersImport ordersImportExecutor) *RunManualWooImportUseCase {
	return &RunManualWooImportUseCase{productsImport: productsImport, ordersImport: ordersImport}
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
