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
vi.mock('../../../api/reviews', () => ({ listReviews: vi.fn(), submitReview: vi.fn() }))

import * as productsApi from '../../../api/products'
import * as cartApi from '../../../api/cart'
import * as reviewsApi from '../../../api/reviews'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  vi.mocked(reviewsApi.listReviews).mockResolvedValue({ reviews: [], summary: { average: 0, count: 0 } })
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

describe('ProductDetailsBlock reviews', () => {
  it('shows the review summary and existing reviews', async () => {
    vi.mocked(productsApi.getProduct).mockResolvedValue(product)
    vi.mocked(reviewsApi.listReviews).mockResolvedValue({
      reviews: [{ id: 'review-1', customer_name: 'Ada', rating: 5, title: 'Great', body: 'Loved it', verified: true, created_at: '2026-06-01T00:00:00Z' }],
      summary: { average: 5, count: 1 },
    })

    renderAt('p1')

    expect(await screen.findByText('Ada')).toBeInTheDocument()
    expect(screen.getByText('Loved it')).toBeInTheDocument()
    expect(screen.getByText('Verified purchase')).toBeInTheDocument()
    expect(screen.getByText(/5.0 out of 5/)).toBeInTheDocument()
  })

  it('shows an empty state with no reviews yet', async () => {
    vi.mocked(productsApi.getProduct).mockResolvedValue(product)

    renderAt('p1')

    expect(await screen.findByText(/No reviews yet/)).toBeInTheDocument()
  })

  it('submits a review and refreshes the list', async () => {
    const user = userEvent.setup()
    vi.mocked(productsApi.getProduct).mockResolvedValue(product)
    vi.mocked(reviewsApi.submitReview).mockResolvedValue({
      id: 'review-2', customer_name: 'Bob', rating: 4, title: '', body: 'Good stuff', verified: false, created_at: '2026-06-02T00:00:00Z',
    })

    renderAt('p1')
    await screen.findByText(/No reviews yet/)

    await user.click(screen.getByRole('button', { name: 'Write a review' }))
    await user.type(screen.getByLabelText('Name'), 'Bob')
    await user.type(screen.getByLabelText('Email'), 'bob@example.com')
    await user.type(screen.getByLabelText('Review'), 'Good stuff')

    vi.mocked(reviewsApi.listReviews).mockResolvedValue({
      reviews: [{ id: 'review-2', customer_name: 'Bob', rating: 5, title: '', body: 'Good stuff', verified: false, created_at: '2026-06-02T00:00:00Z' }],
      summary: { average: 5, count: 1 },
    })
    await user.click(screen.getByRole('button', { name: 'Submit review' }))

    await waitFor(() => expect(reviewsApi.submitReview).toHaveBeenCalledWith('p1', expect.objectContaining({
      customer_name: 'Bob', customer_email: 'bob@example.com', body: 'Good stuff',
    })))
    expect(await screen.findByText('Bob')).toBeInTheDocument()
  })

  it('shows an error when submission fails', async () => {
    const user = userEvent.setup()
    vi.mocked(productsApi.getProduct).mockResolvedValue(product)
    vi.mocked(reviewsApi.submitReview).mockRejectedValue(new Error('rating must be between 1 and 5'))

    renderAt('p1')
    await screen.findByText(/No reviews yet/)

    await user.click(screen.getByRole('button', { name: 'Write a review' }))
    await user.type(screen.getByLabelText('Name'), 'Bob')
    await user.type(screen.getByLabelText('Email'), 'bob@example.com')
    await user.type(screen.getByLabelText('Review'), 'Good stuff')
    await user.click(screen.getByRole('button', { name: 'Submit review' }))

    expect(await screen.findByText('rating must be between 1 and 5')).toBeInTheDocument()
  })
})
