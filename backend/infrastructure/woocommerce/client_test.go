package woocommerce

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	wc "github.com/adirkuhn/go-woocommerce-api"

	integrationsDomain "github.com/adirkuhn/commerceos/domain/integrations"
)

func TestToWcImages_mapsURLAndAltInOrder(t *testing.T) {
	images := []integrationsDomain.WooProductImage{
		{URL: "https://cdn.example.com/a.jpg", Alt: "Front"},
		{URL: "https://cdn.example.com/b.jpg", Alt: "Back"},
	}

	out := toWcImages(images)

	require.Len(t, out, 2)
	require.Equal(t, "https://cdn.example.com/a.jpg", out[0].Source)
	require.Equal(t, "Front", out[0].Alt)
	require.Equal(t, "https://cdn.example.com/b.jpg", out[1].Source)
	require.Equal(t, "Back", out[1].Alt)
}

// TestToWcImages_nilForNoImages proves an empty gallery sends no images
// field at all (via omitempty), not an empty array that could clear an
// existing gallery on Woo's side for an UpdateProduct call whose media just
// hasn't loaded.
func TestToWcImages_nilForNoImages(t *testing.T) {
	require.Nil(t, toWcImages(nil))
	require.Nil(t, toWcImages([]integrationsDomain.WooProductImage{}))
}

func TestToWooProduct_mapsGTINFromGlobalUniqueID(t *testing.T) {
	result := toWooProduct(wc.Product{ID: 1, GlobalUniqueID: "0012345678905"})

	require.Equal(t, "0012345678905", result.GTIN)
}

func TestToWooVariation_mapsGTINFromGlobalUniqueID(t *testing.T) {
	result := toWooVariation(wc.ProductVariation{ID: 1, GlobalUniqueID: "0012345678905"})

	require.Equal(t, "0012345678905", result.GTIN)
}

// newTestWooClient stands up a real wooClient against an httptest server, so
// CreateProduct/UpdateProduct/CreateVariation/UpdateVariation's outbound
// field mapping can be proven against the actual JSON wire format instead
// of just the intermediate wc.Product/wc.ProductVariation struct.
func newTestWooClient(t *testing.T, handler http.HandlerFunc) *wooClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := wc.New(wc.Config{ShopURL: server.URL, ConsumerKey: "ck", ConsumerSecret: "cs"})
	require.NoError(t, err)
	return &wooClient{client: client, storeURL: server.URL, consumerKey: "ck", consumerSecret: "cs"}
}

func TestCreateProduct_sendsGTINAsGlobalUniqueID(t *testing.T) {
	var body wc.Product
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		json.NewEncoder(w).Encode(wc.Product{ID: 20, GlobalUniqueID: body.GlobalUniqueID})
	})

	result, err := c.CreateProduct(context.Background(), integrationsDomain.WooProduct{Name: "T-Shirt", GTIN: "0012345678905"})

	require.NoError(t, err)
	require.Equal(t, "0012345678905", body.GlobalUniqueID, "GTIN must reach the wire as global_unique_id")
	require.Equal(t, "0012345678905", result.GTIN)
}

// TestUpdateProductStock_simpleProduct_assertsManageStockAndHitsProductEndpoint
// proves a plain product stock push (no parent id) PUTs /products/{id} and
// always sets manage_stock=true -- without it, Woo ignores stock_quantity
// for stock_status purposes whenever manage_stock happens to be off on its
// side (e.g. a product imported from Woo rather than created by
// CommerceOS), so a recount to zero would silently never show as out of
// stock.
func TestUpdateProductStock_simpleProduct_assertsManageStockAndHitsProductEndpoint(t *testing.T) {
	var body wc.Product
	var gotPath string
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		json.NewEncoder(w).Encode(wc.Product{ID: 555})
	})

	err := c.UpdateProductStock(context.Background(), "", "555", 0, nil)

	require.NoError(t, err)
	require.Contains(t, gotPath, "/products/555")
	require.NotContains(t, gotPath, "variations")
	require.True(t, body.ManageStock)
	require.NotNil(t, body.StockQuantity)
	require.Equal(t, 0, *body.StockQuantity)
}

// TestUpdateProductStock_variation_routesThroughParentVariationsEndpoint
// proves a variation stock push (parentExternalID set) PUTs
// /products/{parent}/variations/{id} -- the standalone /products/{id}
// endpoint used for a simple product does not exist for a variation id, so
// routing it there would silently miss the real Woo resource.
func TestUpdateProductStock_variation_routesThroughParentVariationsEndpoint(t *testing.T) {
	var body wc.ProductVariation
	var gotPath string
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		json.NewEncoder(w).Encode(wc.ProductVariation{ID: 901})
	})

	err := c.UpdateProductStock(context.Background(), "900", "901", 0, nil)

	require.NoError(t, err)
	require.Contains(t, gotPath, "/products/900/variations/901")
	require.True(t, body.ManageStock)
	require.NotNil(t, body.StockQuantity)
	require.Equal(t, 0, *body.StockQuantity)
}

func TestCreateVariation_sendsGTINAsGlobalUniqueID(t *testing.T) {
	var body wc.ProductVariation
	requestCount := 0
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		if requestCount == 1 {
			// ensureAttributeOptionDeclared's parent GET, ahead of Create.
			json.NewEncoder(w).Encode(wc.Product{ID: 900, Attributes: []wc.ProductAttributes{
				{Name: "Variant", Options: []string{"367g"}},
			}})
			return
		}
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		json.NewEncoder(w).Encode(wc.ProductVariation{ID: 901, GlobalUniqueID: body.GlobalUniqueID})
	})

	result, err := c.CreateVariation(context.Background(), "900", integrationsDomain.WooVariation{
		GTIN: "0012345678905", AttributeName: "Variant", AttributeValue: "367g",
	})

	require.NoError(t, err)
	require.Equal(t, "0012345678905", body.GlobalUniqueID, "GTIN must reach the wire as global_unique_id")
	require.Equal(t, "0012345678905", result.GTIN)
}
