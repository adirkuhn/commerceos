package oms_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func decodePayload(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var m map[string]any
	require.NoError(t, json.Unmarshal(raw, &m))
	return m
}

// substitutedEvents decodes every oms.sales_order_line_item.substituted the stub recorded.
func substitutedEvents(t *testing.T, o *stubOutboxRepo) []map[string]any {
	t.Helper()
	var out []map[string]any
	for i, subject := range o.calls {
		if subject == "oms.sales_order_line_item.substituted" {
			out = append(out, decodePayload(t, o.payloads[i]))
		}
	}
	return out
}
