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
  // The company name for a company; else the person, billing company or email.
  // Optional: older responses and fixtures omit them.
  display_name?: string
  customer_type?: 'private' | 'company'
  company_name?: string
  client_number?: string
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
