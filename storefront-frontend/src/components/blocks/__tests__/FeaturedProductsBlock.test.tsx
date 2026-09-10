// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { FeaturedProductsBlock } from '../FeaturedProductsBlock'
import { WishlistProvider } from '../../../wishlist/WishlistContext'
import type { StorefrontProductListResult } from '../../../types'

vi.mock('../../../api/products', () => ({ listProducts: vi.fn(), getProduct: vi.fn() }))
vi.mock('../../../api/wishlist', () => ({ createWishlist: vi.fn(), getWishlist: vi.fn(), toggleWishlistItem: vi.fn() }))

import * as productsApi from '../../../api/products'

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
})

interface RenderProps {
  title: string
  subtitle?: string
  categoryId?: string
  productIds?: string[]
  excludeProductIds?: string[]
  limit: number
  itemsPerRowDesktop?: number
  itemsPerRowMobile?: number
}

function renderBlock(props: RenderProps) {
  return render(<MemoryRouter><WishlistProvider><FeaturedProductsBlock {...props} /></WishlistProvider></MemoryRouter>)
}

describe('FeaturedProductsBlock', () => {
  it('shows its own title and subtitle above the product list', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({ items: [], total: 0, category_facets: [] })

    renderBlock({ title: 'Bebidas', subtitle: 'Sucos e refrigerantes', limit: 8 })

    expect(await screen.findByRole('heading', { name: 'Bebidas' })).toBeInTheDocument()
    expect(screen.getByText('Sucos e refrigerantes')).toBeInTheDocument()
  })

  it('category mode: fetches by category, no search/category sidebar', async () => {
    const result: StorefrontProductListResult = {
      items: [{ id: 'p1', slug: 'p1', name: 'Guaraná', category_id: 'cat-1', tags: [], image_url: null, price_minor_units: 349, currency_code: 'EUR' }],
      total: 1,
      category_facets: [],
    }
    vi.mocked(productsApi.listProducts).mockResolvedValue(result)

    renderBlock({ title: 'Bebidas', categoryId: 'cat-1', limit: 4 })

    expect(await screen.findByText('Guaraná')).toBeInTheDocument()
    expect(productsApi.listProducts).toHaveBeenCalledWith({
      categoryId: 'cat-1', productIds: undefined, excludeIds: undefined, perPage: 4,
    })
    expect(screen.queryByLabelText('Search products')).not.toBeInTheDocument()
  })

  it('manual mode: productIds wins over categoryId', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({ items: [], total: 0, category_facets: [] })

    renderBlock({ title: 'Curated', categoryId: 'cat-1', productIds: ['p1', 'p2'], limit: 4 })

    await waitFor(() => expect(productsApi.listProducts).toHaveBeenCalledWith({
      categoryId: undefined, productIds: ['p1', 'p2'], excludeIds: undefined, perPage: 4,
    }))
  })

  it('passes excludeProductIds through regardless of mode', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({ items: [], total: 0, category_facets: [] })

    renderBlock({ title: 'Bebidas', categoryId: 'cat-1', excludeProductIds: ['p9'], limit: 4 })

    await waitFor(() => expect(productsApi.listProducts).toHaveBeenCalledWith({
      categoryId: 'cat-1', productIds: undefined, excludeIds: ['p9'], perPage: 4,
    }))
  })

  it('re-fetches when the category or limit prop changes', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({ items: [], total: 0, category_facets: [] })

    const { rerender } = renderBlock({ title: 'Bebidas', categoryId: 'cat-1', limit: 4 })
    await screen.findByText('No products found.')

    rerender(<MemoryRouter><WishlistProvider><FeaturedProductsBlock title="Bebidas" categoryId="cat-2" limit={4} /></WishlistProvider></MemoryRouter>)

    await waitFor(() => expect(productsApi.listProducts).toHaveBeenLastCalledWith({
      categoryId: 'cat-2', productIds: undefined, excludeIds: undefined, perPage: 4,
    }))
  })

  it('shows an empty state when there are no products', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({ items: [], total: 0, category_facets: [] })

    renderBlock({ title: 'Bebidas', limit: 8 })

    expect(await screen.findByText('No products found.')).toBeInTheDocument()
  })

  it('applies items-per-row as grid column classes, clamped to a safe range', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({
      items: [{ id: 'p1', slug: 'p1', name: 'Widget', category_id: null, tags: [], image_url: null }],
      total: 1,
      category_facets: [],
    })

    renderBlock({ title: 'Bebidas', limit: 4, itemsPerRowDesktop: 3, itemsPerRowMobile: 2 })

    const grid = (await screen.findByText('Widget')).closest('.grid')
    expect(grid).toHaveClass('grid-cols-2', 'sm:grid-cols-3')
  })

  it('clamps out-of-range items-per-row to the supported 1-6 / 1-3 bounds', async () => {
    vi.mocked(productsApi.listProducts).mockResolvedValue({
      items: [{ id: 'p1', slug: 'p1', name: 'Widget', category_id: null, tags: [], image_url: null }],
      total: 1,
      category_facets: [],
    })

    renderBlock({ title: 'Bebidas', limit: 4, itemsPerRowDesktop: 99, itemsPerRowMobile: 0 })

    const grid = (await screen.findByText('Widget')).closest('.grid')
    expect(grid).toHaveClass('grid-cols-1', 'sm:grid-cols-6')
  })
})
