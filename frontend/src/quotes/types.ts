// Quote editor and saved quotes (backend/api/quotes/openapi.yaml).

// One quote line as kept in the URL; unitPrice is set only when the price
// was typed over the discounted one.
export interface QuoteLine {
  productId: string
  skuId: string
  quantity: number
  unitPrice?: number
}

// Mirrors backend/api/quotes/openapi.yaml.
export interface QuoteCustomer {
  id: string
  name: string
  email: string
}

export interface SavedQuoteSummary {
  id: string
  number: number
  reference: string
  name: string
  customer: QuoteCustomer | null
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
  customer_id?: string | null
  discount_basis_points: number
  currency_code?: string
  lines: Omit<SavedQuoteLine, 'line_total_minor_units'>[]
}

export interface SavedQuotePage {
  quotes: SavedQuoteSummary[]
  total: number
}

export interface QuoteStockCheckLine {
  product_id: string
  sku_id: string
  quantity: number
  available: number
  short: number
}

export interface QuoteStockCheck {
  lines: QuoteStockCheckLine[]
  has_shortfall: boolean
}

export interface ConvertQuoteResult {
  sales_order_id: string
  number: string
  needs_stock_review: boolean
}
