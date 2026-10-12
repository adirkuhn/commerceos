package messagecenter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	customersDomain "github.com/adirkuhn/commerceos/domain/customers"
	"github.com/adirkuhn/commerceos/domain/locale"
	omsDomain "github.com/adirkuhn/commerceos/domain/oms"
	settingsDomain "github.com/adirkuhn/commerceos/domain/settings"
	shippingDomain "github.com/adirkuhn/commerceos/domain/shipping"
)

func TestStoreVars(t *testing.T) {
	vars := storeVars(&settingsDomain.CompanyProfile{
		LegalName: "Cantinho BV", TradingName: "Cantinho Brasileiro", Email: "hi@c.nl", Phone: "+31 20 1",
		Address: settingsDomain.CompanyAddress{Street: "Damrak", HouseNumber: "1", PostalCode: "1012 LG", City: "Amsterdam", Country: "NL"},
	})
	assert.Equal(t, map[string]any{
		"store_name": "Cantinho Brasileiro", "store_email": "hi@c.nl", "store_phone": "+31 20 1",
		"store_address": "Damrak 1, 1012 LG Amsterdam, NL",
	}, vars)

	assert.Equal(t, "Cantinho BV", storeVars(&settingsDomain.CompanyProfile{LegalName: "Cantinho BV"})["store_name"], "legal name when no trading name")
	assert.Equal(t, "", storeVars(nil)["store_name"])
}

func sampleLoadedOrder() *loadedOrder {
	return &loadedOrder{
		order: &omsDomain.SalesOrder{
			Number: "1001", Status: omsDomain.SalesOrderStatusShipped, Source: omsDomain.SalesOrderSourceWooCommerce, CurrencyCode: "EUR",
			OccurredAt:              time.Date(2026, 10, 9, 15, 0, 0, 0, time.UTC),
			ShippingTotalMinorUnits: 495, DiscountTotalMinorUnits: 100, TaxTotalMinorUnits: 173, TotalMinorUnits: 1393,
			Billing:             omsDomain.Address{FirstName: "Bia", LastName: "Lima", Email: "bia@x.nl"},
			Shipping:            omsDomain.Address{FirstName: "Ana", LastName: "Souza", Address1: "Kalverstraat 10", Postcode: "1012 PA", City: "Amsterdam", Country: "NL"},
			ShippingMethodTitle: "PostNL standaard", PaymentMethodTitle: "iDEAL", CustomerNote: "ring twice",
		},
		customer: &customersDomain.Customer{FirstName: "Ana", LastName: "Souza", Email: "ana@x.nl"},
		lines: []omsDomain.SalesOrderLineItem{
			{Name: "Pão de queijo", SKUCode: "PQ-1", Quantity: 2, PriceMinorUnits: 499, TotalMinorUnits: 998},
		},
		shipment:    &shippingDomain.Shipment{Carrier: shippingDomain.CarrierPostNL, Barcode: "3S1"},
		trackingURL: "https://t/3S1",
	}
}

func TestOrderVars(t *testing.T) {
	vars := orderVars(sampleLoadedOrder(), locale.Dutch, settingsDomain.DateFormatDayMonthYear)

	assert.Equal(t, "Ana", vars["customer_name"])
	assert.Equal(t, "Souza", vars["customer_last_name"])
	assert.Equal(t, "ana@x.nl", vars["customer_email"])
	assert.Equal(t, "1001", vars["order_number"])
	assert.Equal(t, "09/10/2026", vars["order_date"])
	assert.Equal(t, "shipped", vars["order_status"])
	assert.Equal(t, "woocommerce", vars["order_source"], "rules can skip marketplace orders")
	assert.Equal(t, "Verzonden", vars["order_status_label"])
	assert.Equal(t, []any{map[string]any{"name": "Pão de queijo", "sku": "PQ-1", "quantity": int64(2), "unit_price": "€ 4,99", "total": "€ 9,98"}}, vars["items"])
	assert.Equal(t, 1, vars["item_count"])
	assert.Equal(t, "€ 9,98", vars["subtotal"])
	assert.Equal(t, "€ 4,95", vars["shipping_total"])
	assert.Equal(t, "€ 1,00", vars["discount_total"])
	assert.Equal(t, "€ 1,73", vars["tax_total"])
	assert.Equal(t, "€ 13,93", vars["order_total"])
	assert.Equal(t, "Ana Souza", vars["shipping_name"])
	assert.Equal(t, "Kalverstraat 10, 1012 PA Amsterdam, NL", vars["shipping_address"])
	assert.Equal(t, "PostNL standaard", vars["shipping_method"])
	assert.Equal(t, "iDEAL", vars["payment_method"])
	assert.Equal(t, "ring twice", vars["customer_note"])
	assert.Equal(t, "PostNL", vars["carrier"])
	assert.Equal(t, "3S1", vars["tracking_barcode"])
	assert.Equal(t, "https://t/3S1", vars["tracking_url"])
}

func TestOrderVars_guestUsesBillingContact(t *testing.T) {
	lo := sampleLoadedOrder()
	lo.customer = nil
	vars := orderVars(lo, locale.English, settingsDomain.DateFormatISO)

	assert.Equal(t, "Bia", vars["customer_name"])
	assert.Equal(t, "bia@x.nl", vars["customer_email"])
	assert.Equal(t, "2026-10-09", vars["order_date"])
}

func TestOrderVars_noShipmentLeavesTrackingEmpty(t *testing.T) {
	lo := sampleLoadedOrder()
	lo.shipment, lo.trackingURL = nil, ""
	vars := orderVars(lo, locale.English, "")

	assert.Equal(t, "", vars["carrier"])
	assert.Equal(t, "", vars["tracking_barcode"])
	assert.Equal(t, "", vars["tracking_url"])
}

func TestEventVars_refundAmountInOrderCurrency(t *testing.T) {
	vars := eventVars(map[string]any{"amount_minor_units": float64(499)}, sampleLoadedOrder(), locale.English)
	assert.Equal(t, "€4.99", vars["refund_amount"])
	assert.Empty(t, eventVars(map[string]any{}, nil, locale.English))
}
