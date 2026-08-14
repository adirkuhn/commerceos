package channels

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteAllImportedProductsInput struct {
	ConnectionID string
}

type DeleteAllImportedProductsResult struct {
	DeletedSalesOrders int
	DeletedProducts    int
	DeletedCategories  int
	Errors             []string
}

// deleteSalesOrderUC is application/oms's DeleteSalesOrderUseCase, narrowed
// to point of use (ISP) -- reused rather than reimplemented so an
// imported order's stock impact is reversed the exact same way a staff
// deletion would (see DeleteSalesOrderUseCase's own doc comment), instead
// of this use case having to duplicate that restock logic.
type deleteSalesOrderUC interface {
	Execute(ctx context.Context, id string) error
}

// DeleteAllImportedProductsUseCase undoes a WooCommerce import. Order
// matters: this connection's own sales orders are deleted first (each
// via DeleteSalesOrderUseCase, so stock is restocked before the order and
// its line items go) -- otherwise sales_order_line_items.sku_id's
// ON DELETE RESTRICT (deliberate: it's real sales history, never silently
// dropped by a cascade) blocks deleting any product that was ever sold.
// Only then are products deleted (DB cascades take care of their variants,
// SKUs, remaining stock movements, media, and the listing row itself --
// see migrations 002_pms.sql/006_inventory.sql/007_channels.sql), then any
// category that import created and is now empty.
type DeleteAllImportedProductsUseCase struct {
	db               *gorm.DB
	listingRepo      channelsDomain.ProductChannelListingRepository
	productRepo      pmsDomain.ProductRepository
	categoryRepo     pmsDomain.CategoryRepository
	salesOrderRepo   omsDomain.SalesOrderRepository
	deleteSalesOrder deleteSalesOrderUC
}

func NewDeleteAllImportedProductsUseCase(
	db *gorm.DB,
	listingRepo channelsDomain.ProductChannelListingRepository,
	productRepo pmsDomain.ProductRepository,
	categoryRepo pmsDomain.CategoryRepository,
	salesOrderRepo omsDomain.SalesOrderRepository,
	deleteSalesOrder deleteSalesOrderUC,
) *DeleteAllImportedProductsUseCase {
	return &DeleteAllImportedProductsUseCase{
		db: db, listingRepo: listingRepo, productRepo: productRepo, categoryRepo: categoryRepo,
		salesOrderRepo: salesOrderRepo, deleteSalesOrder: deleteSalesOrder,
	}
}

func (uc *DeleteAllImportedProductsUseCase) Execute(ctx context.Context, in DeleteAllImportedProductsInput) (*DeleteAllImportedProductsResult, error) {
	result := &DeleteAllImportedProductsResult{}

	var orders []omsDomain.SalesOrder
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		orders, err = uc.salesOrderRepo.ListByConnection(ctx, in.ConnectionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("deleteAllImportedProducts: %w", err)
	}
	for _, order := range orders {
		if err := uc.deleteSalesOrder.Execute(ctx, order.ID); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("sales order %s: %v", order.ID, err))
			continue
		}
		result.DeletedSalesOrders++
	}

	var listings []channelsDomain.ProductChannelListing
	err = tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		listings, err = uc.listingRepo.ListByConnection(ctx, in.ConnectionID)
		return err
	})
	if err != nil {
		return nil, fmt.Errorf("deleteAllImportedProducts: %w", err)
	}

	categoryIDs := map[string]bool{}

	for _, listing := range listings {
		err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			product, err := uc.productRepo.FindByID(ctx, listing.ProductID)
			if err != nil {
				return err
			}
			if product.CategoryID != nil {
				categoryIDs[*product.CategoryID] = true
			}
			return uc.productRepo.Delete(ctx, listing.ProductID)
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("product %s: %v", listing.ProductID, err))
			continue
		}
		result.DeletedProducts++
	}

	for categoryID := range categoryIDs {
		var deleted bool
		err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
			_, total, err := uc.productRepo.List(ctx, pmsDomain.ProductFilter{CategoryID: &categoryID, Limit: 1})
			if err != nil {
				return err
			}
			if total > 0 {
				return nil // still in use elsewhere, leave it
			}
			if err := uc.categoryRepo.Delete(ctx, categoryID); err != nil {
				return err
			}
			deleted = true
			return nil
		})
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("category %s: %v", categoryID, err))
			continue
		}
		if deleted {
			result.DeletedCategories++
		}
	}

	return result, nil
}
