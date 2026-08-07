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

func TestOpenAIDocumentParser_ExtractLineItems_success(t *testing.T) {
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/responses", r.URL.Path)
		require.Equal(t, "Bearer test-key", r.Header.Get("Authorization"))
		require.NoError(t, json.NewDecoder(r.Body).Decode(&capturedBody))

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{
							"type": "output_text",
							"text": `{"line_items":[{"raw_sku":"WID-001","raw_description":"Widget A","quantity":10,"unit_cost_minor_units":500}],"supplier_name":"Acme Foods, LLC"}`,
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewOpenAIDocumentParser("test-key").WithBaseURL(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.NoError(t, err)
	require.Len(t, result.LineItems, 1)
	assert.Equal(t, procurementDomain.ExtractedLineItem{
		RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500,
	}, result.LineItems[0])
	assert.Equal(t, "Acme Foods, LLC", result.SupplierName)

	assert.Equal(t, "gpt-4o", capturedBody["model"])
}

func TestOpenAIDocumentParser_ExtractLineItems_classifiesChargeRows(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{
							"type": "output_text",
							"text": `{"line_items":[
								{"raw_sku":"WID-001","raw_description":"Widget A","quantity":10,"unit_cost_minor_units":500,"line_type":"product"},
								{"raw_sku":"","raw_description":"Shipping","quantity":1,"unit_cost_minor_units":1500,"line_type":"shipping"}
							],"supplier_name":"Acme Foods"}`,
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewOpenAIDocumentParser("test-key").WithBaseURL(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.NoError(t, err)
	require.Len(t, result.LineItems, 2)
	assert.Equal(t, procurementDomain.LineItemTypeProduct, result.LineItems[0].LineType)
	assert.Equal(t, procurementDomain.LineItemTypeShipping, result.LineItems[1].LineType)
}

func TestOpenAIDocumentParser_ExtractLineItems_fallsBackToProductOnUnrecognizedLineType(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{
					"type": "message",
					"content": []map[string]any{
						{
							"type": "output_text",
							"text": `{"line_items":[{"raw_sku":"WID-001","raw_description":"Widget A","quantity":10,"unit_cost_minor_units":500,"line_type":"something_the_model_made_up"}],"supplier_name":"Acme Foods"}`,
						},
					},
				},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewOpenAIDocumentParser("test-key").WithBaseURL(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.NoError(t, err)
	require.Len(t, result.LineItems, 1)
	assert.Equal(t, procurementDomain.LineItemType(""), result.LineItems[0].LineType, "an unrecognized value must fall back to product, not be persisted verbatim")
}

func TestOpenAIDocumentParser_ExtractLineItems_returnsErrorOnAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "invalid api key"},
		})
	}))
	defer server.Close()

	parser := ai.NewOpenAIDocumentParser("bad-key").WithBaseURL(server.URL)
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid api key")
}

func TestOpenAIDocumentParser_ExtractLineItems_returnsErrorWhenNoOutputText(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"output": []map[string]any{}})
	}))
	defer server.Close()

	parser := ai.NewOpenAIDocumentParser("test-key").WithBaseURL(server.URL)
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no output_text")
}

func TestOpenAIDocumentParser_ExtractLineItems_returnsErrorOnMalformedLineItemsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{"type": "message", "content": []map[string]any{{"type": "output_text", "text": "not json"}}},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewOpenAIDocumentParser("test-key").WithBaseURL(server.URL)
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.Error(t, err)
}

func TestOpenAIDocumentParser_ExtractLineItems_emptyLineItemsReturnsEmptySlice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"output": []map[string]any{
				{"type": "message", "content": []map[string]any{{"type": "output_text", "text": `{"line_items":[]}`}}},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewOpenAIDocumentParser("test-key").WithBaseURL(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.NoError(t, err)
	assert.Empty(t, result.LineItems)
	assert.Empty(t, result.SupplierName, "no supplier_name field in the response means empty, not an error")
}
