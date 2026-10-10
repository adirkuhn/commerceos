package messagecenter

import (
	"strings"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/locale"
	mcDomain "github.com/adirkuhn/commerceos/domain/messagecenter"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

// loadedOrder is everything a message about one sales order can show:
// the order, its customer (nil for a guest), its lines, and its current
// shipment (nil before one exists) with the link to track it.
type loadedOrder struct {
	order       *omsDomain.SalesOrder
	customer    *customersDomain.Customer
	lines       []omsDomain.SalesOrderLineItem
	shipment    *shippingDomain.Shipment
	trackingURL string
}

// storeVars are the store's own details -- domain/messagecenter.StoreVariables.
func storeVars(p *settingsDomain.CompanyProfile) map[string]any {
	if p == nil {
		p = &settingsDomain.CompanyProfile{}
	}
	name := p.TradingName
	if name == "" {
		name = p.LegalName
	}
	a := p.Address
	address := joinNonEmpty(", ",
		joinNonEmpty(" ", a.Street, a.HouseNumber),
		joinNonEmpty(" ", a.PostalCode, a.City),
		a.Country)
	return map[string]any{"store_name": name, "store_email": p.Email, "store_phone": p.Phone, "store_address": address}
}

// orderVars are one order's details -- domain/messagecenter.OrderVariables
// -- with money in the order's currency and dates in the tenant's format,
// both written in lang.
func orderVars(lo *loadedOrder, lang locale.Locale, dateFormat settingsDomain.DateFormat) map[string]any {
	o := lo.order
	money := func(minor int64) string { return lang.FormatMoney(minor, o.CurrencyCode) }

	first, last, email := o.Billing.FirstName, o.Billing.LastName, o.Billing.Email
	if c := lo.customer; c != nil {
		first, last = c.FirstName, c.LastName
		if c.Email != "" {
			email = c.Email
		}
	}

	items := make([]any, len(lo.lines))
	var subtotal int64
	for i, l := range lo.lines {
		subtotal += l.TotalMinorUnits
		items[i] = map[string]any{
			"name": l.Name, "sku": l.SKUCode, "quantity": l.Quantity,
			"unit_price": money(l.PriceMinorUnits), "total": money(l.TotalMinorUnits),
		}
	}

	s := o.Shipping
	carrier, barcode := "", ""
	if lo.shipment != nil {
		carrier, barcode = lo.shipment.Carrier.String(), lo.shipment.Barcode
	}

	return map[string]any{
		"customer_name": first, "customer_first_name": first, "customer_last_name": last, "customer_email": email,
		"order_number": o.Number, "order_date": dateFormat.Format(o.OccurredAt),
		"order_status": string(o.Status), "order_status_label": mcDomain.OrderStatusLabel(string(o.Status), lang),
		"items": items, "item_count": len(lo.lines),
		"subtotal": money(subtotal), "shipping_total": money(o.ShippingTotalMinorUnits),
		"discount_total": money(o.DiscountTotalMinorUnits), "tax_total": money(o.TaxTotalMinorUnits),
		"order_total":   money(o.TotalMinorUnits),
		"shipping_name": joinNonEmpty(" ", s.FirstName, s.LastName),
		"shipping_address": joinNonEmpty(", ",
			joinNonEmpty(" ", s.Address1, s.Address2),
			joinNonEmpty(" ", s.Postcode, s.City),
			s.Country),
		"shipping_method": o.ShippingMethodTitle, "payment_method": o.PaymentMethodTitle, "customer_note": o.CustomerNote,
		"carrier": carrier, "tracking_barcode": barcode, "tracking_url": lo.trackingURL,
	}
}

// eventVars are values derived from an event's own payload, e.g. a
// refund's amount in the order's currency.
func eventVars(payload map[string]any, lo *loadedOrder, lang locale.Locale) map[string]any {
	out := map[string]any{}
	if amount, ok := payload["amount_minor_units"].(float64); ok && lo != nil {
		out["refund_amount"] = lang.FormatMoney(int64(amount), lo.order.CurrencyCode)
	}
	return out
}

func joinNonEmpty(sep string, parts ...string) string {
	kept := parts[:0:0]
	for _, p := range parts {
		if strings.TrimSpace(p) != "" {
			kept = append(kept, p)
		}
	}
	return strings.Join(kept, sep)
}
