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

// CouponCurrencyAmount is a manual fixed-amount override for one coupon
// in one currency -- coupons are tenant-wide (not channel-scoped), so a
// single code can be redeemed on channels running in different currencies.
// Only meaningful for discount_type "fixed"; absent an override, the
// checkout flow converts discount_value via the tenant's configured
// exchange rate instead (see Settings > Localization).
export interface CouponCurrencyAmount {
  coupon_id: string
  currency_code: string
  amount_minor_units: number
}

export interface UpsertCouponCurrencyAmountInput {
  currency_code: string
  amount_minor_units: number
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
