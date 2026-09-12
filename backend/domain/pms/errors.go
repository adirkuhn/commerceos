package pms

import "errors"

var (
	ErrProductNotFound  = errors.New("product not found")
	ErrCategoryNotFound = errors.New("category not found")
	ErrVariantNotFound  = errors.New("variant not found")
	ErrSKUNotFound      = errors.New("sku not found")
	ErrDuplicateSlug    = errors.New("slug already exists")
	// ErrInvalidSlug guards an explicit slug edit (UpdateProductUseCase):
	// it must already be in slugified form (lowercase alphanumeric,
	// single dashes, no leading/trailing dash) -- same shape Slugify
	// produces from a name, so the URL stays well-formed.
	ErrInvalidSlug      = errors.New("slug must be lowercase alphanumeric with single dashes")
	ErrDuplicateSKUCode = errors.New("sku code already exists")
	// ErrDuplicateBarcode guards SKU.Barcode: a barcode scan must resolve
	// to exactly one SKU within a tenant (skus_tenant_barcode_idx).
	ErrDuplicateBarcode   = errors.New("barcode already in use by another sku")
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
	// ErrInvalidCostPrice guards SetManualCostPriceUseCase's input: a cost
	// price cannot be negative.
	ErrInvalidCostPrice     = errors.New("cost price cannot be negative")
	ErrMediaNotFound        = errors.New("media not found")
	ErrUnsupportedMediaType = errors.New("unsupported file type")
	ErrMediaTargetRequired  = errors.New("media must be attached to a product or a variant")
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
	// ErrPortionedFromSKUCannotBeSelf guards against a SKU declaring
	// itself as its own bulk pool.
	ErrPortionedFromSKUCannotBeSelf = errors.New("a SKU cannot be portioned from itself")
	// ErrPortionedFromSKUMustTrackWeight guards SKU.PortionedFromSKUID:
	// the referenced SKU must be a weight pool (UnitType=kilo) --
	// portioning only makes sense against a bulk lot.
	ErrPortionedFromSKUMustTrackWeight = errors.New("portioned_from_sku_id must reference a SKU with unit_type kilo")
	// ErrBulkSKUCannotBePortioned guards against a kilo (weight-pool) SKU
	// itself having a PortionedFromSKUID -- the relationship is
	// depth-limited to one level, same reasoning as ErrCaseOfSKUCannotChain.
	ErrBulkSKUCannotBePortioned = errors.New("a bulk (kilo) SKU cannot itself be portioned from another SKU")
	// ErrInvalidSKUUnitType guards SKU.UnitType against any value outside
	// unit/box/kilo.
	ErrInvalidSKUUnitType = errors.New("invalid sku unit_type")
	// ErrProductHasReceivingHistory guards product/SKU deletion: a SKU
	// that was scanned into a purchase-order booking has booking_scans
	// rows referencing it (ON DELETE RESTRICT, deliberately -- it's the
	// receiving audit trail), so the product can't be deleted while that
	// history exists.
	ErrProductHasReceivingHistory = errors.New("product has purchase-order receiving history and cannot be deleted")
	// ErrProductHasSalesOrderHistory guards product/SKU deletion the same
	// way as ErrProductHasReceivingHistory: a SKU that was ever sold has
	// sales_order_line_items rows referencing it (ON DELETE RESTRICT,
	// deliberately -- it's the sales history), so the product can't be
	// deleted while that history exists.
	ErrProductHasSalesOrderHistory = errors.New("product has sales order history and cannot be deleted")
	// ErrInvalidTargetMarkupBasisPoints guards Category.TargetMarkupBasisPoints
	// and settings.TenantSettings.DefaultTargetMarkupBasisPoints: must be
	// >= 0. Unlike margin, markup (profit / cost) has no upper bound.
	ErrInvalidTargetMarkupBasisPoints = errors.New("target markup basis points must be at least 0")
	// ErrExchangeRateRequired is returned by ResolveSKUPrice when a SKU
	// needs converting into a currency that has neither an explicit
	// SKUCurrencyPrice override nor a configured fx.ExchangeRate -- a
	// misconfigured channel (its CurrencyCode's rate got deleted after
	// the channel was created) rather than a normal "no price" case.
	ErrExchangeRateRequired = errors.New("no exchange rate or price override configured for this currency")
	// ErrInvalidSKUPricingMode guards SKU.PricingMode against any value
	// outside fixed/per_weight.
	ErrInvalidSKUPricingMode = errors.New("invalid sku pricing_mode")
	// ErrPricePerKiloRequiredForPerWeight guards SKU.PricePerKiloMinorUnits:
	// PricingMode=per_weight requires it to be set (and positive).
	ErrPricePerKiloRequiredForPerWeight = errors.New("price_per_kilo_minor_units is required when pricing_mode is per_weight")
	// ErrPricePerKiloOnlyForPerWeight guards the reverse: a fixed-pricing
	// SKU (the default) must not carry a per-kilo rate that would never be
	// read, since it'd be silently stale/misleading data.
	ErrPricePerKiloOnlyForPerWeight = errors.New("price_per_kilo_minor_units can only be set when pricing_mode is per_weight")
	// ErrInvalidPricePerKilo guards SKU.PricePerKiloMinorUnits: when set,
	// it must be positive.
	ErrInvalidPricePerKilo = errors.New("price_per_kilo_minor_units must be positive")
	// ErrDuplicateScaleCode guards SKU.ScaleCode: a scale-decoded barcode
	// must resolve to exactly one SKU within a tenant
	// (skus_tenant_scale_code_idx).
	ErrDuplicateScaleCode = errors.New("scale_code already in use by another sku")
)
