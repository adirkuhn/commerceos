// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { buildPuckConfig } from '../config'

vi.mock('../../components/blocks/ProductGridBlock', () => ({ ProductGridBlock: () => <div>product-grid-marker</div> }))
vi.mock('../../components/blocks/FeaturedProductsBlock', () => ({
  FeaturedProductsBlock: (props: {
    title: string; subtitle?: string; categoryId?: string; productIds?: string[]; excludeProductIds?: string[]
    limit: number; itemsPerRowDesktop: number; itemsPerRowMobile: number
  }) => <div data-testid="featured-products-marker" data-props={JSON.stringify(props)} />,
}))
vi.mock('../../components/blocks/ProductDetailsBlock', () => ({ ProductDetailsBlock: () => <div>product-details-marker</div> }))
vi.mock('../../components/blocks/CartSummaryBlock', () => ({ CartSummaryBlock: () => <div>cart-summary-marker</div> }))
vi.mock('../../components/blocks/CheckoutFormBlock', () => ({ CheckoutFormBlock: () => <div>checkout-form-marker</div> }))
vi.mock('../../components/blocks/OrderStatusBlock', () => ({ OrderStatusBlock: ({ pageSlug }: { pageSlug: string }) => <div>order-status-marker:{pageSlug}</div> }))
vi.mock('../../components/blocks/WishlistBlock', () => ({ WishlistBlock: () => <div>wishlist-marker</div> }))
vi.mock('../../components/blocks/LoginFormBlock', () => ({ LoginFormBlock: () => <div>login-form-marker</div> }))
vi.mock('../../components/blocks/AccountBlock', () => ({ AccountBlock: () => <div>account-marker</div> }))
vi.mock('../../components/blocks/NavMenuBlock', () => ({
  NavMenuBlock: ({ links }: { links: { label: string; url: string }[] }) => <div data-testid="nav-menu-marker" data-links={JSON.stringify(links)} />,
}))
vi.mock('../../components/blocks/NavActionsBlock', () => ({ NavActionsBlock: () => <div>nav-actions-marker</div> }))

describe('buildPuckConfig', () => {
  it('registers every storefront block type as a Puck component', () => {
    const config = buildPuckConfig('home')
    expect(Object.keys(config.components ?? {}).sort()).toEqual([
      'Account', 'CartSummary', 'CheckoutForm', 'CustomHTML', 'FeaturedProducts', 'Hero', 'LoginForm', 'NavActions', 'NavMenu',
      'OrderStatus', 'ProductDetails', 'ProductGrid', 'RichText', 'Wishlist',
    ])
  })

  it('renders Hero with its own editable heading/subheading', () => {
    const config = buildPuckConfig('home')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<>{(config.components!.Hero.render as any)({ heading: 'Big sale', subheading: 'This week only', imageUrl: '' })}</>)

    expect(screen.getByText('Big sale')).toBeInTheDocument()
    expect(screen.getByText('This week only')).toBeInTheDocument()
  })

  it('renders Hero\'s CTA button only when both text and link are set', () => {
    const config = buildPuckConfig('home')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<>{(config.components!.Hero.render as any)({ heading: 'Sale', ctaText: 'Shop now', ctaUrl: '/products' })}</>)

    expect(screen.getByRole('link', { name: 'Shop now' })).toHaveAttribute('href', '/products')
  })

  it('renders the real, live FeaturedProducts component with every prop, parsing comma-separated id lists', () => {
    const config = buildPuckConfig('home')
    render(<>{(config.components!.FeaturedProducts.render as
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      any)({
      title: 'Bebidas', subtitle: 'Sucos', categoryId: 'cat-1', productIds: ' p1, p2 ,', excludeProductIds: 'p3',
      limit: 4, itemsPerRowDesktop: 3, itemsPerRowMobile: 1,
    })}</>)

    const props = JSON.parse(screen.getByTestId('featured-products-marker').dataset.props!)
    expect(props).toEqual({
      title: 'Bebidas', subtitle: 'Sucos', categoryId: 'cat-1', productIds: ['p1', 'p2'], excludeProductIds: ['p3'],
      limit: 4, itemsPerRowDesktop: 3, itemsPerRowMobile: 1,
    })
  })

  it('FeaturedProducts falls back to sensible defaults for missing/empty fields', () => {
    const config = buildPuckConfig('home')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<>{(config.components!.FeaturedProducts.render as any)({ title: 'Snacks', subtitle: '', categoryId: '', productIds: '', excludeProductIds: '' })}</>)

    const props = JSON.parse(screen.getByTestId('featured-products-marker').dataset.props!)
    expect(props).toEqual({
      title: 'Snacks', limit: 8, itemsPerRowDesktop: 4, itemsPerRowMobile: 2,
    })
  })

  it('builds FeaturedProducts\' category field options from the given categories, plus "All categories"', () => {
    const config = buildPuckConfig('home', [{ id: 'cat-1', name: 'Bebidas' }, { id: 'cat-2', name: 'Snacks' }])

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const categoryField = (config.components!.FeaturedProducts.fields as any).categoryId
    expect(categoryField.options).toEqual([
      { label: 'All categories', value: '' },
      { label: 'Bebidas', value: 'cat-1' },
      { label: 'Snacks', value: 'cat-2' },
    ])
  })

  it('renders RichText content verbatim', () => {
    const config = buildPuckConfig('home')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<>{(config.components!.RichText.render as any)({ content: 'Hand-authored copy' })}</>)

    expect(screen.getByText('Hand-authored copy')).toBeInTheDocument()
  })

  it('renders CustomHTML as real HTML, not escaped text', () => {
    const config = buildPuckConfig('home')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const { container } = render(<>{(config.components!.CustomHTML.render as any)({ html: '<strong>bold</strong>' })}</>)

    expect(container.querySelector('strong')).toBeInTheDocument()
  })

  it('renders the real, live ProductGrid component (not a placeholder)', () => {
    const config = buildPuckConfig('home')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<>{(config.components!.ProductGrid.render as any)({})}</>)

    expect(screen.getByText('product-grid-marker')).toBeInTheDocument()
  })

  it('passes order-confirmation to OrderStatus on any slug except order-lookup', () => {
    const config = buildPuckConfig('order-confirmation')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<>{(config.components!.OrderStatus.render as any)({})}</>)

    expect(screen.getByText('order-status-marker:order-confirmation')).toBeInTheDocument()
  })

  it('passes order-lookup to OrderStatus when editing the order-lookup page', () => {
    const config = buildPuckConfig('order-lookup')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<>{(config.components!.OrderStatus.render as any)({})}</>)

    expect(screen.getByText('order-status-marker:order-lookup')).toBeInTheDocument()
  })

  it('renders NavMenu with its links, defaulting to an empty list', () => {
    const config = buildPuckConfig('header')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<>{(config.components!.NavMenu.render as any)({ links: [{ label: 'Bebidas', url: '/products?category=cat-1' }] })}</>)

    expect(screen.getByTestId('nav-menu-marker').dataset.links).toBe('[{"label":"Bebidas","url":"/products?category=cat-1"}]')

    render(<>{(config.components!.NavMenu.render as any)({ links: undefined })}</>)
    expect(screen.getAllByTestId('nav-menu-marker')[1].dataset.links).toBe('[]')
  })

  it('NavMenu\'s array field has label/url subfields and a summary derived from the label', () => {
    const config = buildPuckConfig('header')

    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const linksField = (config.components!.NavMenu.fields as any).links
    expect(linksField.type).toBe('array')
    expect(Object.keys(linksField.arrayFields)).toEqual(['label', 'url'])
    expect(linksField.getItemSummary({ label: 'Bebidas' })).toBe('Bebidas')
    expect(linksField.getItemSummary({ label: '' })).toBe('Link')
  })

  it('renders the real, live NavActions component', () => {
    const config = buildPuckConfig('header')
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    render(<>{(config.components!.NavActions.render as any)({})}</>)

    expect(screen.getByText('nav-actions-marker')).toBeInTheDocument()
  })
})
