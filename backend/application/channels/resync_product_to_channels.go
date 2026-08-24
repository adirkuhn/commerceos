package channels

import (
	"context"
	"fmt"
	"log"

	"gorm.io/gorm"

	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// ── Use-case interfaces (defined at point of use — ISP) ────────────────────

type productFieldsSyncer interface {
	Execute(ctx context.Context, productID string) error
}

type productStockForceSyncer interface {
	ForceExecute(ctx context.Context, skuID string) error
}

// productEnroller is the one method this use case needs from
// EnrollProductInChannelsUseCase (ISP) -- run here so "Resync to channels"
// also creates a first listing for any channel the product is enrolled in
// but has never successfully synced to, not just refresh existing ones.
type productEnroller interface {
	Execute(ctx context.Context, productID string) error
}

// ResyncProductToChannelsUseCase force-pushes a product's current fields and
// stock to every channel it's already linked to, regardless of whether
// anything actually changed since the last sync -- the "Resync to
// channels" product-page action, and the per-product unit of work the
// periodic bulk resync job drives. Unlike the event-driven paths
// (SyncProductToChannelsUseCase, already unconditional for fields, and
// SyncStockToChannelsUseCase.Execute, which skips a push when the last
// successful sync already reported the same values), stock here always
// goes through ForceExecute so a deliberate resync can never be a silent
// no-op.
//
// It also runs enroll (EnrollProductInChannelsUseCase) first: syncFields
// only ever refreshes a channel the product ALREADY has a listing row
// for, and hard no-ops when there are zero listings -- correct for its
// real job reacting to pms.product.changed on every edit, but wrong for
// this button, whose point is "push this everywhere it should be,"
// including a channel the product is enrolled in but that never got a
// first listing created (e.g. every Bol product before the
// CreateOffer/UpdateOffer wire-format fix: enrolled, but the first push
// always silently failed to produce a listing, so a plain refresh of
// "existing" listings had nothing to refresh).
type ResyncProductToChannelsUseCase struct {
	db          *gorm.DB
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
	syncFields  productFieldsSyncer
	syncStock   productStockForceSyncer
	enroll      productEnroller
}

func NewResyncProductToChannelsUseCase(
	db *gorm.DB,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	syncFields productFieldsSyncer,
	syncStock productStockForceSyncer,
	enroll productEnroller,
) *ResyncProductToChannelsUseCase {
	return &ResyncProductToChannelsUseCase{
		db: db, variantRepo: variantRepo, skuRepo: skuRepo, syncFields: syncFields, syncStock: syncStock, enroll: enroll,
	}
}

// Execute runs enroll first (creates a first listing for any enrolled-but-
// unlinked channel; a failure there is logged, not fatal -- one channel's
// trouble must not block the fields/stock refresh below), then pushes
// fields, then stock per SKU. A stock push failure on one SKU is logged,
// not fatal to the others -- same "one flaky store can't block the rest"
// reasoning as the event-driven paths this reuses.
func (uc *ResyncProductToChannelsUseCase) Execute(ctx context.Context, productID string) error {
	if err := uc.enroll.Execute(ctx, productID); err != nil {
		log.Printf("resyncProductToChannels: enroll %s: %v", productID, err)
	}

	if err := uc.syncFields.Execute(ctx, productID); err != nil {
		return fmt.Errorf("resyncProductToChannels: %w", err)
	}

	var skuIDs []string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		variants, err := uc.variantRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		for _, v := range variants {
			skus, err := uc.skuRepo.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			for _, s := range skus {
				skuIDs = append(skuIDs, s.ID)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("resyncProductToChannels: %w", err)
	}

	for _, skuID := range skuIDs {
		if err := uc.syncStock.ForceExecute(ctx, skuID); err != nil {
			log.Printf("resyncProductToChannels: sku %s: %v", skuID, err)
		}
	}
	return nil
}
