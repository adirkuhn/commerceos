package channels

import (
	"context"
	"fmt"
	"log"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
	"gorm.io/gorm"
)

// EnrollProductInChannelsUseCase is what actually fixes "new products never
// sync to Woo": in response to pms.product.changed (raised on both create
// and update), it evaluates every active SalesChannel's enrollment rule
// against this product and creates the first ProductChannelListing for
// every channel it newly matches. It only ever creates a link, never
// updates or removes one -- SyncProductToChannelsUseCase (a separate,
// pre-existing use case, unchanged) re-pushes updates to whatever's already
// linked, including the link this use case just created. A channel a
// product no longer matches is left exactly as-is (see SalesChannel's doc
// comment: this module never auto-delists).
type EnrollProductInChannelsUseCase struct {
	db           *gorm.DB
	productRepo  pmsDomain.ProductRepository
	channelRepo  channelsDomain.SalesChannelRepository
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository
	pusher       productPusher
	bolPusher    bolProductPusher
}

func NewEnrollProductInChannelsUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	channelRepo channelsDomain.SalesChannelRepository,
	overrideRepo channelsDomain.SalesChannelProductOverrideRepository,
	pusher productPusher,
	bolPusher bolProductPusher,
) *EnrollProductInChannelsUseCase {
	return &EnrollProductInChannelsUseCase{
		db: db, productRepo: productRepo, channelRepo: channelRepo, overrideRepo: overrideRepo,
		pusher: pusher, bolPusher: bolPusher,
	}
}

func (uc *EnrollProductInChannelsUseCase) Execute(ctx context.Context, productID string) error {
	var product pmsDomain.Product
	var candidateChannels []channelsDomain.SalesChannel
	overrideByChannel := map[string]*channelsDomain.SalesChannelProductOverride{}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		p, err := uc.productRepo.FindByID(ctx, productID)
		if err != nil {
			return err
		}
		product = *p

		channels, err := uc.channelRepo.ListByTenant(ctx)
		if err != nil {
			return err
		}
		for _, ch := range channels {
			// Inactive channels and channel types with nothing to push
			// through yet (no ConnectionID -- only a future built-in
			// storefront type would have this) are never candidates.
			if !ch.Active || ch.ConnectionID == nil {
				continue
			}
			candidateChannels = append(candidateChannels, ch)
			o, err := uc.overrideRepo.FindByChannelAndProduct(ctx, ch.ID, productID)
			if err != nil {
				return err
			}
			if o != nil {
				overrideByChannel[ch.ID] = o
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("enrollProductInChannels: %w", err)
	}

	for _, ch := range candidateChannels {
		if !ch.IsProductEnrolled(product.CategoryID, product.Tags, overrideByChannel[ch.ID]) {
			log.Printf("enrollProductInChannels: product %s not enrolled in channel %s (%s, mode=%s) -- skipping", productID, ch.ID, ch.Type, ch.EnrollmentMode)
			continue
		}
		// One channel rejecting/failing this product (e.g. it has variants,
		// or the store is unreachable) must not block enrollment in every
		// other channel -- logged, not returned, same tolerance
		// SyncProductToChannelsUseCase applies per-listing.
		switch ch.Type {
		case channelsDomain.ChannelTypeBol:
			if uc.bolPusher == nil {
				log.Printf("enrollProductInChannels: product %s -> bol channel %s: no bol pusher wired, skipping", productID, ch.ID)
				continue
			}
			log.Printf("enrollProductInChannels: product %s enrolled in bol channel %s -> pushing to connection %s", productID, ch.ID, *ch.ConnectionID)
			if _, err := uc.bolPusher.Execute(ctx, PushProductToBolInput{ProductID: productID, ConnectionID: *ch.ConnectionID}); err != nil {
				log.Printf("enrollProductInChannels: product %s -> bol channel %s: %v", productID, ch.ID, err)
			} else {
				log.Printf("enrollProductInChannels: product %s -> bol channel %s: pushed successfully", productID, ch.ID)
			}
		case channelsDomain.ChannelTypeWooCommerce:
			log.Printf("enrollProductInChannels: product %s enrolled in woo channel %s -> pushing to connection %s", productID, ch.ID, *ch.ConnectionID)
			if _, err := uc.pusher.Execute(ctx, PushProductToWooInput{ProductID: productID, ConnectionID: *ch.ConnectionID}); err != nil {
				log.Printf("enrollProductInChannels: product %s -> woo channel %s: %v", productID, ch.ID, err)
			} else {
				log.Printf("enrollProductInChannels: product %s -> woo channel %s: pushed successfully", productID, ch.ID)
			}
		default:
			log.Printf("enrollProductInChannels: product %s enrolled in channel %s but type %q has no push adapter -- skipping", productID, ch.ID, ch.Type)
		}
	}
	return nil
}
