// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { WishlistBlock } from '../WishlistBlock'
import { WishlistProvider } from '../../../wishlist/WishlistContext'
import type { StorefrontProduct } from '../../../types'

vi.mock('../../../api/products', () => ({ listProducts: vi.fn(), getProduct: vi.fn() }))
vi.mock('../../../api/wishlist', () => ({ createWishlist: vi.fn(), getWishlist: vi.fn(), toggleWishlistItem: vi.fn() }))

import * as productsApi from '../../../api/products'
import * as wishlistApi from '../../../api/wishlist'

const STORAGE_KEY = 'commerceos_wishlist_token'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

function renderBlock() {
  return render(<MemoryRouter><WishlistProvider><WishlistBlock /></WishlistProvider></MemoryRouter>)
}

const product: StorefrontProduct = {
  id: 'p1', slug: 'p1', name: 'Widget', description: '', category_id: null, tags: [], images: [],
  variants: [{ id: 'v1', name: 'Default', skus: [{ id: 's1', sku_code: 'SKU-1', price_minor_units: 1000, currency_code: 'EUR', stock_quantity: 5 }] }],
}

describe('WishlistBlock', () => {
  it('shows an empty state with no stored wishlist', async () => {
    renderBlock()
    expect(await screen.findByText('Your wishlist is empty.')).toBeInTheDocument()
  })

  it('resolves and lists every saved product', async () => {
    localStorage.setItem(STORAGE_KEY, 'wishlist-1')
    vi.mocked(wishlistApi.getWishlist).mockResolvedValue({ token: 'wishlist-1', product_ids: ['p1'] })
    vi.mocked(productsApi.getProduct).mockResolvedValue(product)

    renderBlock()

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(screen.getByText('€10.00')).toBeInTheDocument()
  })

  it('removing an item calls toggle and drops it from the list', async () => {
    const user = userEvent.setup()
    localStorage.setItem(STORAGE_KEY, 'wishlist-1')
    vi.mocked(wishlistApi.getWishlist).mockResolvedValue({ token: 'wishlist-1', product_ids: ['p1'] })
    vi.mocked(productsApi.getProduct).mockResolvedValue(product)
    vi.mocked(wishlistApi.toggleWishlistItem).mockResolvedValue({ token: 'wishlist-1', product_ids: [] })

    renderBlock()
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Remove' }))

    await waitFor(() => expect(wishlistApi.toggleWishlistItem).toHaveBeenCalledWith('wishlist-1', 'p1'))
    expect(await screen.findByText('Your wishlist is empty.')).toBeInTheDocument()
  })

  it('skips a product id that no longer resolves', async () => {
    localStorage.setItem(STORAGE_KEY, 'wishlist-1')
    vi.mocked(wishlistApi.getWishlist).mockResolvedValue({ token: 'wishlist-1', product_ids: ['missing-product'] })
    vi.mocked(productsApi.getProduct).mockRejectedValue(new Error('not found'))

    renderBlock()

    expect(await screen.findByText('Your wishlist is empty.')).toBeInTheDocument()
  })
})
