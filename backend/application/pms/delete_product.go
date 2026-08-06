package pms

import (
	"context"
	"fmt"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/outbox"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

type DeleteProductUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
	listingRepo channelsDomain.ProductChannelListingRepository
	outboxRepo  outbox.Repository
}

func NewDeleteProductUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	outboxRepo outbox.Repository,
) *DeleteProductUseCase {
	return &DeleteProductUseCase{db: db, productRepo: productRepo, listingRepo: listingRepo, outboxRepo: outboxRepo}
}

// Execute deletes the product; variants/skus/channel listings/overrides all
// cascade via the DB's ON DELETE CASCADE (migrations/002_pms.sql,
// 007_channels.sql, 037_sales_channels.sql). The product's channel listings
// are snapshotted *before* the delete and carried on the raised
// pms.product.deleted event -- the row is cascade-gone by the time any
// consumer could otherwise read it, but the product must still be removed
// from each external channel (e.g. WooCommerce) it was actually synced to.
func (uc *DeleteProductUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		if _, err := uc.productRepo.FindByID(ctx, id); err != nil {
			return err
		}
		listings, err := uc.listingRepo.ListByProduct(ctx, id)
		if err != nil {
			return err
		}
		if err := uc.productRepo.Delete(ctx, id); err != nil {
			return err
		}
		return enqueueProductDeleted(ctx, uc.outboxRepo, id, listings)
	})
	if err != nil {
		return fmt.Errorf("deleteProduct: %w", err)
	}
	return nil
}
