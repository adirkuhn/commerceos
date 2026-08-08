package pms

import "errors"

var (
	ErrProductNotFound    = errors.New("product not found")
	ErrCategoryNotFound   = errors.New("category not found")
	ErrVariantNotFound    = errors.New("variant not found")
	ErrSKUNotFound        = errors.New("sku not found")
	ErrDuplicateSlug      = errors.New("slug already exists")
	ErrDuplicateSKUCode   = errors.New("sku code already exists")
	ErrVariantsNotEnabled = errors.New("product does not have variants enabled")
	// ErrVirtualSKUCannotHaveDimensions guards the split between virtual
	// (digital, non-shippable) and physical SKUs: a virtual SKU never has
	// weight/dimensions, since it never ships.
	ErrVirtualSKUCannotHaveDimensions = errors.New("virtual SKU cannot have weight or dimensions")
	// ErrSalePriceExceedsPrice guards SKU.SalePriceMinorUnits: when set,
	// it must not exceed PriceMinorUnits.
	ErrSalePriceExceedsPrice = errors.New("sale price cannot exceed regular price")
	// ErrSaleEndDateRequiresSalePrice guards SKU.SaleEndDate: it can only
	// be set alongside a SalePriceMinorUnits.
	ErrSaleEndDateRequiresSalePrice = errors.New("sale end date requires a sale price to be set")
	ErrMediaNotFound                = errors.New("media not found")
	ErrUnsupportedMediaType         = errors.New("unsupported file type")
	ErrMediaTargetRequired          = errors.New("media must be attached to a product or a variant")
	// ErrFeaturedMediaProductMismatch guards a variant's FeaturedMediaID:
	// it must reference a Media row belonging to the variant's own product.
	ErrFeaturedMediaProductMismatch = errors.New("featured media does not belong to this variant's product")
	ErrBarcodeAliasNotFound         = errors.New("barcode alias not found")
	// ErrCaseOfSKURequiresUnitsPerCase guards SKU.CaseOfSKUID/UnitsPerCase:
	// both must be set together, or neither.
	ErrCaseOfSKURequiresUnitsPerCase = errors.New("case_of_sku_id and units_per_case must be set together")
	// ErrCaseOfSKUCannotBeSelf guards against a SKU declaring itself as its
	// own case-of relationship.
	ErrCaseOfSKUCannotBeSelf = errors.New("a SKU cannot be a case of itself")
	// ErrCaseOfSKUCannotChain guards against a case SKU pointing at another
	// case SKU -- the relationship is depth-limited to one level.
	ErrCaseOfSKUCannotChain = errors.New("a case SKU cannot itself be a case of another case SKU")
	// ErrUnitsPerCaseMustBePositive guards SKU.UnitsPerCase: a case of zero
	// or negative units isn't one.
	ErrUnitsPerCaseMustBePositive = errors.New("units_per_case must be positive")
)
