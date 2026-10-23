// Mirrors backend/api/pms/openapi.yaml (listPriceList).

export interface PriceListRow {
  product_id: string
  product_name: string
  variant_name: string
  sku_id: string
  sku_code: string
  unit_type: 'unit' | 'box' | 'kilo'
  currency_code: string
  // Stored prices: tax-inclusive when PriceListPage.prices_include_tax.
  price_minor_units: number
  sale_price_minor_units?: number | null
  // Excludes tax; null when unknown.
  cost_price_minor_units?: number | null
  // Grams for unit_type kilo.
  available_quantity: number
  earliest_expiry?: string | null
  tax_rate_basis_points: number
  target_markup_basis_points: number
}

export interface PriceListPage {
  rows: PriceListRow[]
  // Products, not rows.
  total: number
  pricesIncludeTax: boolean
}

export interface PriceListQuery {
  page?: number
  perPage?: number
  search?: string
  productIds?: string[]
}

// One quote line as kept in the URL; unitPrice is set only when the price
// was typed over the discounted one.
export interface QuoteLine {
  productId: string
  skuId: string
  quantity: number
  unitPrice?: number
}
