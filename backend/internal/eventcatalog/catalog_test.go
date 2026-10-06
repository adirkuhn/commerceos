package eventcatalog_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/internal/eventcatalog"
)

var (
	eventsDir = filepath.Join("..", "..", "domain", "events")
	docPath   = filepath.Join("..", "..", "..", "docs", "events.md")
)

func TestCollect(t *testing.T) {
	entries, err := eventcatalog.Collect(eventsDir)
	require.NoError(t, err)

	bySubject := map[string]eventcatalog.Entry{}
	for _, e := range entries {
		bySubject[e.Subject] = e
	}
	goods := bySubject["procurement.goods.received"]
	assert.Equal(t, "typed", goods.Kind)
	assert.Equal(t, "GoodsReceived", goods.GoType)
	assert.Contains(t, goods.Description, "purchase-order line item")
	assert.Equal(t, []string{"event_id", "occurred_at", "tenant_id", "sku_id", "warehouse_id", "batch_id",
		"purchase_order_line_item_id", "quantity", "landed_unit_cost_minor_units"}, goods.Fields)

	po := bySubject["procurement.purchase_order.created"]
	assert.Equal(t, "action", po.Kind)
	assert.NotEmpty(t, po.Description)

	// Not cut at "e.g." inside a sentence.
	assert.NotContains(t, bySubject["integrations.connection.connected"].Description, "(e.g.|")
	assert.True(t, len(bySubject["integrations.connection.connected"].Description) > 80)

	// Multi-line EventName bodies are found too.
	assert.Contains(t, bySubject, "payments.refund.manual_review_required")
}

// docs/events.md is generated -- regenerate with `go run ./cmd/eventcatalog`.
func TestDocIsUpToDate(t *testing.T) {
	entries, err := eventcatalog.Collect(eventsDir)
	require.NoError(t, err)
	current, err := os.ReadFile(docPath)
	require.NoError(t, err, "run `go run ./cmd/eventcatalog` in backend/")
	assert.Equal(t, eventcatalog.Markdown(entries), normalize(string(current)),
		"docs/events.md is out of date: run `go run ./cmd/eventcatalog` in backend/")
}

// Working copies may be CRLF (core.autocrlf).
func normalize(s string) string {
	out := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		if s[i] == '\r' && i+1 < len(s) && s[i+1] == '\n' {
			continue
		}
		out = append(out, s[i])
	}
	return string(out)
}
