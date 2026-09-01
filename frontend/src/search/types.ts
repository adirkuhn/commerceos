export interface SearchProductResult {
  id: string
  name: string
  slug: string
}

export interface SearchSupplierResult {
  id: string
  name: string
  email?: string
}

export interface SearchCustomerResult {
  id: string
  first_name: string
  last_name: string
  email?: string
}

export interface SearchSalesOrderResult {
  id: string
  number: string
  billing_first_name?: string
  billing_last_name?: string
  status: string
}

export interface GlobalSearchResult {
  products: SearchProductResult[]
  suppliers: SearchSupplierResult[]
  customers: SearchCustomerResult[]
  sales_orders: SearchSalesOrderResult[]
}

export interface APIResponse<T> {
  data: T | null
  error: { code: string; message: string } | null
}
