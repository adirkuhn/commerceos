package integrations

import (
	"context"
	"io"
	"time"
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

// WooVariation is the subset of a WooCommerce product variation this
// codebase maps to. Distinct from WooProduct: a variation has no name,
// description, images, or categories of its own -- it inherits all of
// that from its parent product. Only price/stock/physical attributes and
// which attribute option it represents are per-variation.
type WooVariation struct {
	ID            string
	SKU           string
	RegularPrice  string
	SalePrice     string
	SaleEndDate   string
	ManageStock   bool
	StockQuantity *int
	Weight        string
	Dimensions    WooProductDimensions
	// AttributeName/AttributeValue select which of the parent product's
	// declared attribute options this variation represents -- e.g.
	// "Weight"/"367g". AttributeValue must already be listed in that
	// attribute's Options on the parent (see CreateVariableProduct) --
	// CreateVariation adds it there automatically if it isn't yet.
	AttributeName  string
	AttributeValue string
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

// WooOrder is the subset of a WooCommerce order this codebase maps onto
// domain/oms.SalesOrder -- deliberately excludes refunds/coupon lines/fee
// lines/tax lines/meta_data, none of which oms.SalesOrder represents today.
// Date* fields are UTC ("Y-m-d\TH:i:s" with no offset, read from Woo's
// *_gmt fields, not its store-local-time ones) -- converted to time.Time at
// the import use case, not here. Empty string means Woo has no value for
// that field.
type WooOrder struct {
	ID                 int
	Number             string
	OrderKey           string
	Status             string // woo: pending/processing/on-hold/completed/cancelled/refunded/failed -- byte-for-byte oms.SalesOrderStatus
	CustomerID         int    // 0 = guest checkout, no linked Woo customer
	Billing            WooAddress
	Shipping           WooAddress
	CustomerNote       string
	Currency           string
	DiscountTotal      string
	ShippingTotal      string
	TotalTax           string
	Total              string
	PaymentMethod      string
	PaymentMethodTitle string
	TransactionID      string
	DateCreated        string
	DateModified       string
	DatePaid           string // "" if unpaid
	DateCompleted      string // "" if not completed
	LineItems          []WooOrderLineItem
}

type WooAddress struct {
	FirstName, LastName, Company, Address1, Address2, City, State, Postcode, Country, Email, Phone string
}

type WooOrderLineItem struct {
	ProductID   int
	VariationID int // 0 if this line is a simple product, not a variation
	SKU         string
	Name        string
	Quantity    int64
	// Price/Subtotal/SubtotalTax/Total/TotalTax are raw decimal strings,
	// same convention as WooProduct.RegularPrice -- converted to minor
	// units at the import use case.
	Price       string
	Subtotal    string
	SubtotalTax string
	Total       string
	TotalTax    string
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
	// CreateVariableProduct creates the PARENT product with type "variable"
	// and declares one variation-attribute (attributeName) with the given
	// possible values -- must be called before any CreateVariation call
	// for this product. p.SKU/RegularPrice/SalePrice/ManageStock/
	// StockQuantity are ignored: a variable product's parent carries none
	// of those, only its variations do.
	CreateVariableProduct(ctx context.Context, p WooProduct, attributeName string, attributeValues []string) (*WooProduct, error)
	// CreateVariation creates a new variation under an existing variable
	// product (parentExternalID). If v.AttributeValue isn't already
	// declared in the parent's attribute options, the parent is updated
	// first to add it -- WooCommerce rejects a variation whose attribute
	// value isn't declared on the parent.
	CreateVariation(ctx context.Context, parentExternalID string, v WooVariation) (*WooVariation, error)
	// UpdateVariation pushes this codebase's current view of one variation
	// to an already-created Woo variation, identified by its external id.
	UpdateVariation(ctx context.Context, parentExternalID, variationExternalID string, v WooVariation) (*WooVariation, error)
	// DeleteVariation permanently removes one variation from its parent
	// variable product.
	DeleteVariation(ctx context.Context, parentExternalID, variationExternalID string) error
	// DeleteProduct permanently removes a product from the store, identified
	// by its external (Woo) id. Used to clean up a channel listing after the
	// product has been deleted on the CommerceOS side.
	DeleteProduct(ctx context.Context, externalProductID string) error
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
	// ListOrders pages through the store's orders, newest-modified last
	// (Woo's default sort). modifiedAfter, if non-nil, filters to orders
	// changed since that instant -- the incremental sync path; nil fetches
	// every order (full import / periodic reconciliation pass), so no
	// order is ever permanently missed by an incremental cursor that
	// drifts.
	ListOrders(ctx context.Context, page, perPage int, modifiedAfter *time.Time) ([]WooOrder, error)
}

// WooClientFactory builds a WooClient scoped to one store's credentials.
type WooClientFactory interface {
	NewClient(storeURL, consumerKey, consumerSecret string) (WooClient, error)
}
