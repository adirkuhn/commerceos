package woocommerce

import (
	"context"
	"encoding/json"
	"errors"
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

// TestUpdateProductStock_neverTouchesSalePrice is a regression test for a
// live bug: UpdateProductStock never sets WooProduct.SalePrice, so its
// zero value ("") used to get marshaled as an explicit "sale_price":"" on
// every stock-only push once go-woocommerce-api's SalePrice field lost its
// omitempty (fixed for a *different* bug -- clearing a sale price wasn't
// reaching Woo at all). Every recount silently wiped the product's real
// sale price on Woo's side. Fixed by making SalePrice a pointer: nil
// (never set here) is omitted from the wire; only an explicit push that
// means to convey price (UpdateProduct/UpdateVariation) sends a non-nil
// pointer.
func TestUpdateProductStock_neverTouchesSalePrice(t *testing.T) {
	var raw map[string]json.RawMessage
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
		json.NewEncoder(w).Encode(wc.Product{ID: 555})
	})

	err := c.UpdateProductStock(context.Background(), "", "555", 0, nil)

	require.NoError(t, err)
	_, present := raw["sale_price"]
	require.False(t, present, "stock-only push must never send sale_price -- would clear the real sale price on Woo's side")
}

// TestUpdateProductStock_variation_neverTouchesSalePrice is the variation
// counterpart of TestUpdateProductStock_neverTouchesSalePrice.
func TestUpdateProductStock_variation_neverTouchesSalePrice(t *testing.T) {
	var raw map[string]json.RawMessage
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
		json.NewEncoder(w).Encode(wc.ProductVariation{ID: 901})
	})

	err := c.UpdateProductStock(context.Background(), "900", "901", 0, nil)

	require.NoError(t, err)
	_, present := raw["sale_price"]
	require.False(t, present, "stock-only push must never send sale_price -- would clear the real sale price on Woo's side")
}

// TestUpdateProduct_alwaysSendsSalePriceExplicitly proves the field-fields
// sync path (unlike UpdateProductStock) always sends sale_price on the
// wire, even when clearing it to "" -- this is the path that owns
// conveying the product's current sale price state to Woo.
// TestCreateProduct_400Failure_wrapsAsErrWooDuplicateSKU proves any 400
// from a create gets translated to integrationsDomain.ErrWooDuplicateSKU,
// not left as an opaque wrapped HTTP error -- this is what lets
// PushProductToWooUseCase's self-heal path (search by SKU, link) actually
// get a chance to run. See translateDuplicateSKUError's own doc for why
// this gates on HTTP status alone rather than Woo's REST error code: two
// attempts at matching a specific code both kept missing in production
// because the real code was never directly observable from the logged
// error, and self-heal's own miss-is-harmless fallback makes the broader
// gate safe.
func TestCreateProduct_400Failure_wrapsAsErrWooDuplicateSKU(t *testing.T) {
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"code": "product_invalid_sku", "message": "SKU already exists", "data": map[string]any{"status": 400},
		})
	})

	_, err := c.CreateProduct(context.Background(), integrationsDomain.WooProduct{Name: "T-Shirt", SKU: "TSHIRT-1"})

	require.Error(t, err)
	require.ErrorIs(t, err, integrationsDomain.ErrWooDuplicateSKU)
}

// TestCreateProduct_nonBadRequestFailure_doesNotBecomeErrWooDuplicateSKU
// proves the translation is scoped to HTTP 400 -- a different status (e.g.
// a 500 from the store itself) must not be misidentified as self-healable.
func TestCreateProduct_nonBadRequestFailure_doesNotBecomeErrWooDuplicateSKU(t *testing.T) {
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"code": "internal_server_error", "message": "something else is wrong", "data": map[string]any{"status": 500},
		})
	})

	_, err := c.CreateProduct(context.Background(), integrationsDomain.WooProduct{Name: "T-Shirt", SKU: "TSHIRT-1"})

	require.Error(t, err)
	require.False(t, errors.Is(err, integrationsDomain.ErrWooDuplicateSKU))
}

func TestFindProductBySKU_returnsTheMatch(t *testing.T) {
	var gotQuery string
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("sku")
		json.NewEncoder(w).Encode([]wc.Product{{ID: 42, Sku: "TSHIRT-1"}})
	})

	found, err := c.FindProductBySKU(context.Background(), "TSHIRT-1")

	require.NoError(t, err)
	require.Equal(t, "TSHIRT-1", gotQuery)
	require.NotNil(t, found)
	require.Equal(t, "42", found.ID)
	require.Equal(t, "TSHIRT-1", found.SKU)
}

func TestFindProductBySKU_returnsNilNotErrorWhenNothingMatches(t *testing.T) {
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]wc.Product{})
	})

	found, err := c.FindProductBySKU(context.Background(), "NO-SUCH-SKU")

	require.NoError(t, err)
	require.Nil(t, found)
}

func TestUpdateProduct_alwaysSendsSalePriceExplicitly(t *testing.T) {
	var raw map[string]json.RawMessage
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
		json.NewEncoder(w).Encode(wc.Product{ID: 20})
	})

	_, err := c.UpdateProduct(context.Background(), "20", integrationsDomain.WooProduct{RegularPrice: "29.99", SalePrice: ""})

	require.NoError(t, err)
	v, present := raw["sale_price"]
	require.True(t, present, "product-fields sync must always convey sale price, including clearing it")
	require.Equal(t, `""`, string(v))
}

// TestUpdateProduct_sendsSKU is a regression test for a live bug:
// UpdateProduct never mapped WooProduct.SKU onto the wc.Product wire struct
// (CreateProduct always did), so an in-app SKU edit made after a product's
// initial push to Woo silently never reached Woo on any subsequent sync.
func TestUpdateProduct_sendsSKU(t *testing.T) {
	var raw map[string]json.RawMessage
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
		json.NewEncoder(w).Encode(wc.Product{ID: 20})
	})

	_, err := c.UpdateProduct(context.Background(), "20", integrationsDomain.WooProduct{SKU: "TSHIRT-1-NEW"})

	require.NoError(t, err)
	v, present := raw["sku"]
	require.True(t, present, "SKU edit must reach Woo on update, not just on initial create")
	require.Equal(t, `"TSHIRT-1-NEW"`, string(v))
}

// TestUpdateVariation_sendsSKU is the variation counterpart of
// TestUpdateProduct_sendsSKU -- same bug, same fix, on the
// ProductVariations.Update wire call.
func TestUpdateVariation_sendsSKU(t *testing.T) {
	var raw map[string]json.RawMessage
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
		json.NewEncoder(w).Encode(wc.ProductVariation{ID: 901})
	})

	_, err := c.UpdateVariation(context.Background(), "900", "901", integrationsDomain.WooVariation{SKU: "CARNE-378-NEW"})

	require.NoError(t, err)
	v, present := raw["sku"]
	require.True(t, present, "SKU edit must reach Woo on variation update, not just on initial create")
	require.Equal(t, `"CARNE-378-NEW"`, string(v))
}

// TestUpdateOrderStatus_sendsOnlyStatusToTheOrdersEndpoint proves the
// event-driven Woo status push (see application/channels.
// PushSalesOrderStatusToChannelsUseCase) hits PUT /orders/{id} with only
// the status field -- Woo's partial-update semantics mean nothing else
// about the order gets touched by this call.
func TestUpdateOrderStatus_sendsOnlyStatusToTheOrdersEndpoint(t *testing.T) {
	var raw map[string]json.RawMessage
	var gotPath string
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		require.NoError(t, json.NewDecoder(r.Body).Decode(&raw))
		json.NewEncoder(w).Encode(wc.Order{ID: 501, Status: "on-hold"})
	})

	err := c.UpdateOrderStatus(context.Background(), "501", "on-hold")

	require.NoError(t, err)
	require.Contains(t, gotPath, "/orders/501")
	v, present := raw["status"]
	require.True(t, present)
	require.Equal(t, `"on-hold"`, string(v))
	_, priceSent := raw["total"]
	require.False(t, priceSent, "only status is sent -- nothing else about the order")
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

func TestCreateWebhook_sendsTopicDeliveryURLAndSecret(t *testing.T) {
	var body wc.Webhook
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodPost, r.Method)
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		json.NewEncoder(w).Encode(wc.Webhook{ID: 7, Topic: body.Topic, DeliveryURL: body.DeliveryURL, Status: "active"})
	})

	result, err := c.CreateWebhook(context.Background(), "order.updated", "https://commerceos.example.com/webhooks/v1/tenant/woocommerce", "shh")

	require.NoError(t, err)
	require.Equal(t, "order.updated", body.Topic)
	require.Equal(t, "https://commerceos.example.com/webhooks/v1/tenant/woocommerce", body.DeliveryURL)
	require.Equal(t, "shh", body.Secret)
	require.Equal(t, integrationsDomain.WooWebhook{
		ID: "7", Topic: "order.updated", DeliveryURL: "https://commerceos.example.com/webhooks/v1/tenant/woocommerce", Status: "active",
	}, result)
}

func TestListWebhooks_parsesEveryTopic(t *testing.T) {
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, http.MethodGet, r.Method)
		json.NewEncoder(w).Encode([]wc.Webhook{
			{ID: 1, Topic: "order.updated", DeliveryURL: "https://a.example.com", Status: "active"},
			{ID: 2, Topic: "product.updated", DeliveryURL: "https://b.example.com", Status: "disabled"},
		})
	})

	result, err := c.ListWebhooks(context.Background())

	require.NoError(t, err)
	require.Len(t, result, 2)
	require.Equal(t, integrationsDomain.WooWebhook{ID: "1", Topic: "order.updated", DeliveryURL: "https://a.example.com", Status: "active"}, result[0])
	require.Equal(t, integrationsDomain.WooWebhook{ID: "2", Topic: "product.updated", DeliveryURL: "https://b.example.com", Status: "disabled"}, result[1])
}

func TestDeleteWebhook_deletesByID(t *testing.T) {
	var gotMethod, gotPath string
	c := newTestWooClient(t, func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		json.NewEncoder(w).Encode(wc.Webhook{ID: 7})
	})

	err := c.DeleteWebhook(context.Background(), "7")

	require.NoError(t, err)
	require.Equal(t, http.MethodDelete, gotMethod)
	require.Equal(t, "/wp-json/wc/v3/webhooks/7", gotPath)
}
