package assistanttools

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestOperationsTools_wiresEveryOperationsTool(t *testing.T) {
	// Construction only wires adapters -- no query runs.
	var names []string
	for _, tool := range operationsTools(&gorm.DB{}) {
		names = append(names, tool.Definition().Name)
	}

	assert.ElementsMatch(t, []string{
		"search_suppliers", "get_supplier", "list_purchase_orders", "get_purchase_order",
		"list_batches", "list_expiring_stock", "list_integration_connections", "list_sync_logs",
	}, names)
}
