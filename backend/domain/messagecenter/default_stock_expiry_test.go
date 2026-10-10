package messagecenter_test

import (
	"html"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/adirkuhn/commerceos/domain/locale"
	mc "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
)

func TestDefaultStockExpiry_rendersEveryRequestedField(t *testing.T) {
	data := map[string]any{
		"window_days": float64(30), "min_days_left": float64(8), "item_count": float64(2),
		"items": []any{
			map[string]any{"product_name": "Pão de queijo", "sku": "PQ-1", "image_url": "https://cdn/pq.jpg", "quantity": float64(7),
				"expiry_date": "30/10/2026", "days_left": float64(19), "price": "€ 4,99", "sale_price": "€ 3,99"},
			map[string]any{"product_name": "Guaraná", "sku": "G-1", "image_url": "", "quantity": float64(3),
				"expiry_date": "01/11/2026", "days_left": float64(21), "price": "€ 1,50", "sale_price": ""},
		},
	}
	for _, lang := range []locale.Locale{locale.English, locale.Portuguese, locale.Dutch} {
		d := mc.DefaultStockExpiry(lang)
		assert.Equal(t, mc.CategoryStockInfo, d.Template.Category)
		assert.Equal(t, []messaging.Channel{messaging.ChannelEmail}, d.Template.Channels)
		assert.True(t, d.Template.Enabled)
		assert.Equal(t, mc.TriggerStockCloseToExpiry, d.Rule.TriggerSubject)
		assert.True(t, d.Rule.Enabled)

		subject := mc.Render(d.Template.Subject, data, func(s string) string { return s })
		body := mc.Render(d.Template.Body, data, html.EscapeString)

		assert.Contains(t, subject, "30", lang)
		assert.NotContains(t, subject+body, "{{", "every token resolves (%s)", lang)
		for _, want := range []string{"Pão de queijo", "PQ-1", `src="https://cdn/pq.jpg"`, ">7<", "30/10/2026", "<s>€ 4,99</s>", "€ 3,99", "Guaraná", "€ 1,50"} {
			assert.Contains(t, body, want, lang)
		}
		assert.Equal(t, 1, strings.Count(body, "<img"), "no image tag for a product without a photo (%s)", lang)
		if lang != locale.English {
			assert.NotEqual(t, mc.DefaultStockExpiry(locale.English).Template.Subject, d.Template.Subject, "translated (%s)", lang)
		}
	}
}

func TestDefaultStockExpiry_unsetLanguageIsEnglish(t *testing.T) {
	assert.Equal(t, mc.DefaultStockExpiry(locale.English), mc.DefaultStockExpiry(""))
}
