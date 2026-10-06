package channels

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	pmsDomain "github.com/adirkuhn/commerceos/domain/pms"
	"github.com/adirkuhn/commerceos/internal/tenant"
)

// recordSyncLog writes an audit entry for one sync attempt against one
// listing — shared by every use case that talks to a channel (push,
// import, stock import/export), so "everything that's synced" ends up in
// one place regardless of which direction or entity moved. Diffs fields
// against that listing's most recent entry — on the very first entry for a
// listing, every field shows as changed from "" (nothing existed before).
//
// A successful sync whose fields are identical to the last recorded entry
// is skipped entirely — no row written. Without this, a single product
// edit that touches both the product and its SKU (two separate API calls,
// two separate pms.product.changed events) produces two syncs moments
// apart: the second one finds nothing left to change and would otherwise
// show up as a confusing "No changes" row right under the real one. A
// failed sync is always recorded, even if fields match, since the failure
// itself is the thing worth an audit trail.
//
// Every written entry is also raised as channels.product_listing.synced
// (same transaction), so a skipped no-change success raises nothing either.
//
// Must be called from its own tenant.WithTenant block — logging failures
// are only logged, never propagated, since the actual sync this entry
// describes has already happened by the time this runs.
func recordSyncLog(
	ctx context.Context, db *gorm.DB, syncLogRepo channelsDomain.SyncLogRepository, outbox eventrecord.Enqueuer,
	listing channelsDomain.ProductChannelListing,
	direction channelsDomain.SyncDirectionFlow, status channelsDomain.SyncLogStatus, errMsg string, fields map[string]string,
) {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		// listing.ID is "" when this records a failure that happened before
		// any ProductChannelListing row ever existed (e.g. the very first
		// CreateOffer call for a product failing validation) -- there is
		// nothing to dedupe against yet, and querying/storing "" would hit
		// the listing_id foreign key (it's a UUID column; "" is neither a
		// valid UUID nor an existing row).
		var listingID *string
		var prevFields map[string]string
		hasPrev := false
		if listing.ID != "" {
			id := listing.ID
			listingID = &id
			prev, err := syncLogRepo.ListByListing(ctx, listing.ID, 1)
			if err != nil {
				return err
			}
			hasPrev = len(prev) > 0
			if hasPrev {
				prevFields = prev[0].Fields
			}
		}
		changes := diffFields(prevFields, fields)
		if status == channelsDomain.SyncLogStatusSuccess && hasPrev && len(changes) == 0 {
			return nil
		}

		entry := channelsDomain.SyncLogEntry{
			TenantID: tenant.IDFromContext(ctx), ListingID: listingID, ProductID: listing.ProductID,
			ConnectionID: listing.ConnectionID, Provider: listing.Provider, ExternalID: listing.ExternalID,
			Direction: direction, Status: status, ErrorMessage: errMsg,
			Fields: fields, Changes: changes, SyncedAt: time.Now(),
		}
		if err := syncLogRepo.Create(ctx, &entry); err != nil {
			return err
		}
		return eventrecord.Record(ctx, outbox, events.ProductListingSynced, listing.ID, eventrecord.Data{
			"product_id": listing.ProductID, "connection_id": listing.ConnectionID, "provider": listing.Provider,
			"external_id": listing.ExternalID, "direction": string(direction), "status": string(status), "error_message": errMsg,
		}, syncLogChanges(changes))
	})
	if err != nil {
		log.Printf("recordSyncLog: writing entry for listing %s: %v", listing.ID, err)
	}
}

// diffFields returns only the fields whose value changed between prev and
// curr — a field present in curr but not prev is treated as changing from
// "". Nil prev (first-ever sync) means "everything in curr is new".
func diffFields(prev, curr map[string]string) map[string]channelsDomain.FieldChange {
	changes := map[string]channelsDomain.FieldChange{}
	for k, to := range curr {
		from := prev[k]
		if from != to {
			changes[k] = channelsDomain.FieldChange{From: from, To: to}
		}
	}
	return changes
}

// importedFieldsFromWoo is the import-direction counterpart to
// wooProductToFields, populated from the CommerceOS-side record of what
// was just imported. Deliberately uses different key names for
// weight/dimensions (weight_grams/length_mm/... vs push's store-unit
// weight/length/...) — the two directions store measurements in different
// units, so giving them the same key would make an import->push diff look
// like a real value change when it's actually just a unit conversion.
func importedFieldsFromWoo(wp integrationsDomain.WooProduct, product pmsDomain.Product, sku pmsDomain.SKU) map[string]string {
	fields := map[string]string{
		"name":          product.Name,
		"description":   product.Description,
		"status":        wp.Status,
		"regular_price": wp.RegularPrice,
		"virtual":       fmt.Sprintf("%t", sku.IsVirtual),
	}
	if sku.WeightGrams != nil {
		fields["weight_grams"] = strconv.FormatInt(*sku.WeightGrams, 10)
	}
	if sku.LengthMM != nil {
		fields["length_mm"] = strconv.FormatInt(*sku.LengthMM, 10)
	}
	if sku.WidthMM != nil {
		fields["width_mm"] = strconv.FormatInt(*sku.WidthMM, 10)
	}
	if sku.HeightMM != nil {
		fields["height_mm"] = strconv.FormatInt(*sku.HeightMM, 10)
	}
	return fields
}

// bolOfferToFields is wooProductToFields's Bol counterpart -- the offer
// payload actually sent to Bol, snapshotted for diffing/logging.
// mergeFields combines two field snapshots for one recordSyncLog call --
// e.g. Bol's offer fields (price/stock/etc) plus its product-content
// attributes, which are a separate Bol resource pushed in the same
// pushOne call but weren't part of the diffed snapshot at all until this
// fix. Content-only changes (no price/stock movement) were invisible to
// the diff in bolOfferToFields alone, so recordSyncLog's "identical to
// last entry, skip" dedup (see its doc comment) silently swallowed every
// content-only sync: the push to Bol genuinely succeeded, but no row ever
// appeared on /sync-log for it (caught 2026-08-25: manufacturer-field
// fixes deployed and pushed with no error, nothing showed up in the log).
// No key collisions expected -- offer keys are lowercase snake_case,
// content keys are Bol's title-case attribute ids.
func mergeFields(maps ...map[string]string) map[string]string {
	merged := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			merged[k] = v
		}
	}
	return merged
}

func bolOfferToFields(o integrationsDomain.BolOfferInput) map[string]string {
	return map[string]string{
		"ean":                o.EAN,
		"reference":          o.Reference,
		"condition_category": o.ConditionCategory,
		"fulfilment_method":  o.FulfilmentMethod,
		"delivery_code":      o.DeliveryCode,
		"price":              strconv.FormatFloat(o.UnitPrice, 'f', 2, 64),
		"stock_quantity":     strconv.Itoa(o.StockAmount),
	}
}

// wooProductToFields flattens the fields this codebase actually manages
// into a snapshot suitable for diffing/logging — deliberately excludes
// anything not sent (categories, images, SEO meta).
func wooProductToFields(p integrationsDomain.WooProduct) map[string]string {
	return map[string]string{
		"name":          p.Name,
		"description":   p.Description,
		"status":        p.Status,
		"regular_price": p.RegularPrice,
		"sale_price":    p.SalePrice,
		"sale_end_date": p.SaleEndDate,
		"virtual":       fmt.Sprintf("%t", p.Virtual),
		"weight":        p.Weight,
		"length":        p.Dimensions.Length,
		"width":         p.Dimensions.Width,
		"height":        p.Dimensions.Height,
	}
}
