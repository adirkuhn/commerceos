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

type variantFinderForDeleteSKU interface {
	FindByID(ctx context.Context, id string) (*pmsDomain.Variant, error)
}

type DeleteSKUUseCase struct {
	db         *gorm.DB
	skuRepo    pmsDomain.SKURepository
	variants   variantFinderForDeleteSKU
	listings   listingsForUnlist
	outboxRepo outbox.Repository
}

func NewDeleteSKUUseCase(db *gorm.DB, skuRepo pmsDomain.SKURepository, variants variantFinderForDeleteSKU, listings listingsForUnlist, outboxRepo outbox.Repository) *DeleteSKUUseCase {
	return &DeleteSKUUseCase{db: db, skuRepo: skuRepo, variants: variants, listings: listings, outboxRepo: outboxRepo}
}

// Execute deletes the SKU. When it was its variant's last SKU, the variant
// has nothing left to sell: its channel listing rows are removed in the
// same transaction and raised as pms.variant.unlisted, so a consumer takes
// it off each external channel (see DeleteVariantUseCase).
func (uc *DeleteSKUUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		variant, err := uc.variants.FindByID(ctx, sku.VariantID)
		if err != nil {
			return err
		}
		siblings, err := uc.skuRepo.ListByVariant(ctx, sku.VariantID)
		if err != nil {
			return err
		}
		var refs []channelsDomain.VariantListingRef
		if len(siblings) <= 1 {
			all, err := uc.listings.ListByProduct(ctx, variant.ProductID)
			if err != nil {
				return err
			}
			refs = channelsDomain.ListingsOfVariant(all, variant.ID)
			for _, r := range refs {
				if err := uc.listings.Delete(ctx, r.ListingID); err != nil {
					return err
				}
			}
		}
		if err := uc.skuRepo.Delete(ctx, id); err != nil {
			return err
		}
		if err := enqueueVariantUnlisted(ctx, uc.outboxRepo, variant.ProductID, variant.ID, refs); err != nil {
			return err
		}
		return enqueueProductChanged(ctx, uc.outboxRepo, variant.ProductID)
	})
	if err != nil {
		return fmt.Errorf("deleteSKU: %w", err)
	}
	return nil
}
