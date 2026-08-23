package channels

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/crypto"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

type PushProductToBolInput struct {
	ProductID    string
	ConnectionID string
}

// PushProductToBolUseCase pushes a product to Bol as one offer per
// non-bulk-pool SKU -- unlike PushProductToWooUseCase, Bol has no
// "variation" concept to migrate between, so this is always the flat
// "separate-products" shape: single-SKU products get one offer with
// VariantID nil, multi-variant products get one offer per Variant with
// VariantID set. Additive alongside PushProductToWooUseCase -- doesn't
// touch it.
//
// An EAN unknown to Bol's catalog is handled by submitting Product Content
// first (see domain/integrations.BolClient.SubmitProductContent's doc
// comment on the unverified parts of this) then retrying the offer create.
type PushProductToBolUseCase struct {
	db           *gorm.DB
	connRepo     integrationsDomain.ConnectionRepository
	bolFactory   integrationsDomain.BolClientFactory
	encSecret    string
	productRepo  pmsDomain.ProductRepository
	variantRepo  pmsDomain.VariantRepository
	skuRepo      pmsDomain.SKURepository
	listingRepo  channelsDomain.ProductChannelListingRepository
	overrideRepo channelsDomain.BolProductOverrideRepository
	stockLevel   stockLevelGetter
}

func NewPushProductToBolUseCase(
	db *gorm.DB,
	connRepo integrationsDomain.ConnectionRepository,
	bolFactory integrationsDomain.BolClientFactory,
	encryptionSecret string,
	productRepo pmsDomain.ProductRepository,
	variantRepo pmsDomain.VariantRepository,
	skuRepo pmsDomain.SKURepository,
	listingRepo channelsDomain.ProductChannelListingRepository,
	overrideRepo channelsDomain.BolProductOverrideRepository,
	stockLevel stockLevelGetter,
) *PushProductToBolUseCase {
	return &PushProductToBolUseCase{
		db: db, connRepo: connRepo, bolFactory: bolFactory, encSecret: encryptionSecret,
		productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		listingRepo: listingRepo, overrideRepo: overrideRepo, stockLevel: stockLevel,
	}
}

func (uc *PushProductToBolUseCase) Execute(ctx context.Context, in PushProductToBolInput) ([]channelsDomain.ProductChannelListing, error) {
	var conn integrationsDomain.Connection
	var product pmsDomain.Product
	var pushes []variantPush
	var override *channelsDomain.BolProductOverride
	existingByVariant := map[string]channelsDomain.ProductChannelListing{}

	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, in.ConnectionID)
		if err != nil {
			return err
		}
		conn = *c

		p, err := uc.productRepo.FindByID(ctx, in.ProductID)
		if err != nil {
			return err
		}
		product = *p

		variants, err := uc.variantRepo.ListByProduct(ctx, in.ProductID)
		if err != nil {
			return err
		}
		if len(variants) == 0 {
			return channelsDomain.ErrCannotPushVariableProduct
		}
		for _, v := range variants {
			skus, err := uc.skuRepo.ListByVariant(ctx, v.ID)
			if err != nil {
				return err
			}
			if len(skus) != 1 {
				return channelsDomain.ErrCannotPushVariableProduct
			}
			pushes = append(pushes, variantPush{Variant: v, SKU: skus[0]})
		}

		// A bulk pool (UnitType=kilo) SKU is never itself sellable -- same
		// exclusion PushProductToWooUseCase applies.
		sellable := pushes[:0]
		for _, p := range pushes {
			if p.SKU.UnitType != pmsDomain.SKUUnitTypeKilo {
				sellable = append(sellable, p)
			}
		}
		pushes = sellable
		if len(pushes) == 0 {
			return channelsDomain.ErrCannotPushBulkPoolSKU
		}

		override, err = uc.overrideRepo.FindByProductAndConnection(ctx, in.ProductID, in.ConnectionID)
		if err != nil {
			return err
		}

		for _, pu := range pushes {
			variantID := bolListingVariantID(product, pu.Variant.ID)
			l, err := uc.listingRepo.FindByProductVariantAndConnection(ctx, in.ProductID, variantID, in.ConnectionID)
			if err == nil {
				existingByVariant[pu.Variant.ID] = *l
			} else if !errors.Is(err, channelsDomain.ErrListingNotFound) {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("pushProductToBol: %w", err)
	}

	secret, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("pushProductToBol: decrypt secret: %w", err)
	}
	client, err := uc.bolFactory.NewClient(conn.ConsumerKey, secret)
	if err != nil {
		return nil, fmt.Errorf("pushProductToBol: build client: %w", err)
	}

	economicOperatorID := ""
	if conn.BolEconomicOperatorID != nil {
		economicOperatorID = *conn.BolEconomicOperatorID
	}
	condition := string(channelsDomain.BolConditionNew)
	fulfilment := string(channelsDomain.BolFulfilmentFBR)
	gpcCode, reference := "", ""
	if override != nil {
		if override.ConditionCategory != "" {
			condition = string(override.ConditionCategory)
		}
		if override.FulfilmentMethod != "" {
			fulfilment = string(override.FulfilmentMethod)
		}
		if override.GPCCode != nil {
			gpcCode = *override.GPCCode
		}
		if override.ReferenceOverride != nil {
			reference = *override.ReferenceOverride
		}
	}

	var result []channelsDomain.ProductChannelListing
	for _, pu := range pushes {
		listing, err := uc.pushOne(ctx, client, product, pu, override, existingByVariant[pu.Variant.ID],
			in.ConnectionID, economicOperatorID, condition, fulfilment, gpcCode, reference)
		if err != nil {
			return nil, err
		}
		result = append(result, listing)
	}
	return result, nil
}

func (uc *PushProductToBolUseCase) pushOne(
	ctx context.Context, client integrationsDomain.BolClient, product pmsDomain.Product, pu variantPush,
	override *channelsDomain.BolProductOverride, existing channelsDomain.ProductChannelListing,
	connectionID, economicOperatorID, condition, fulfilment, gpcCode, reference string,
) (channelsDomain.ProductChannelListing, error) {
	ean := pu.SKU.Barcode
	if override != nil && override.EANOverride != nil && *override.EANOverride != "" {
		ean = *override.EANOverride
	}
	if ean == "" {
		return channelsDomain.ProductChannelListing{}, fmt.Errorf("pushProductToBol: %w", channelsDomain.ErrBolMissingEAN)
	}

	stockAmount := 0
	if uc.stockLevel != nil {
		if level, err := uc.stockLevel.Execute(ctx, pu.SKU.ID); err == nil && level != nil {
			stockAmount = int(level.Syncable)
		}
	}

	offerIn := integrationsDomain.BolOfferInput{
		EAN: ean, EconomicOperatorID: economicOperatorID, Reference: reference,
		ConditionCategory: condition, FulfilmentMethod: fulfilment,
		UnitPrice:   float64(pu.SKU.PriceMinorUnits) / 100,
		StockAmount: stockAmount, StockManagedByRetailer: true,
	}

	var offerID string
	if existing.ExternalID != "" {
		if err := client.UpdateOffer(ctx, existing.ExternalID, offerIn); err != nil {
			return channelsDomain.ProductChannelListing{}, fmt.Errorf("pushProductToBol: update offer: %w", err)
		}
		offerID = existing.ExternalID
	} else {
		id, createErr := client.CreateOffer(ctx, offerIn)
		if createErr != nil {
			if !client.IsUnknownEAN(createErr) {
				return channelsDomain.ProductChannelListing{}, fmt.Errorf("pushProductToBol: create offer: %w", createErr)
			}
			if contentErr := client.SubmitProductContent(ctx, integrationsDomain.BolProductContentInput{
				EAN: ean, GPCCode: gpcCode, Title: product.Name, Description: product.Description,
			}); contentErr != nil {
				return channelsDomain.ProductChannelListing{}, fmt.Errorf("pushProductToBol: submit product content: %w", contentErr)
			}
			id, createErr = client.CreateOffer(ctx, offerIn)
			if createErr != nil {
				return channelsDomain.ProductChannelListing{}, fmt.Errorf("pushProductToBol: create offer after product content: %w", createErr)
			}
		}
		offerID = id
	}

	variantID := bolListingVariantID(product, pu.Variant.ID)
	if existing.ID != "" {
		existing.ExternalID = offerID
		existing.SyncStatus = channelsDomain.SyncStatusSynced
		if err := uc.saveListing(ctx, &existing, false); err != nil {
			return channelsDomain.ProductChannelListing{}, err
		}
		return existing, nil
	}

	listing := channelsDomain.ProductChannelListing{
		ProductID: product.ID, ConnectionID: connectionID,
		Provider: string(integrationsDomain.ProviderBol), ExternalID: offerID,
		VariantID: variantID, IsVariation: false, SyncStatus: channelsDomain.SyncStatusSynced,
	}
	if err := uc.saveListing(ctx, &listing, true); err != nil {
		return channelsDomain.ProductChannelListing{}, err
	}
	return listing, nil
}

func (uc *PushProductToBolUseCase) saveListing(ctx context.Context, l *channelsDomain.ProductChannelListing, create bool) error {
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		l.TenantID = tenant.IDFromContext(ctx)
		if create {
			return uc.listingRepo.Create(ctx, l)
		}
		return uc.listingRepo.Update(ctx, l)
	})
}

// bolListingVariantID mirrors ProductChannelListing.VariantID's convention:
// nil for a single-SKU product's one listing, set for each variant of a
// multi-variant product. Bol has no "variations" mode to migrate between,
// so there's no third case.
func bolListingVariantID(product pmsDomain.Product, variantID string) *string {
	if !product.HasVariants {
		return nil
	}
	v := variantID
	return &v
}
