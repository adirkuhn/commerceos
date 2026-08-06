export type SupplierStatus = 'active' | 'inactive'

export interface Supplier {
  id: string
  tenant_id: string
  name: string
  email: string
  phone: string
  country: string
  lead_time_days?: number | null
  status: SupplierStatus
  notes: string
  created_at: string
  updated_at: string
}

export interface SupplierInput {
  name: string
  email?: string
  phone?: string
  country?: string
  lead_time_days?: number | null
  status: SupplierStatus
  notes?: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
