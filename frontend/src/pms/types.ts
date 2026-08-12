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
  status: SKUStatus
  created_at: string
  updated_at: string
}

export interface ResolveBarcodeResult {
  sku: SKU
  // 1 for a direct unit-barcode match; the case/box multiplier when
  // resolved via a BarcodeAlias.
  units_per_scan: number
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

export type UpdateCategoryPayload = CreateCategoryPayload

export interface CreateProductPayload {
  name: string
  description?: string
  category_id?: string | null
  status: ProductStatus
  has_variants?: boolean
  sync_variants_as_woo_variations?: boolean
  tags?: string[]
}

export type UpdateProductPayload = CreateProductPayload

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
  case_of_sku_id?: string | null
  units_per_case?: number | null
  unit_type?: SKUUnitType
  portioned_from_sku_id?: string | null
  status: SKUStatus
}

export interface UpdateSKUPayload {
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
  case_of_sku_id?: string | null
  units_per_case?: number | null
  unit_type?: SKUUnitType
  portioned_from_sku_id?: string | null
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
