// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { BlockRenderer } from '../BlockRenderer'
import type { Block, PageSlug } from '../../types'

vi.mock('../blocks/ProductGridBlock', () => ({ ProductGridBlock: () => <div>product-grid-block</div> }))
vi.mock('../blocks/FeaturedProductsBlock', () => ({
  FeaturedProductsBlock: (props: {
    title: string; subtitle?: string; categoryId?: string; productIds?: string[]; excludeProductIds?: string[]
    limit: number; itemsPerRowDesktop: number; itemsPerRowMobile: number
  }) => <div data-testid="featured-products-block" data-props={JSON.stringify(props)} />,
}))
vi.mock('../blocks/ProductDetailsBlock', () => ({ ProductDetailsBlock: () => <div>product-details-block</div> }))
vi.mock('../blocks/CartSummaryBlock', () => ({ CartSummaryBlock: () => <div>cart-summary-block</div> }))
vi.mock('../blocks/CheckoutFormBlock', () => ({ CheckoutFormBlock: () => <div>checkout-form-block</div> }))
vi.mock('../blocks/OrderStatusBlock', () => ({ OrderStatusBlock: ({ pageSlug }: { pageSlug: string }) => <div>order-status-block:{pageSlug}</div> }))
vi.mock('../blocks/NavMenuBlock', () => ({
  NavMenuBlock: ({ links }: { links: { label: string; url: string }[] }) => <div>nav-menu-block:{JSON.stringify(links)}</div>,
}))
vi.mock('../blocks/NavActionsBlock', () => ({ NavActionsBlock: () => <div>nav-actions-block</div> }))

function block(overrides: Partial<Block>): Block {
  return {
    id: 'b1',
    type: 'hero',
    layout: { desktop: { x: 0, y: 0, width: 100, height: 100 }, mobile: null },
    props: {},
    ...overrides,
  }
}

function renderBlock(b: Block, pageSlug: PageSlug = 'home') {
  return render(<MemoryRouter><BlockRenderer block={b} pageSlug={pageSlug} /></MemoryRouter>)
}

describe('BlockRenderer', () => {
  it("renders a hero block's heading and subheading", () => {
    renderBlock(block({ type: 'hero', props: { heading: 'Big sale', subheading: '50% off' } }))

    expect(screen.getByText('Big sale')).toBeInTheDocument()
    expect(screen.getByText('50% off')).toBeInTheDocument()
  })

  it("renders a hero block's CTA button, only when both text and link are set", () => {
    renderBlock(block({ type: 'hero', props: { heading: 'Sale', ctaText: 'Shop now', ctaUrl: '/products' } }))

    expect(screen.getByRole('link', { name: 'Shop now' })).toHaveAttribute('href', '/products')
  })

  it("omits a hero block's CTA button when only one of text/link is set", () => {
    renderBlock(block({ type: 'hero', props: { heading: 'Sale', ctaText: 'Shop now' } }))

    expect(screen.queryByRole('link')).not.toBeInTheDocument()
  })

  it('renders featured_products with every prop passed through', () => {
    renderBlock(block({
      type: 'featured_products',
      props: {
        title: 'Bebidas', subtitle: 'Sucos', categoryId: 'cat-1', productIds: ['p1', 'p2'], excludeProductIds: ['p3'],
        limit: 4, itemsPerRowDesktop: 3, itemsPerRowMobile: 1,
      },
    }))
    const props = JSON.parse(screen.getByTestId('featured-products-block').dataset.props!)
    expect(props).toEqual({
      title: 'Bebidas', subtitle: 'Sucos', categoryId: 'cat-1', productIds: ['p1', 'p2'], excludeProductIds: ['p3'],
      limit: 4, itemsPerRowDesktop: 3, itemsPerRowMobile: 1,
    })
  })

  it('renders featured_products with sensible defaults when only title is set', () => {
    renderBlock(block({ type: 'featured_products', props: { title: 'Snacks' } }))
    const props = JSON.parse(screen.getByTestId('featured-products-block').dataset.props!)
    expect(props).toEqual({
      title: 'Snacks', subtitle: undefined, categoryId: undefined, productIds: undefined, excludeProductIds: undefined,
      limit: 8, itemsPerRowDesktop: 4, itemsPerRowMobile: 2,
    })
  })

  it("renders a rich_text block's content", () => {
    renderBlock(block({ type: 'rich_text', props: { content: 'Hello shoppers' } }))

    expect(screen.getByText('Hello shoppers')).toBeInTheDocument()
  })

  it("renders a custom_html block's markup", () => {
    renderBlock(block({ type: 'custom_html', props: { html: '<strong>Sale!</strong>' } }))

    expect(screen.getByText('Sale!').tagName).toBe('STRONG')
  })

  it('dispatches each functional block type to its real component', () => {
    renderBlock(block({ type: 'product_grid' }))
    expect(screen.getByText('product-grid-block')).toBeInTheDocument()
  })

  it('dispatches order_status with the current page slug', () => {
    renderBlock(block({ type: 'order_status' }), 'order-lookup')
    expect(screen.getByText('order-status-block:order-lookup')).toBeInTheDocument()
  })

  it('renders nav_menu with its links prop, ignoring malformed entries', () => {
    renderBlock(block({ type: 'nav_menu', props: { links: [{ label: 'Bebidas', url: '/products?category=cat-1' }, 'not-a-link', { label: 42 }] } }))
    expect(screen.getByText('nav-menu-block:[{"label":"Bebidas","url":"/products?category=cat-1"}]')).toBeInTheDocument()
  })

  it('dispatches nav_actions to the real, live component', () => {
    renderBlock(block({ type: 'nav_actions' }))
    expect(screen.getByText('nav-actions-block')).toBeInTheDocument()
  })
})
