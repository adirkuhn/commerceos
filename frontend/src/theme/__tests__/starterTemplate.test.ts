import { describe, expect, it } from 'vitest'
import {
  buildStarterFooterBlocks, buildStarterHeaderBlocks, buildStarterHomeBlocks,
  buildEditorialFooterBlocks, buildEditorialHeaderBlocks, buildEditorialHomeBlocks,
  buildMercadoFooterBlocks, buildMercadoHeaderBlocks, buildMercadoHomeBlocks,
  STARTER_TEMPLATES, STARTER_TEMPLATE_CSS, combinedStarterCSS,
} from '../starterTemplate'

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

describe('buildEditorialHomeBlocks', () => {
  it('returns a hero, an about blurb, and the product catalog, in reading order', () => {
    const blocks = buildEditorialHomeBlocks('Cantinho Brasileiro')

    expect(blocks.map(b => b.type)).toEqual(['hero', 'rich_text', 'product_grid'])
    for (let i = 1; i < blocks.length; i++) {
      expect(blocks[i].layout.desktop.y).toBeGreaterThan(blocks[i - 1].layout.desktop.y)
    }
  })

  it('lays out category sections two-up instead of stacked full-width', () => {
    const blocks = buildEditorialHomeBlocks('Store', [{ id: 'cat-1', name: 'Bebidas' }, { id: 'cat-2', name: 'Snacks' }])
    const [first, second] = blocks.filter(b => b.type === 'featured_products')

    expect(first.layout.desktop.y).toBe(second.layout.desktop.y)
    expect(first.layout.desktop.x).toBeLessThan(second.layout.desktop.x)
    expect(first.layout.desktop.width).toBeLessThan(1440)
  })

  it('caps category sections at 6', () => {
    const categories = Array.from({ length: 10 }, (_, i) => ({ id: `cat-${i}`, name: `Category ${i}` }))
    const blocks = buildEditorialHomeBlocks('Store', categories)

    expect(blocks.filter(b => b.type === 'featured_products')).toHaveLength(6)
  })
})

describe('buildEditorialHeaderBlocks', () => {
  it('returns a logo and a nav menu, linking to real categories when any exist', () => {
    const blocks = buildEditorialHeaderBlocks('Store', [{ id: 'cat-1', name: 'Bebidas' }])

    expect(blocks.map(b => b.type)).toEqual(['hero', 'nav_menu'])
    expect(blocks[1].props.links).toEqual([{ label: 'Bebidas', url: '/products?category=cat-1' }])
  })
})

describe('buildEditorialFooterBlocks', () => {
  it('returns a tagline, a newsletter block, and a legal nav menu', () => {
    const blocks = buildEditorialFooterBlocks('Cantinho Brasileiro')

    expect(blocks.map(b => b.type)).toEqual(['rich_text', 'custom_html', 'nav_menu'])
    expect(blocks[0].props.content).toContain('Cantinho Brasileiro')
    expect(blocks[2].props.links).toEqual([
      { label: 'Privacy policy', url: '/' },
      { label: 'Terms & conditions', url: '/' },
      { label: 'Returns & refunds', url: '/' },
    ])
  })
})

describe('combinedStarterCSS', () => {
  it('appends nothing for classic -- palette CSS unchanged', () => {
    expect(combinedStarterCSS(':root { --brand-500: #000; }', 'classic')).toBe(':root { --brand-500: #000; }')
  })

  it('appends serif headings + softened corners for editorial, so the two templates look visually distinct even with identical block content', () => {
    const result = combinedStarterCSS(':root { --brand-500: #000; }', 'editorial')

    expect(result).toContain(':root { --brand-500: #000; }')
    expect(result).toContain('serif')
    expect(result).toContain('border-radius')
    expect(result).not.toBe(STARTER_TEMPLATE_CSS.classic)
    expect(result).not.toBe('')
  })

  it('classic and editorial produce different CSS for the same palette', () => {
    const palette = ':root { --brand-500: #1f8a3d; }'
    expect(combinedStarterCSS(palette, 'classic')).not.toBe(combinedStarterCSS(palette, 'editorial'))
  })
})

describe('buildMercadoHomeBlocks', () => {
  it('returns a hero, a floating badge, and the product catalog, in reading order', () => {
    const blocks = buildMercadoHomeBlocks('Cantinho Brasileiro')

    expect(blocks.map(b => b.type)).toEqual(['hero', 'custom_html', 'product_grid', 'custom_html', 'custom_html', 'custom_html'])
  })

  it('lays out category sections three-up', () => {
    const blocks = buildMercadoHomeBlocks('Store', [
      { id: 'cat-1', name: 'Bebidas' }, { id: 'cat-2', name: 'Snacks' }, { id: 'cat-3', name: 'Doces' },
    ])
    const featured = blocks.filter(b => b.type === 'featured_products')

    expect(featured).toHaveLength(3)
    expect(featured[0].layout.desktop.y).toBe(featured[1].layout.desktop.y)
    expect(featured[0].layout.desktop.y).toBe(featured[2].layout.desktop.y)
    expect(featured[0].layout.desktop.x).toBeLessThan(featured[1].layout.desktop.x)
    expect(featured[1].layout.desktop.x).toBeLessThan(featured[2].layout.desktop.x)
  })

  it('caps category sections at 6', () => {
    const categories = Array.from({ length: 10 }, (_, i) => ({ id: `cat-${i}`, name: `Category ${i}` }))
    const blocks = buildMercadoHomeBlocks('Store', categories)

    expect(blocks.filter(b => b.type === 'featured_products')).toHaveLength(6)
  })

  it('always ends with two promo banners and a trust-badge row, below the catalog', () => {
    const blocks = buildMercadoHomeBlocks('Store', [{ id: 'cat-1', name: 'Bebidas' }])
    const trailing = blocks.slice(-3)

    expect(trailing.every(b => b.type === 'custom_html')).toBe(true)
    const catalogBottom = Math.max(...blocks.filter(b => b.type === 'featured_products').map(b => b.layout.desktop.y + b.layout.desktop.height))
    expect(trailing[0].layout.desktop.y).toBeGreaterThanOrEqual(catalogBottom)
  })
})

describe('buildMercadoHeaderBlocks', () => {
  it('returns an announcement bar, a logo, and a nav menu, in that order', () => {
    const blocks = buildMercadoHeaderBlocks('Store', [{ id: 'cat-1', name: 'Bebidas' }])

    expect(blocks.map(b => b.type)).toEqual(['custom_html', 'hero', 'nav_menu'])
    expect(blocks[2].props.links).toEqual([{ label: 'Bebidas', url: '/products?category=cat-1' }])
  })
})

describe('buildMercadoFooterBlocks', () => {
  it('returns a store-name band and a legal nav menu', () => {
    const blocks = buildMercadoFooterBlocks('Cantinho Brasileiro')

    expect(blocks.map(b => b.type)).toEqual(['custom_html', 'nav_menu'])
    expect(blocks[0].props.html).toContain('Cantinho Brasileiro')
    expect(blocks[1].props.links).toEqual([
      { label: 'Privacy policy', url: '/' },
      { label: 'Terms & conditions', url: '/' },
      { label: 'Returns & refunds', url: '/' },
    ])
  })
})

describe('STARTER_TEMPLATE_CSS mercado', () => {
  it('gives Mercado its own serif/rounded/accent overrides, distinct from Editorial', () => {
    expect(STARTER_TEMPLATE_CSS.mercado).toContain('serif')
    expect(STARTER_TEMPLATE_CSS.mercado).toContain('border-radius')
    expect(STARTER_TEMPLATE_CSS.mercado).toContain('#e06c5c')
    expect(STARTER_TEMPLATE_CSS.mercado).not.toBe(STARTER_TEMPLATE_CSS.editorial)
  })
})

describe('STARTER_TEMPLATES', () => {
  it('lists classic, editorial, and mercado, each producing header/home/footer blocks', () => {
    expect(STARTER_TEMPLATES.map(t => t.id)).toEqual(['classic', 'editorial', 'mercado'])
    for (const template of STARTER_TEMPLATES) {
      expect(template.buildHeader('Store').length).toBeGreaterThan(0)
      expect(template.buildHome('Store').length).toBeGreaterThan(0)
      expect(template.buildFooter('Store').length).toBeGreaterThan(0)
    }
  })
})
