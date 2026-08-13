// Mirrors WooCommerce's customer/address object shape -- see the backend's
// domain/customers.Customer doc comment.
export interface Address {
  first_name?: string
  last_name?: string
  company?: string
  address_1?: string
  address_2?: string
  city?: string
  state?: string
  postcode?: string
  country?: string
  // Only meaningful on billing.
  email?: string
  phone?: string
}

export interface Customer {
  id: string
  tenant_id: string
  woo_customer_id?: number | null
  email: string
  first_name: string
  last_name: string
  username?: string
  role?: string
  billing?: Address
  shipping?: Address
  is_paying_customer: boolean
  avatar_url?: string
  date_created: string
  date_modified: string
}

export interface CustomerInput {
  woo_customer_id?: number | null
  email: string
  first_name?: string
  last_name?: string
  username?: string
  role?: string
  billing?: Address
  shipping?: Address
  is_paying_customer?: boolean
  avatar_url?: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
