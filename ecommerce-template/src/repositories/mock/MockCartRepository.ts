import type { Cart, CartItem, AppliedCoupon } from '@/types'
import type { CartRepository } from '../types'
import { simulateLatency } from './latency'
import { computeTotals, computeCouponDiscount } from './cartMath'
import { COUPONS } from '@/config/demoData'

interface CartState {
  items: CartItem[]
  savedItems: CartItem[]
  coupons: string[]
}

// In-memory per-cartId store -- persists for the life of the tab, same as
// a real cart would persist server-side keyed by session/customer. See
// stores/cartStore.ts for the client-side cart id + UI state (drawer
// open/closed) this repository is called from.
const carts = new Map<string, CartState>()

function stateFor(cartId: string): CartState {
  let state = carts.get(cartId)
  if (!state) {
    state = { items: [], savedItems: [], coupons: [] }
    carts.set(cartId, state)
  }
  return state
}

function toCart(cartId: string, state: CartState): Cart {
  const coupons: AppliedCoupon[] = state.coupons.map(code => {
    const meta = COUPONS.find(c => c.code === code)
    return {
      code,
      label: meta?.label ?? code,
      discountAmount: computeCouponDiscount(state.items.reduce((s, i) => s + i.price * i.quantity, 0), state.items, code),
    }
  })
  return {
    id: cartId,
    items: state.items,
    savedItems: state.savedItems,
    coupons,
    currency: 'USD',
    totals: computeTotals(state.items, coupons),
  }
}

let lineSeq = 0

export class MockCartRepository implements CartRepository {
  async get(cartId: string): Promise<Cart> {
    await simulateLatency()
    return toCart(cartId, stateFor(cartId))
  }

  async addItem(cartId: string, item: Omit<CartItem, 'id'>): Promise<Cart> {
    await simulateLatency()
    const state = stateFor(cartId)
    const existing = state.items.find(i => i.productId === item.productId && i.variantId === item.variantId)
    if (existing) {
      existing.quantity += item.quantity
    } else {
      lineSeq += 1
      state.items.push({ ...item, id: `line-${lineSeq}` })
    }
    return toCart(cartId, state)
  }

  async updateQuantity(cartId: string, lineId: string, quantity: number): Promise<Cart> {
    await simulateLatency()
    const state = stateFor(cartId)
    if (quantity <= 0) {
      state.items = state.items.filter(i => i.id !== lineId)
    } else {
      const line = state.items.find(i => i.id === lineId)
      if (line) line.quantity = quantity
    }
    return toCart(cartId, state)
  }

  async removeItem(cartId: string, lineId: string): Promise<Cart> {
    await simulateLatency()
    const state = stateFor(cartId)
    state.items = state.items.filter(i => i.id !== lineId)
    return toCart(cartId, state)
  }

  async saveForLater(cartId: string, lineId: string): Promise<Cart> {
    await simulateLatency()
    const state = stateFor(cartId)
    const line = state.items.find(i => i.id === lineId)
    if (line) {
      state.items = state.items.filter(i => i.id !== lineId)
      state.savedItems.push({ ...line, savedForLater: true })
    }
    return toCart(cartId, state)
  }

  async moveToCart(cartId: string, lineId: string): Promise<Cart> {
    await simulateLatency()
    const state = stateFor(cartId)
    const line = state.savedItems.find(i => i.id === lineId)
    if (line) {
      state.savedItems = state.savedItems.filter(i => i.id !== lineId)
      state.items.push({ ...line, savedForLater: false })
    }
    return toCart(cartId, state)
  }

  async applyCoupon(cartId: string, code: string): Promise<Cart> {
    await simulateLatency()
    const state = stateFor(cartId)
    const upper = code.trim().toUpperCase()
    const coupon = COUPONS.find(c => c.code === upper)
    if (!coupon) throw new Error(`"${code}" is not a valid coupon code.`)
    if (state.coupons.includes(upper)) throw new Error('That coupon is already applied.')
    const subtotal = state.items.reduce((s, i) => s + i.price * i.quantity, 0)
    if (coupon.minSubtotal && subtotal < coupon.minSubtotal) {
      throw new Error(`${coupon.code} requires a subtotal of at least $${(coupon.minSubtotal / 100).toFixed(2)}.`)
    }
    state.coupons.push(upper)
    return toCart(cartId, state)
  }

  async removeCoupon(cartId: string, code: string): Promise<Cart> {
    await simulateLatency()
    const state = stateFor(cartId)
    state.coupons = state.coupons.filter(c => c !== code)
    return toCart(cartId, state)
  }

  async clear(cartId: string): Promise<Cart> {
    await simulateLatency()
    const state = stateFor(cartId)
    state.items = []
    state.coupons = []
    return toCart(cartId, state)
  }
}
