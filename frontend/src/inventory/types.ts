// Address/contact fields mirror Shopify's Location resource.
export interface Warehouse {
  id: string
  tenant_id: string
  name: string
  address1: string
  address2: string
  city: string
  province: string
  zip: string
  country: string
  phone: string
  active: boolean
  created_at: string
  updated_at: string
}

export interface WarehouseInput {
  name: string
  address1?: string
  address2?: string
  city?: string
  province?: string
  zip?: string
  country?: string
  phone?: string
  active?: boolean
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
