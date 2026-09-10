export type CouponKind = 'percentage' | 'fixed' | 'free_shipping' | 'buy_x_get_y'

export interface Coupon {
  code: string
  kind: CouponKind
  label: string
  value?: number // percentage points, or minor-unit fixed discount
  minSubtotal?: number
  buyQuantity?: number
  getQuantity?: number
  expiresAt?: string
}

export interface QuantityDiscountTier {
  minQuantity: number
  discountPercent: number
}

export interface FlashSale {
  id: string
  title: string
  productIds: string[]
  discountPercent: number
  endsAt: string
}
