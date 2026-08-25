package channels

import (
	"context"
	"errors"
	"fmt"
	"strconv"

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
) *PushProductToBolUseCase {
	return &PushProductToBolUseCase{
		db: db, connRepo: connRepo, bolFactory: bolFactory, encSecret: encryptionSecret,
		productRepo: productRepo, variantRepo: variantRepo, skuRepo: skuRepo,
		listingRepo: listingRepo, overrideRepo: overrideRepo, contentRepo: contentRepo,
		syncLogRepo: syncLogRepo, stockLevel: stockLevel,
	}
}

func (uc *PushProductToBolUseCase) Execute(ctx context.Context, in PushProductToBolInput) ([]channelsDomain.ProductChannelListing, error) {
	var conn integrationsDomain.Connection
	var product pmsDomain.Product
	var pushes []variantPush
	var override *channelsDomain.BolProductOverride
	var content *channelsDomain.BolProductContent
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

		content, err = uc.contentRepo.FindByProductAndConnection(ctx, in.ProductID, in.ConnectionID)
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
		listing, err := uc.pushOne(ctx, client, product, pu, override, content, existingByVariant[pu.Variant.ID],
			in.ConnectionID, economicOperatorID, condition, fulfilment, deliveryCode, gpcCode, reference)
		if err != nil {
			return nil, err
		}
		result = append(result, listing)
	}
	return result, nil
}

// bolProductContentAttributes flattens BolProductContent into the id/value
// map SubmitProductContent sends -- see BolProductContentInput.Attributes'
// doc comment on why these keys are the Dutch labels from Bol's own UI
// rather than a verified attribute id. nil content (nothing filled in
// yet) returns nil, not an error -- SubmitProductContent still sends
// EAN/GPC Code/Title/Description on their own.
func bolProductContentAttributes(c *channelsDomain.BolProductContent) map[string]string {
	if c == nil {
		return nil
	}
	attrs := map[string]string{}
	setStr := func(key string, v *string) {
		if v != nil && *v != "" {
			attrs[key] = *v
		}
	}
	setInt := func(key string, v *int32) {
		if v != nil {
			attrs[key] = strconv.FormatInt(int64(*v), 10)
		}
	}
	setBool := func(key string, v *bool) {
		if v != nil {
			attrs[key] = strconv.FormatBool(*v)
		}
	}

	setStr("E-mailadres of online contactformulier van de fabrikant", c.ManufacturerContact)
	setStr("Fabrikant Adres", c.ManufacturerAddress)
	setStr("Fabrikant Naam", c.ManufacturerName)
	setStr("Houdbaarheid gesloten product", c.ShelfLifeClosed)
	setStr("Houdbaarheid periode na opening", c.ShelfLifeAfterOpening)
	setStr("Ingrediënten", c.Ingredients)
	setStr("Inhoud", c.ContentAmount)
	setStr("Inhoud eenheid", c.ContentUnit)
	setStr("Land van oorsprong", c.CountryOfOrigin)
	setStr("Substantie", c.Substance)
	setStr("Taal handleiding", c.InstructionLanguage)
	setStr("Verpakkingstaal", c.PackagingLanguage)

	setInt("Aantal artikelen in verpakking", c.ItemsPerPackage)
	setStr("Bijsluiter", c.PackageInsertURL)
	setBool("Cadeau", c.IsGift)
	setBool("Dermatologisch getest", c.DermatologicallyTested)
	setStr("Gevarenaanduiding (H-zinnen)", c.HazardStatements)
	setStr("Huidtype", c.SkinType)
	setBool("Hypo-allergeen", c.Hypoallergenic)
	setBool("Inclusief Zinkoxide", c.ContainsZincOxide)
	setStr("Inhoud per dispenser", c.ContentPerDispenserAmount)
	setStr("Inhoud per dispenser eenheid", c.ContentPerDispenserUnit)
	setStr("Merk", c.Brand)
	setBool("Parfumvrij", c.FragranceFree)
	setBool("pH-neutraal", c.PHNeutral)
	setBool("Verkoelende werking", c.CoolingEffect)
	setInt("Verpakking breedte", c.PackageWidthMM)
	setInt("Verpakking hoogte", c.PackageHeightMM)
	setInt("Verpakking lengte", c.PackageLengthMM)
	setInt("Verpakkingsgewicht", c.PackageWeightG)
	setStr("Voorzorgsaanduiding (P-zinnen)", c.PrecautionStatements)
	setStr("Voorzorgsmaatregelen", c.Precautions)
	setStr("Waarschuwingen", c.Warnings)
	setStr("Video", c.VideoURL)

	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

func (uc *PushProductToBolUseCase) pushOne(
	ctx context.Context, client integrationsDomain.BolClient, product pmsDomain.Product, pu variantPush,
	override *channelsDomain.BolProductOverride, content *channelsDomain.BolProductContent, existing channelsDomain.ProductChannelListing,
	connectionID, economicOperatorID, condition, fulfilment, deliveryCode, gpcCode, reference string,
) (channelsDomain.ProductChannelListing, error) {
	// fail records this attempt as a failure in Sync History before
	// returning the error -- without this, a push failure (e.g. Bol
	// rejecting the offer payload) was a Go error the caller saw in logs
	// but the UI had zero record of, since recordSyncLog used to only ever
	// be called on the success paths below. existing may have no ID yet
	// (the very first push for this product/channel never got as far as
	// creating a listing) -- recordSyncLog treats an empty listing.ID as
	// "no listing to link this entry to" rather than erroring.
	fail := func(err error, fields map[string]string) (channelsDomain.ProductChannelListing, error) {
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, channelsDomain.ProductChannelListing{
			ID: existing.ID, ProductID: product.ID, ConnectionID: connectionID,
			Provider: string(integrationsDomain.ProviderBol), ExternalID: existing.ExternalID,
		}, channelsDomain.SyncFlowPush, channelsDomain.SyncLogStatusError, err.Error(), fields)
		return channelsDomain.ProductChannelListing{}, err
	}

	ean := pu.SKU.Barcode
	if override != nil && override.EANOverride != nil && *override.EANOverride != "" {
		ean = *override.EANOverride
	}
	if ean == "" {
		return fail(fmt.Errorf("pushProductToBol: %w", channelsDomain.ErrBolMissingEAN), nil)
	}

	stockAmount := 0
	if uc.stockLevel != nil {
		if level, err := uc.stockLevel.Execute(ctx, pu.SKU.ID); err == nil && level != nil {
			stockAmount = int(level.Syncable)
		}
	}

	priceMinorUnits := pu.SKU.PriceMinorUnits
	if override != nil && override.CustomPriceMinorUnits != nil {
		priceMinorUnits = *override.CustomPriceMinorUnits
	}

	offerIn := integrationsDomain.BolOfferInput{
		EAN: ean, EconomicOperatorID: economicOperatorID, Reference: reference,
		ConditionCategory: condition, FulfilmentMethod: fulfilment, DeliveryCode: deliveryCode,
		UnitPrice:   float64(priceMinorUnits) / 100,
		StockAmount: stockAmount, StockManagedByRetailer: true,
	}

	offerFields := bolOfferToFields(offerIn)

	// Product content (manufacturer info, ingredients, etc -- the Bol tab's
	// "Product content" section) is submitted whenever the merchant has
	// filled any of it in, unconditionally -- not just on the unknown-EAN
	// create-retry path below. That path only ever fires for a genuinely
	// new offer; a product whose offer already exists takes the update
	// branch and would otherwise never submit content at all, no matter
	// how many fields were saved on the Bol tab. Content and the offer are
	// separate Bol resources (Product Content API vs Offers API), so this
	// keeps them in sync on every push regardless of offer create/update.
	contentAttrs := bolProductContentAttributes(content)
	contentSubmitted := false
	if contentAttrs != nil {
		if contentErr := client.SubmitProductContent(ctx, integrationsDomain.BolProductContentInput{
			EAN: ean, GPCCode: gpcCode, Title: product.Name, Description: product.Description,
			Attributes: contentAttrs,
		}); contentErr != nil {
			return fail(fmt.Errorf("pushProductToBol: submit product content: %w", contentErr), offerFields)
		}
		contentSubmitted = true
	}

	// No local listing yet doesn't mean no Bol offer exists yet -- it may
	// predate this integration, or a local listing row was lost while the
	// real offer survived. Blindly POSTing in that case 409 Conflicts
	// against the existing offer; checking first lets this link and update
	// it instead, same as if the local listing row had never gone missing.
	if existing.ExternalID == "" {
		foundID, found, err := client.FindOfferByEAN(ctx, ean)
		if err != nil {
			return fail(fmt.Errorf("pushProductToBol: find offer by ean: %w", err), offerFields)
		}
		if found {
			existing.ExternalID = foundID
		}
	}

	var offerID string
	if existing.ExternalID != "" {
		if err := client.UpdateOffer(ctx, existing.ExternalID, offerIn); err != nil {
			return fail(fmt.Errorf("pushProductToBol: update offer: %w", err), offerFields)
		}
		offerID = existing.ExternalID
	} else {
		id, createErr := client.CreateOffer(ctx, offerIn)
		if createErr != nil {
			if !client.IsUnknownEAN(createErr) {
				return fail(fmt.Errorf("pushProductToBol: create offer: %w", createErr), offerFields)
			}
			// Content was already submitted above whenever the merchant had
			// any configured -- only fall back to this minimal EAN/GPC
			// Code/Title/Description submission when there was nothing to
			// submit yet (content == nil), so Bol has at least enough to
			// recognize the EAN.
			if !contentSubmitted {
				if contentErr := client.SubmitProductContent(ctx, integrationsDomain.BolProductContentInput{
					EAN: ean, GPCCode: gpcCode, Title: product.Name, Description: product.Description,
				}); contentErr != nil {
					return fail(fmt.Errorf("pushProductToBol: submit product content: %w", contentErr), offerFields)
				}
			}
			id, createErr = client.CreateOffer(ctx, offerIn)
			if createErr != nil {
				return fail(fmt.Errorf("pushProductToBol: create offer after product content: %w", createErr), offerFields)
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
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, existing, channelsDomain.SyncFlowPush,
			channelsDomain.SyncLogStatusSuccess, "", bolOfferToFields(offerIn))
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
	recordSyncLog(ctx, uc.db, uc.syncLogRepo, listing, channelsDomain.SyncFlowPush,
		channelsDomain.SyncLogStatusSuccess, "", bolOfferToFields(offerIn))
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
