// Package tools holds the assistant's read-only tools: each wraps one or
// more existing read use cases (which open their own tenant transaction)
// and maps the result to a compact JSON view for the model. Views are
// explicit structs -- never a marshalled domain struct -- so nothing the
// model shouldn't see (credentials, keys, tokens) can leak in by a new
// field. Money stays in minor units, in fields ending _minor_units.
package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	defaultLimit = 10
	maxLimit     = 20
)

// ErrInvalidArguments: the model's arguments don't match the tool's
// schema. The message goes back to the model so it can correct itself.
var ErrInvalidArguments = errors.New("invalid arguments")

// object builds a strict JSON-schema object: only the listed properties,
// with required ones named.
func object(props map[string]any, required ...string) map[string]any {
	if required == nil {
		required = []string{}
	}
	return map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             required,
		"additionalProperties": false,
	}
}

func str(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}

func enum(description string, values ...string) map[string]any {
	return map[string]any{"type": "string", "description": description, "enum": values}
}

func limitProp() map[string]any {
	return map[string]any{"type": "integer", "minimum": 1, "maximum": maxLimit,
		"description": fmt.Sprintf("How many results (default %d, max %d).", defaultLimit, maxLimit)}
}

// decode reads the model's arguments strictly: unknown fields are an
// error, and empty/null args decode to the zero value.
func decode(args json.RawMessage, v any) error {
	if len(bytes.TrimSpace(args)) == 0 || string(bytes.TrimSpace(args)) == "null" {
		return nil
	}
	dec := json.NewDecoder(bytes.NewReader(args))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidArguments, err)
	}
	return nil
}

func required(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%w: %s is required", ErrInvalidArguments, name)
	}
	return nil
}

func clampLimit(n int) int {
	if n <= 0 {
		return defaultLimit
	}
	return min(n, maxLimit)
}

func toJSON(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("encode result: %w", err)
	}
	return string(b), nil
}

// formatTime renders a time for the model; zero is empty.
func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339)
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return formatTime(*t)
}
