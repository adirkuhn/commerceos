package integrations

import (
	"context"
	"io"
)

// WooProduct is the subset of a WooCommerce product this codebase actually
// maps to its own generic Product/Variant/SKU model — deliberately excludes
// the large tail of SEO/ads/affiliate/plugin meta_data fields real Woo
// stores carry, none of which any other channel would ever have.
type WooProduct struct {
	ID           string
	Name         string
	Slug         string
	SKU          string
	Description  string
	Status       string // woo: publish/draft/private/pending
	Type         string // woo: simple/variable/grouped/external
	RegularPrice string
	// SalePrice/SaleEndDate mirror this codebase's SKU.SalePriceMinorUnits/
	// SaleEndDate -- empty string means no sale. SaleEndDate is
	// "YYYY-MM-DD" (Woo's date_on_sale_to). Same round-trip caveat as
	// Virtual below: the underlying library's omitempty means explicitly
	// clearing a previously-set SalePrice back to "" does not actually
	// clear it on the Woo side (a pre-existing library limitation, not new
	// to these two fields).
	SalePrice     string
	SaleEndDate   string
	Virtual       bool // digital/non-shippable — never has weight or dimensions
	ManageStock   bool
	StockQuantity *int
	// Weight/Dimensions are raw numeric strings in whatever unit the store
	// is configured for (WooCommerce doesn't return the unit per-product —
	// see WooClient.GetProductUnits). Empty string means Woo has no value
	// set for that field.
	Weight     string
	Dimensions WooProductDimensions
	Images     []WooProductImage
	Categories []WooProductCategory
}

type WooProductDimensions struct {
	Length string
	Width  string
	Height string
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

// WooCategory is a full entry from Woo's Product Categories resource
// (/wc/v3/products/categories) — unlike the category info embedded on a
// product, this includes ParentExternalID, which is what makes hierarchy
// reconstruction possible. ParentExternalID is "" for a top-level category.
type WooCategory struct {
	ExternalID       string
	Name             string
	Slug             string
	ParentExternalID string
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
	// UpdateProduct pushes this codebase's current view of a product to an
	// already-linked Woo product, identified by its external id — used by
	// the event-driven product->channel sync (see application/channels'
	// SyncProductToChannelsUseCase). Only fields this codebase manages are
	// sent (name/description/status/price/virtual/weight/dimensions);
	// categories/images/SEO meta etc. are left untouched on the Woo side.
	// Bool fields (Virtual) rely on the underlying library's own omitempty
	// JSON tags, so explicitly clearing a previously-true flag back to
	// false does not round-trip — a pre-existing library limitation.
	UpdateProduct(ctx context.Context, externalProductID string, p WooProduct) (*WooProduct, error)
	// ListCategories returns every category in the store, flat — callers
	// reconstruct the tree themselves via ParentExternalID. Used by import
	// to preserve Woo's category hierarchy instead of importing everything
	// as a single flat level.
	ListCategories(ctx context.Context) ([]WooCategory, error)
	// GetProductUnits returns the store's configured weight unit
	// (kg/g/lb/oz) and dimension unit (m/cm/mm/in/yd) — WooCommerce's
	// product endpoint returns raw numeric strings with no unit attached;
	// these are a store-wide setting, fetched separately, needed to
	// convert Weight/Dimensions into WeightGrams/*MM correctly instead of
	// assuming a unit and silently importing wrong values.
	GetProductUnits(ctx context.Context) (weightUnit, dimensionUnit string, err error)
	// DownloadImage fetches a product image from its Woo-hosted URL — a
	// plain HTTPS GET (product images aren't behind the store's REST API
	// auth), routed through this port anyway so all of an import's outbound
	// network calls stay behind one seam. Caller must close the returned
	// reader.
	DownloadImage(ctx context.Context, url string) (data io.ReadCloser, contentType string, err error)
}

// WooClientFactory builds a WooClient scoped to one store's credentials.
type WooClientFactory interface {
	NewClient(storeURL, consumerKey, consumerSecret string) (WooClient, error)
}
