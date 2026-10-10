package messagecenter

import (
	"github.com/adirkuhn/commerceos/domain/locale"
	"github.com/adirkuhn/commerceos/domain/messaging"
)

// DefaultMessage is a ready-made template + rule a tenant starts with and
// then edits freely; Rule.TemplateID is filled in once the template exists.
type DefaultMessage struct {
	Template MessageTemplate
	Rule     AutomationRule
}

// DefaultStockExpiry is the default "stock close to expiry" email (one per
// report window, see TriggerStockCloseToExpiry) in lang: a table with each
// product's photo, name, SKU, quantity, expiry date and price -- the
// regular price struck through when a sale runs.
func DefaultStockExpiry(lang locale.Locale) DefaultMessage {
	t, ok := stockExpiryTexts[lang]
	if !ok {
		t = stockExpiryTexts[locale.English]
	}
	return DefaultMessage{
		Template: MessageTemplate{
			Name: t.templateName, Category: CategoryStockInfo, Channels: []messaging.Channel{messaging.ChannelEmail},
			Subject: t.subject, Body: t.body, Enabled: true,
		},
		Rule: AutomationRule{Name: t.ruleName, TriggerSubject: TriggerStockCloseToExpiry, Enabled: true},
	}
}

type stockExpiryText struct {
	templateName, ruleName, subject, body string
}

var stockExpiryTexts = map[locale.Locale]stockExpiryText{
	locale.English: {
		templateName: "Stock close to expiry",
		ruleName:     "Stock close to expiry report",
		subject:      "{{item_count}} products expire within {{window_days}} days",
		body: `<div style="font-family:Arial,sans-serif;color:#222">
<h2 style="margin:0 0 8px">Stock close to expiry: {{window_days}} days</h2>
<p>These products have {{min_days_left}} to {{window_days}} days left before they expire.</p>
<table style="border-collapse:collapse;width:100%;font-size:14px">
<thead><tr style="background:#f5f5f5"><th style="padding:8px;text-align:left">Photo</th><th style="padding:8px;text-align:left">Product</th><th style="padding:8px;text-align:left">SKU</th><th style="padding:8px;text-align:right">Qty</th><th style="padding:8px;text-align:left">Expires</th><th style="padding:8px;text-align:right">Price</th></tr></thead>
<tbody>
{{#items}}
<tr>
<td style="padding:8px;border-bottom:1px solid #eee;width:64px">{{#image_url}}<img src="{{image_url}}" alt="" width="56" height="56" style="display:block;width:56px;height:56px;object-fit:cover;border-radius:6px">{{/image_url}}</td>
<td style="padding:8px;border-bottom:1px solid #eee">{{product_name}}</td>
<td style="padding:8px;border-bottom:1px solid #eee;font-family:monospace">{{sku}}</td>
<td style="padding:8px;border-bottom:1px solid #eee;text-align:right">{{quantity}}</td>
<td style="padding:8px;border-bottom:1px solid #eee">{{expiry_date}} ({{days_left}} days)</td>
<td style="padding:8px;border-bottom:1px solid #eee;text-align:right">{{#sale_price}}<s>{{price}}</s> <strong>{{sale_price}}</strong>{{/sale_price}}{{^sale_price}}{{price}}{{/sale_price}}</td>
</tr>
{{/items}}
</tbody>
</table>
</div>`,
	},
	locale.Portuguese: {
		templateName: "Estoque perto do vencimento",
		ruleName:     "Relatório de estoque perto do vencimento",
		subject:      "{{item_count}} produtos vencem em até {{window_days}} dias",
		body: `<div style="font-family:Arial,sans-serif;color:#222">
<h2 style="margin:0 0 8px">Estoque perto do vencimento: {{window_days}} dias</h2>
<p>Estes produtos têm de {{min_days_left}} a {{window_days}} dias até o vencimento.</p>
<table style="border-collapse:collapse;width:100%;font-size:14px">
<thead><tr style="background:#f5f5f5"><th style="padding:8px;text-align:left">Foto</th><th style="padding:8px;text-align:left">Produto</th><th style="padding:8px;text-align:left">SKU</th><th style="padding:8px;text-align:right">Qtd</th><th style="padding:8px;text-align:left">Vence em</th><th style="padding:8px;text-align:right">Preço</th></tr></thead>
<tbody>
{{#items}}
<tr>
<td style="padding:8px;border-bottom:1px solid #eee;width:64px">{{#image_url}}<img src="{{image_url}}" alt="" width="56" height="56" style="display:block;width:56px;height:56px;object-fit:cover;border-radius:6px">{{/image_url}}</td>
<td style="padding:8px;border-bottom:1px solid #eee">{{product_name}}</td>
<td style="padding:8px;border-bottom:1px solid #eee;font-family:monospace">{{sku}}</td>
<td style="padding:8px;border-bottom:1px solid #eee;text-align:right">{{quantity}}</td>
<td style="padding:8px;border-bottom:1px solid #eee">{{expiry_date}} ({{days_left}} dias)</td>
<td style="padding:8px;border-bottom:1px solid #eee;text-align:right">{{#sale_price}}<s>{{price}}</s> <strong>{{sale_price}}</strong>{{/sale_price}}{{^sale_price}}{{price}}{{/sale_price}}</td>
</tr>
{{/items}}
</tbody>
</table>
</div>`,
	},
	locale.Dutch: {
		templateName: "Voorraad bijna over datum",
		ruleName:     "Rapport voorraad bijna over datum",
		subject:      "{{item_count}} producten verlopen binnen {{window_days}} dagen",
		body: `<div style="font-family:Arial,sans-serif;color:#222">
<h2 style="margin:0 0 8px">Voorraad bijna over datum: {{window_days}} dagen</h2>
<p>Deze producten hebben nog {{min_days_left}} tot {{window_days}} dagen tot de vervaldatum.</p>
<table style="border-collapse:collapse;width:100%;font-size:14px">
<thead><tr style="background:#f5f5f5"><th style="padding:8px;text-align:left">Foto</th><th style="padding:8px;text-align:left">Product</th><th style="padding:8px;text-align:left">SKU</th><th style="padding:8px;text-align:right">Aantal</th><th style="padding:8px;text-align:left">Vervalt</th><th style="padding:8px;text-align:right">Prijs</th></tr></thead>
<tbody>
{{#items}}
<tr>
<td style="padding:8px;border-bottom:1px solid #eee;width:64px">{{#image_url}}<img src="{{image_url}}" alt="" width="56" height="56" style="display:block;width:56px;height:56px;object-fit:cover;border-radius:6px">{{/image_url}}</td>
<td style="padding:8px;border-bottom:1px solid #eee">{{product_name}}</td>
<td style="padding:8px;border-bottom:1px solid #eee;font-family:monospace">{{sku}}</td>
<td style="padding:8px;border-bottom:1px solid #eee;text-align:right">{{quantity}}</td>
<td style="padding:8px;border-bottom:1px solid #eee">{{expiry_date}} ({{days_left}} dagen)</td>
<td style="padding:8px;border-bottom:1px solid #eee;text-align:right">{{#sale_price}}<s>{{price}}</s> <strong>{{sale_price}}</strong>{{/sale_price}}{{^sale_price}}{{price}}{{/sale_price}}</td>
</tr>
{{/items}}
</tbody>
</table>
</div>`,
	},
}
