package messagecenter_test

import (
	"html"
	"regexp"
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/locale"
	mc "github.com/adirkuhn/commerceos/domain/messagecenter"
	"github.com/adirkuhn/commerceos/domain/messaging"
)

var allLocales = []locale.Locale{locale.English, locale.Portuguese, locale.Dutch}

func TestDefaultTemplates_catalog(t *testing.T) {
	var keys []string
	for _, d := range mc.DefaultTemplates(locale.English) {
		keys = append(keys, d.Key)
	}
	assert.Equal(t, []string{
		mc.DefaultOrderConfirmation, mc.DefaultOrderStatus, mc.DefaultOrderShipped,
		mc.DefaultRefundIssued, mc.DefaultRefundReview, mc.DefaultStockCloseToExpiry,
	}, keys)
}

func TestDefaultTemplates_everyDefaultIsAValidTemplateInEveryLanguage(t *testing.T) {
	for _, lang := range allLocales {
		for _, d := range mc.DefaultTemplates(lang) {
			assert.NotEmpty(t, d.Template.Name, "%s %s", d.Key, lang)
			assert.True(t, d.Template.Category.Valid(), "%s %s", d.Key, lang)
			assert.True(t, mc.ValidTriggerSubject(d.TriggerSubject), "%s %s", d.Key, lang)
			assert.Equal(t, d.Key, d.Template.DefaultKey)
			assert.True(t, d.Template.Enabled)
			assert.NoError(t, mc.ValidateContent(d.Template.Channels, d.Template.Body, d.Template.TextBody), "%s %s", d.Key, lang)
			if slices.Contains(d.Template.Channels, messaging.ChannelEmail) {
				assert.NotEmpty(t, d.Template.Subject, "%s %s", d.Key, lang)
			}
			if lang != locale.English {
				en, _ := mc.DefaultTemplateFor(d.Key, locale.English)
				assert.NotEqual(t, en.Template.Subject, d.Template.Subject, "translated: %s %s", d.Key, lang)
			}
		}
	}
}

var tokenPattern = regexp.MustCompile(`\{\{[#^/]?\s*([a-z_]+)\s*\}\}|data-each="([a-z_]+)"`)

// Every token a default uses must be a variable its trigger offers, and
// rendering with the trigger's sample data must leave nothing unresolved.
func TestDefaultTemplates_useOnlyTheirTriggersVariables(t *testing.T) {
	for _, lang := range allLocales {
		for _, d := range mc.DefaultTemplates(lang) {
			trigger, ok := mc.TriggerFor(d.TriggerSubject)
			require.True(t, ok, d.Key)
			known := map[string]bool{}
			for _, v := range trigger.Variables() {
				known[v.Name] = true
				for _, f := range v.Fields {
					known[f.Name] = true
				}
			}
			all := d.Template.Subject + d.Template.Body + d.Template.TextBody
			for _, m := range tokenPattern.FindAllStringSubmatch(all, -1) {
				name := m[1] + m[2]
				assert.True(t, known[name], "%s (%s) uses unknown variable %q", d.Key, lang, name)
			}
			data := trigger.SampleData()
			rendered := mc.Render(d.Template.Subject, data, func(s string) string { return s }) +
				mc.Render(d.Template.Body, data, html.EscapeString) +
				mc.Render(d.Template.TextBody, data, func(s string) string { return s })
			assert.NotContains(t, rendered, "{{", "%s (%s)", d.Key, lang)
			assert.NotContains(t, rendered, "data-each", "%s (%s)", d.Key, lang)
		}
	}
}

func TestDefaultTemplates_orderEmailsShowTheOrder(t *testing.T) {
	d, ok := mc.DefaultTemplateFor(mc.DefaultOrderConfirmation, locale.Dutch)
	require.True(t, ok)
	trigger, _ := mc.TriggerFor(d.TriggerSubject)

	body := mc.Render(d.Template.Body, trigger.SampleData(), html.EscapeString)

	for _, want := range []string{"Cantinho Brasileiro", "1001", "Pão de queijo 400g", "€ 14,93", "Kalverstraat 10"} {
		assert.Contains(t, body, want)
	}
}

func TestDefaultTemplateFor_unknownKey(t *testing.T) {
	_, ok := mc.DefaultTemplateFor("nope", locale.English)
	assert.False(t, ok)
}

func TestDefaultTemplates_unsetLanguageIsEnglish(t *testing.T) {
	assert.Equal(t, mc.DefaultTemplates(locale.English), mc.DefaultTemplates(""))
}

func TestDefaultStockExpiry_isTheCatalogDefault(t *testing.T) {
	d, ok := mc.DefaultTemplateFor(mc.DefaultStockCloseToExpiry, locale.Portuguese)
	require.True(t, ok)
	assert.Equal(t, d.Template, mc.DefaultStockExpiry(locale.Portuguese).Template)
}

func TestDefaultTemplates_eachHasADescriptionAndAnEnabledEmailRule(t *testing.T) {
	for _, lang := range allLocales {
		for _, d := range mc.DefaultTemplates(lang) {
			assert.NotEmpty(t, d.Description, "%s %s", d.Key, lang)
			assert.Equal(t, d.TriggerSubject, d.Rule.TriggerSubject, d.Key)
			assert.Equal(t, d.Template.Name, d.Rule.Name, d.Key)
			assert.True(t, d.Rule.Enabled, d.Key)
			require.NotNil(t, d.Rule.ChannelOverride, d.Key)
			assert.Equal(t, messaging.ChannelEmail, *d.Rule.ChannelOverride, "seeded rules send email only (%s)", d.Key)
			for _, c := range d.Rule.Conditions {
				assert.True(t, c.Field != "" && c.Operator.Valid(), d.Key)
			}
		}
	}
}

// Marketplaces email their own customers: default customer rules skip
// their orders so nobody gets two emails.
func TestDefaultTemplates_customerRulesSkipMarketplaceOrders(t *testing.T) {
	for _, d := range mc.DefaultTemplates(locale.English) {
		if mc.AudienceFor(d.TriggerSubject).Kind != mc.AudienceCustomer {
			assert.Empty(t, d.Rule.Conditions, d.Key)
			continue
		}
		trigger, _ := mc.TriggerFor(d.TriggerSubject)
		data := trigger.SampleData()
		data["status_code"] = float64(3)
		for source, want := range map[string]bool{"internal": true, "pos": true, "woocommerce": false, "bol": false} {
			data["order_source"] = source
			assert.Equal(t, want, d.Rule.Matches(data), "%s from %s", d.Key, source)
		}
	}
}

func TestDefaultTemplates_shippedFiresOnlyWhenTheCarrierHasIt(t *testing.T) {
	d, _ := mc.DefaultTemplateFor(mc.DefaultOrderShipped, locale.English)
	data := map[string]any{"order_source": "internal", "kind": "shipment_status", "status_code": float64(3)}
	assert.True(t, d.Rule.Matches(data))
	data["status_code"] = float64(7)
	assert.False(t, d.Rule.Matches(data), "not on every later status update")
}
