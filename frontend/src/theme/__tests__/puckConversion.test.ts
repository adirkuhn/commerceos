import { describe, expect, it } from 'vitest'
import { blocksToPuckData } from '../puckConversion'
import type { Block } from '../types'

function block(type: Block['type'], props: Record<string, unknown> = {}, id = 'b1'): Block {
  return { id, type, layout: { desktop: { x: 0, y: 0, width: 100, height: 100 }, mobile: null }, props }
}

describe('blocksToPuckData', () => {
  it('maps each block type to its registered Puck component name', () => {
    const blocks: Block[] = [
      block('hero', {}, 'b1'), block('product_grid', {}, 'b2'), block('featured_products', {}, 'b3'),
      block('rich_text', {}, 'b4'), block('custom_html', {}, 'b5'), block('nav_menu', {}, 'b6'),
      block('nav_actions', {}, 'b7'),
    ]

    const data = blocksToPuckData(blocks)

    expect((data.content as { type: string }[]).map(c => c.type)).toEqual([
      'Hero', 'ProductGrid', 'FeaturedProducts', 'RichText', 'CustomHTML', 'NavMenu', 'NavActions',
    ])
  })

  it('carries the block id into props.id, and passes hero/rich_text/custom_html/nav_menu props through unchanged', () => {
    const blocks: Block[] = [
      block('hero', { heading: 'Big sale', ctaText: 'Shop', ctaUrl: '/products' }, 'hero-1'),
      block('nav_menu', { links: [{ label: 'Products', url: '/products' }] }, 'nav-1'),
    ]

    const data = blocksToPuckData(blocks)
    const content = data.content as { type: string; props: Record<string, unknown> }[]

    expect(content[0].props).toMatchObject({ id: 'hero-1', heading: 'Big sale', ctaText: 'Shop', ctaUrl: '/products' })
    expect(content[1].props).toMatchObject({ id: 'nav-1', links: [{ label: 'Products', url: '/products' }] })
  })

  it('joins featured_products id-list arrays into comma-separated strings -- Puck\'s own field shape', () => {
    const blocks: Block[] = [
      block('featured_products', { title: 'Bebidas', productIds: ['p1', 'p2'], excludeProductIds: ['p9'] }),
    ]

    const data = blocksToPuckData(blocks)
    const props = (data.content as { props: Record<string, unknown> }[])[0].props

    expect(props.productIds).toBe('p1, p2')
    expect(props.excludeProductIds).toBe('p9')
  })

  it('gives a functional block with no props an empty-but-present props object plus its id', () => {
    const data = blocksToPuckData([block('cart_summary', {}, 'cart-1')])
    const props = (data.content as { props: Record<string, unknown> }[])[0].props

    expect(props).toEqual({ id: 'cart-1' })
  })

  it('preserves block order (Puck has no position concept, only document order)', () => {
    const blocks: Block[] = [block('hero', {}, 'first'), block('rich_text', {}, 'second'), block('product_grid', {}, 'third')]

    const data = blocksToPuckData(blocks)

    expect((data.content as { props: { id: string } }[]).map(c => c.props.id)).toEqual(['first', 'second', 'third'])
  })

  it('returns an empty root, and an empty content array for no blocks', () => {
    const data = blocksToPuckData([])

    expect(data).toEqual({ content: [], root: {} })
  })
})
