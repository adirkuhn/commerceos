// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import type { ReactNode } from 'react'
import { rendererFor } from '../renderers'
import { registry } from '../registry'
import { RenderProvider, type RenderEnv } from '../../runtime/RenderContext'
import { ApiProvider } from '../../api/ApiContext'
import { StorefrontAPI, type ProductSummary } from '../../api/storefront'

const ASSET = '0b0c7d1e-6f7a-4b8c-9d0e-1f2a3b4c5d6e'
const assets = { [ASSET]: { url: 'https://cdn.test/hero.jpg', width: 1200, height: 600, alt: 'Hero' } }

function renderWith(el: ReactNode, env: Partial<RenderEnv> = {}, api?: StorefrontAPI) {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  const store = api ?? new StorefrontAPI({ baseURL: 'x', storeKey: 'k', fetch: vi.fn() })
  return render(
    <QueryClientProvider client={qc}>
      <ApiProvider api={store}>
        <RenderProvider value={{ assets, storePath: p => `/store/k/${p}`, ...env }}>{el}</RenderProvider>
      </ApiProvider>
    </QueryClientProvider>,
  )
}

const R = (type: string) => rendererFor(type)!

describe('image', () => {
  it('renders the referenced asset with its size and alt', () => {
    const Image = R('image')
    const { container } = renderWith(<Image node={{ id: 'i', type: 'image' }} props={{ image: `asset:${ASSET}`, alt: '', link: '/pages/about', fit: 'cover' }} className="n-i" />)
    const img = container.querySelector('img')!
    expect(img).toHaveAttribute('src', 'https://cdn.test/hero.jpg')
    expect(img).toHaveAttribute('alt', 'Hero')
    expect(img).toHaveAttribute('width', '1200')
    expect(container.querySelector('a')).toHaveAttribute('href', '/pages/about')
  })

  it('renders nothing live and a placeholder in the editor when no image is chosen', () => {
    const Image = R('image')
    const props = { image: '', alt: '', link: '', fit: 'cover' }
    expect(renderWith(<Image node={{ id: 'i', type: 'image' }} props={props} className="" />).container.innerHTML).toBe('')
    expect(renderWith(<Image node={{ id: 'i', type: 'image' }} props={props} className="" />, { mode: 'editor' }).container.textContent).toBe('Choose an image')
  })

  it('never links to unsafe targets', () => {
    const Image = R('image')
    const { container } = renderWith(<Image node={{ id: 'i', type: 'image' }} props={{ image: `asset:${ASSET}`, alt: '', link: 'javascript:alert(1)', fit: 'cover' }} className="" />)
    expect(container.querySelector('a')).toBeNull()
  })
})

describe('button', () => {
  it('links live, with new-tab safety', () => {
    const Button = R('button')
    renderWith(<Button node={{ id: 'b', type: 'button' }} props={{ label: 'Shop', link: 'https://x.test', variant: 'outline', new_tab: true }} className="n-b" />)
    const a = screen.getByText('Shop')
    expect(a).toHaveAttribute('href', 'https://x.test')
    expect(a).toHaveAttribute('rel', 'noopener noreferrer')
    expect(a.className).toContain('cos-button-outline')
  })

  it('replaces unsafe links and is inert in the editor', () => {
    const Button = R('button')
    const props = { label: 'Go', link: 'javascript:alert(1)', variant: 'primary', new_tab: false }
    expect(renderWith(<Button node={{ id: 'b', type: 'button' }} props={props} className="" />).getByText('Go')).toHaveAttribute('href', '#')
    expect(renderWith(<Button node={{ id: 'b2', type: 'button' }} props={props} className="" />, { mode: 'editor' }).getAllByText('Go')[1]).not.toHaveAttribute('href')
  })
})

describe('layout', () => {
  it('columns and column carry their layout classes', () => {
    expect(registry.get('columns')!.element!({ stack_on: 'tablet' }).className).toBe('cos-columns cos-stack-tablet')
    expect(registry.get('column')!.element!({ span: 9 }).className).toBe('cos-column cos-span-4')
  })

  it('spacer uses the spacing scale', () => {
    const Spacer = R('spacer')
    const { container } = renderWith(<Spacer node={{ id: 's', type: 'spacer' }} props={{ size: 'bogus' }} className="" />)
    expect(container.firstElementChild!.className).toContain('cos-spacer-lg')
  })
})

const product = (o: Partial<ProductSummary>): ProductSummary => ({
  id: 'p1', name: 'Red shoes', slug: 'red-shoes', category_id: null, tags: [], image_url: null,
  price_minor_units: 2999, sale_price_minor_units: null, currency_code: 'EUR', ...o,
})

function apiReturning(items: ProductSummary[]) {
  const listProducts = vi.fn(async () => ({ items, total: items.length }))
  return { api: Object.assign(new StorefrontAPI({ baseURL: 'x', storeKey: 'k' }), { listProducts }), listProducts }
}

describe('product grid', () => {
  it('shows products in the merchant\'s chosen order with prices and links', async () => {
    const { api, listProducts } = apiReturning([product({ id: 'p1' }), product({ id: 'p2', name: 'Blue hat', slug: 'blue-hat', sale_price_minor_units: 1500 })])
    const Grid = R('product-grid')
    renderWith(<Grid node={{ id: 'g', type: 'product-grid' }} props={{ source: 'products', product_ids: ['p2', 'p1'], limit: 8, columns: 3, show_price: true }} className="n-g" />, {}, api)

    const links = await screen.findAllByRole('link')
    expect(links.map(l => l.textContent)).toEqual([expect.stringContaining('Blue hat'), expect.stringContaining('Red shoes')])
    expect(links[0]).toHaveAttribute('href', '/store/k/products/blue-hat')
    expect(links[0].textContent).toContain('€ 15,00')
    expect(listProducts).toHaveBeenCalledWith({ productIds: ['p2', 'p1'], perPage: 2 })
  })

  it('does not call the API for an unconfigured source and explains it in the editor', async () => {
    const { api, listProducts } = apiReturning([])
    const Grid = R('product-grid')
    renderWith(<Grid node={{ id: 'g', type: 'product-grid' }} props={{ source: 'category', category_id: '', product_ids: [], limit: 8, columns: 0, show_price: true }} className="" />, { mode: 'editor' }, api)
    expect(await screen.findByText(/No products to show yet/)).toBeInTheDocument()
    expect(listProducts).not.toHaveBeenCalled()
  })
})

describe('product card', () => {
  it('shows the chosen product', async () => {
    const { api } = apiReturning([product({})])
    const Card = R('product-card')
    renderWith(<Card node={{ id: 'c', type: 'product-card' }} props={{ product_id: 'p1', show_price: false }} className="" />, {}, api)
    expect(await screen.findByText('Red shoes')).toBeInTheDocument()
    expect(screen.queryByText(/€/)).toBeNull()
  })
})
