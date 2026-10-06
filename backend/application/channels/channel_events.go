package channels

import (
	"context"

	"github.com/adirkuhn/commerceos/application/eventrecord"
	channelsDomain "github.com/adirkuhn/commerceos/domain/channels"
	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/events"
	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
)

// How an external category link was made -- data.source of
// channels.external_category_link.saved.
const (
	CategoryLinkSourceManual    = "manual"
	CategoryLinkSourceSlugMatch = "slug_match"
)

// salesChannelChanges are the fields an update changed on a sales channel.
func salesChannelChanges(before, after channelsDomain.SalesChannel) eventrecord.Changes {
	return eventrecord.NewChanges().
		Set("name", before.Name, after.Name).
		Set("connection_id", before.ConnectionID, after.ConnectionID).
		Set("enrollment_mode", string(before.EnrollmentMode), string(after.EnrollmentMode)).
		Set("enrollment_category_id", before.EnrollmentCategoryID, after.EnrollmentCategoryID).
		Set("enrollment_tag", before.EnrollmentTag, after.EnrollmentTag).
		Set("price_adjustment_basis_points", before.PriceAdjustmentBasisPoints, after.PriceAdjustmentBasisPoints).
		Set("quantity_breaks", before.QuantityBreaks, after.QuantityBreaks).
		Set("default_warehouse_id", before.DefaultWarehouseID, after.DefaultWarehouseID).
		Set("storefront_mode", string(before.StorefrontMode), string(after.StorefrontMode)).
		Set("sign_in_url", before.SignInURL, after.SignInURL).
		Set("storefront_locale", string(before.StorefrontLocale), string(after.StorefrontLocale)).
		Set("storefront_locales", before.StorefrontLocales, after.StorefrontLocales).
		Set("active", before.Active, after.Active).
		Set("payment_connection_id", before.PaymentConnectionID, after.PaymentConnectionID)
}

func syncScheduleData(s channelsDomain.SyncSchedule) eventrecord.Data {
	return eventrecord.Data{
		"connection_id": s.ConnectionID, "entity_type": string(s.EntityType), "direction": string(s.Direction),
		"enabled": s.Enabled, "interval_minutes": s.IntervalMinutes,
		"default_warehouse_id": s.DefaultWarehouseID, "default_currency": s.DefaultCurrency,
	}
}

func recordCategoryLinkSaved(ctx context.Context, outbox eventrecord.Enqueuer, l channelsDomain.ExternalCategoryLink, source string) error {
	return eventrecord.Record(ctx, outbox, events.ExternalCategoryLinkSaved, l.ID, eventrecord.Data{
		"connection_id": l.ConnectionID, "provider": l.Provider, "external_id": l.ExternalID,
		"category_id": l.CategoryID, "source": source,
	})
}

// Why a customer was imported -- data.source of channels.customer.imported.
const (
	CustomerImportSourceCustomers = "customer_import"
	CustomerImportSourceOrders    = "order_import"
)

func recordCustomerImported(ctx context.Context, outbox eventrecord.Enqueuer, c customersDomain.Customer, source string) error {
	return eventrecord.Record(ctx, outbox, events.ChannelCustomerImported, c.ID, eventrecord.Data{
		"provider": string(integrationsDomain.ProviderWooCommerce), "external_customer_id": c.WooCustomerID,
		"email": c.Email, "source": source,
	})
}

// recordOrderImported records a sales order created from a channel order.
func recordOrderImported(ctx context.Context, outbox eventrecord.Enqueuer, o omsDomain.SalesOrder, lineCount, unlinkedLines int) error {
	return eventrecord.Record(ctx, outbox, events.ChannelOrderImported, o.ID, eventrecord.Data{
		"provider": string(o.Source), "connection_id": o.ConnectionID, "external_order_id": o.ExternalOrderID,
		"number": o.Number, "status": string(o.Status), "currency_code": o.CurrencyCode,
		"total_minor_units": o.TotalMinorUnits, "line_count": lineCount,
		"unlinked_line_count": unlinkedLines, "needs_sku_review": o.NeedsSKUReview,
	})
}
