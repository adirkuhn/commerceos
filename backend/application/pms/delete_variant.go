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

type DeleteVariantUseCase struct {
	db          *gorm.DB
	variantRepo pmsDomain.VariantRepository
	listings    listingsForUnlist
	outboxRepo  outbox.Repository
}

func NewDeleteVariantUseCase(db *gorm.DB, variantRepo pmsDomain.VariantRepository, listings listingsForUnlist, outboxRepo outbox.Repository) *DeleteVariantUseCase {
	return &DeleteVariantUseCase{db: db, variantRepo: variantRepo, listings: listings, outboxRepo: outboxRepo}
}

// Execute deletes the variant; its skus and channel listing rows cascade
// via the DB's ON DELETE CASCADE. The listings are snapshotted first and
// raised as pms.variant.unlisted, so a consumer takes the variant off each
// external channel too -- the cascade alone would leave e.g. its Woo
// product live and sellable with nothing behind it.
func (uc *DeleteVariantUseCase) Execute(ctx context.Context, id string) error {
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		variant, err := uc.variantRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}
		all, err := uc.listings.ListByProduct(ctx, variant.ProductID)
		if err != nil {
			return err
		}
		refs := channelsDomain.ListingsOfVariant(all, id)
		if err := uc.variantRepo.Delete(ctx, id); err != nil {
			return err
		}
		if err := enqueueVariantUnlisted(ctx, uc.outboxRepo, variant.ProductID, id, refs); err != nil {
			return err
		}
		return enqueueProductChanged(ctx, uc.outboxRepo, variant.ProductID)
	})
	if err != nil {
		return fmt.Errorf("deleteVariant: %w", err)
	}
	return nil
}
