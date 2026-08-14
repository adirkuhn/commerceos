package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/internal/tenant"
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
}

type RunManualWooImportResult struct {
	Products *ImportResult
	// Orders is nil unless this was the connection's first-ever products
	// import -- see RunManualWooImportUseCase's doc comment.
	Orders *ImportOrdersResult
}

// RunManualWooImportUseCase wraps the manual "Import" button's products
// import with a one-time orders bootstrap: the very first time a
// connection ever imports products (no ProductChannelListing rows for it
// yet), it also runs a full orders import right after, so a newly
// connected store's order history shows up in CommerceOS without a
// separate manual step. That bootstrap orders import never touches
// inventory (ImportWooOrdersInput.SkipStockMovements) -- the products
// import that just ran already set every SKU's stock to Woo's current,
// already-net-of-those-orders quantity, so decrementing again would
// double-count. Every import after the first is products only, exactly as
// before; orders keep syncing on their own schedule from then on.
type RunManualWooImportUseCase struct {
	db             *gorm.DB
	listingRepo    channelsDomain.ProductChannelListingRepository
	productsImport productsImportExecutor
	ordersImport   ordersImportExecutor
}

func NewRunManualWooImportUseCase(
	db *gorm.DB,
	listingRepo channelsDomain.ProductChannelListingRepository,
	productsImport productsImportExecutor,
	ordersImport ordersImportExecutor,
) *RunManualWooImportUseCase {
	return &RunManualWooImportUseCase{db: db, listingRepo: listingRepo, productsImport: productsImport, ordersImport: ordersImport}
}

func (uc *RunManualWooImportUseCase) Execute(ctx context.Context, in RunManualWooImportInput) (*RunManualWooImportResult, error) {
	isFirstImport, err := uc.isFirstImport(ctx, in.ConnectionID)
	if err != nil {
		return nil, fmt.Errorf("runManualWooImport: %w", err)
	}

	productsResult, err := uc.productsImport.Execute(ctx, ImportWooProductsInput{
		ConnectionID: in.ConnectionID, DefaultWarehouseID: in.DefaultWarehouseID, DefaultCurrency: in.DefaultCurrency,
	})
	if err != nil {
		return nil, err
	}

	result := &RunManualWooImportResult{Products: productsResult}
	if !isFirstImport {
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

func (uc *RunManualWooImportUseCase) isFirstImport(ctx context.Context, connectionID string) (bool, error) {
	var listings []channelsDomain.ProductChannelListing
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		listings, err = uc.listingRepo.ListByConnection(ctx, connectionID)
		return err
	})
	if err != nil {
		return false, err
	}
	return len(listings) == 0, nil
}
