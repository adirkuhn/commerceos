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

// Mirrors backend/api/quotes/openapi.yaml.
export interface SavedQuoteSummary {
  id: string
  number: number
  reference: string
  name: string
  currency_code: string
  discount_basis_points: number
  line_count: number
  total_minor_units: number
  created_at: string
  updated_at: string
}

export interface SavedQuoteLine {
  product_id: string
  sku_id: string
  quantity: number
  unit_price_minor_units: number
  line_total_minor_units: number
}

export interface SavedQuote extends SavedQuoteSummary {
  lines: SavedQuoteLine[]
}

export interface QuoteInput {
  name: string
  discount_basis_points: number
  currency_code?: string
  lines: Omit<SavedQuoteLine, 'line_total_minor_units'>[]
}

export interface SavedQuotePage {
  quotes: SavedQuoteSummary[]
  total: number
}
