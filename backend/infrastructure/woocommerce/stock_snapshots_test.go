package woocommerce

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListProductAndVariationStock_mapStockAndMetaAsStrings(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/wp-json/wc/v3/products":
			assert.Equal(t, "2", r.URL.Query().Get("page"))
			assert.Equal(t, "100", r.URL.Query().Get("per_page"))
			_, _ = w.Write([]byte(`[{"id":23981,"type":"simple","manage_stock":true,"stock_quantity":7,
				"meta_data":[{"id":1,"key":"lot_number","value":"L42"},{"id":2,"key":"expiry_date","value":"2027-01-31"},{"id":3,"key":"_n","value":3}]},
				{"id":500,"type":"variable","manage_stock":false,"stock_quantity":null,"meta_data":[]}]`))
		case "/wp-json/wc/v3/products/500/variations":
			_, _ = w.Write([]byte(`[{"id":501,"manage_stock":true,"stock_quantity":0,"meta_data":[{"key":"lot_number","value":""}]}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	client, err := (&Factory{}).NewClient(server.URL, "ck", "cs")
	require.NoError(t, err)

	products, err := client.ListProductStock(context.Background(), 2, 100)
	require.NoError(t, err)
	require.Len(t, products, 2)
	assert.Equal(t, "23981", products[0].ID)
	assert.Equal(t, "simple", products[0].Type)
	assert.True(t, products[0].ManageStock)
	require.NotNil(t, products[0].StockQuantity)
	assert.Equal(t, 7, *products[0].StockQuantity)
	assert.Equal(t, map[string]string{"lot_number": "L42", "expiry_date": "2027-01-31", "_n": "3"}, products[0].Meta)
	assert.Nil(t, products[1].StockQuantity)

	variations, err := client.ListVariationStock(context.Background(), "500", 1, 100)
	require.NoError(t, err)
	require.Len(t, variations, 1)
	assert.Equal(t, "501", variations[0].ID)
	assert.Equal(t, 0, *variations[0].StockQuantity)
	assert.Equal(t, "", variations[0].Meta["lot_number"])
}
