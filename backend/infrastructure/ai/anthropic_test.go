package ai_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adirkuhn/commerceos/infrastructure/ai"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAnthropicDocumentParser_ExtractLineItems_success(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/v1/messages", r.URL.Path)
		require.Equal(t, "test-key", r.Header.Get("X-Api-Key"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
			"content": []map[string]any{
				{
					"type": "tool_use", "id": "toolu_1", "name": "extract_line_items",
					"input": map[string]any{
						"line_items": []map[string]any{
							{"raw_sku": "WID-001", "raw_description": "Widget A", "quantity": 10, "unit_cost_minor_units": 500},
						},
						"supplier_name": "Acme Foods, LLC",
					},
				},
			},
			"stop_reason": "tool_use",
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	parser := ai.NewAnthropicDocumentParser("test-key").WithBaseURL(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.NoError(t, err)
	require.Len(t, result.LineItems, 1)
	assert.Equal(t, procurementDomain.ExtractedLineItem{
		RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500,
	}, result.LineItems[0])
	assert.Equal(t, "Acme Foods, LLC", result.SupplierName)
	assert.Equal(t, "claude-sonnet-5", capturedBody["model"])
}

func TestAnthropicDocumentParser_ExtractLineItems_extractsBatchCodeAndExpiryDate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
			"content": []map[string]any{
				{
					"type": "tool_use", "id": "toolu_1", "name": "extract_line_items",
					"input": map[string]any{
						"line_items": []map[string]any{
							{
								"raw_sku": "PT26004", "raw_description": "Farofa Premium", "quantity": 36,
								"unit_cost_minor_units": 179, "batch_code": "PT26004", "expiry_date": "2027-03-01",
							},
						},
						"supplier_name": "Bom Gosto",
					},
				},
			},
			"stop_reason": "tool_use",
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	parser := ai.NewAnthropicDocumentParser("test-key").WithBaseURL(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "invoice.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.NoError(t, err)
	require.Len(t, result.LineItems, 1)
	assert.Equal(t, "PT26004", result.LineItems[0].BatchCode)
	require.NotNil(t, result.LineItems[0].ExpiryDate)
	assert.Equal(t, "2027-03-01", result.LineItems[0].ExpiryDate.Format("2006-01-02"))
}

func TestAnthropicDocumentParser_ExtractLineItems_returnsErrorOnAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"type": "error", "error": map[string]any{"type": "authentication_error", "message": "invalid x-api-key"},
		})
	}))
	defer server.Close()

	parser := ai.NewAnthropicDocumentParser("bad-key").WithBaseURL(server.URL)
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid x-api-key")
}

func TestAnthropicDocumentParser_ExtractLineItems_returnsErrorWhenNoToolUseBlock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
			"content":     []map[string]any{{"type": "text", "text": "I couldn't extract anything."}},
			"stop_reason": "end_turn",
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	parser := ai.NewAnthropicDocumentParser("test-key").WithBaseURL(server.URL)
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no extract_line_items tool_use block")
}

func TestAnthropicDocumentParser_ExtractLineItems_returnsErrorOnMalformedLineItemsInput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
			"content": []map[string]any{
				{"type": "tool_use", "id": "toolu_1", "name": "extract_line_items", "input": map[string]any{"line_items": "not an array"}},
			},
			"stop_reason": "tool_use",
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	parser := ai.NewAnthropicDocumentParser("test-key").WithBaseURL(server.URL)
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.Error(t, err)
}

func TestAnthropicDocumentParser_ExtractLineItems_emptyLineItemsReturnsEmptySlice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-sonnet-5",
			"content": []map[string]any{
				{"type": "tool_use", "id": "toolu_1", "name": "extract_line_items", "input": map[string]any{"line_items": []map[string]any{}, "supplier_name": ""}},
			},
			"stop_reason": "tool_use",
			"usage":       map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	}))
	defer server.Close()

	parser := ai.NewAnthropicDocumentParser("test-key").WithBaseURL(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.NoError(t, err)
	assert.Empty(t, result.LineItems)
	assert.Empty(t, result.SupplierName)
}
