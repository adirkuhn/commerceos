// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { rendererFor } from '../renderers'
import { RenderProvider } from '../../runtime/RenderContext'
import { ApiProvider } from '../../api/ApiContext'
import { StorefrontAPI } from '../../api/storefront'
import { productGridQuery } from '../data'
import { ProductTile } from '../commerce/ProductTile'

function wrap(el: ReactNode, api = new StorefrontAPI({ baseURL: 'x', storeKey: 'k', fetch: vi.fn() })) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  return render(<QueryClientProvider client={qc}><ApiProvider api={api}><RenderProvider value={{ storePath: p => `/store/k/${p}` }}>{el}</RenderProvider></ApiProvider></QueryClientProvider>)
}

const product = { id: 'p1', name: 'Cheese bread', slug: 'cheese-bread', category_id: 'c1', tags: [], image_url: null,
  price_minor_units: 500, currency_code: 'EUR', sku_code: 'CB', sku_count: 1, in_stock: true, category_name: 'Bakery' }

describe('product tiles', () => {
  it('can show the category and a small "+ Add" beside the price', () => {
    const { container } = wrap(<ProductTile product={product} showPrice showAddToCart showCategory addButton="compact" />)
    expect(screen.getByText('Bakery')).toHaveClass('cos-product-category')
    expect(screen.getByRole('button').textContent).toMatch(/^\+\s*Add$/)
    expect(container.querySelector('.cos-tile-compact')).not.toBeNull()
  })

  it('keeps the wide "Add to cart" and no category by default', () => {
    wrap(<ProductTile product={product} showPrice showAddToCart />)
    expect(screen.queryByText('Bakery')).toBeNull()
    expect(screen.getByRole('button')).toHaveTextContent('Add to cart')
  })
})

describe('product grid data', () => {
  it('adds category names when the grid shows them', async () => {
    const api = new StorefrontAPI({ baseURL: 'x', storeKey: 'k', fetch: vi.fn() })
    api.listProducts = vi.fn(async () => ({ items: [{ ...product, category_name: undefined }], total: 1 }))
    api.listCategories = vi.fn(async () => [{ id: 'c1', parent_id: null, name: 'Bakery', slug: 'bakery', position: 0 }])
    const props = { source: 'all', limit: 4, show_category: true } as never
    const list = await new QueryClient().fetchQuery(productGridQuery(api, props))
    expect(list.items[0].category_name).toBe('Bakery')

    api.listCategories = vi.fn()
    await new QueryClient().fetchQuery(productGridQuery(api, { source: 'all', limit: 4 } as never))
    expect(api.listCategories).not.toHaveBeenCalled()
  })
})

describe('icon', () => {
  it('draws the chosen line icon, decorative only', () => {
    const Icon = rendererFor('icon')!
    const { container } = wrap(<Icon node={{ id: 'i', type: 'icon' }} props={{ name: 'clock', size: 'lg' }} className="n-i" />)
    const el = container.querySelector('.cos-icon')!
    expect(el).toHaveClass('cos-icon-lg')
    expect(el).toHaveAttribute('aria-hidden', 'true')
    expect(el.querySelector('svg circle')).not.toBeNull()
  })
})
