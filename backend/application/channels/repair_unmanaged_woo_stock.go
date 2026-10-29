package channels

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type RepairUnmanagedWooStockInput struct {
	ConnectionID string
	// DryRun lists what would be repaired without pushing anything.
	DryRun bool
}

// UnmanagedWooListing is one Woo product found without stock management
// although its CommerceOS SKU is physical.
type UnmanagedWooListing struct {
	ExternalID string
	Name       string
	SKUID      string
}

type RepairUnmanagedWooStockResult struct {
	// Checked is how many top-level Woo product listings were compared
	// against Woo's catalog.
	Checked   int
	Unmanaged []UnmanagedWooListing
	Repaired  int
	// SkippedVariations counts variation listings: the Woo client can't
	// list variations, so those aren't checked.
	SkippedVariations int
	Errors            []string
}

// RepairUnmanagedWooStockUseCase is a one-time repair for Woo products
// created without manage_stock although their SKU is physical -- what
// SyncProductToChannelsUseCase did for a variant added to an already-listed
// product (e.g. "split into packages") before it sent stock on create. Woo
// sells such a product as in stock with no limit. Scans the connection's
// Woo catalog once and force-syncs stock (SyncStockToChannelsUseCase
// .ForceExecute, which always sets manage_stock) for each unmanaged one.
// Changes no CommerceOS state itself; every push it triggers records its
// own sync log entry. Safe to run again: a repaired product is managed and
// no longer matches.
type RepairUnmanagedWooStockUseCase struct {
	db          *gorm.DB
	connRepo    integrationsDomain.ConnectionRepository
	wooFactory  integrationsDomain.WooClientFactory
	encSecret   string
	listingRepo channelsDomain.ProductChannelListingRepository
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
	stockSyncer productStockForceSyncer
}

func NewRepairUnmanagedWooStockUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	listingRepo channelsDomain.ProductChannelListingRepository,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	stockSyncer productStockForceSyncer,
) *RepairUnmanagedWooStockUseCase {
	return &RepairUnmanagedWooStockUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret,
		listingRepo: listingRepo, productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		stockSyncer: stockSyncer,
	}
}

func (uc *RepairUnmanagedWooStockUseCase) Execute(ctx context.Context, in RepairUnmanagedWooStockInput) (*RepairUnmanagedWooStockResult, error) {
	result := &RepairUnmanagedWooStockResult{}
	var conn integrationsDomain.Connection
	var secret string
	// physicalSKUByExternalID: top-level Woo product ID -> its physical SKU.
	physicalSKUByExternalID := map[string]string{}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		existing, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *existing
		secret, err = crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
		if err != nil {
			return err
		}
		listings, err := uc.listingRepo.ListByConnection(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		for _, l := range listings {
			if l.IsVariation {
				result.SkippedVariations++
				continue
			}
			if l.IsGated() {
				continue
			}
			sku, ok, err := uc.listingSKU(ctx, l)
			if err != nil {
				return err
			}
			if !ok {
				continue
			}
			result.Checked++
			if !sku.IsVirtual {
				physicalSKUByExternalID[l.ExternalID] = sku.ID
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("repairUnmanagedWooStock: %w", err)
	}
	if len(physicalSKUByExternalID) == 0 {
		return result, nil
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("repairUnmanagedWooStock: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
	}
	for page := 1; page <= maxBackfillPages; page++ {
		products, err := client.ListProducts(ctx, page, 50)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("page %d: %v", page, err))
			break
		}
		if len(products) == 0 {
			break
		}
		for _, wp := range products {
			skuID, ok := physicalSKUByExternalID[wp.ID]
			if !ok || wp.ManageStock || wp.Virtual {
				continue
			}
			result.Unmanaged = append(result.Unmanaged, UnmanagedWooListing{ExternalID: wp.ID, Name: wp.Name, SKUID: skuID})
			if in.DryRun {
				continue
			}
			if err := uc.stockSyncer.ForceExecute(ctx, skuID); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("product %s (%s): %v", wp.ID, wp.Name, err))
				continue
			}
			result.Repaired++
		}
	}
	return result, nil
}

// listingSKU resolves the one SKU behind a top-level listing: the
// variant's SKU for a separate-products row, the product's only SKU for a
// single-SKU row. ok=false for a variations-mode parent (no SKU of its
// own) or any row not holding exactly one SKU -- the same invariant every
// push enforces.
func (uc *RepairUnmanagedWooStockUseCase) listingSKU(ctx context.Context, l channelsDomain.ProductChannelListing) (pmsDomain.SKU, bool, error) {
	variantID := ""
	if l.VariantID != nil {
		variantID = *l.VariantID
	} else {
		product, err := uc.productRepo.FindByID(ctx, l.ProductID)
		if err != nil {
			return pmsDomain.SKU{}, false, err
		}
		if product.HasVariants {
			return pmsDomain.SKU{}, false, nil
		}
		variants, err := uc.variantRepo.ListByProduct(ctx, l.ProductID)
		if err != nil {
			return pmsDomain.SKU{}, false, err
		}
		if len(variants) != 1 {
			return pmsDomain.SKU{}, false, nil
		}
		variantID = variants[0].ID
	}
	skus, err := uc.skuRepo.ListByVariant(ctx, variantID)
	if err != nil {
		return pmsDomain.SKU{}, false, err
	}
	if len(skus) != 1 {
		return pmsDomain.SKU{}, false, nil
	}
	return skus[0], true, nil
}
