package pms

import "context"

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
}

type ProductFilter struct {
	CategoryID *string
	Status     *ProductStatus
	// Search matches against name or slug, case- and accent-insensitively
	// (e.g. "acai" matches "Açaí").
	Search *string
	Limit  int
	Offset int
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
