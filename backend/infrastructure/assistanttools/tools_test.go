package assistanttools_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/infrastructure/assistanttools"
)

func TestReadTools_everyToolHasAUniqueName(t *testing.T) {
	// Construction only wires adapters -- no query runs -- so a bare
	// *gorm.DB is enough here.
	ts := assistanttools.ReadTools(&gorm.DB{})

	seen := map[string]bool{}
	for _, tool := range ts {
		name := tool.Definition().Name
		assert.False(t, seen[name], "duplicate tool %q", name)
		seen[name] = true
	}
	// The core tools; each area's wiring test checks its own (tools_*_test.go).
	assert.Subset(t, keys(seen), []string{
		"search", "get_sales_order", "list_sales_orders", "get_quote", "list_quotes", "get_product",
		"list_sales_channels", "list_product_channel_listings", "get_tenant_settings", "get_company_profile", "list_tax_rates",
	})
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
