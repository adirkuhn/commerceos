package pms

import (
	"context"
	"time"
)

type CategoryRepository interface {
	Create(ctx context.Context, c *Category) error
	FindByID(ctx context.Context, id string) (*Category, error)
	FindBySlug(ctx context.Context, slug string) (*Category, error)
	List(ctx context.Context) ([]Category, error)
	Update(ctx context.Context, c *Category) error
	Delete(ctx context.Context, id string) error
}

type ProductRepository interface {
	Create(ctx context.Context, p *Product) error
	FindByID(ctx context.Context, id string) (*Product, error)
	FindBySlug(ctx context.Context, slug string) (*Product, error)
	List(ctx context.Context, filter ProductFilter) ([]Product, int64, error)
	Update(ctx context.Context, p *Product) error
	Delete(ctx context.Context, id string) error
	// SearchByAnyNameToken returns every product whose (case/accent
	// -insensitive) name contains at least one of tokens -- a deliberately
	// wide candidate net. It exists for procurement's fuzzy name-matching
	// step (see procurement.MatchExtractedLineItemsUseCase): real-world
	// document text and a catalog's canonical product name rarely contain
	// each other verbatim (word order, brand suffixes, pack-size text all
	// differ), so an exact-phrase Search can't find the candidates at all.
	// Real scoring/ranking of the results happens in Go, not SQL.
	SearchByAnyNameToken(ctx context.Context, tokens []string, limit int) ([]Product, error)
	// CountByCategory groups filter's matching products by category_id --
	// used by application/storefront's search-facets support to show a
	// count next to each category a shopper could switch to. See
	// infrastructure/repositories.ProductRepo.CountByCategory's doc
	// comment for why AndCategoryID is deliberately ignored here.
	CountByCategory(ctx context.Context, filter ProductFilter) (map[string]int64, error)
}

type ProductFilter struct {
	CategoryID *string
	// Tag matches products whose Tags array contains this value exactly
	// (JSONB containment, not a substring match) -- same field
	// SalesChannel's "by tag" enrollment rule matches against.
	Tag    *string
	Status *ProductStatus
	// Search matches against name or slug (case- and accent-insensitively,
	// e.g. "acai" matches "Açaí"), or any of the product's SKUs' sku_code
	// or barcode (case-insensitive, exact substring -- these are codes, not
	// natural-language text, so no accent-folding needed there).
	Search *string
	// IncludeIDs, CategoryID and Tag combine with OR into one group (a
	// product matches the group if it satisfies ANY of the ones that are
	// set), which then ANDs with every other filter below -- this is what
	// lets application/storefront express "category X's products, PLUS
	// these specific manually-included ones" (or tag/manual-only) as a
	// single, exactly-paginated query instead of over-fetching and
	// filtering in Go. nil means "not constrained by an explicit id list"
	// (omitted from the group entirely) -- a non-nil pointer, even to an
	// empty slice, means "constrained to exactly this list", so when
	// CategoryID and Tag are both nil too, an empty IncludeIDs correctly
	// matches nothing rather than silently matching everything.
	IncludeIDs *[]string
	// ExcludeIDs removes these product ids from the result regardless of
	// every other filter -- applied last, unconditionally.
	ExcludeIDs []string
	// AndCategoryID is a plain, independent "AND category_id = ?" filter --
	// unlike CategoryID (which is part of the enrollment OR-group above),
	// this narrows the result regardless of IncludeIDs/Tag. Lets
	// application/storefront support "browse this category" on top of
	// "products enrolled on this channel" as one exact query, without the
	// two concepts interfering (a manually-included product from a
	// different category correctly stays hidden while browsing a specific
	// category, but still shows up when browsing "all").
	AndCategoryID *string
	// AndIncludeIDs is AndCategoryID's own counterpart for an explicit id
	// list -- a plain, independent "AND id IN (...)" filter, unlike
	// IncludeIDs (part of the enrollment OR-group above). Lets
	// application/storefront support a merchant's manually-curated
	// hand-picked product list (e.g. a storefront Product Grid block)
	// on top of "products enrolled on this channel" as one exact query.
	// Same nil-vs-empty convention as IncludeIDs: nil means unconstrained,
	// a non-nil pointer to an empty slice means "constrained to nothing".
	AndIncludeIDs *[]string
	// OrderByIDs, when set (and there's no search term), orders results by
	// their position in this list -- e.g. a collection's own sort order,
	// kept intact across pages.
	OrderByIDs []string
	// UpdatedAfter restricts to products modified after this time --
	// incremental-sync filter for external integrations pulling the
	// catalog (e.g. a POS desktop), same idea as WooCommerce's own
	// modified_after. Compared against products.updated_at, which is
	// bumped by UpdateProductUseCase but NOT by anything stock-related
	// (stock lives in a separate ledger, not a Product field).
	UpdatedAfter *time.Time
	// StockLevel, when not StockLevelAny, keeps only low or out-of-stock
	// products -- see StockLevel.
	StockLevel StockLevel
	Limit      int
	Offset     int
}

type VariantRepository interface {
	Create(ctx context.Context, v *Variant) error
	FindByID(ctx context.Context, id string) (*Variant, error)
	ListByProduct(ctx context.Context, productID string) ([]Variant, error)
	Update(ctx context.Context, v *Variant) error
	Delete(ctx context.Context, id string) error
}

type SKURepository interface {
	Create(ctx context.Context, s *SKU) error
	FindByID(ctx context.Context, id string) (*SKU, error)
	FindByCode(ctx context.Context, code string) (*SKU, error)
	// FindByBarcode is used by the scanner app to resolve a scanned unit
	// barcode straight to a SKU. Returns ErrSKUNotFound when nothing
	// matches -- same as FindByCode, no separate error type needed.
	FindByBarcode(ctx context.Context, barcode string) (*SKU, error)
	// FindByScaleCode resolves a SKU by its 5-digit ScaleCode -- used by
	// ResolveBarcodeUseCase once a scanned code has been decoded as a
	// weight-embedded barcode (see DecodeWeightEmbeddedBarcode), whose
	// full barcode never matches FindByBarcode since the weight digits
	// vary per package. Returns ErrSKUNotFound when nothing matches.
	FindByScaleCode(ctx context.Context, scaleCode string) (*SKU, error)
	// FindByExactProductName resolves a SKU via its parent Product's Name
	// (normalized: case/punctuation/whitespace-insensitive, same convention
	// as suppliers.FindByName -- NOT a fuzzy/substring search). Used by
	// ImportWooOrdersUseCase as a fallback when Woo's own SKU string is
	// blank or doesn't match anything (a common WooCommerce quirk on order
	// line snapshots even when the product itself has a real SKU) -- the
	// line item's own Woo-sent name is compared against the catalog
	// product's name instead. Returns ErrSKUNotFound both when nothing
	// matches AND when more than one SKU would match (a same-named product
	// with multiple variants/SKUs) -- ambiguous is treated the same as
	// not-found, since guessing wrong here would silently mislink an order
	// to the wrong product.
	FindByExactProductName(ctx context.Context, name string) (*SKU, error)
	ListByVariant(ctx context.Context, variantID string) ([]SKU, error)
	Update(ctx context.Context, s *SKU) error
	Delete(ctx context.Context, id string) error
	// ListWithExpiredSale returns every SKU (across all tenants -- meant
	// to be called under a bypass_rls/SuperAdmin context, see
	// cmd/scheduler) whose sale has a SaleEndDate in the past and still
	// has a SalePriceMinorUnits set.
	ListWithExpiredSale(ctx context.Context) ([]SKU, error)
	// ExistsWithTaxClass reports whether any SKU explicitly references the
	// given tax class -- used to guard tax.TaxClass deletion (see
	// application/tax.DeleteTaxClassUseCase) without loading every SKU.
	ExistsWithTaxClass(ctx context.Context, taxClassID string) (bool, error)
	// ListActive returns every active SKU for the current tenant -- the
	// daily cost-price cron's (RecalculateCostPricesUseCase) iteration
	// source. Inactive SKUs are skipped: nothing meaningful to recompute
	// for a discontinued product.
	ListActive(ctx context.Context) ([]SKU, error)
	// UpdateCostPrice writes only CostPriceMinorUnits -- deliberately
	// narrower than Update, so RecalculateCostPricesUseCase is the sole
	// writer of this field and an ordinary SKU edit (UpdateSKUUseCase,
	// which never reads/round-trips a cost price) can't accidentally
	// clobber it back to nil. Does not touch CostPriceSource -- the cron
	// only ever calls this after confirming the SKU's source isn't manual
	// (see RecalculateCostPricesUseCase.Execute).
	UpdateCostPrice(ctx context.Context, skuID string, costPriceMinorUnits int64) error
	// SetManualCostPrice writes CostPriceMinorUnits and flips CostPriceSource
	// to manual in one call -- see SetManualCostPriceUseCase. The cron
	// (RecalculateCostPricesUseCase) checks CostPriceSource and skips any
	// SKU set this way entirely, so the value sticks until
	// ClearCostPriceOverride runs.
	SetManualCostPrice(ctx context.Context, skuID string, costPriceMinorUnits int64) error
	// ClearCostPriceOverride resets CostPriceSource back to computed --
	// CostPriceMinorUnits is left as-is (still the last manual value) until
	// the next cron run recomputes it; see ClearCostPriceOverrideUseCase.
	ClearCostPriceOverride(ctx context.Context, skuID string) error
}

// SKUCostPriceHistoryRepository backs the margin-trend-over-time chart --
// append-only in spirit (a day's row is never edited by a human), but the
// cron itself must be able to safely rerun the same day (crash/redeploy
// mid-run), hence Upsert rather than a plain Create.
type SKUCostPriceHistoryRepository interface {
	// Upsert writes h, replacing any existing row for the same
	// (tenant_id, sku_id, occurred_on) -- makes a same-day cron rerun a
	// no-op instead of a duplicate/error.
	Upsert(ctx context.Context, h *SKUCostPriceHistory) error
	// ListBySKU returns a SKU's history rows, most recent first, capped at
	// limit (0 means the repository's own default cap).
	ListBySKU(ctx context.Context, skuID string, limit int) ([]SKUCostPriceHistory, error)
}

// SKUCostPriceLedgerRepository is defined in cost_price_ledger.go, next to
// the SKUCostPriceLedgerEntry type it backs.

// SKUPriceHistoryRepository backs the immutable, effective-dated
// selling-price audit trail -- see SKUPriceHistory. Unlike
// SKUCostPriceHistoryRepository (a periodic snapshot upserted daily), a row
// here is only ever written when price/sale price actually changes, and
// ValidUntil is filled in retroactively by CloseOpen when superseded --
// never edited otherwise.
type SKUPriceHistoryRepository interface {
	// Create inserts a new open-ended interval (ValidUntil nil).
	Create(ctx context.Context, h *SKUPriceHistory) error
	// CloseOpen closes whatever interval is currently open (ValidUntil IS
	// NULL) for skuID by setting ValidUntil = until. A no-op (not an
	// error) if none is open -- e.g. the SKU's very first price change,
	// or a SKU that existed before this table did.
	CloseOpen(ctx context.Context, skuID string, until time.Time) error
	// ListBySKU returns skuID's price history, most recent (by ValidFrom)
	// first, capped at limit (0 means the repository's own default cap).
	ListBySKU(ctx context.Context, skuID string, limit int) ([]SKUPriceHistory, error)
}

type MediaRepository interface {
	Attach(ctx context.Context, m *Media) error
	FindByID(ctx context.Context, id string) (*Media, error)
	ListByProduct(ctx context.Context, productID string) ([]Media, error)
	Update(ctx context.Context, m *Media) error
	Delete(ctx context.Context, id string) error
}

// BarcodeAliasRepository backs the scanner app's case/box-barcode
// resolution -- see BarcodeAlias.
type BarcodeAliasRepository interface {
	Create(ctx context.Context, a *BarcodeAlias) error
	FindByCode(ctx context.Context, code string) (*BarcodeAlias, error)
	ListBySKU(ctx context.Context, skuID string) ([]BarcodeAlias, error)
	// ListBySKUIDs bulk-loads aliases for multiple SKUs in one query --
	// same N+1-avoidance rationale as VariantRepo.ListByProductIDs (see
	// ListPOSProductCatalogUseCase, which syncs every SKU's aliases to
	// external POS integrations).
	ListBySKUIDs(ctx context.Context, skuIDs []string) ([]BarcodeAlias, error)
	Delete(ctx context.Context, id string) error
}

// ScaleCodeAliasRepository backs additional (non-primary) scale codes for
// a SKU -- see ScaleCodeAlias.
type ScaleCodeAliasRepository interface {
	Create(ctx context.Context, a *ScaleCodeAlias) error
	FindByCode(ctx context.Context, code string) (*ScaleCodeAlias, error)
	ListBySKU(ctx context.Context, skuID string) ([]ScaleCodeAlias, error)
	// ListBySKUIDs bulk-loads aliases for multiple SKUs in one query --
	// see BarcodeAliasRepository.ListBySKUIDs.
	ListBySKUIDs(ctx context.Context, skuIDs []string) ([]ScaleCodeAlias, error)
	Delete(ctx context.Context, id string) error
}

type AttributeRepository interface {
	Create(ctx context.Context, a *AttributeDefinition) error
	FindByID(ctx context.Context, id string) (*AttributeDefinition, error)
	List(ctx context.Context) ([]AttributeDefinition, error)
	Delete(ctx context.Context, id string) error
}

// ProductLink is the least needed to link to a product page: a sitemap
// entry.
type ProductLink struct {
	Slug      string
	UpdatedAt time.Time
}

// ProductLinkLister lists links for every product matching filter (its
// Limit caps the result), newest change first -- without loading whole
// products, so a sitemap of tens of thousands stays cheap.
type ProductLinkLister interface {
	ListLinks(ctx context.Context, filter ProductFilter) ([]ProductLink, error)
}
