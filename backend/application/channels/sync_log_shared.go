package channels

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"gorm.io/gorm"

	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
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
// Must be called from its own tenant.WithTenant block — logging failures
// are only logged, never propagated, since the actual sync this entry
// describes has already happened by the time this runs.
func recordSyncLog(
	ctx context.Context, db *gorm.DB, syncLogRepo channelsDomain.SyncLogRepository,
	listing channelsDomain.ProductChannelListing,
	direction channelsDomain.SyncDirectionFlow, status channelsDomain.SyncLogStatus, errMsg string, fields map[string]string,
) {
	err := tenant.WithTenant(ctx, db, func(ctx context.Context) error {
		prev, err := syncLogRepo.ListByListing(ctx, listing.ID, 1)
		if err != nil {
			return err
		}
		var prevFields map[string]string
		if len(prev) > 0 {
			prevFields = prev[0].Fields
		}
		entry := channelsDomain.SyncLogEntry{
			TenantID: tenant.IDFromContext(ctx), ListingID: listing.ID, ProductID: listing.ProductID,
			ConnectionID: listing.ConnectionID, Provider: listing.Provider, ExternalID: listing.ExternalID,
			Direction: direction, Status: status, ErrorMessage: errMsg,
			Fields: fields, Changes: diffFields(prevFields, fields), SyncedAt: time.Now(),
		}
		return syncLogRepo.Create(ctx, &entry)
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

// wooProductToFields flattens the fields this codebase actually manages
// into a snapshot suitable for diffing/logging — deliberately excludes
// anything not sent (categories, images, SEO meta).
func wooProductToFields(p integrationsDomain.WooProduct) map[string]string {
	return map[string]string{
		"name":          p.Name,
		"description":   p.Description,
		"status":        p.Status,
		"regular_price": p.RegularPrice,
		"virtual":       fmt.Sprintf("%t", p.Virtual),
		"weight":        p.Weight,
		"length":        p.Dimensions.Length,
		"width":         p.Dimensions.Width,
		"height":        p.Dimensions.Height,
	}
}
