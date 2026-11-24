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

// A company is named by its company name and has contact persons; a private
// customer by the person.
export type CustomerType = 'private' | 'company'

export interface Customer {
  id: string
  tenant_id: string
  woo_customer_id?: number | null
  // Optional: older responses and fixtures omit them.
  customer_type?: CustomerType
  // The company name for a company; else the person, billing company or email.
  display_name?: string
  company_name?: string
  // The shop's own number for the customer, unique per shop.
  client_number?: string
  kvk_number?: string
  vat_number?: string
  email: string
  first_name: string
  last_name: string
  username?: string
  role?: string
  billing?: Address
  shipping?: Address
  /** Read-only: worked out from the orders (a paid, not cancelled or refunded one). */
  is_paying_customer: boolean
  avatar_url?: string
  customer_note?: string
  date_created: string
  date_modified: string
}

export interface CustomerInput {
  woo_customer_id?: number | null
  customer_type?: CustomerType
  // Required for a company.
  company_name?: string
  // Generated when empty on create.
  client_number?: string
  kvk_number?: string
  vat_number?: string
  // Added with a new customer; the first is the main one.
  contact_persons?: ContactPersonInput[]
  // Optional: a customer needs a name, company, email or phone.
  email?: string
  first_name?: string
  last_name?: string
  username?: string
  role?: string
  billing?: Address
  shipping?: Address
  avatar_url?: string
  customer_note?: string
}

// Someone to deal with at a company customer.
export interface ContactPerson {
  id: string
  customer_id: string
  first_name: string
  last_name: string
  // Free text, e.g. "Purchasing".
  role: string
  email: string
  phone: string
  created_at: string
}

// A first or last name is required; the rest is optional.
export interface ContactPersonInput {
  first_name?: string
  last_name?: string
  role?: string
  email?: string
  phone?: string
}

export type WhatsAppMessageDirection = 'inbound' | 'outbound'

// One entry in a customer's WhatsApp conversation log (both inbound
// messages and outbound replies) -- see backend/domain/whatsapp.Message.
export interface WhatsAppMessage {
  id: string
  phone_e164: string
  direction: WhatsAppMessageDirection
  body: string
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
