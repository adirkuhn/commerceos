import { describe, it, expect, vi, beforeEach } from 'vitest'
import { createCart, getCart, updateCartItems, checkoutCart, createPaymentIntent } from '../cart'
import type { APIResponse, Cart, Checkout, PaymentIntent } from '../../types'

function mockFetch<T>(data: T, ok = true, status = 200) {
  const body: APIResponse<T> = { data: ok ? data : null, error: ok ? null : { code: 'ERR', message: 'failed' } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok, status, json: () => Promise.resolve(body) }))
}

beforeEach(() => { vi.restoreAllMocks() })

const sampleCart: Cart = { token: 'cart-1', items: [], subtotal_minor_units: 0, discount_minor_units: 0 }

describe('cart api', () => {
  it('createCart POSTs to /carts', async () => {
    mockFetch(sampleCart)
    const result = await createCart()
    expect(result).toEqual(sampleCart)
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/storefront/v1/carts')
    expect(options?.method).toBe('POST')
  })

  it('getCart fetches by token', async () => {
    mockFetch(sampleCart)
    await getCart('cart-1')
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/carts/cart-1')
  })

  it('updateCartItems PUTs the whole line list', async () => {
    mockFetch(sampleCart)
    await updateCartItems('cart-1', [{ sku_code: 'SKU-1', quantity: 2 }])
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/carts/cart-1/items')
    expect(options?.method).toBe('PUT')
    expect(JSON.parse(options?.body as string)).toEqual({ items: [{ sku_code: 'SKU-1', quantity: 2 }] })
  })

  it('checkoutCart POSTs the payload and returns the order', async () => {
    const checkout: Checkout = { order_number: 'WEB-1', total_minor_units: 1000, currency_code: 'EUR' }
    mockFetch(checkout)
    const payload = {
      email: 'shopper@example.com',
      billing: { first_name: 'A', last_name: 'B', address1: 'St 1', city: 'C', postcode: '1000', country: 'NL' },
      shipping: { first_name: 'A', last_name: 'B', address1: 'St 1', city: 'C', postcode: '1000', country: 'NL' },
    }
    const result = await checkoutCart('cart-1', payload)
    expect(result).toEqual(checkout)
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/carts/cart-1/checkout')
    expect(options?.method).toBe('POST')
  })

  it('createPaymentIntent POSTs the shipping address and returns the intent', async () => {
    const intent: PaymentIntent = { client_secret: 'pi_secret_1', publishable_key: 'pk_test_1', amount_minor_units: 1000, currency_code: 'EUR' }
    mockFetch(intent)
    const shipping = { first_name: 'A', last_name: 'B', address1: 'St 1', city: 'C', postcode: '1000', country: 'NL' }
    const result = await createPaymentIntent('cart-1', { shipping })
    expect(result).toEqual(intent)
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/carts/cart-1/payment-intent')
    expect(options?.method).toBe('POST')
    expect(JSON.parse(options?.body as string)).toEqual({ shipping })
  })

  it('throws with the server error message on failure', async () => {
    mockFetch(null, false, 409)
    await expect(getCart('cart-1')).rejects.toThrow('failed')
  })
})
