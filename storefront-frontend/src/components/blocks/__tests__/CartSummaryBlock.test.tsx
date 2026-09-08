// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { CartSummaryBlock } from '../CartSummaryBlock'
import { CartProvider } from '../../../cart/CartContext'
import type { Cart } from '../../../types'

vi.mock('../../../api/cart', () => ({ createCart: vi.fn(), getCart: vi.fn(), updateCartItems: vi.fn(), checkoutCart: vi.fn() }))

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
    const cart: Cart = { token: 'cart-1', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 2, price_minor_units: 1000, currency_code: 'EUR' }], subtotal_minor_units: 2000 }
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)

    renderBlock()

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.getAllByText('€20.00')).toHaveLength(2) // line total + subtotal, same amount for a single line
  })

  it('removing an item calls updateCartItems with it dropped', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'cart-1')
    const cart: Cart = { token: 'cart-1', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 1, price_minor_units: 1000, currency_code: 'EUR' }], subtotal_minor_units: 1000 }
    vi.mocked(cartApi.getCart).mockResolvedValue(cart)
    vi.mocked(cartApi.updateCartItems).mockResolvedValue({ token: 'cart-1', items: [], subtotal_minor_units: 0 })

    renderBlock()
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Remove SKU-1' }))

    await waitFor(() => expect(cartApi.updateCartItems).toHaveBeenCalledWith('cart-1', []))
  })
})
