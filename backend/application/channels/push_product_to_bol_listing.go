package channels

import (
	"context"
	"fmt"
	"log"
	"strings"

	appnotifications "github.com/adirkuhn/commerceos/application/notifications"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	notificationsDomain "github.com/adirkuhn/commerceos/domain/notifications"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

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
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, channelsDomain.ProductChannelListing{
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

	// title wins over product.Name when the merchant set a Bol-only name
	// override (Bol tab) -- feeds only the Title submitted to Bol's Product
	// Content API below, nothing else reads it, so this never touches Woo
	// or any other channel's copy of the product name.
	title := product.Name
	if override != nil && override.NameOverride != nil && *override.NameOverride != "" {
		title = *override.NameOverride
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
	// Internal Reference and Product Classification are two of Bol's
	// enrichmentLevel-0 (mandatory-for-sale) content attributes that were
	// never sent at all -- confirmed 2026-08-25 against Bol's real data
	// model (chunks[].attributes filtered to enrichmentLevel 0, sampled
	// across several categories: EAN, Internal Reference, Manufacturer
	// Name/Address/Email, Name, Product Classification is the stable
	// core set). Internal Reference is the offer's own reference/SKU code
	// (already computed above as `reference`); Product Classification is
	// a free-text category label (e.g. "Blush") -- this is what the
	// override's GPC Code field actually maps onto (the numeric GPC/chunk
	// id itself, gpc.chunkId, is a read-only classification Bol assigns,
	// not something this endpoint accepts). Both merge into contentAttrs
	// unconditionally (independent of whether the merchant filled in the
	// optional Bol tab content fields) since they're mandatory whenever
	// known.
	if reference != "" {
		if contentAttrs == nil {
			contentAttrs = map[string]string{}
		}
		contentAttrs["Internal Reference"] = reference
	}
	if gpcCode != "" {
		if contentAttrs == nil {
			contentAttrs = map[string]string{}
		}
		contentAttrs["Product Classification"] = gpcCode
	}
	contentSubmitted := false
	if len(contentAttrs) > 0 {
		if contentErr := client.SubmitProductContent(ctx, integrationsDomain.BolProductContentInput{
			EAN: ean, Title: title, Description: product.Description,
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
					EAN: ean, Title: title, Description: product.Description,
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

	// Bol can 200/202 every call above with no error at all and still
	// refuse to actually list the offer for sale (e.g. a still-missing
	// mandatory content attribute) -- there is no other signal for this
	// anywhere in the push. Best-effort: a failure here is a diagnostic
	// call failing, not the push itself, so it's logged and swallowed
	// rather than turning a genuinely successful push into a failure.
	note := uc.checkNotForSale(ctx, client, offerID)

	variantID := bolListingVariantID(product, pu.Variant.ID)
	if existing.ID != "" {
		noteChanged := note != existing.NotForSaleNote
		before := existing
		existing.ExternalID = offerID
		existing.SyncStatus = channelsDomain.SyncStatusSynced
		existing.NotForSaleNote = note
		if err := uc.saveListing(ctx, &existing, &before); err != nil {
			return channelsDomain.ProductChannelListing{}, err
		}
		if noteChanged && note != "" {
			uc.notifyNotForSale(ctx, product, note)
		}
		recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, existing, channelsDomain.SyncFlowPush,
			channelsDomain.SyncLogStatusSuccess, "", mergeFields(bolOfferToFields(offerIn), contentAttrs))
		return existing, nil
	}

	listing := channelsDomain.ProductChannelListing{
		ProductID: product.ID, ConnectionID: connectionID,
		Provider: string(integrationsDomain.ProviderBol), ExternalID: offerID,
		VariantID: variantID, IsVariation: false, SyncStatus: channelsDomain.SyncStatusSynced,
		NotForSaleNote: note,
	}
	if err := uc.saveListing(ctx, &listing, nil); err != nil {
		return channelsDomain.ProductChannelListing{}, err
	}
	if note != "" {
		uc.notifyNotForSale(ctx, product, note)
	}
	recordSyncLog(ctx, uc.db, uc.syncLogRepo, uc.outbox, listing, channelsDomain.SyncFlowPush,
		channelsDomain.SyncLogStatusSuccess, "", mergeFields(bolOfferToFields(offerIn), contentAttrs))
	return listing, nil
}

// checkNotForSale calls Bol's own not-for-sale-reasons diagnostic and
// flattens it into one note string, or "" if the offer is for sale
// everywhere (or the check itself failed -- logged, not propagated, see
// the call site's comment).
func (uc *PushProductToBolUseCase) checkNotForSale(ctx context.Context, client integrationsDomain.BolClient, offerID string) string {
	reasons, err := client.GetNotForSaleReasons(ctx, offerID)
	if err != nil {
		log.Printf("pushProductToBol: get not-for-sale reasons for offer %s: %v", offerID, err)
		return ""
	}
	if len(reasons) == 0 {
		return ""
	}
	parts := make([]string, len(reasons))
	for i, r := range reasons {
		part := r.CountryCode + ": " + r.Description
		if len(r.MissingAttributes) > 0 {
			part += " (missing: " + strings.Join(r.MissingAttributes, ", ") + ")"
		}
		parts[i] = part
	}
	return strings.Join(parts, "; ")
}

// notifyNotForSale is best-effort and non-transactional, called after
// saveListing's own transaction has already committed -- same pattern as
// CreateImportJobUseCase's notifier (see its doc comment): nesting another
// tenant.WithTenant call inside an outer transaction's closure would open
// a second, unrelated transaction rather than participating in the first.
// A notification failure must never undo or fail a push that already
// succeeded.
func (uc *PushProductToBolUseCase) notifyNotForSale(ctx context.Context, product pmsDomain.Product, note string) {
	if _, err := uc.notifier.Execute(ctx, appnotifications.CreateNotificationInput{
		Type:    notificationsDomain.TypeBolNotForSale,
		Title:   "Bol: \"" + product.Name + "\" is not for sale",
		Message: note,
		Link:    "/products/" + product.ID,
	}); err != nil {
		log.Printf("pushProductToBol: notify not-for-sale for product %s: %v", product.ID, err)
	}
}

// saveListing writes the listing; before is its previous state on an
// update (nil for a create).
func (uc *PushProductToBolUseCase) saveListing(ctx context.Context, l *channelsDomain.ProductChannelListing, before *channelsDomain.ProductChannelListing) error {
	return tenant.WithTenant(ctx, uc.db, func(ctx context.Context) error {
		l.TenantID = tenant.IDFromContext(ctx)
		if before == nil {
			if err := uc.listingRepo.Create(ctx, l); err != nil {
				return err
			}
			return recordListingCreated(ctx, uc.outbox, *l, ListingSourcePush)
		}
		if err := uc.listingRepo.Update(ctx, l); err != nil {
			return err
		}
		return recordListingUpdated(ctx, uc.outbox, *before, *l)
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
