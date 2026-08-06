package ai_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/adirkuhn/commerceos/infrastructure/ai"

	procurementDomain "github.com/adirkuhn/commerceos/domain/procurement"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeNDJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/x-ndjson")
	b, _ := json.Marshal(v)
	fmt.Fprintln(w, string(b))
}

func TestOllamaDocumentParser_ExtractLineItems_success(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		writeNDJSON(w, map[string]any{
			"model": "llama3.2", "created_at": "2026-01-01T00:00:00Z", "done": true,
			"message": map[string]any{
				"role":    "assistant",
				"content": `{"line_items":[{"raw_sku":"WID-001","raw_description":"Widget A","quantity":10,"unit_cost_minor_units":500}],"supplier_name":"Acme Foods, LLC"}`,
			},
		})
	}))
	defer server.Close()

	parser := ai.NewOllamaDocumentParser(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", ExtractedText: "WID-001 Widget A qty 10 $5.00",
	})

	require.NoError(t, err)
	require.Len(t, result.LineItems, 1)
	assert.Equal(t, procurementDomain.ExtractedLineItem{
		RawSKU: "WID-001", RawDescription: "Widget A", Quantity: 10, UnitCostMinorUnits: 500,
	}, result.LineItems[0])
	assert.Equal(t, "Acme Foods, LLC", result.SupplierName)
	assert.Equal(t, "/api/chat", capturedPath)
}

func TestOllamaDocumentParser_ExtractLineItems_returnsErrorWhenNoExtractedText(t *testing.T) {
	parser := ai.NewOllamaDocumentParser("http://localhost:11434")
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no extracted text")
}

func TestOllamaDocumentParser_ExtractLineItems_returnsErrorOnAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		writeNDJSON(w, map[string]any{"error": "model \"llama3.2\" not found"})
	}))
	defer server.Close()

	parser := ai.NewOllamaDocumentParser(server.URL)
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", ExtractedText: "some text",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestOllamaDocumentParser_ExtractLineItems_returnsErrorOnMalformedLineItemsJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeNDJSON(w, map[string]any{
			"model": "llama3.2", "created_at": "2026-01-01T00:00:00Z", "done": true,
			"message": map[string]any{"role": "assistant", "content": "not json"},
		})
	}))
	defer server.Close()

	parser := ai.NewOllamaDocumentParser(server.URL)
	_, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", ExtractedText: "some text",
	})

	require.Error(t, err)
}

func TestOllamaDocumentParser_ExtractLineItems_emptyLineItemsReturnsEmptySlice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		writeNDJSON(w, map[string]any{
			"model": "llama3.2", "created_at": "2026-01-01T00:00:00Z", "done": true,
			"message": map[string]any{"role": "assistant", "content": `{"line_items":[],"supplier_name":""}`},
		})
	}))
	defer server.Close()

	parser := ai.NewOllamaDocumentParser(server.URL)
	result, err := parser.ExtractLineItems(context.Background(), procurementDomain.DocumentInput{
		FileName: "order.pdf", ContentType: "application/pdf", ExtractedText: "some text",
	})

	require.NoError(t, err)
	assert.Empty(t, result.LineItems)
	assert.Empty(t, result.SupplierName)
}
