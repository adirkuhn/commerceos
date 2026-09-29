// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { Suspense, type ReactNode } from 'react'
import { rendererFor } from '../renderers'
import { RenderProvider, type RenderEnv } from '../../runtime/RenderContext'
import { ApiProvider } from '../../api/ApiContext'
import { StorefrontAPI, type Cart, type ProductDetail } from '../../api/storefront'
import { ProductProvider } from '../../runtime/page/ProductContext'
import { ListingProvider, type ListingPage } from '../../runtime/page/ListingContext'
import { CartProvider } from '../../runtime/page/CartContext'

const product: ProductDetail = {
  id: 'p1', name: 'Red shoes', slug: 'red-shoes', description: '<p>Comfy &amp; red</p>', category_id: null, tags: [],
  images: ['https://cdn/1.jpg', 'https://cdn/2.jpg'],
  variants: [
    { id: 'v1', name: 'Size 40', skus: [{ id: 's1', sku_code: 'RS-40', price_minor_units: 5000, sale_price_minor_units: 4000, currency_code: 'EUR', stock_quantity: 0 }] },
    { id: 'v2', name: 'Size 41', skus: [{ id: 's2', sku_code: 'RS-41', price_minor_units: 5000, currency_code: 'EUR', stock_quantity: 3 }] },
  ],
}

function fakeAPI() {
  const api = new StorefrontAPI({ baseURL: 'x', storeKey: 'k' })
  let cart: Cart = { token: 'tok', items: [], subtotal_minor_units: 0, discount_minor_units: 0 }
  api.createCart = vi.fn(async () => cart)
  api.getCart = vi.fn(async () => cart)
  api.setCartItems = vi.fn(async (_t: string, items: { sku_code: string; quantity: number }[]) => {
    cart = { ...cart, items: items.map(i => ({ ...i, product_name: 'Red shoes', price_minor_units: 5000, currency_code: 'EUR' })), subtotal_minor_units: items.reduce((s, i) => s + i.quantity * 5000, 0) }
    return cart
  })
  api.listReviews = vi.fn(async () => ({ reviews: [{ id: 'r1', customer_name: 'Ann', rating: 4, title: 'Nice', body: 'Good fit', verified: true, created_at: '2026-09-01T00:00:00Z' }], summary: { average: 4, count: 1 } }))
  api.getRecommendations = vi.fn(async () => [])
  return api
}

function setup(el: ReactNode, opts: { env?: Partial<RenderEnv>; api?: StorefrontAPI; product?: boolean; listing?: ListingPage } = {}) {
  const api = opts.api ?? fakeAPI()
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } })
  let tree = el
  if (opts.product) tree = <ProductProvider product={product}>{tree}</ProductProvider>
  if (opts.listing) tree = <ListingProvider value={opts.listing}>{tree}</ListingProvider>
  return {
    api,
    ...render(
      <QueryClientProvider client={qc}>
        <ApiProvider api={api}>
          <RenderProvider value={{ storePath: p => `/store/k/${p}`, ...opts.env }}>
            <CartProvider api={api} storeKey="k"><Suspense fallback={null}>{tree}</Suspense></CartProvider>
          </RenderProvider>
        </ApiProvider>
      </QueryClientProvider>,
    ),
  }
}

const R = (t: string) => rendererFor(t)!
const node = (type: string) => ({ id: type, type })

beforeEach(() => window.localStorage.clear())

describe('product page', () => {
  it('starts on the first variant in stock and shows its price', () => {
    const [Price, Variants] = [R('product-price'), R('variant-selector')]
    setup(<><Price node={node('product-price')} props={{ show_compare_at: true }} className="" /><Variants node={node('variant-selector')} props={{ style: 'buttons' }} className="" /></>, { product: true })
    expect(screen.getByRole('button', { name: 'Size 41' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByText('€50.00')).toBeInTheDocument()
  })

  it('shows a sale price and a sold-out add to cart for a sold-out variant', async () => {
    const [Price, Variants, Add] = [R('product-price'), R('variant-selector'), R('add-to-cart')]
    setup(<><Price node={node('product-price')} props={{ show_compare_at: true }} className="" /><Variants node={node('variant-selector')} props={{ style: 'buttons' }} className="" /><Add node={node('add-to-cart')} props={{ label: 'Add', sold_out_label: 'Gone' }} className="" /></>, { product: true })
    await userEvent.click(screen.getByRole('button', { name: 'Size 40' }))
    expect(screen.getByText('€40.00')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Gone' })).toBeDisabled()
  })

  it('adds the chosen variant and quantity to a new cart', async () => {
    const [Qty, Add] = [R('quantity-selector'), R('add-to-cart')]
    const { api } = setup(<><Qty node={node('quantity-selector')} props={{ max: 5 }} className="" /><Add node={node('add-to-cart')} props={{ label: 'Add', sold_out_label: '' }} className="" /></>, { product: true })
    await userEvent.click(screen.getByRole('button', { name: 'More' }))
    await userEvent.click(screen.getByRole('button', { name: 'Add' }))

    await waitFor(() => expect(screen.getByRole('status')).toHaveTextContent('Added to your cart.'))
    expect(api.createCart).toHaveBeenCalled()
    expect(api.setCartItems).toHaveBeenCalledWith('tok', [{ sku_code: 'RS-41', quantity: 2 }])
    expect(window.localStorage.getItem('cos-cart:k')).toBe('tok')
  })

  it('renders the description as text, never HTML', () => {
    const Desc = R('product-description')
    const { container } = setup(<Desc node={node('product-description')} props={{}} className="" />, { product: true })
    expect(container.querySelector('.cos-description')!.innerHTML).toBe('<p>Comfy &amp; red</p>')
  })

  it('switches gallery images', async () => {
    const Gallery = R('product-gallery')
    const { container } = setup(<Gallery node={node('product-gallery')} props={{ layout: 'thumbs_below' }} className="" />, { product: true })
    await userEvent.click(screen.getByRole('button', { name: 'Show image 2' }))
    expect(container.querySelector('.cos-gallery-main')).toHaveAttribute('src', 'https://cdn/2.jpg')
  })

  it('lists reviews with the rating summary', async () => {
    const Reviews = R('product-reviews')
    setup(<Reviews node={node('product-reviews')} props={{ title: 'Reviews', limit: 5 }} className="" />, { product: true })
    expect(await screen.findByText('Good fit')).toBeInTheDocument()
    expect(screen.getByText(/Verified buyer/)).toBeInTheDocument()
  })

  it('explains itself in the editor when there is no product yet', () => {
    const Title = R('product-title')
    setup(<Title node={node('product-title')} props={{ level: 1 }} className="" />, { env: { mode: 'editor' } })
    expect(screen.getByText(/product this page is about/)).toBeInTheDocument()
  })
})

const listing: ListingPage = {
  title: 'Summer', description: 'Hot stuff', imageUrl: null, page: 2, pageCount: 3, activeCategoryId: 'c1',
  listing: { items: [{ id: 'p1', name: 'Red shoes', slug: 'red-shoes', category_id: null, tags: [], image_url: null, price_minor_units: 5000, currency_code: 'EUR' }], total: 50,
    category_facets: [{ category_id: 'c1', category_name: 'Shoes', count: 12 }] },
  hrefForPage: p => `/c?page=${p}`, hrefForCategory: id => (id ? `/c?category=${id}` : '/c'),
}

describe('collection page', () => {
  it('renders header, products, filters and pagination from the listing', () => {
    const [Header, Products, Filters, Pages] = ['collection-header', 'collection-products', 'collection-filters', 'pagination'].map(R)
    setup(<>
      <Header node={node('h')} props={{ show_image: true, show_description: true }} className="" />
      <Products node={node('p')} props={{ per_page: 24, columns: 0, show_price: true }} className="" />
      <Filters node={node('f')} props={{ title: 'Filter' }} className="" />
      <Pages node={node('pg')} props={{}} className="" />
    </>, { listing })
    expect(screen.getByRole('heading', { name: 'Summer' })).toBeInTheDocument()
    expect(screen.getByText('Red shoes')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: /Shoes/ })).toHaveAttribute('aria-current', 'true')
    expect(screen.getByRole('link', { name: 'Next' })).toHaveAttribute('href', '/c?page=3')
    expect(screen.getByRole('link', { name: 'Previous' })).toHaveAttribute('href', '/c?page=1')
  })

  it('breadcrumbs follow the page context', () => {
    const Crumbs = R('breadcrumbs')
    setup(<Crumbs node={node('b')} props={{ home_label: 'Shop' }} className="" />, { listing })
    const nav = screen.getByRole('navigation', { name: 'Breadcrumb' })
    expect(within(nav).getByRole('link', { name: 'Shop' })).toHaveAttribute('href', '/store/k/')
    expect(within(nav).getByText('Summer')).toBeInTheDocument()
  })
})

describe('cart', () => {
  it('loads the stored cart, changes quantities and links to checkout', async () => {
    window.localStorage.setItem('cos-cart:k', 'tok')
    const api = fakeAPI()
    await api.setCartItems('tok', [{ sku_code: 'RS-41', quantity: 1 }])
    const [Items, Summary, Checkout] = ['cart-items', 'cart-summary', 'checkout-button'].map(R)
    setup(<>
      <Items node={node('i')} props={{ empty_message: 'Empty' }} className="" />
      <Summary node={node('s')} props={{ show_coupon: false }} className="" />
      <Checkout node={node('c')} props={{ label: 'Pay' }} className="" />
    </>, { api })

    expect(await screen.findByText('Red shoes')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Pay' })).toHaveAttribute('href', '/store/k/checkout')
    await userEvent.click(screen.getByRole('button', { name: 'More' }))
    await waitFor(() => expect(api.setCartItems).toHaveBeenLastCalledWith('tok', [{ sku_code: 'RS-41', quantity: 2 }]))
    expect(await screen.findAllByText('€100.00')).not.toHaveLength(0)
  })

  it('shows the empty message and disables checkout for an empty cart', async () => {
    const [Items, Checkout] = ['cart-items', 'checkout-button'].map(R)
    setup(<><Items node={node('i')} props={{ empty_message: 'Nothing here' }} className="" /><Checkout node={node('c')} props={{ label: 'Pay' }} className="" /></>)
    expect(await screen.findByText('Nothing here')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Pay' })).not.toHaveAttribute('href')
  })
})
