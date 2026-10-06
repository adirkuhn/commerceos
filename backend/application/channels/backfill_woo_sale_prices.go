package channels

import (
	"context"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"
	"github.com/adirkuhn/commerceos/domain/events"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

const maxBackfillPages = 100 // mirrors maxImportPages's safety cap

type BackfillWooSalePricesInput struct {
	ConnectionID string
}

type BackfillWooSalePricesResult struct {
	// Candidates is how many already-imported, single-SKU products on this
	// connection had no sale price recorded yet, before checking Woo's
	// current data for each.
	Candidates int
	Updated    int
	Errors     []string
}

// BackfillWooSalePricesUseCase is a one-time reconciliation for products
// imported before ImportWooProductsUseCase mapped Woo's sale_price/
// on_sale/date_on_sale_to fields at all -- every SKU imported before that
// fix has SalePriceMinorUnits=nil regardless of whether Woo actually had
// an active sale configured for it. Re-scans the connection's current Woo
// catalog and fills in SalePriceMinorUnits/SaleEndDate for exactly the
// SKUs still missing it, using the same OnSale/<=regular-price gating as
// the fixed import path (see wooImportedSaleFields). Never touches a SKU
// that already has a sale price recorded (whether imported since the fix,
// pushed there by CommerceOS, or edited manually), and never
// creates/updates anything else about the product -- deliberately narrow,
// not a general re-import.
//
// Scoped to single-SKU products only (VariantID nil on the listing row,
// and the product itself has HasVariants=false): ImportWooProductsUseCase
// never imports variable/variation products at all (see
// ShouldImportWooProductType), so those were never subject to this gap in
// the first place -- a variations-mode parent listing row also happens to
// have VariantID nil, which is why product.HasVariants is checked too, not
// just the listing row shape.
type BackfillWooSalePricesUseCase struct {
	db          *gorm.DB
	connRepo    integrationsDomain.ConnectionRepository
	wooFactory  integrationsDomain.WooClientFactory
	encSecret   string
	listingRepo channelsDomain.ProductChannelListingRepository
	productRepo pmsDomain.ProductRepository
	variantRepo pmsDomain.VariantRepository
	skuRepo     pmsDomain.SKURepository
	outbox      eventrecord.Enqueuer
}

func NewBackfillWooSalePricesUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	wooFactory integrationsDomain.WooClientFactory,
	encryptionSecret string,
	listingRepo channelsDomain.ProductChannelListingRepository,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	outbox eventrecord.Enqueuer,
) *BackfillWooSalePricesUseCase {
	return &BackfillWooSalePricesUseCase{
		db: db, connRepo: connRepo, wooFactory: wooFactory, encSecret: encryptionSecret,
		listingRepo: listingRepo, productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		outbox: outbox,
	}
}

// backfillTarget is one already-imported, single-SKU product still missing
// a sale price -- keyed by its Woo external product ID (see Execute) so a
// single paginated Woo catalog scan can match against every candidate in
// one pass, instead of one Woo request per candidate.
type backfillTarget struct {
	skuID           string
	priceMinorUnits int64
}

func (uc *BackfillWooSalePricesUseCase) Execute(ctx context.Context, in BackfillWooSalePricesInput) (*BackfillWooSalePricesResult, error) {
	var conn integrationsDomain.Connection
	var secret string
	targets := map[string]backfillTarget{}

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
			if l.VariantID != nil {
				continue
			}
			product, err := uc.productRepo.FindByID(ctx, l.ProductID)
			if err != nil {
				return err
			}
			if product.HasVariants {
				continue
			}
			variants, err := uc.variantRepo.ListByProduct(ctx, l.ProductID)
			if err != nil {
				return err
			}
			if len(variants) != 1 {
				continue
			}
			skus, err := uc.skuRepo.ListByVariant(ctx, variants[0].ID)
			if err != nil {
				return err
			}
			if len(skus) != 1 || skus[0].SalePriceMinorUnits != nil {
				continue
			}
			targets[l.ExternalID] = backfillTarget{skuID: skus[0].ID, priceMinorUnits: skus[0].PriceMinorUnits}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("backfillWooSalePrices: %w", err)
	}

	result := &BackfillWooSalePricesResult{Candidates: len(targets)}
	if len(targets) == 0 {
		return result, nil
	}

	client, err := uc.wooFactory.NewClient(conn.StoreURL.String(), conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("backfillWooSalePrices: %w: %w", integrationsDomain.ErrWooRequestFailed, err)
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
			target, ok := targets[wp.ID]
			if !ok {
				continue
			}
			salePriceMinorUnits, saleEndDate, err := wooImportedSaleFields(wp, target.priceMinorUnits)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("product %s: %v", wp.ID, err))
				continue
			}
			if salePriceMinorUnits == nil {
				continue
			}
			if err := uc.applySalePrice(ctx, in.ConnectionID, wp.ID, target.skuID, salePriceMinorUnits, saleEndDate); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("sku %s: %v", target.skuID, err))
				continue
			}
			result.Updated++
		}
	}
	return result, nil
}

func (uc *BackfillWooSalePricesUseCase) applySalePrice(ctx context.Context, connectionID, externalID, skuID string, salePriceMinorUnits *int64, saleEndDate *time.Time) error {
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		sku, err := uc.skuRepo.FindByID(ctx, skuID)
		if err != nil {
			return err
		}
		if sku.SalePriceMinorUnits != nil {
			return nil // something else set it since the scan -- don't clobber
		}
		before := *sku
		sku.SalePriceMinorUnits = salePriceMinorUnits
		sku.SaleEndDate = saleEndDate
		sku.UpdatedAt = time.Now()
		if err := uc.skuRepo.Update(ctx, sku); err != nil {
			return err
		}
		return eventrecord.Record(ctx, uc.outbox, events.ChannelSalePriceBackfilled, sku.ID,
			eventrecord.Data{"connection_id": connectionID, "external_id": externalID},
			eventrecord.NewChanges().
				Set("sale_price_minor_units", before.SalePriceMinorUnits, sku.SalePriceMinorUnits).
				Set("sale_end_date", before.SaleEndDate, sku.SaleEndDate))
	})
}
