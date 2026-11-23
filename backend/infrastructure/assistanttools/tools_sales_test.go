package assistanttools_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"

	"github.com/adirkuhn/commerceos/infrastructure/assistanttools"
)

func TestReadTools_includesTheSalesTools(t *testing.T) {
	names := map[string]bool{}
	for _, tool := range assistanttools.ReadTools(&gorm.DB{}) {
		names[tool.Definition().Name] = true
	}

	for _, want := range []string{"get_order_shipments", "search_customers", "get_customer", "get_stock_by_warehouse"} {
		assert.True(t, names[want], "%s is wired", want)
	}
}
