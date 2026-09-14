package messagecenter_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	appmc "github.com/adirkuhn/commerceos/application/messagecenter"
)

func TestRenderTemplate_substitutesKnownTokens(t *testing.T) {
	out := appmc.RenderTemplate("Hi {{customer_name}}, order {{sales_order_id}} update", map[string]string{
		"customer_name": "Jane", "sales_order_id": "order-1",
	})
	assert.Equal(t, "Hi Jane, order order-1 update", out)
}

func TestRenderTemplate_leavesUnknownTokenUnresolved(t *testing.T) {
	out := appmc.RenderTemplate("Hi {{customer_name}}, {{typo_token}}", map[string]string{"customer_name": "Jane"})
	assert.Equal(t, "Hi Jane, {{typo_token}}", out)
}
