export interface CartItem {
  id: string // line id, distinct from productId -- two lines can share a product with different variants
  productId: string
  variantId?: string
  name: string
  variantLabel?: string
  imageUrl?: string
  sku: string
  price: number
  quantity: number
  savedForLater?: boolean
}

export interface AppliedCoupon {
  code: string
  label: string
  discountAmount: number // resolved minor-unit discount for the current cart
}

export interface CartTotals {
  subtotal: number
  discount: number
  shipping: number
  tax: number
  total: number
  freeShippingThreshold: number
  freeShippingRemaining: number
}

export interface Cart {
  id: string
  items: CartItem[]
  savedItems: CartItem[]
  coupons: AppliedCoupon[]
  giftCardCode?: string
  currency: string
  totals: CartTotals
}
