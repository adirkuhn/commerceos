import { createContext, useCallback, useContext, useEffect, useState } from 'react'
import type { ReactNode } from 'react'
import * as cartApi from '../api/cart'
import type { Cart, CheckoutRequest, Checkout, UpdateCartItemLine } from '../types'

const STORAGE_KEY = 'commerceos_cart_token'

interface CartContextValue {
  cart: Cart | null
  loading: boolean
  error: string | null
  addItem: (skuCode: string, quantity: number) => Promise<void>
  setItemQuantity: (skuCode: string, quantity: number) => Promise<void>
  removeItem: (skuCode: string) => Promise<void>
  // Passing an empty string removes any coupon already attached to the cart.
  applyCoupon: (code: string) => Promise<void>
  checkout: (payload: CheckoutRequest) => Promise<Checkout>
}

const CartContext = createContext<CartContextValue | null>(null)

export function CartProvider({ children }: { children: ReactNode }) {
  const [cart, setCart] = useState<Cart | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    const token = localStorage.getItem(STORAGE_KEY)
    if (!token) {
      setLoading(false)
      return
    }
    cartApi.getCart(token)
      .then(setCart)
      .catch(() => localStorage.removeItem(STORAGE_KEY)) // stale/expired/checked-out cart
      .finally(() => setLoading(false))
  }, [])

  // Every cart mutation replaces the whole line list (see the backend's
  // updateCartItems doc comment) -- ensureCart guarantees a token exists
  // (creating one lazily on first add) before any of them run.
  const ensureCart = useCallback(async (): Promise<Cart> => {
    if (cart) return cart
    const existingToken = localStorage.getItem(STORAGE_KEY)
    if (existingToken) {
      try {
        const existing = await cartApi.getCart(existingToken)
        setCart(existing)
        return existing
      } catch {
        localStorage.removeItem(STORAGE_KEY)
      }
    }
    const created = await cartApi.createCart()
    localStorage.setItem(STORAGE_KEY, created.token)
    setCart(created)
    return created
  }, [cart])

  // Takes the cart to update explicitly (rather than re-deriving it via
  // ensureCart) so a caller that already resolved the cart this tick --
  // addItem/setItemQuantity, right after their own ensureCart() call --
  // can't have a second, stale-closure ensureCart() call race it and
  // apply the update against the wrong cart.
  const applyItems = useCallback(async (token: string, items: UpdateCartItemLine[]) => {
    setError(null)
    try {
      const updated = await cartApi.updateCartItems(token, items)
      setCart(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
      throw e
    }
  }, [])

  const addItem = useCallback(async (skuCode: string, quantity: number) => {
    const current = await ensureCart()
    const lines = current.items.map(i => ({ sku_code: i.sku_code, quantity: i.quantity }))
    const existing = lines.find(l => l.sku_code === skuCode)
    if (existing) {
      existing.quantity += quantity
    } else {
      lines.push({ sku_code: skuCode, quantity })
    }
    await applyItems(current.token, lines)
  }, [ensureCart, applyItems])

  const setItemQuantity = useCallback(async (skuCode: string, quantity: number) => {
    const current = await ensureCart()
    const lines = current.items
      .map(i => ({ sku_code: i.sku_code, quantity: i.sku_code === skuCode ? quantity : i.quantity }))
      .filter(l => l.quantity > 0)
    await applyItems(current.token, lines)
  }, [ensureCart, applyItems])

  const removeItem = useCallback((skuCode: string) => setItemQuantity(skuCode, 0), [setItemQuantity])

  const applyCoupon = useCallback(async (code: string) => {
    const current = await ensureCart()
    setError(null)
    try {
      const updated = await cartApi.applyCoupon(current.token, code)
      setCart(updated)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
      throw e
    }
  }, [ensureCart])

  const checkout = useCallback(async (payload: CheckoutRequest): Promise<Checkout> => {
    if (!cart) throw new Error('Your cart is empty.')
    setError(null)
    try {
      const result = await cartApi.checkoutCart(cart.token, payload)
      localStorage.removeItem(STORAGE_KEY)
      setCart(null)
      return result
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
      throw e
    }
  }, [cart])

  return (
    <CartContext.Provider value={{ cart, loading, error, addItem, setItemQuantity, removeItem, applyCoupon, checkout }}>
      {children}
    </CartContext.Provider>
  )
}

export function useCart(): CartContextValue {
  const ctx = useContext(CartContext)
  if (!ctx) throw new Error('useCart must be used within a CartProvider')
  return ctx
}
