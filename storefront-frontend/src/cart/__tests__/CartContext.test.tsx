// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { CartProvider, useCart } from '../CartContext'
import type { Cart } from '../../types'

vi.mock('../../api/cart', () => ({
  createCart: vi.fn(),
  getCart: vi.fn(),
  updateCartItems: vi.fn(),
  checkoutCart: vi.fn(),
  applyCoupon: vi.fn(),
}))

import * as cartApi from '../../api/cart'

const STORAGE_KEY = 'commerceos_cart_token'

function Harness() {
  const { cart, addItem, setItemQuantity, checkout, error } = useCart()
  return (
    <div>
      <p data-testid="items">{cart ? JSON.stringify(cart.items) : 'no cart'}</p>
      {error && <p data-testid="error">{error}</p>}
      <button onClick={() => addItem('SKU-1', 1)}>Add SKU-1</button>
      <button onClick={() => setItemQuantity('SKU-1', 0)}>Remove SKU-1</button>
      <button onClick={() => checkout({
        email: 'a@b.com',
        billing: { first_name: 'A', last_name: 'B', address1: 'St', city: 'C', postcode: '1', country: 'NL' },
        shipping: { first_name: 'A', last_name: 'B', address1: 'St', city: 'C', postcode: '1', country: 'NL' },
      })}>
        Checkout
      </button>
    </div>
  )
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

describe('CartContext', () => {
  it('starts empty with no stored token', async () => {
    render(<CartProvider><Harness /></CartProvider>)
    await waitFor(() => expect(screen.getByTestId('items')).toHaveTextContent('no cart'))
    expect(cartApi.getCart).not.toHaveBeenCalled()
  })

  it('creates a cart lazily on the first addItem and persists the token', async () => {
    const user = userEvent.setup()
    const created: Cart = { token: 'cart-1', items: [], subtotal_minor_units: 0, discount_minor_units: 0 }
    const updated: Cart = { token: 'cart-1', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 1, price_minor_units: 500, currency_code: 'EUR' }], subtotal_minor_units: 500, discount_minor_units: 0 }
    vi.mocked(cartApi.createCart).mockResolvedValue(created)
    vi.mocked(cartApi.updateCartItems).mockResolvedValue(updated)

    render(<CartProvider><Harness /></CartProvider>)
    await waitFor(() => expect(screen.getByTestId('items')).toHaveTextContent('no cart'))

    await user.click(screen.getByText('Add SKU-1'))

    await waitFor(() => expect(screen.getByTestId('items')).toHaveTextContent('SKU-1'))
    expect(cartApi.updateCartItems).toHaveBeenCalledWith('cart-1', [{ sku_code: 'SKU-1', quantity: 1 }])
    expect(localStorage.getItem(STORAGE_KEY)).toBe('cart-1')
  })

  it('loads the existing cart on mount when a token is already stored', async () => {
    localStorage.setItem(STORAGE_KEY, 'cart-9')
    const existing: Cart = { token: 'cart-9', items: [{ sku_code: 'SKU-2', product_name: 'Gizmo', quantity: 2, price_minor_units: 200, currency_code: 'EUR' }], subtotal_minor_units: 400, discount_minor_units: 0 }
    vi.mocked(cartApi.getCart).mockResolvedValue(existing)

    render(<CartProvider><Harness /></CartProvider>)

    await waitFor(() => expect(screen.getByTestId('items')).toHaveTextContent('SKU-2'))
    expect(cartApi.getCart).toHaveBeenCalledWith('cart-9')
  })

  it('drops an item once its quantity reaches zero', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-9')
    const existing: Cart = { token: 'cart-9', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 1, price_minor_units: 500, currency_code: 'EUR' }], subtotal_minor_units: 500, discount_minor_units: 0 }
    vi.mocked(cartApi.getCart).mockResolvedValue(existing)
    vi.mocked(cartApi.updateCartItems).mockResolvedValue({ token: 'cart-9', items: [], subtotal_minor_units: 0, discount_minor_units: 0 })

    render(<CartProvider><Harness /></CartProvider>)
    await waitFor(() => expect(screen.getByTestId('items')).toHaveTextContent('SKU-1'))

    await user.click(screen.getByText('Remove SKU-1'))

    await waitFor(() => expect(cartApi.updateCartItems).toHaveBeenCalledWith('cart-9', []))
  })

  it('clears the cart and stored token after checkout', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-9')
    const existing: Cart = { token: 'cart-9', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 1, price_minor_units: 500, currency_code: 'EUR' }], subtotal_minor_units: 500, discount_minor_units: 0 }
    vi.mocked(cartApi.getCart).mockResolvedValue(existing)
    vi.mocked(cartApi.checkoutCart).mockResolvedValue({ order_number: 'WEB-1', total_minor_units: 500, currency_code: 'EUR' })

    render(<CartProvider><Harness /></CartProvider>)
    await waitFor(() => expect(screen.getByTestId('items')).toHaveTextContent('SKU-1'))

    await user.click(screen.getByText('Checkout'))

    await waitFor(() => expect(screen.getByTestId('items')).toHaveTextContent('no cart'))
    expect(localStorage.getItem(STORAGE_KEY)).toBeNull()
  })
})
