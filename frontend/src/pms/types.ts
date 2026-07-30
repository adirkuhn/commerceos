export type ProductStatus = 'draft' | 'active' | 'archived'
export type SKUStatus = 'active' | 'inactive'

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
  currency_code: string
  weight_grams?: number | null
  length_mm?: number | null
  width_mm?: number | null
  height_mm?: number | null
  is_virtual: boolean
  status: SKUStatus
  created_at: string
  updated_at: string
}

export interface Variant {
  id: string
  tenant_id: string
  product_id: string
  name: string
  position: number
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
}

export type UpdateProductPayload = CreateProductPayload

export interface CreateVariantPayload {
  name: string
  position?: number
}

export type UpdateVariantPayload = CreateVariantPayload

export interface CreateSKUPayload {
  sku_code: string
  barcode?: string
  price_minor_units: number
  currency_code: string
  weight_grams?: number | null
  length_mm?: number | null
  width_mm?: number | null
  height_mm?: number | null
  is_virtual?: boolean
  status: SKUStatus
}

export interface UpdateSKUPayload {
  barcode?: string
  price_minor_units: number
  currency_code: string
  weight_grams?: number | null
  length_mm?: number | null
  width_mm?: number | null
  height_mm?: number | null
  is_virtual?: boolean
  status: SKUStatus
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
