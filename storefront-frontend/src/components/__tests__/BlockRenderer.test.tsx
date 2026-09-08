// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'
import { BlockRenderer } from '../BlockRenderer'
import type { Block, PageSlug } from '../../types'

vi.mock('../blocks/ProductGridBlock', () => ({ ProductGridBlock: () => <div>product-grid-block</div> }))
vi.mock('../blocks/ProductDetailsBlock', () => ({ ProductDetailsBlock: () => <div>product-details-block</div> }))
vi.mock('../blocks/CartSummaryBlock', () => ({ CartSummaryBlock: () => <div>cart-summary-block</div> }))
vi.mock('../blocks/CheckoutFormBlock', () => ({ CheckoutFormBlock: () => <div>checkout-form-block</div> }))
vi.mock('../blocks/OrderStatusBlock', () => ({ OrderStatusBlock: ({ pageSlug }: { pageSlug: string }) => <div>order-status-block:{pageSlug}</div> }))

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
})
