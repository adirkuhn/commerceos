package messagecenter

import (
	"strings"

	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/messaging"
)

// Keys of the built-in default templates (MessageTemplate.DefaultKey).
const (
	DefaultOrderConfirmation  = "order_confirmation"
	DefaultOrderStatus        = "order_status"
	DefaultOrderShipped       = "order_shipped"
	DefaultRefundIssued       = "refund_issued"
	DefaultRefundReview       = "refund_review"
	DefaultStockCloseToExpiry = "stock_close_to_expiry"
)

// DefaultTemplate is a ready-made template a tenant can start from and then
// customize; TriggerSubject is the trigger it is written for (its
// variables, and the rule it belongs in).
type DefaultTemplate struct {
	Key            string
	TriggerSubject string
	// Description says in one line what the message is and when it goes.
	Description string
	Template    MessageTemplate
	// Rule is the automation rule a tenant gets with the template: on,
	// email only, and for customer messages skipping marketplace orders.
	Rule AutomationRule
}

// marketplaceSources email their own customers; default customer rules
// skip their orders so nobody gets the same message twice.
var marketplaceSources = []string{"woocommerce", "bol"}

func skipMarketplaces(extra ...Condition) []Condition {
	out := make([]Condition, 0, len(marketplaceSources)+len(extra))
	for _, s := range marketplaceSources {
		out = append(out, Condition{Field: "order_source", Operator: OperatorNeq, Value: s})
	}
	return append(out, extra...)
}

func defaultRule(name, trigger string, conditions []Condition) AutomationRule {
	email := messaging.ChannelEmail
	return AutomationRule{
		Name: name, TriggerSubject: trigger, Conditions: conditions, ChannelOverride: &email,
		ExtraRecipients: []string{}, Enabled: true,
	}
}

type defaultSpec struct {
	key        string
	trigger    string
	conditions []Condition
	category   Category
	content    func(l layoutTexts) string
	texts      func(d defaultTexts) messageTexts
}

var defaultSpecs = []defaultSpec{
	{DefaultOrderConfirmation, "storefront.order.placed", skipMarketplaces(), CategoryPurchase,
		func(l layoutTexts) string { return orderItemsBlock(l) + shippingAddressBlock(l) },
		func(d defaultTexts) messageTexts { return d.orderConfirmation }},
	{DefaultOrderStatus, "oms.sales_order.status_changed", skipMarketplaces(), CategoryOrderStatus,
		orderItemsBlock,
		func(d defaultTexts) messageTexts { return d.orderStatus }},
	// MyParcel status 3: handed to the carrier -- once, not on every update.
	{DefaultOrderShipped, "shipping.shipment.fulfillment_signal", skipMarketplaces(
		Condition{Field: "kind", Operator: OperatorEq, Value: "shipment_status"},
		Condition{Field: "status_code", Operator: OperatorEq, Value: 3},
	), CategoryTracking,
		func(l layoutTexts) string { return trackingBlock(l) + shippingAddressBlock(l) + orderItemsBlock(l) },
		func(d defaultTexts) messageTexts { return d.orderShipped }},
	{DefaultRefundIssued, "payments.refund.issued", skipMarketplaces(), CategoryPurchase,
		func(layoutTexts) string { return "" },
		func(d defaultTexts) messageTexts { return d.refundIssued }},
	{DefaultRefundReview, "payments.refund.manual_review_required", skipMarketplaces(), CategoryPurchase,
		func(layoutTexts) string { return "" },
		func(d defaultTexts) messageTexts { return d.refundReview }},
}

// DefaultTemplates is every built-in default in lang (English when unset
// or unknown), in the order a gallery shows them.
func DefaultTemplates(lang locale.Locale) []DefaultTemplate {
	texts, ok := defaultTextsByLocale[lang]
	if !ok {
		texts = defaultTextsByLocale[locale.English]
	}
	out := make([]DefaultTemplate, 0, len(defaultSpecs)+1)
	for _, s := range defaultSpecs {
		m := s.texts(texts)
		out = append(out, DefaultTemplate{Key: s.key, TriggerSubject: s.trigger, Description: m.description,
			Rule: defaultRule(m.name, s.trigger, s.conditions), Template: MessageTemplate{
				Name: m.name, Category: s.category,
				Channels: []messaging.Channel{messaging.ChannelEmail, messaging.ChannelWhatsApp},
				Subject:  m.subject, Body: emailLayout(texts.layout, heading(m.heading)+paragraph(m.intro)+s.content(texts.layout)),
				TextBody: m.text, DefaultKey: s.key, Enabled: true,
			}})
	}
	stock := DefaultStockExpiry(lang)
	return append(out, DefaultTemplate{
		Key: DefaultStockCloseToExpiry, TriggerSubject: TriggerStockCloseToExpiry, Description: texts.stockDescription,
		Template: stock.Template, Rule: defaultRule(stock.Template.Name, TriggerStockCloseToExpiry, nil),
	})
}

// DefaultTemplateFor is the built-in default key in lang.
func DefaultTemplateFor(key string, lang locale.Locale) (DefaultTemplate, bool) {
	for _, d := range DefaultTemplates(lang) {
		if d.Key == key {
			return d, true
		}
	}
	return DefaultTemplate{}, false
}

// --- email building blocks: table layout + inline styles, so the defaults
// render the same in mail clients and open as editable blocks in the
// visual editor. Loops use data-each (see textrender.ExpandEach).

const (
	cellStyle  = "padding:8px 0;border-bottom:1px solid #eeeeee"
	labelStyle = "margin:0 0 4px;font-size:12px;color:#777777;text-transform:uppercase;letter-spacing:0.5px"
)

func emailLayout(l layoutTexts, content string) string {
	return `<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="background-color:#f4f4f5;padding:24px 0;font-family:Arial,Helvetica,sans-serif">
<tr><td align="center">
<table role="presentation" width="600" cellpadding="0" cellspacing="0" style="max-width:600px;width:100%;background-color:#ffffff;border-radius:8px">
<tr><td style="background-color:#1f2937;color:#ffffff;padding:20px 24px;font-size:20px;font-weight:bold;border-radius:8px 8px 0 0">{{store_name}}</td></tr>
<tr><td style="padding:24px;color:#222222;font-size:14px;line-height:1.5">
<p style="margin:0 0 12px">` + l.greeting + `</p>
` + content + `
</td></tr>
<tr><td style="padding:16px 24px;background-color:#fafafa;color:#777777;font-size:12px;text-align:center;border-radius:0 0 8px 8px">{{store_name}}{{#store_address}} · {{store_address}}{{/store_address}}<br>{{#store_email}}<a href="mailto:{{store_email}}" style="color:#777777">{{store_email}}</a>{{/store_email}}{{#store_phone}} · {{store_phone}}{{/store_phone}}</td></tr>
</table>
</td></tr>
</table>`
}

func heading(text string) string {
	return `<h1 style="margin:0 0 12px;font-size:20px;color:#111111">` + text + "</h1>\n"
}

func paragraph(text string) string {
	return `<p style="margin:0 0 16px">` + text + "</p>\n"
}

func orderItemsBlock(l layoutTexts) string {
	totalRow := func(label, value, extra string) string {
		return `<tr><td colspan="2" style="padding:4px 0;text-align:right` + extra + `">` + label + `</td><td style="padding:4px 0;text-align:right` + extra + `">` + value + "</td></tr>\n"
	}
	return `<p style="` + labelStyle + `">` + l.order + ` #{{order_number}} · {{order_date}}</p>
<table role="presentation" width="100%" cellpadding="0" cellspacing="0" style="font-size:14px;margin:0 0 16px">
<tr><th style="` + cellStyle + `;text-align:left">` + l.product + `</th><th style="` + cellStyle + `;text-align:right">` + l.quantity + `</th><th style="` + cellStyle + `;text-align:right">` + l.total + `</th></tr>
<tr data-each="items"><td style="` + cellStyle + `">{{name}}</td><td style="` + cellStyle + `;text-align:right">{{quantity}}</td><td style="` + cellStyle + `;text-align:right">{{total}}</td></tr>
` + totalRow(l.subtotal, "{{subtotal}}", "") + totalRow(l.shipping, "{{shipping_total}}", "") +
		totalRow(l.orderTotal, "{{order_total}}", ";font-weight:bold;font-size:16px") + "</table>\n"
}

func shippingAddressBlock(l layoutTexts) string {
	return `{{#shipping_address}}<p style="` + labelStyle + `">` + l.shippingTo + `</p>
<p style="margin:0 0 16px">{{shipping_name}}<br>{{shipping_address}}</p>{{/shipping_address}}
`
}

func trackingBlock(l layoutTexts) string {
	return `{{#tracking_url}}<p style="margin:0 0 16px"><a href="{{tracking_url}}" style="display:inline-block;background-color:#1f2937;color:#ffffff;text-decoration:none;padding:10px 18px;border-radius:6px;font-weight:bold">` + l.track + `</a></p>{{/tracking_url}}
{{#tracking_barcode}}<p style="margin:0 0 16px;color:#555555">{{carrier}} · {{tracking_barcode}}</p>{{/tracking_barcode}}
`
}

// joinLines keeps the WhatsApp/SMS texts readable in source.
func joinLines(lines ...string) string { return strings.Join(lines, "\n") }
