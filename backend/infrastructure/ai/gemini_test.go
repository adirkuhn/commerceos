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

func TestGeminiDocumentParser_ExtractLineItems_success(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{
					"content": map[string]any{
						"role": "model",
						"parts": []map[string]any{
							{"text": `{"line_items":[{"raw_sku":"WID-001","raw_description":"Widget A","quantity":10,"unit_cost_minor_units":500}],"supplier_name":"Acme Foods, LLC"}`},
						},
					},
					"finishReason": "STOP",
				},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewGeminiDocumentParser("test-key").WithBaseURL(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.NoError(t, err)
	require.Len(t, result.LineItems, 1)
	assert.Equal(t, procurementDomain.ExtractedLineItem{
		RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500,
	}, result.LineItems[0])
	assert.Equal(t, "Acme Foods, LLC", result.SupplierName)
	assert.Contains(t, capturedPath, "generateContent")
}

func TestGeminiDocumentParser_ExtractLineItems_returnsErrorOnAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "API key not valid", "status": "UNAUTHENTICATED"},
		})
	}))
	defer server.Close()

	parser := ai.NewGeminiDocumentParser("bad-key").WithBaseURL(server.URL)
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "API key not valid")
}

func TestGeminiDocumentParser_ExtractLineItems_returnsErrorOnMalformedLineItemsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"role": "model", "parts": []map[string]any{{"text": "not json"}}}, "finishReason": "STOP"},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewGeminiDocumentParser("test-key").WithBaseURL(server.URL)
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.Error(t, err)
}

func TestGeminiDocumentParser_ExtractLineItems_emptyLineItemsReturnsEmptySlice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"candidates": []map[string]any{
				{"content": map[string]any{"role": "model", "parts": []map[string]any{{"text": `{"line_items":[],"supplier_name":""}`}}}, "finishReason": "STOP"},
			},
		})
	}))
	defer server.Close()

	parser := ai.NewGeminiDocumentParser("test-key").WithBaseURL(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", Data: []byte("%PDF-fake"),
	})

	require.NoError(t, err)
	assert.Empty(t, result.LineItems)
	assert.Empty(t, result.SupplierName)
}
