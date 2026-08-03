package integrations

import "context"

// WooProduct is the subset of a WooCommerce product this codebase actually
// maps to its own generic Product/Variant/SKU model — deliberately excludes
// the large tail of SEO/ads/affiliate/plugin meta_data fields real Woo
// stores carry, none of which any other channel would ever have.
type WooProduct struct {
	ID            string
	Name          string
	Slug          string
	SKU           string
	Description   string
	Status        string // woo: publish/draft/private/pending
	Type          string // woo: simple/variable/grouped/external
	RegularPrice  string
	Virtual       bool // digital/non-shippable — never has weight or dimensions
	ManageStock   bool
	StockQuantity *int
	Images        []WooProductImage
	Categories    []WooProductCategory
}

type WooProductImage struct {
	URL string
	Alt string
}

type WooProductCategory struct {
	ExternalID string
	Name       string
	Slug       string
}

// WooClient talks to one WooCommerce store's REST API on behalf of a single
// stored connection. A pure port — infrastructure implements it against the
// real WooCommerce client library, so domain/application never import any
// third-party HTTP client.
type WooClient interface {
	TestConnection(ctx context.Context) error
	ListProducts(ctx context.Context, page, perPage int) ([]WooProduct, error)
	// CreateProduct pushes a new product to the store. p.ID is ignored (Woo
	// assigns it); the returned WooProduct's ID is the new external id.
	CreateProduct(ctx context.Context, p WooProduct) (*WooProduct, error)
	// UpdateProductStock sets the stock quantity on an existing product,
	// identified by its external (Woo) id. Used by the stock-export sync
	// job — never touches any other product field.
	UpdateProductStock(ctx context.Context, externalProductID string, quantity int) error
}

// WooClientFactory builds a WooClient scoped to one store's credentials.
type WooClientFactory interface {
	NewClient(storeURL, consumerKey, consumerSecret string) (WooClient, error)
}
