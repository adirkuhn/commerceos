export type ProductStatus = 'draft' | 'active' | 'archived'
export type SKUStatus = 'active' | 'inactive'
// unit/box are both whole-count stock (box is purely descriptive, distinct
// from case_of_sku_id/units_per_case, a real multiplier relationship to
// another SKU). kilo means stock quantity is grams, not a count.
export type SKUUnitType = 'unit' | 'box' | 'kilo'

export interface Category {
  id: string
  tenant_id: string
  parent_id?: string | null
  name: string
  slug: string
  position: number
  // Overrides the tenant's default target markup for every product in this
  // category, when set (basis points, no upper bound, e.g. 3000 = 30%
  // markup). Null means "use the tenant default".
  target_markup_basis_points?: number | null
  created_at: string
  updated_at: string
}

export interface Product {
  id: string
  tenant_id: string
  category_id?: string | null
  name: string
  slug: string
  description: string
  status: ProductStatus
  has_variants: boolean
  // Only matters when has_variants is true, and only for channels that
  // support it (WooCommerce today). false (default): each variant pushes
  // as its own separate simple Woo product. true: all variants push as ONE
  // Woo "variable" product with one Woo variation per variant. Toggling
  // this on an already-pushed product triggers a migration (old external
  // representation deleted, recreated in the new shape) rather than a
  // plain field update.
  sync_variants_as_woo_variations?: boolean
  // Free-form labels this tenant assigns to the product -- used by a Sales
  // Channel's "by tag" enrollment rule to decide which products auto-sync
  // to that channel.
  tags: string[]
  created_at: string
  updated_at: string
  // Only ever populated on the list response (GET /products) -- the
  // featured (lowest-position) media and the first variant's SKU price,
  // for the list's thumbnail/price columns. Absent on detail/create/update
  // responses, which carry the full variant/SKU breakdown instead.
  image_url?: string | null
  price_minor_units?: number | null
  sale_price_minor_units?: number | null
  currency_code?: string | null
}

export interface SKU {
  id: string
  tenant_id: string
  variant_id: string
  sku_code: string
  barcode?: string | null
  price_minor_units: number
  sale_price_minor_units?: number | null
  sale_end_date?: string | null
  currency_code: string
  weight_grams?: number | null
  length_mm?: number | null
  width_mm?: number | null
  height_mm?: number | null
  is_virtual: boolean
  // Whether this SKU's inventory batches carry a meaningful expiry date --
  // advisory only. Always present on a real API response (backend requires
  // it); optional here only so hand-written test fixtures that don't care
  // about it don't all need updating. When updating a SKU, always pass this
  // through from the loaded value -- UpdateSKU is a full replace, so
  // omitting it silently resets it to false.
  tracks_expiry?: boolean
  // Both null unless this SKU is a sealed case/box of another SKU -- set
  // together or not at all. See the inventory module's split-case action
  // for converting case stock into unit stock.
  case_of_sku_id?: string | null
  units_per_case?: number | null
  // unit_type defaults to 'unit'. 'kilo' marks this SKU as a weight pool:
  // its stock quantity is grams, not whole units (e.g. a 20kg bulk lot
  // booked as one stock-in of 20000).
  unit_type?: SKUUnitType
  // portioned_from_sku_id is set when this SKU is a sellable item peeled
  // off a kilo (weight-pool) SKU's pool -- see the inventory module's
  // portion-from-bulk action.
  portioned_from_sku_id?: string | null
  // tax_class_id is null unless this SKU has its own tax class assigned --
  // null means "use the tenant's default class" (see the Tax settings
  // section's TaxClass.is_default).
  tax_class_id?: string | null
  // woo_tax_class_slug/woo_tax_status are informational provenance only,
  // set once by WooCommerce import -- never affects tax calculation. Both
  // null/undefined for a SKU never imported from Woo. "" for
  // woo_tax_class_slug means Woo's own Standard rate.
  woo_tax_class_slug?: string | null
  woo_tax_status?: string | null
  status: SKUStatus
  // Read-only. Weighted average of on-hand batches' landed cost, recomputed
  // daily by the cost-price cron. Null until the cron has run at least once
  // with a cost basis to average.
  cost_price_minor_units?: number | null
  // Read-only. Whether cost_price_minor_units came from the daily cron
  // ('computed', the default) or a human's manual override ('manual', see
  // pmsApi.setManualCostPrice/clearCostPriceOverride).
  cost_price_source?: SKUCostPriceSource
  created_at: string
  updated_at: string
}

export type SKUCostPriceSource = 'computed' | 'manual'

export interface SKUMarginInsight {
  cost_price_available: boolean
  cost_price_minor_units: number
  cost_price_source: SKUCostPriceSource
  effective_target_markup_basis_points: number
  recommended_price_minor_units: number
  profit_at_price_minor_units: number
  // margin = profit / price, markup = profit / cost -- same profit,
  // expressed two ways.
  margin_at_price_basis_points: number
  markup_at_price_basis_points: number
  profit_at_sale_price_minor_units?: number | null
  margin_at_sale_price_basis_points?: number | null
  markup_at_sale_price_basis_points?: number | null
  total_on_hand_quantity: number
  total_profit_potential_minor_units: number
  shrinkage_minor_units: number
}

export interface SKUCostPriceHistoryEntry {
  id: string
  sku_id: string
  cost_price_minor_units: number
  recommended_price_minor_units: number
  occurred_on: string
}

export type SKUPriceHistoryReason = 'created' | 'manual_update' | 'sale_expired'

export interface SKUPriceHistoryEntry {
  id: string
  sku_id: string
  price_minor_units: number
  sale_price_minor_units?: number | null
  currency_code: string
  valid_from: string
  // Null means this interval is the one currently in effect.
  valid_until?: string | null
  reason: SKUPriceHistoryReason
}

export type ProductHistoryEntityType = 'product' | 'variant'
export type ProductHistoryField = 'name' | 'status'

export interface ProductHistoryEntry {
  id: string
  entity_type: ProductHistoryEntityType
  entity_id: string
  field: ProductHistoryField
  old_value: string
  new_value: string
  changed_at: string
}

export interface MarginReportRow {
  sku_id: string
  sku_code: string
  product_name?: string | null
  cost_price_minor_units: number
  price_minor_units: number
  profit_minor_units: number
  margin_basis_points: number
  markup_basis_points: number
  target_markup_basis_points: number
  recommended_price_minor_units: number
  below_target: boolean
}

export interface ResolveBarcodeResult {
  sku: SKU
  // 1 for a direct unit-barcode match; the case/box multiplier when
  // resolved via a BarcodeAlias.
  units_per_scan: number
  // The resolved SKU's product's name -- scanner apps show this instead of
  // the SKU code alone, which is rarely human readable on its own.
  product_name: string
}

export interface Variant {
  id: string
  tenant_id: string
  product_id: string
  name: string
  position: number
  featured_media_id?: string | null
  created_at: string
  updated_at: string
}

export interface VariantWithSKUs extends Variant {
  skus?: SKU[]
}

export interface ProductDetail {
  product: Product
  variants: VariantWithSKUs[]
}

export interface CreateCategoryPayload {
  name: string
  parent_id?: string | null
}

export interface UpdateCategoryPayload extends CreateCategoryPayload {
  target_markup_basis_points?: number | null
}

export interface CreateProductPayload {
  name: string
  description?: string
  category_id?: string | null
  status: ProductStatus
  has_variants?: boolean
  sync_variants_as_woo_variations?: boolean
  tags?: string[]
}

// slug is optional here (unlike a create, where it's always derived from
// name): omit it to leave the product's current slug unchanged. When sent,
// it must already be in slugified form (lowercase alphanumeric, single
// dashes) and unique across the tenant's products -- this is the fix that
// lets a duplicated product's permanent "-copy" slug be corrected.
export type UpdateProductPayload = CreateProductPayload & { slug?: string }

export interface CreateVariantPayload {
  name: string
  position?: number
}

export type UpdateVariantPayload = CreateVariantPayload & { featured_media_id?: string | null }

export interface CreateSKUPayload {
  sku_code: string
  barcode?: string
  price_minor_units: number
  sale_price_minor_units?: number | null
  sale_end_date?: string | null
  currency_code: string
  weight_grams?: number | null
  length_mm?: number | null
  width_mm?: number | null
  height_mm?: number | null
  is_virtual?: boolean
  tracks_expiry?: boolean
  case_of_sku_id?: string | null
  units_per_case?: number | null
  unit_type?: SKUUnitType
  portioned_from_sku_id?: string | null
  tax_class_id?: string | null
  status: SKUStatus
}

export interface UpdateSKUPayload {
  sku_code: string
  barcode?: string
  price_minor_units: number
  sale_price_minor_units?: number | null
  sale_end_date?: string | null
  currency_code: string
  weight_grams?: number | null
  length_mm?: number | null
  width_mm?: number | null
  height_mm?: number | null
  is_virtual?: boolean
  tracks_expiry?: boolean
  case_of_sku_id?: string | null
  units_per_case?: number | null
  unit_type?: SKUUnitType
  portioned_from_sku_id?: string | null
  tax_class_id?: string | null
  status: SKUStatus
}

export type MediaType = 'image' | 'video'

export interface Media {
  id: string
  tenant_id: string
  product_id?: string | null
  variant_id?: string | null
  url: string
  type: MediaType
  position: number
  alt_text: string
  created_at: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
