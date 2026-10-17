package channels

import (
	"context"
	"errors"
	"fmt"
	"github.com/adirkuhn/commerceos/application/eventrecord"

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
	contentRepo  channelsDomain.BolProductContentRepository
	syncLogRepo  channelsDomain.SyncLogRepository
	stockLevel   stockLevelGetter
	notifier     notifier
	outbox       eventrecord.Enqueuer
	vatRates     vatRateResolver
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
	contentRepo channelsDomain.BolProductContentRepository,
	syncLogRepo channelsDomain.SyncLogRepository,
	stockLevel stockLevelGetter,
	notifier notifier,
	outbox eventrecord.Enqueuer,
	vatRates vatRateResolver,
) *PushProductToBolUseCase {
	return &PushProductToBolUseCase{
		db: db, connRepo: connRepo, bolFactory: bolFactory, encSecret: encryptionSecret,
		productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		listingRepo: listingRepo, overrideRepo: overrideRepo, contentRepo: contentRepo,
		syncLogRepo: syncLogRepo, stockLevel: stockLevel, notifier: notifier,
		outbox: outbox, vatRates: vatRates,
	}
}

// SetOnHold toggles one existing Bol offer's onHoldByRetailer flag via a
// minimal PATCH -- unlike Execute, it never touches price/stock/content, so
// it's safe to call without recomputing any of pushOne's inputs. Used by
// SyncProductToChannelsUseCase to gate/ungate a Bol listing when the local
// product's status crosses the Active boundary.
//
//events:delegated SyncProductToChannelsUseCase.setGatedAt records channels.product_listing.gated/ungated for this on-hold flip; the push itself records its listing and sync events in push_product_to_bol_listing.go
func (uc *PushProductToBolUseCase) SetOnHold(ctx context.Context, connectionID, externalID string, onHold bool) error {
	var conn integrationsDomain.Connection
	err := tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		c, err := uc.connRepo.FindByID(ctx, connectionID)
		if err != nil {
			return err
		}
		conn = *c
		return nil
	})
	if err != nil {
		return fmt.Errorf("pushProductToBol.SetOnHold: %w", err)
	}
	secret, err := crypto.Decrypt(uc.encSecret, conn.ConsumerSecretEncrypted)
	if err != nil {
		return fmt.Errorf("pushProductToBol.SetOnHold: decrypt secret: %w", err)
	}
	client, err := uc.bolFactory.NewClient(conn.ConsumerKey, secret)
	if err != nil {
		return fmt.Errorf("pushProductToBol.SetOnHold: build client: %w", err)
	}
	return client.SetOfferOnHold(ctx, externalID, onHold)
}

func (uc *PushProductToBolUseCase) Execute(ctx context.Context, in PushProductToBolInput) ([]channelsDomain.ProductChannelListing, error) {
	var conn integrationsDomain.Connection
	var product pmsDomain.Product
	var pushes []variantPush
	var override *channelsDomain.BolProductOverride
	var content *channelsDomain.BolProductContent
	existingByVariant := map[string]channelsDomain.ProductChannelListing{}
	vatBySKU := map[string]int64{}

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
		if !product.IsSyncableToChannels() {
			return channelsDomain.ErrProductNotSyncable
		}

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

		content, err = uc.contentRepo.FindByProductAndConnection(ctx, in.ProductID, in.ConnectionID)
		if err != nil {
			return err
		}

		for _, pu := range pushes {
			vat, err := uc.vatRates.RateBasisPoints(ctx, pu.SKU.TaxClassID, bolVATCountry)
			if err != nil {
				return err
			}
			vatBySKU[pu.SKU.ID] = vat

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
	deliveryCode := integrationsDomain.DefaultBolDeliveryCode
	if conn.BolDeliveryCode != nil && *conn.BolDeliveryCode != "" {
		deliveryCode = *conn.BolDeliveryCode
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
		priceIn := channelsDomain.BolPriceInputs{
			RegularPriceMinorUnits: pu.SKU.PriceMinorUnits, ProductVATBasisPoints: vatBySKU[pu.SKU.ID],
			ShippingMinorUnits: conn.BolShippingCostMinorUnits,
		}
		listing, err := uc.pushOne(ctx, client, product, pu, override, content, existingByVariant[pu.Variant.ID],
			in.ConnectionID, economicOperatorID, condition, fulfilment, deliveryCode, gpcCode, reference, priceIn)
		if err != nil {
			return nil, err
		}
		result = append(result, listing)
	}
	return result, nil
}
