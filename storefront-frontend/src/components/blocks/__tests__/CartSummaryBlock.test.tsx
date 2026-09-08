// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { CartSummaryBlock } from '../CartSummaryBlock'
import { CartProvider } from '../../../cart/CartContext'
import type { Cart } from '../../../types'

vi.mock('../../../api/cart', () => ({ createCart: vi.fn(), getCart: vi.fn(), updateCartItems: vi.fn(), checkoutCart: vi.fn(), applyCoupon: vi.fn() }))

import * as cartApi from '../../../api/cart'

const STORAGE_KEY = 'commerceos_cart_token'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

function renderBlock() {
  return render(<MemoryRouter><CartProvider><CartSummaryBlock /></CartProvider></MemoryRouter>)
}

describe('CartSummaryBlock', () => {
  it('shows an empty-cart message with no stored cart', async () => {
    renderBlock()
    expect(await screen.findByText('Your cart is empty.')).toBeInTheDocument()
  })

  it("lists the cart's items and subtotal", async () => {
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    const cart: Cart = { token: 'cart-1', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 2, price_minor_units: 1000, currency_code: 'EUR' }], subtotal_minor_units: 2000, discount_minor_units: 0 }
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)

    renderBlock()

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.getAllByText('€20.00')).toHaveLength(3) // line total + subtotal + total, same amount with no discount
  })

  it('removing an item calls updateCartItems with it dropped', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    const cart: Cart = { token: 'cart-1', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 1, price_minor_units: 1000, currency_code: 'EUR' }], subtotal_minor_units: 1000, discount_minor_units: 0 }
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(cartApi.updateCartItems).mockResolvedValue({ token: 'cart-1', items: [], subtotal_minor_units: 0, discount_minor_units: 0 })

    renderBlock()
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Remove SKU-1' }))

    await waitFor(() => expect(cartApi.updateCartItems).toHaveBeenCalledWith('cart-1', []))
  })

  it('applying a coupon code shows the discount and updated total', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    const cart: Cart = { token: 'cart-1', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 1, price_minor_units: 1000, currency_code: 'EUR' }], subtotal_minor_units: 1000, discount_minor_units: 0 }
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(cartApi.applyCoupon).mockResolvedValue({ ...cart, coupon_code: 'SUMMER10', discount_minor_units: 100 })

    renderBlock()
    await screen.findByText('Widget')

    await user.type(screen.getByLabelText('Coupon code'), 'summer10')
    await user.click(screen.getByRole('button', { name: 'Apply' }))

    await waitFor(() => expect(cartApi.applyCoupon).toHaveBeenCalledWith('cart-1', 'summer10'))
    expect(await screen.findByText('SUMMER10')).toBeInTheDocument()
    expect(screen.getByText('-€1.00')).toBeInTheDocument()
    expect(screen.getByText('€9.00')).toBeInTheDocument()
  })

  it('shows an error when the coupon code is rejected', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    const cart: Cart = { token: 'cart-1', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 1, price_minor_units: 1000, currency_code: 'EUR' }], subtotal_minor_units: 1000, discount_minor_units: 0 }
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(cartApi.applyCoupon).mockRejectedValue(new Error('coupon is not redeemable'))

    renderBlock()
    await screen.findByText('Widget')

    await user.type(screen.getByLabelText('Coupon code'), 'EXPIRED')
    await user.click(screen.getByRole('button', { name: 'Apply' }))

    expect(await screen.findAllByText('coupon is not redeemable')).not.toHaveLength(0)
  })

  it('removing an applied coupon calls applyCoupon with an empty code', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    const cart: Cart = {
      token: 'cart-1', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 1, price_minor_units: 1000, currency_code: 'EUR' }],
      subtotal_minor_units: 1000, coupon_code: 'SUMMER10', discount_minor_units: 100,
    }
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(cartApi.applyCoupon).mockResolvedValue({ ...cart, coupon_code: null, discount_minor_units: 0 })

    renderBlock()
    await screen.findByText('SUMMER10')

    await user.click(screen.getByRole('button', { name: 'Remove' }))

    await waitFor(() => expect(cartApi.applyCoupon).toHaveBeenCalledWith('cart-1', ''))
  })
})
