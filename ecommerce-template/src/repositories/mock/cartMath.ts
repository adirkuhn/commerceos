import type { CartItem, AppliedCoupon, CartTotals } from '@/types'
import { COUPONS } from '@/config/demoData'

export const FREE_SHIPPING_THRESHOLD = 10000 // $100
export const STANDARD_SHIPPING = 900 // $9
export const TAX_RATE = 0.08

export function computeCouponDiscount(subtotal: number, items: CartItem[], code: string): number {
  const coupon = COUPONS.find(c => c.code === code)
  if (!coupon) return 0
  if (coupon.minSubtotal && subtotal < coupon.minSubtotal) return 0
  switch (coupon.kind) {
    case 'percentage':
      return Math.round(subtotal * ((coupon.value ?? 0) / 100))
    case 'fixed':
      return Math.min(coupon.value ?? 0, subtotal)
    case 'buy_x_get_y': {
      // Cheapest eligible line becomes free for every (buy+get) group met.
      const buy = coupon.buyQuantity ?? 1
      const get = coupon.getQuantity ?? 1
      const totalQty = items.reduce((sum, i) => sum + i.quantity, 0)
      const groups = Math.floor(totalQty / (buy + get))
      if (groups === 0) return 0
      const cheapestPrice = items.length > 0 ? Math.min(...items.map(i => i.price)) : 0
      return cheapestPrice * get * groups
    }
    default:
      return 0
  }
}

export function computeTotals(items: CartItem[], coupons: AppliedCoupon[]): CartTotals {
  const subtotal = items.reduce((sum, i) => sum + i.price * i.quantity, 0)
  const discount = coupons.reduce((sum, c) => sum + c.discountAmount, 0)
  const hasFreeShipping = coupons.some(c => COUPONS.find(x => x.code === c.code)?.kind === 'free_shipping')
  const afterDiscount = Math.max(0, subtotal - discount)
  const shipping = items.length === 0 || hasFreeShipping || afterDiscount >= FREE_SHIPPING_THRESHOLD ? 0 : STANDARD_SHIPPING
  const tax = Math.round(afterDiscount * TAX_RATE)
  const total = afterDiscount + shipping + tax
  const freeShippingRemaining = Math.max(0, FREE_SHIPPING_THRESHOLD - afterDiscount)
  return { subtotal, discount, shipping, tax, total, freeShippingThreshold: FREE_SHIPPING_THRESHOLD, freeShippingRemaining }
}
