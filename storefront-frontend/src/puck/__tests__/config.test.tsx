// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { buildPuckConfig } from '../config'

vi.mock('../../components/blocks/ProductGridBlock', () => ({ ProductGridBlock: () => <div>product-grid-marker</div> }))
vi.mock('../../components/blocks/ProductDetailsBlock', () => ({ ProductDetailsBlock: () => <div>product-details-marker</div> }))
vi.mock('../../components/blocks/CartSummaryBlock', () => ({ CartSummaryBlock: () => <div>cart-summary-marker</div> }))
vi.mock('../../components/blocks/CheckoutFormBlock', () => ({ CheckoutFormBlock: () => <div>checkout-form-marker</div> }))
vi.mock('../../components/blocks/OrderStatusBlock', () => ({ OrderStatusBlock: ({ pageSlug }: { pageSlug: string }) => <div>order-status-marker:{pageSlug}</div> }))
vi.mock('../../components/blocks/WishlistBlock', () => ({ WishlistBlock: () => <div>wishlist-marker</div> }))
vi.mock('../../components/blocks/LoginFormBlock', () => ({ LoginFormBlock: () => <div>login-form-marker</div> }))
vi.mock('../../components/blocks/AccountBlock', () => ({ AccountBlock: () => <div>account-marker</div> }))

describe('buildPuckConfig', () => {
  it('registers every storefront block type as a Puck component', () => {
    const config = buildPuckConfig('home')
    expect(Object.keys(config.components ?? {}).sort()).toEqual([
      'Account', 'CartSummary', 'CheckoutForm', 'CustomHTML', 'Hero', 'LoginForm',
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
})
