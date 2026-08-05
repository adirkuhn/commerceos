package channels

import (
	"context"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// SyncProductToChannelsUseCase re-pushes a product's current state to every
// channel it's already linked to, in response to a pms.product.changed
// event. A product with no listings is a cheap no-op — most products
// aren't linked to any channel, and this runs on every core product/SKU
// edit, not just ones that happen to matter to a connected store.
//
// Only single-SKU, non-variable products are ever synced this way — same
// invariant PushProductToWooUseCase enforces on the initial push. A product
// that's grown variants after being linked is silently skipped rather than
// erroring, since there's no single SKU left to represent it.
//
// Stock quantity is deliberately not pushed here — that's the sync
// schedule's job (ExportWooStockLevelsUseCase); duplicating it here would
// just mean two code paths racing to set the same field.
type SyncProductToChannelsUseCase struct {
	db          *gorm.DB
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
	listingRepo channelsDomain.ProductChannelListingRepository
	syncLogRepo channelsDomain.SyncLogRepository
	connRepo    integrationsDomain.ConnectionRepository
	wooFactory  integrationsDomain.WooClientFactory
	encSecret   string
}

func NewSyncProductToChannelsUseCase(
	db *gorm.DB,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	syncLogRepo channelsDomain.SyncLogRepository,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
) *SyncProductToChannelsUseCase {
	return &SyncProductToChannelsUseCase{
		db: db, productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		listingRepo: listingRepo, syncLogRepo: syncLogRepo,
		connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret,
	}
}

// Execute loads the product's fresh state and fans it out to every linked
// channel. The returned error is reserved for infra-level failures (can't
// read from the DB) that warrant the caller nacking/retrying the whole
// event — a single channel being unreachable is recorded on that listing's
// SyncStatus instead, never returned, so one flaky store can't block
// (or endlessly retry) the whole event.
func (uc *SyncProductToChannelsUseCase) Execute(ctx context.Context, productID string) error {
	var product pmsDomain.Product
	var sku pmsDomain.SKU
	var hasSingleSKU bool
	var listings []channelsDomain.ProductChannelListing

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		var err error
		listings, err = uc.listingRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		if len(listings) == 0 {
			return nil
		}

		p, err := uc.productRepo.FindByID(ctx, productID)
		if err != nil {
			return err
		}
		product = *p
		if product.HasVariants {
			return nil
		}

		variants, err := uc.variantRepo.ListByProduct(ctx, productID)
		if err != nil {
			return err
		}
		if len(variants) != 1 {
			return nil
		}

		skus, err := uc.skuRepo.ListByVariant(ctx, variants[0].ID)
		if err != nil {
			return err
		}
		if len(skus) != 1 {
			return nil
		}
		sku = skus[0]
		hasSingleSKU = true
		return nil
	})
	if err != nil {
		return fmt.Errorf("syncProductToChannels: %w", err)
	}
	if len(listings) == 0 || !hasSingleSKU {
		return nil
	}

	for _, listing := range listings {
		uc.syncOne(ctx, product, sku, listing)
	}
	return nil
}

func (uc *SyncProductToChannelsUseCase) syncOne(ctx context.Context, product pmsDomain.Product, sku pmsDomain.SKU, listing channelsDomain.ProductChannelListing) {
	var syncErr error
	var fields map[string]string
	switch listing.Provider {
	case string(integrationsDomain.ProviderWooCommerce):
		fields, syncErr = uc.pushToWoo(ctx, product, sku, listing)
	default:
		return // no other provider implemented yet
	}

	now := time.Now()
	listing.LastSyncedAt = &now
	listing.UpdatedAt = now
	status := channelsDomain.SyncLogStatusSuccess
	errMsg := ""
	if syncErr != nil {
		listing.SyncStatus = channelsDomain.SyncStatusError
		listing.LastSyncError = syncErr.Error()
		status = channelsDomain.SyncLogStatusError
		errMsg = syncErr.Error()
		log.Printf("syncProductToChannels: product %s -> connection %s: %v", product.ID, listing.ConnectionID, syncErr)
	} else {
		listing.SyncStatus = channelsDomain.SyncStatusSynced
		listing.LastSyncError = ""
	}

	recordSyncLog(ctx, uc.db, uc.syncLogRepo, listing, channelsDomain.SyncFlowPush, status, errMsg, fields)

	if uErr := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		return uc.listingRepo.Update(ctx, &listing)
	}); uErr != nil {
		log.Printf("syncProductToChannels: recording sync result for listing %s: %v", listing.ID, uErr)
	}
}

func (uc *SyncProductToChannelsUseCase) pushToWoo(ctx context.Context, product pmsDomain.Product, sku pmsDomain.SKU, listing channelsDomain.ProductChannelListing) (map[string]string, error) {
	var conn integrationsDomain.Connection
	var secret string
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, listing.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c
		secret, err = crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
		return err
	})
	if err != nil {
		return nil, err
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}

	salePrice, saleEndDate := WooSaleFields(sku)
	wooProduct := integrationsDomain.WooProduct{
		Name: product.Name, Description: product.Description,
		Status:       MapCommerceStatusToWoo(product.Status),
		RegularPrice: FormatMinorUnitsAsWooPrice(sku.PriceMinorUnits),
		SalePrice:    salePrice,
		SaleEndDate:  saleEndDate,
		Virtual:      sku.IsVirtual,
	}
	// Weight/dimensions need the store's configured unit to convert into —
	// a failure to fetch it just means weight/dimensions aren't pushed this
	// round, not a fatal error for the whole sync.
	if !sku.IsVirtual {
		if weightUnit, dimensionUnit, err := client.GetProductUnits(ctx); err == nil {
			if sku.WeightGrams != nil {
				if w, ok := gramsToWooWeight(*sku.WeightGrams, weightUnit); ok {
					wooProduct.Weight = w
				}
			}
			var dims integrationsDomain.WooProductDimensions
			if sku.LengthMM != nil {
				if v, ok := mmToWooLength(*sku.LengthMM, dimensionUnit); ok {
					dims.Length = v
				}
			}
			if sku.WidthMM != nil {
				if v, ok := mmToWooLength(*sku.WidthMM, dimensionUnit); ok {
					dims.Width = v
				}
			}
			if sku.HeightMM != nil {
				if v, ok := mmToWooLength(*sku.HeightMM, dimensionUnit); ok {
					dims.Height = v
				}
			}
			wooProduct.Dimensions = dims
		}
	}

	fields := wooProductToFields(wooProduct)
	if _, err := client.UpdateProduct(ctx, listing.ExternalID, wooProduct); err != nil {
		return fields, fmt.Errorf("%w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	return fields, nil
}
