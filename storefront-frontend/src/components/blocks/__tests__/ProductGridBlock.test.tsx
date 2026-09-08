// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { ProductGridBlock } from '../ProductGridBlock'
import { WishlistProvider } from '../../../wishlist/WishlistContext'
import type { StorefrontProductListResult } from '../../../types'

vi.mock('../../../api/products', () => ({ listProducts: vi.fn(), getProduct: vi.fn() }))
vi.mock('../../../api/wishlist', () => ({ createWishlist: vi.fn(), getWishlist: vi.fn(), toggleWishlistItem: vi.fn() }))

import * as productsApi from '../../../api/products'
import * as wishlistApi from '../../../api/wishlist'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

function renderBlock() {
  return render(<MemoryRouter><WishlistProvider><ProductGridBlock /></WishlistProvider></MemoryRouter>)
}

describe('ProductGridBlock', () => {
  it('renders each enrolled product with its price', async () => {
    const result: StorefrontProductListResult = {
      items: [{ id: 'p1', slug: 'p1', name: 'Widget', category_id: null, tags: [], image_url: null, price_minor_units: 1999, currency_code: 'EUR' }],
      total: 1,
      category_facets: [],
    }
    vi.mocked(productsApi.listProducts).mockResolvedValue(result)

    renderBlock()

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.getByText('€19.99')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Widget/ })).toHaveAttribute('href', '/products/p1')
  })

  it('shows a sale price struck through against the original', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({
      items: [{ id: 'p1', slug: 'p1', name: 'Widget', category_id: null, tags: [], image_url: null, price_minor_units: 2000, sale_price_minor_units: 1500, currency_code: 'EUR' }],
      total: 1,
      category_facets: [],
    })

    renderBlock()

    expect(await screen.findByText('€20.00')).toBeInTheDocument()
    expect(screen.getByText('€15.00')).toBeInTheDocument()
  })

  it('shows an empty state when there are no products', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({ items: [], total: 0, category_facets: [] })

    renderBlock()

    expect(await screen.findByText('No products found.')).toBeInTheDocument()
  })

  it('toggles a product onto the wishlist without navigating', async () => {
    const user = userEvent.setup()
    vi.mocked(productsApi.listProducts).mockResolvedValue({
      items: [{ id: 'p1', slug: 'p1', name: 'Widget', category_id: null, tags: [], image_url: null, price_minor_units: 1999, currency_code: 'EUR' }],
      total: 1,
      category_facets: [],
    })
    vi.mocked(wishlistApi.createWishlist).mockResolvedValue({ token: 'wishlist-1', product_ids: [] })
    vi.mocked(wishlistApi.toggleWishlistItem).mockResolvedValue({ token: 'wishlist-1', product_ids: ['p1'] })

    renderBlock()
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Save to wishlist' }))

    expect(wishlistApi.toggleWishlistItem).toHaveBeenCalledWith('wishlist-1', 'p1')
    expect(await screen.findByRole('button', { name: 'Remove from wishlist' })).toBeInTheDocument()
  })

  it('shows category facets with counts and re-fetches when one is clicked', async () => {
    const user = userEvent.setup()
    vi.mocked(productsApi.listProducts).mockResolvedValue({
      items: [{ id: 'p1', slug: 'p1', name: 'Widget', category_id: null, tags: [], image_url: null, price_minor_units: 1999, currency_code: 'EUR' }],
      total: 1,
      category_facets: [{ category_id: 'cat-1', category_name: 'Shoes', count: 3 }],
    })

    renderBlock()

    expect(await screen.findByText('Shoes (3)')).toBeInTheDocument()

    await user.click(screen.getByText('Shoes (3)'))

    await waitFor(() => expect(productsApi.listProducts).toHaveBeenLastCalledWith(
      expect.objectContaining({ categoryId: 'cat-1' }),
    ))
  })

  it('searches products by typed text', async () => {
    const user = userEvent.setup()
    vi.mocked(productsApi.listProducts).mockResolvedValue({ items: [], total: 0, category_facets: [] })

    renderBlock()
    await screen.findByText('No products found.')

    await user.type(screen.getByLabelText('Search products'), 'widget{Enter}')

    await waitFor(() => expect(productsApi.listProducts).toHaveBeenLastCalledWith(
      expect.objectContaining({ search: 'widget' }),
    ))
  })
})
