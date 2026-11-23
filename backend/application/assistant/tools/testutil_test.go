package tools_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/adirkuhn/commerceos/domain/assistant"
)

// assertStrictDefinition checks a tool exposes a strict object schema
// under the given name.
func assertStrictDefinition(t *testing.T, tool assistant.Tool, name string) {
	t.Helper()
	d := tool.Definition()
	assert.Equal(t, name, d.Name)
	assert.NotEmpty(t, d.Description)
	assert.Equal(t, "object", d.Parameters["type"])
	assert.Equal(t, false, d.Parameters["additionalProperties"])
	_, ok := d.Parameters["properties"].(map[string]any)
	assert.True(t, ok, "properties is an object")
	_, ok = d.Parameters["required"].([]string)
	assert.True(t, ok, "required is listed")
	_, err := json.Marshal(d.Parameters)
	require.NoError(t, err)
}

// call runs a tool and decodes its JSON result.
func call(t *testing.T, tool assistant.Tool, args string) map[string]any {
	t.Helper()
	out, err := tool.Call(context.Background(), json.RawMessage(args))
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal([]byte(out), &decoded), out)
	return decoded
}

func callErr(t *testing.T, tool assistant.Tool, args string) error {
	t.Helper()
	_, err := tool.Call(context.Background(), json.RawMessage(args))
	require.Error(t, err)
	return err
}

func ptr[T any](v T) *T { return &v }
