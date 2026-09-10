import { describe, expect, it } from 'vitest'
import { buildStarterFooterBlocks, buildStarterHeaderBlocks, buildStarterHomeBlocks } from '../starterTemplate'

describe('buildStarterHomeBlocks', () => {
  it('returns a hero, an about blurb, and the product catalog, in reading order', () => {
    const blocks = buildStarterHomeBlocks('Cantinho Brasileiro')

    expect(blocks.map(b => b.type)).toEqual(['hero', 'rich_text', 'product_grid'])
    // Reading order top-to-bottom -- each block starts below the one before it.
    for (let i = 1; i < blocks.length; i++) {
      expect(blocks[i].layout.desktop.y).toBeGreaterThan(blocks[i - 1].layout.desktop.y)
    }
  })

  it('uses the given store name in the hero heading and the about blurb', () => {
    const blocks = buildStarterHomeBlocks('Cantinho Brasileiro')

    expect(blocks[0].props.heading).toBe('Cantinho Brasileiro')
    expect(blocks[1].props.content).toContain('Cantinho Brasileiro')
  })

  it('falls back to generic copy when no store name is given', () => {
    const blocks = buildStarterHomeBlocks('')

    expect(blocks[0].props.heading).toBe('Welcome to our store')
  })

  it('gives every block a unique id', () => {
    const blocks = buildStarterHomeBlocks('Store')

    expect(new Set(blocks.map(b => b.id)).size).toBe(blocks.length)
  })

  it('replaces the single product grid with one Featured Products section per real category, when any exist', () => {
    const blocks = buildStarterHomeBlocks('Store', [{ id: 'cat-1', name: 'Bebidas' }, { id: 'cat-2', name: 'Snacks' }])

    expect(blocks.map(b => b.type)).toEqual(['hero', 'rich_text', 'featured_products', 'featured_products'])
    expect(blocks[2].props).toMatchObject({ title: 'Bebidas', categoryId: 'cat-1' })
    expect(blocks[3].props).toMatchObject({ title: 'Snacks', categoryId: 'cat-2' })
  })

  it('caps category sections at 6, not one per category for a huge catalog', () => {
    const categories = Array.from({ length: 10 }, (_, i) => ({ id: `cat-${i}`, name: `Category ${i}` }))
    const blocks = buildStarterHomeBlocks('Store', categories)

    expect(blocks.filter(b => b.type === 'featured_products')).toHaveLength(6)
  })
})

describe('buildStarterHeaderBlocks', () => {
  it('returns a logo (hero) and a nav menu of generic links when there are no categories yet', () => {
    const blocks = buildStarterHeaderBlocks('Cantinho Brasileiro')

    expect(blocks.map(b => b.type)).toEqual(['hero', 'nav_menu'])
    expect(blocks[0].props.heading).toBe('Cantinho Brasileiro')
    expect(blocks[1].props.links).toEqual([
      { label: 'Products', url: '/products' },
      { label: 'Track order', url: '/order-lookup' },
    ])
  })

  it('links the nav to the tenant\'s own real categories when any exist', () => {
    const blocks = buildStarterHeaderBlocks('Store', [{ id: 'cat-1', name: 'Bebidas' }, { id: 'cat-2', name: 'Snacks' }])

    expect(blocks[1].props.links).toEqual([
      { label: 'Bebidas', url: '/products?category=cat-1' },
      { label: 'Snacks', url: '/products?category=cat-2' },
    ])
  })

  it('falls back to generic copy when no store name is given', () => {
    const blocks = buildStarterHeaderBlocks('')

    expect(blocks[0].props.heading).toBe('Store')
  })
})

describe('buildStarterFooterBlocks', () => {
  it('returns a tagline, a legal nav menu, and a generic contact/social info block, not any real business\' details', () => {
    const blocks = buildStarterFooterBlocks('Cantinho Brasileiro')

    expect(blocks.map(b => b.type)).toEqual(['rich_text', 'nav_menu', 'custom_html'])
    expect(blocks[0].props.content).toContain('Cantinho Brasileiro')
    expect(blocks[1].props.links).toEqual([
      { label: 'Privacy policy', url: '/' },
      { label: 'Terms & conditions', url: '/' },
      { label: 'Returns & refunds', url: '/' },
    ])
    expect(blocks[2].props.html).toContain('email@example.com')
    expect(blocks[2].props.html).not.toContain('cantinhobrasileironl')
  })

  it('gives every block a unique id', () => {
    const blocks = buildStarterFooterBlocks('Store')

    expect(new Set(blocks.map(b => b.id)).size).toBe(blocks.length)
  })
})
