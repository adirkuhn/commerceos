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
}

type ProductFilter struct {
	CategoryID *string
	Status     *ProductStatus
	// Search matches against name or slug (case- and accent-insensitively,
	// e.g. "acai" matches "Açaí"), or any of the product's SKUs' sku_code
	// or barcode (case-insensitive, exact substring -- these are codes, not
	// natural-language text, so no accent-folding needed there).
	Search *string
	// UpdatedAfter restricts to products modified after this time --
	// incremental-sync filter for external integrations pulling the
	// catalog (e.g. a POS desktop), same idea as WooCommerce's own
	// modified_after. Compared against products.updated_at, which is
	// bumped by UpdateProductUseCase but NOT by anything stock-related
	// (stock lives in a separate ledger, not a Product field).
	UpdatedAfter *time.Time
	Limit        int
	Offset       int
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
	// clobber it back to nil.
	UpdateCostPrice(ctx context.Context, skuID string, costPriceMinorUnits int64) error
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
	Delete(ctx context.Context, id string) error
}

type AttributeRepository interface {
	Create(ctx context.Context, a *AttributeDefinition) error
	FindByID(ctx context.Context, id string) (*AttributeDefinition, error)
	List(ctx context.Context) ([]AttributeDefinition, error)
	Delete(ctx context.Context, id string) error
}
