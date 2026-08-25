package channels

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

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

// bolUnitID translates a free-text unit string (whatever the merchant typed
// into the Bol tab's Content Unit / Content per Dispenser Unit fields) into
// Bol's unece unit code for AttributeValue.unitId. Confirmed against the
// "Capacity"/"Capacity per dispenser" attributes' units lists in
// productdatamodel.s-bol.com/v10/datamodel_v10_en.json. Unrecognized input
// returns "" (no unitId sent) rather than guessing or erroring -- the value
// itself still gets submitted, same as before this fix, just without a
// unit Bol can act on; better than blocking the whole content save over a
// free-text field Bol never validated to begin with.
func bolUnitID(unit string) string {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "ml", "milliliter", "milliliters", "milliliter(s)":
		return "unece.unit.MLT"
	case "l", "liter", "liters", "litre", "litres":
		return "unece.unit.LTR"
	case "cl", "centiliter", "centiliters":
		return "unece.unit.CLT"
	case "g", "gram", "grams":
		return "unece.unit.GRM"
	case "kg", "kilogram", "kilograms":
		return "unece.unit.KGM"
	case "mg", "milligram", "milligrams":
		return "unece.unit.MGM"
	case "stuk", "stuks", "piece", "pieces":
		return "Pieces"
	default:
		return ""
	}
}

// bolProductContentAttributes flattens BolProductContent into the id/value
// map (and, for number+unit attributes, the companion unit map)
// SubmitProductContent sends. nil content (nothing filled in yet) returns
// nil/nil, not an error -- SubmitProductContent still sends EAN/Name/
// Description on their own.
//
// Keys were originally the Dutch UI label text (see git blame), which is
// NOT the real attribute id -- Bol's id is a separate, English, data-model
// key (e.g. "Manufacturer Name", not "Fabrikant Naam"). A wrong id isn't
// rejected: the POST still 202s and Bol silently drops that attribute, so
// this shipped for a while with content-save "succeeding" while fields
// never actually appeared on bol.com (caught 2026-08-25 in prod:
// Manufacturer Name/Address/Email never updated, no error anywhere). Every
// key below is now confirmed against Bol's real, published data model
// (productdatamodel.s-bol.com/v10/datamodel_v10_en.json -- id field --
// cross-referenced against the _nl.json file's name field to translate
// each Dutch label; values are submitted in Dutch since language is "nl").
//
// Two categories need more than a renamed key:
//   - Number+unit attributes (Capacity, Capacity per dispenser,
//     Width/Height/Length, Weight, Number Pieces In Package) -- Bol wants
//     value and unitId as sibling fields on one AttributeValue, not two
//     attributes. ContentAmount+ContentUnit merge into one "Capacity"
//     entry (unit via bolUnitID); same for the dispenser pair. Width/
//     Height/Length/Weight keep their own value but gain a fixed unitId
//     (the DB fields are always mm/g, no user-entered unit to translate).
//     Items-per-package gets a fixed "Pieces" unitId (no unit field exists
//     for it in the domain type, and "how many items" is inherently a
//     piece count).
//   - LOV-constrained attributes (Skin Type, Shelf life sealed/after
//     opening, Country of Origin, Hazard Statements, Substance,
//     Precautionary Statement) -- Bol only accepts values from a fixed
//     list. The id fix alone doesn't guarantee a match: see
//     ProductBolTab.tsx, which now renders these as selects populated from
//     the same real LOV data instead of free text, so a saved value is
//     guaranteed to be one Bol recognizes.
//
// Two fields stay free text deliberately: Language Instructions/Packaging
// only have LOV entries scoped per GPC category (e.g. "Language
// Instructions@30003886"), not a global list, so there's no single static
// dropdown that's correct for every category -- picking one product's
// category-scoped list and presenting it as universal would be its own
// silent-mismatch bug. Package Insert (a document URL) has no LOV at all.
func bolProductContentAttributes(c *channelsDomain.BolProductContent) (attrs map[string]string, units map[string]string) {
	if c == nil {
		return nil, nil
	}
	attrs = map[string]string{}
	units = map[string]string{}
	setStr := func(key string, v *string) {
		if v != nil && *v != "" {
			attrs[key] = *v
		}
	}
	setBool := func(key string, v *bool) {
		if v != nil {
			attrs[key] = strconv.FormatBool(*v)
		}
	}
	setIntWithUnit := func(key string, v *int32, unitID string) {
		if v != nil {
			attrs[key] = strconv.FormatInt(int64(*v), 10)
			units[key] = unitID
		}
	}
	setAmountWithUnit := func(key string, amount, unit *string) {
		if amount == nil || *amount == "" {
			return
		}
		attrs[key] = *amount
		if unit != nil {
			if id := bolUnitID(*unit); id != "" {
				units[key] = id
			}
		}
	}

	setStr("Manufacturer Email", c.ManufacturerContact)
	setStr("Manufacturer Address", c.ManufacturerAddress)
	setStr("Manufacturer Name", c.ManufacturerName)
	setStr("Shelf life sealed product", c.ShelfLifeClosed)
	setStr("Shelf life period after opening", c.ShelfLifeAfterOpening)
	setStr("Ingredients", c.Ingredients)
	setAmountWithUnit("Capacity", c.ContentAmount, c.ContentUnit)
	setStr("Country of Origin", c.CountryOfOrigin)
	setStr("Substance", c.Substance)
	setStr("Language Instructions", c.InstructionLanguage)
	setStr("Language Packaging", c.PackagingLanguage)

	setIntWithUnit("Number Pieces In Package", c.ItemsPerPackage, "Pieces")
	setStr("Package Insert", c.PackageInsertURL)
	setBool("Gift", c.IsGift)
	setBool("Dermatological Tested", c.DermatologicallyTested)
	setStr("Hazard Statements", c.HazardStatements)
	setStr("Skin Type", c.SkinType)
	setBool("Hypo Allergenic", c.Hypoallergenic)
	setBool("Including Zincoxide", c.ContainsZincOxide)
	setAmountWithUnit("Capacity per dispenser", c.ContentPerDispenserAmount, c.ContentPerDispenserUnit)
	setStr("Brand", c.Brand)
	setBool("Fragrance Free", c.FragranceFree)
	setBool("Ph Neutral", c.PHNeutral)
	setBool("Cooling Effect", c.CoolingEffect)
	setIntWithUnit("Width", c.PackageWidthMM, "unece.unit.MMT")
	setIntWithUnit("Height", c.PackageHeightMM, "unece.unit.MMT")
	setIntWithUnit("Length", c.PackageLengthMM, "unece.unit.MMT")
	setIntWithUnit("Weight", c.PackageWeightG, "unece.unit.GRM")
	setStr("Precautionary Statement", c.PrecautionStatements)
	setStr("Precautionary Measures", c.Precautions)
	setStr("Warning Label", c.Warnings)
	// c.VideoURL is deliberately not sent: "Video" is not a real attribute
	// id (searched the full data model, no match, nor any generic
	// product-video attribute) -- it was silently dropped same as
	// everything else in this map before the fix. Bol's Asset (images)
	// mechanism is the most likely real fit for video, but no label value
	// for it is confirmed (spec only documents "FRONT" for images), so
	// this isn't guessed at rather than shipping another unverified guess.

	if len(attrs) == 0 {
		return nil, nil
	}
	return attrs, units
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
	contentAttrs, contentUnits := bolProductContentAttributes(content)
	contentSubmitted := false
	if contentAttrs != nil {
		if contentErr := client.SubmitProductContent(ctx, integrationsDomain.BolProductContentInput{
			EAN: ean, Title: product.Name, Description: product.Description,
			Attributes: contentAttrs, AttributeUnits: contentUnits,
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
			// any configured -- only fall back to this minimal EAN/Title/
			// Description submission when there was nothing to submit yet
			// (content == nil), so Bol has at least enough to recognize
			// the EAN.
			if !contentSubmitted {
				if contentErr := client.SubmitProductContent(ctx, integrationsDomain.BolProductContentInput{
					EAN: ean, Title: product.Name, Description: product.Description,
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
