// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { ProductDetailsBlock } from '../ProductDetailsBlock'
import { CartProvider } from '../../../cart/CartContext'
import type { StorefrontProduct, Cart } from '../../../types'

vi.mock('../../../api/products', () => ({ listProducts: vi.fn(), getProduct: vi.fn() }))
vi.mock('../../../api/cart', () => ({ createCart: vi.fn(), getCart: vi.fn(), updateCartItems: vi.fn(), checkoutCart: vi.fn(), applyCoupon: vi.fn() }))

import * as productsApi from '../../../api/products'
import * as cartApi from '../../../api/cart'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

function renderAt(id: string) {
  return render(
    <MemoryRouter initialEntries={[`/products/${id}`]}>
      <CartProvider>
        <Routes>
          <Route path="/products/:id" element={<ProductDetailsBlock />} />
        </Routes>
      </CartProvider>
    </MemoryRouter>,
  )
}

const product: StorefrontProduct = {
  id: 'p1', slug: 'p1', name: 'Widget', description: 'A fine widget', category_id: null, tags: [], images: [],
  variants: [{ id: 'v1', name: 'Default', skus: [{ id: 's1', sku_code: 'SKU-1', price_minor_units: 1000, currency_code: 'EUR', stock_quantity: 5 }] }],
}

describe('ProductDetailsBlock', () => {
  it("renders the product's name, description and SKUs", async () => {
    vi.mocked(productsApi.getProduct).mockResolvedValue(product)

    renderAt('p1')

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.getByText('A fine widget')).toBeInTheDocument()
    expect(screen.getByText('SKU-1')).toBeInTheDocument()
  })

  it('adds the SKU to the cart and shows it was added', async () => {
    const user = userEvent.setup()
    vi.mocked(productsApi.getProduct).mockResolvedValue(product)
    vi.mocked(cartApi.createCart).mockResolvedValue({ token: 'cart-1', items: [], subtotal_minor_units: 0, discount_minor_units: 0 })
    const updated: Cart = { token: 'cart-1', items: [{ sku_code: 'SKU-1', product_name: 'Widget', quantity: 1, price_minor_units: 1000, currency_code: 'EUR' }], subtotal_minor_units: 1000, discount_minor_units: 0 }
    vi.mocked(cartApi.updateCartItems).mockResolvedValue(updated)

    renderAt('p1')
    await screen.findByText('SKU-1')

    await user.click(screen.getByRole('button', { name: 'Add to cart' }))

    await waitFor(() => expect(screen.getByRole('button', { name: 'Added' })).toBeInTheDocument())
    expect(cartApi.updateCartItems).toHaveBeenCalledWith('cart-1', [{ sku_code: 'SKU-1', quantity: 1 }])
  })

  it('disables the button for an out-of-stock SKU', async () => {
    vi.mocked(productsApi.getProduct).mockResolvedValue({
      ...product,
      variants: [{ id: 'v1', name: 'Default', skus: [{ ...product.variants[0].skus[0], stock_quantity: 0 }] }],
    })

    renderAt('p1')

    expect(await screen.findByRole('button', { name: 'Out of stock' })).toBeDisabled()
  })
})
