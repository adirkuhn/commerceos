import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from 'react'
import type { Cart, StorefrontAPI } from '../../api/storefront'

export interface CartState {
  /** null until loaded in the browser (the server never knows the cart). */
  cart: Cart | null
  busy: boolean
  error: string
  add: (skuCode: string, quantity: number) => Promise<void>
  setQuantity: (skuCode: string, quantity: number) => Promise<void>
  applyCoupon: (code: string) => Promise<void>
  /** Forgets the cart after checkout; the next add starts a new one. */
  clear: () => void
}

const Ctx = createContext<CartState | null>(null)

import { readToken as read, writeToken as write } from './storage'

const readToken = (storeKey: string) => read('cart', storeKey)
const writeToken = (storeKey: string, token: string | null) => write('cart', storeKey, token)

/** The shopper's cart, kept by token in this browser. Loads after mount,
 *  so server and first client render agree (both: no cart yet). */
export function CartProvider({ api, storeKey, children }: { api: StorefrontAPI; storeKey: string; children: ReactNode }) {
  const [cart, setCart] = useState<Cart | null>(null)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    const token = readToken(storeKey)
    if (!token) {
      setCart({ token: '', items: [], subtotal_minor_units: 0, discount_minor_units: 0 })
      return
    }
    api.getCart(token).then(setCart, () => {
      writeToken(storeKey, null) // expired or checked out: start fresh
      setCart({ token: '', items: [], subtotal_minor_units: 0, discount_minor_units: 0 })
    })
  }, [api, storeKey])

  const run = useCallback(async (fn: (token: string) => Promise<Cart>) => {
    setBusy(true)
    setError('')
    try {
      let token = cart?.token || readToken(storeKey)
      if (!token) {
        token = (await api.createCart()).token
        writeToken(storeKey, token)
      }
      setCart(await fn(token))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Something went wrong')
    } finally {
      setBusy(false)
    }
  }, [api, cart, storeKey])

  const lines = useCallback(() => (cart?.items ?? []).map(i => ({ sku_code: i.sku_code, quantity: i.quantity })), [cart])

  const value = useMemo<CartState>(() => ({
    cart, busy, error,
    add: (sku, qty) => run(token => {
      const current = lines()
      const existing = current.find(l => l.sku_code === sku)
      const next = existing ? current.map(l => (l.sku_code === sku ? { ...l, quantity: l.quantity + qty } : l)) : [...current, { sku_code: sku, quantity: qty }]
      return api.setCartItems(token, next)
    }),
    setQuantity: (sku, qty) => run(token => api.setCartItems(token, lines().map(l => (l.sku_code === sku ? { ...l, quantity: qty } : l)).filter(l => l.quantity > 0))),
    applyCoupon: code => run(token => api.applyCoupon(token, code)),
    clear: () => {
      writeToken(storeKey, null)
      setCart({ token: '', items: [], subtotal_minor_units: 0, discount_minor_units: 0 })
    },
  }), [api, busy, cart, error, lines, run, storeKey])

  return <Ctx.Provider value={value}>{children}</Ctx.Provider>
}

export const useCart = () => useContext(Ctx)
