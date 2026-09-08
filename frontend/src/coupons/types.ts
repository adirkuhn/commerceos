export type DiscountType = 'percent' | 'fixed'

export interface Coupon {
  id: string
  code: string
  discount_type: DiscountType
  // Basis points (0-10000) off the cart subtotal for "percent", or a flat
  // minor-currency-units amount for "fixed".
  discount_value: number
  min_subtotal_minor_units?: number | null
  usage_limit?: number | null
  times_used: number
  expires_at?: string | null
  active: boolean
  created_at: string
  updated_at: string
}

export interface CouponInput {
  code: string
  discount_type: DiscountType
  discount_value: number
  min_subtotal_minor_units?: number | null
  usage_limit?: number | null
  expires_at?: string | null
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
