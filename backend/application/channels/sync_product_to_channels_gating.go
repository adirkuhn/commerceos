package channels

import (
	"context"
	"fmt"
	"log"
	"time"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// gateConnection pushes each EXISTING listing row's current fields (Woo:
// full field payload including the product's new draft/archived status;
// Bol: an onHoldByRetailer=true toggle) one last time, then marks every row
// gated so nothing further -- not even a shape migration or a newly added
// variant's create -- reaches this channel until the product is Active
// again. Deliberately never creates or deletes a row: a variant added while
// the product is already Draft/Archived must not be pushed to the channel
// either, only reconciled once ungateConnection's shape-logic fallthrough
// runs on reactivation.
func (uc *SyncProductToChannelsUseCase) gateConnection(ctx context.Context, product pmsDomain.Product, variants []pmsDomain.Variant, skuByVariant map[string]pmsDomain.SKU, connectionID string, listings []channelsDomain.ProductChannelListing, images []integrationsDomain.WooProductImage) {
	switch integrationsDomain.Provider(listings[0].Provider) {
	case integrationsDomain.ProviderWooCommerce:
		uc.gateWoo(ctx, product, variants, skuByVariant, connectionID, listings, images)
	case integrationsDomain.ProviderBol:
		uc.gateBol(ctx, connectionID, listings)
	default:
		now := time.Now()
		uc.setGatedAt(ctx, listings, &now)
	}
}

func (uc *SyncProductToChannelsUseCase) gateWoo(ctx context.Context, product pmsDomain.Product, variants []pmsDomain.Variant, skuByVariant map[string]pmsDomain.SKU, connectionID string, listings []channelsDomain.ProductChannelListing, images []integrationsDomain.WooProductImage) {
	client, err := uc.wooClientFor(ctx, connectionID)
	if err != nil {
		log.Printf("syncProductToChannels: gating product %s on connection %s: %v", product.ID, connectionID, err)
		return
	}
	variantByID := map[string]pmsDomain.Variant{}
	for _, v := range variants {
		variantByID[v.ID] = v
	}
	shape := actualShape(listings)

	for _, l := range listings {
		switch {
		case l.VariantID != nil && l.IsVariation:
			// A variation row has no independent Woo status -- it rides on
			// the parent product row's update below.
			continue
		case l.VariantID == nil && shape == shapeVariations:
			parentProduct := integrationsDomain.WooProduct{
				Name: product.Name, Description: product.Description,
				Status:     MapCommerceStatusToWoo(product.Status),
				Images:     images,
				Categories: uc.resolveCategoryForWoo(ctx, client, connectionID, product.CategoryID),
			}
			if _, err := client.UpdateProduct(ctx, l.ExternalID, parentProduct); err != nil {
				log.Printf("syncProductToChannels: gating parent product %s: %v", l.ExternalID, err)
				continue
			}
			uc.recordSyncResult(ctx, l, nil, wooProductToFields(parentProduct))
		default:
			var name string
			var sku pmsDomain.SKU
			var ok bool
			if l.VariantID == nil {
				name = product.Name
				if len(variants) > 0 {
					sku, ok = skuByVariant[variants[0].ID]
				}
			} else {
				name = fmt.Sprintf("%s - %s", product.Name, variantByID[*l.VariantID].Name)
				sku, ok = skuByVariant[*l.VariantID]
			}
			if !ok {
				continue
			}
			wooProduct := uc.buildWooProduct(ctx, client, connectionID, product, sku, name, images)
			if _, err := client.UpdateProduct(ctx, l.ExternalID, wooProduct); err != nil {
				log.Printf("syncProductToChannels: gating product %s external %s: %v", product.ID, l.ExternalID, err)
				continue
			}
			uc.recordSyncResult(ctx, l, nil, wooProductToFields(wooProduct))
		}
	}

	now := time.Now()
	uc.setGatedAt(ctx, listings, &now)
}

func (uc *SyncProductToChannelsUseCase) gateBol(ctx context.Context, connectionID string, listings []channelsDomain.ProductChannelListing) {
	if uc.bolGater != nil {
		for _, l := range listings {
			if l.ExternalID == "" {
				continue
			}
			if err := uc.bolGater.SetOnHold(ctx, connectionID, l.ExternalID, true); err != nil {
				log.Printf("syncProductToChannels: gating bol offer %s: %v", l.ExternalID, err)
			}
		}
	}
	now := time.Now()
	uc.setGatedAt(ctx, listings, &now)
}

// ungateConnection clears every listing's gate. For Bol it also flips
// onHoldByRetailer back off (Woo needs no equivalent call -- syncConnection's
// normal shape logic, which runs immediately after this returns, already
// re-pushes full fields including the product's now-Active status).
func (uc *SyncProductToChannelsUseCase) ungateConnection(ctx context.Context, connectionID string, listings []channelsDomain.ProductChannelListing) {
	if integrationsDomain.Provider(listings[0].Provider) == integrationsDomain.ProviderBol && uc.bolGater != nil {
		for _, l := range listings {
			if l.ExternalID == "" {
				continue
			}
			if err := uc.bolGater.SetOnHold(ctx, connectionID, l.ExternalID, false); err != nil {
				log.Printf("syncProductToChannels: ungating bol offer %s: %v", l.ExternalID, err)
			}
		}
	}
	uc.setGatedAt(ctx, listings, nil)
}

func (uc *SyncProductToChannelsUseCase) setGatedAt(ctx context.Context, listings []channelsDomain.ProductChannelListing, gatedAt *time.Time) {
	if err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		for _, l := range listings {
			l.GatedAt = gatedAt
			l.UpdatedAt = time.Now()
			if err := uc.listingRepo.Update(ctx, &l); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		log.Printf("syncProductToChannels: updating gate state: %v", err)
	}
}

// forceStockCatchUp re-pushes current stock for every SKU on this product
// after a reactivation -- stock changes that happened while gated never
// reached the channel (SyncStockToChannelsUseCase skips a gated listing
// entirely), so the first sync back to Active must force it rather than
// wait for the next incidental stock movement. stockCatchUp is optional/nil
// -- same convention as bolPusher -- for a caller that hasn't wired it.
func (uc *SyncProductToChannelsUseCase) forceStockCatchUp(ctx context.Context, skuByVariant map[string]pmsDomain.SKU) {
	if uc.stockCatchUp == nil {
		return
	}
	for _, sku := range skuByVariant {
		if err := uc.stockCatchUp.ForceExecute(ctx, sku.ID); err != nil {
			log.Printf("syncProductToChannels: stock catch-up for sku %s: %v", sku.ID, err)
		}
	}
}
