package messagecenter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	mc "github.com/adirkuhn/commerceos/domain/messagecenter"
)

func TestTriggers_coverEveryKnownSubjectOnce(t *testing.T) {
	seen := map[string]bool{}
	for _, tr := range mc.Triggers {
		assert.False(t, seen[tr.Subject], "duplicate %s", tr.Subject)
		seen[tr.Subject] = true
	}
	assert.ElementsMatch(t, mc.KnownTriggerSubjects(), keys(seen))
	assert.True(t, mc.ValidTriggerSubject("oms.sales_order.created"), "orders from every source can trigger")
	assert.False(t, mc.ValidTriggerSubject("nope"))
}

func TestTriggerFor_orderTriggersNameWhereTheOrderIs(t *testing.T) {
	placed, ok := mc.TriggerFor("storefront.order.placed")
	require.True(t, ok)
	assert.Equal(t, "sales_order_id", placed.OrderIDField)
	assert.Equal(t, mc.AudienceCustomer, placed.Audience.Kind)

	created, _ := mc.TriggerFor("oms.sales_order.created")
	assert.Equal(t, "entity_id", created.OrderIDField, "an EntityAction names its order in entity_id")

	expiry, _ := mc.TriggerFor(mc.TriggerStockCloseToExpiry)
	assert.Empty(t, expiry.OrderIDField)
	assert.Equal(t, mc.AudienceStaff, expiry.Audience.Kind)
}

func TestTrigger_variablesIncludeStoreAndOrderData(t *testing.T) {
	tr, _ := mc.TriggerFor("oms.sales_order.status_changed")
	names := variableNames(tr.Variables())
	for _, want := range []string{"store_name", "customer_name", "order_number", "order_status_label", "items", "order_total", "tracking_url"} {
		assert.Contains(t, names, want)
	}
	items := variableNamed(tr.Variables(), "items")
	assert.Equal(t, mc.VariableList, items.Type)
	assert.Contains(t, variableNames(items.Fields), "quantity")

	product, _ := mc.TriggerFor("pms.product.changed")
	assert.Contains(t, variableNames(product.Variables()), "store_name")
	assert.NotContains(t, variableNames(product.Variables()), "order_number", "no order behind a product change")
}

func TestTrigger_SampleDataFillsEveryVariable(t *testing.T) {
	for _, tr := range mc.Triggers {
		data := tr.SampleData()
		for _, v := range tr.Variables() {
			assert.Contains(t, data, v.Name, "%s sample lacks %s", tr.Subject, v.Name)
		}
	}
	tr, _ := mc.TriggerFor("storefront.order.placed")
	items := tr.SampleData()["items"].([]any)
	require.NotEmpty(t, items)
	assert.Contains(t, items[0].(map[string]any), "name")
}

func keys(m map[string]bool) []string {
	out := []string{}
	for k := range m {
		out = append(out, k)
	}
	return out
}

func variableNames(vs []mc.Variable) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = v.Name
	}
	return out
}

func variableNamed(vs []mc.Variable, name string) mc.Variable {
	for _, v := range vs {
		if v.Name == name {
			return v
		}
	}
	return mc.Variable{}
}
