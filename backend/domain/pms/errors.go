package pms

import (
	"github.com/adirkuhn/commerceos/domain/errs"
)

var (
	ErrProductNotFound  = errs.New("PRODUCT_NOT_FOUND", "product not found")
	ErrCategoryNotFound = errs.New("CATEGORY_NOT_FOUND", "category not found")
	ErrVariantNotFound  = errs.New("VARIANT_NOT_FOUND", "variant not found")
	ErrSKUNotFound      = errs.New("SKU_NOT_FOUND", "sku not found")
	ErrDuplicateSlug    = errs.New("SLUG_TAKEN", "slug already exists")
	// ErrInvalidSlug guards an explicit slug edit (UpdateProductUseCase):
	// it must already be in slugified form (lowercase alphanumeric,
	// single dashes, no leading/trailing dash) -- same shape Slugify
	// produces from a name, so the URL stays well-formed.
	ErrInvalidSlug      = errs.New("SLUG_INVALID", "slug must be lowercase alphanumeric with single dashes")
	ErrDuplicateSKUCode = errs.New("SKU_CODE_TAKEN", "sku code already exists")
	// ErrDuplicateBarcode guards SKU.Barcode: a barcode scan must resolve
	// to exactly one SKU within a tenant (skus_tenant_barcode_idx).
	ErrDuplicateBarcode   = errs.New("BARCODE_TAKEN", "barcode already in use by another sku")
	ErrVariantsNotEnabled = errs.New("VARIANTS_NOT_ENABLED", "product does not have variants enabled")
	// ErrVirtualSKUCannotHaveDimensions guards the split between virtual
	// (digital, non-shippable) and physical SKUs: a virtual SKU never has
	// weight/dimensions, since it never ships.
	ErrVirtualSKUCannotHaveDimensions = errs.New("VIRTUAL_SKU_CANNOT_HAVE_DIMENSIONS", "virtual SKU cannot have weight or dimensions")
	// ErrSalePriceExceedsPrice guards SKU.SalePriceMinorUnits: when set,
	// it must not exceed PriceMinorUnits.
	ErrSalePriceExceedsPrice = errs.New("SALE_PRICE_EXCEEDS_PRICE", "sale price cannot exceed regular price")
	// ErrSaleEndDateRequiresSalePrice guards SKU.SaleEndDate: it can only
	// be set alongside a SalePriceMinorUnits.
	ErrSaleEndDateRequiresSalePrice = errs.New("SALE_END_DATE_REQUIRES_SALE_PRICE", "sale end date requires a sale price to be set")
	// ErrInvalidCostPrice guards SetManualCostPriceUseCase's input: a cost
	// price cannot be negative.
	ErrInvalidCostPrice     = errs.New("COST_PRICE_NEGATIVE", "cost price cannot be negative")
	ErrMediaNotFound        = errs.New("MEDIA_NOT_FOUND", "media not found")
	ErrUnsupportedMediaType = errs.New("MEDIA_TYPE_UNSUPPORTED", "unsupported file type")
	ErrMediaTargetRequired  = errs.New("MEDIA_TARGET_REQUIRED", "media must be attached to a product or a variant")
	// ErrFeaturedMediaProductMismatch guards a variant's FeaturedMediaID:
	// it must reference a Media row belonging to the variant's own product.
	ErrFeaturedMediaProductMismatch = errs.New("FEATURED_MEDIA_PRODUCT_MISMATCH", "featured media does not belong to this variant's product")
	ErrBarcodeAliasNotFound         = errs.New("BARCODE_ALIAS_NOT_FOUND", "barcode alias not found")
	// ErrCaseOfSKURequiresUnitsPerCase guards SKU.CaseOfSKUID/UnitsPerCase:
	// both must be set together, or neither.
	ErrCaseOfSKURequiresUnitsPerCase = errs.New("CASE_OF_SKU_REQUIRES_UNITS_PER_CASE", "case_of_sku_id and units_per_case must be set together")
	// ErrCaseOfSKUCannotBeSelf guards against a SKU declaring itself as its
	// own case-of relationship.
	ErrCaseOfSKUCannotBeSelf = errs.New("CASE_OF_SKU_CANNOT_BE_SELF", "a SKU cannot be a case of itself")
	// ErrCaseOfSKUCannotChain guards against a case SKU pointing at another
	// case SKU -- the relationship is depth-limited to one level.
	ErrCaseOfSKUCannotChain = errs.New("CASE_OF_SKU_CANNOT_CHAIN", "a case SKU cannot itself be a case of another case SKU")
	// ErrUnitsPerCaseMustBePositive guards SKU.UnitsPerCase: a case of zero
	// or negative units isn't one.
	ErrUnitsPerCaseMustBePositive = errs.New("UNITS_PER_CASE_MUST_BE_POSITIVE", "units_per_case must be positive")
	// ErrPortionedFromSKUCannotBeSelf guards against a SKU declaring
	// itself as its own bulk pool.
	ErrPortionedFromSKUCannotBeSelf = errs.New("PORTIONED_FROM_SKU_CANNOT_BE_SELF", "a SKU cannot be portioned from itself")
	// ErrPortionedFromSKUMustTrackWeight guards SKU.PortionedFromSKUID:
	// the referenced SKU must be a weight pool (UnitType=kilo) --
	// portioning only makes sense against a bulk lot.
	ErrPortionedFromSKUMustTrackWeight = errs.New("PORTIONED_FROM_SKU_MUST_TRACK_WEIGHT", "portioned_from_sku_id must reference a SKU with unit_type kilo")
	// ErrBulkSKUCannotBePortioned guards against a kilo (weight-pool) SKU
	// itself having a PortionedFromSKUID -- the relationship is
	// depth-limited to one level, same reasoning as ErrCaseOfSKUCannotChain.
	ErrBulkSKUCannotBePortioned = errs.New("BULK_SKU_CANNOT_BE_PORTIONED", "a bulk (kilo) SKU cannot itself be portioned from another SKU")
	// ErrInvalidSKUUnitType guards SKU.UnitType against any value outside
	// unit/box/kilo.
	ErrInvalidSKUUnitType = errs.New("SKU_UNIT_TYPE_INVALID", "invalid sku unit_type")
	// ErrProductHasReceivingHistory guards product/SKU deletion: a SKU
	// that was scanned into a purchase-order booking has booking_scans
	// rows referencing it (ON DELETE RESTRICT, deliberately -- it's the
	// receiving audit trail), so the product can't be deleted while that
	// history exists. It and its siblings below all wrap
	// ErrProductHasHistory.
	ErrProductHasReceivingHistory = errs.NewChild(ErrProductHasHistory, "PRODUCT_HAS_RECEIVING_HISTORY", "product has purchase-order receiving history and cannot be deleted")
	// ErrProductHasSalesOrderHistory: a SKU that was ever sold has
	// sales_order_line_items rows referencing it (ON DELETE RESTRICT,
	// deliberately -- it's the sales history).
	ErrProductHasSalesOrderHistory = errs.NewChild(ErrProductHasHistory, "PRODUCT_HAS_SALES_ORDER_HISTORY", "product has sales order history and cannot be deleted")
	// ErrProductHasPickingHistory: pick_scans rows (ON DELETE RESTRICT).
	ErrProductHasPickingHistory = errs.NewChild(ErrProductHasHistory, "PRODUCT_HAS_PICKING_HISTORY", "product has picking history and cannot be deleted")
	// ErrProductHasDiscardHistory: discard_records rows (ON DELETE RESTRICT).
	ErrProductHasDiscardHistory = errs.NewChild(ErrProductHasHistory, "PRODUCT_HAS_DISCARD_HISTORY", "product has discard history and cannot be deleted")
	// ErrInvalidTargetMarkupBasisPoints guards Category.TargetMarkupBasisPoints
	// and settings.TenantSettings.DefaultTargetMarkupBasisPoints: must be
	// >= 0. Unlike margin, markup (profit / cost) has no upper bound.
	ErrInvalidTargetMarkupBasisPoints = errs.New("CATEGORY_INVALID_TARGET_MARKUP", "target markup basis points must be at least 0")
	// ErrExchangeRateRequired is returned by ResolveSKUPrice when a SKU
	// needs converting into a currency that has neither an explicit
	// SKUCurrencyPrice override nor a configured fx.ExchangeRate -- a
	// misconfigured channel (its CurrencyCode's rate got deleted after
	// the channel was created) rather than a normal "no price" case.
	ErrExchangeRateRequired = errs.New("SKU_PRICE_EXCHANGE_RATE_REQUIRED", "no exchange rate or price override configured for this currency")
	// ErrInvalidSKUPricingMode guards SKU.PricingMode against any value
	// outside fixed/per_weight.
	ErrInvalidSKUPricingMode = errs.New("SKU_PRICING_MODE_INVALID", "invalid sku pricing_mode")
	// ErrPricePerKiloRequiredForPerWeight guards SKU.PricePerKiloMinorUnits:
	// PricingMode=per_weight requires it to be set (and positive).
	ErrPricePerKiloRequiredForPerWeight = errs.New("PRICE_PER_KILO_REQUIRED_FOR_PER_WEIGHT", "price_per_kilo_minor_units is required when pricing_mode is per_weight")
	// ErrPricePerKiloOnlyForPerWeight guards the reverse: a fixed-pricing
	// SKU (the default) must not carry a per-kilo rate that would never be
	// read, since it'd be silently stale/misleading data.
	ErrPricePerKiloOnlyForPerWeight = errs.New("PRICE_PER_KILO_ONLY_FOR_PER_WEIGHT", "price_per_kilo_minor_units can only be set when pricing_mode is per_weight")
	// ErrInvalidPricePerKilo guards SKU.PricePerKiloMinorUnits: when set,
	// it must be positive.
	ErrInvalidPricePerKilo = errs.New("PRICE_PER_KILO_INVALID", "price_per_kilo_minor_units must be positive")
	// ErrDuplicateScaleCode guards SKU.ScaleCode: a scale-decoded barcode
	// must resolve to exactly one SKU within a tenant
	// (skus_tenant_scale_code_idx).
	ErrDuplicateScaleCode = errs.New("SCALE_CODE_TAKEN", "scale_code already in use by another sku")
	// ErrDuplicateBarcodeAliasCode guards BarcodeAlias.Code: a scanned
	// alias must resolve to exactly one SKU within a tenant
	// (barcode_aliases_tenant_id_code_key).
	ErrDuplicateBarcodeAliasCode = errs.New("BARCODE_ALIAS_TAKEN", "barcode already in use by another sku or alias")
	// ErrInvalidUnitsPerScan guards BarcodeAlias.UnitsPerScan: must be at
	// least 1 (1 = a plain alternate unit barcode, >1 = a case/box
	// multiplier).
	ErrInvalidUnitsPerScan = errs.New("UNITS_PER_SCAN_INVALID", "units_per_scan must be at least 1")
	// ErrDuplicateScaleCodeAlias guards ScaleCodeAlias.Code: a decoded
	// weight-embedded barcode must resolve to exactly one SKU within a
	// tenant (scale_code_aliases_tenant_id_code_key).
	ErrDuplicateScaleCodeAlias = errs.New("SCALE_CODE_ALIAS_TAKEN", "scale code already in use by another sku")
	// ErrScaleCodeAliasNotFound is returned by ScaleCodeAliasRepository
	// lookups/deletes that don't match.
	ErrScaleCodeAliasNotFound = errs.New("SCALE_CODE_ALIAS_NOT_FOUND", "scale code alias not found")
	// ErrBarcodeAliasCodeRequired guards CreateBarcodeAliasUseCase's
	// input: a blank/whitespace-only code isn't a barcode.
	ErrBarcodeAliasCodeRequired = errs.New("BARCODE_ALIAS_CODE_REQUIRED", "code is required")
	// ErrScaleCodeAliasCodeRequired guards CreateScaleCodeAliasUseCase's
	// input: a blank/whitespace-only code isn't a scale code.
	ErrScaleCodeAliasCodeRequired = errs.New("SCALE_CODE_ALIAS_CODE_REQUIRED", "code is required")
	// ErrWeightRequiredForPerWeightSKU guards picking/booking scan
	// resolution (oms.ScanPickItemUseCase/ResolvePickScanCodeUseCase,
	// procurement.ScanBookingItemUseCase/ResolveBookingScanCodeUseCase):
	// a SKU priced per_weight can only be resolved by actually decoding
	// its weight-embedded barcode (or a ScaleCodeAlias) -- matching it via
	// a typed SKU code or a line item's raw text carries no real weight,
	// so it's rejected rather than silently letting a picker/receiver
	// record a package that was never actually weighed.
	ErrWeightRequiredForPerWeightSKU = errs.New("WEIGHT_REQUIRED_FOR_PER_WEIGHT_SKU", "this item is priced by weight -- scan its scale barcode, not a product code")
	// ErrNoWeightsProvided guards PackWeightPortionsUseCase: an empty
	// weight list has nothing to create.
	ErrNoWeightsProvided = errs.New("WEIGHTS_REQUIRED", "at least one weight is required")
	// ErrBulkSKUNotWeightPool guards PackWeightPortionsUseCase: the
	// source SKU must actually be a scale-sold weight pool -- UnitType
	// kilo, PricingMode per_weight (so a per-package price can be
	// computed), and a ScaleCode set (so a barcode can be generated for
	// each new package) -- all three, not just one.
	ErrBulkSKUNotWeightPool = errs.New("BULK_SKU_NOT_WEIGHT_POOL", "source sku must be a weight pool with per-weight pricing and a scale code")
	// ErrCostPriceUnknownForCorrection guards CorrectCostUseCase: a value
	// correction adjusts an existing cost baseline, so a SKU whose cost is
	// still unknown (cost_price_source unknown, see
	// SKUCostPriceSourceUnknown) has nothing to correct -- the first real
	// receipt establishes that baseline, not a correction.
	ErrCostPriceUnknownForCorrection = errs.New("COST_PRICE_UNKNOWN_FOR_CORRECTION", "sku has no known cost price to correct -- wait for its first receipt")
	// ErrNoStockOnHandForCorrection guards CorrectCostUseCase: a value
	// correction is meaningless with zero on-hand quantity -- there's
	// nothing to divide the corrected value across.
	ErrNoStockOnHandForCorrection = errs.New("NO_STOCK_ON_HAND_FOR_CORRECTION", "sku has no on-hand stock to apply a cost correction to")
)

// ErrProductHasHistory is what every "can't delete, history references
// it" error wraps -- one kind of failure for callers (the API answers 409
// PRODUCT_HAS_HISTORY and offers archiving instead), each keeping its own
// message.
var ErrProductHasHistory = errs.New("PRODUCT_HAS_HISTORY", "product has history")
