import { describe, it, expect } from 'vitest'
import { cantinhoDocument, collectionInput, MENU_COLLECTIONS, ROW_COLLECTIONS, translate, type CollectionIds, type Node } from '../cantinho.ts'

const n = (id: string, type: string, props?: Record<string, unknown>, ...children: Node[]): Node => ({ id, type, props, children })

// The shape of the mercado preset's home, header and footer (ids matter).
const productRow = (id: string) => n(id, 'section', undefined,
  n(`${id}-head`, 'columns', undefined,
    n(`${id}-title-col`, 'column', undefined, n(`${id}-title`, 'heading', { text: 'X', level: 2 })),
    n(`${id}-link-col`, 'column', undefined, n(`${id}-link`, 'button', { label: 'See all', link: '/search?q=' }))),
  n(`${id}-grid`, 'product-grid', { source: 'all', limit: 10 }))
const home = n('root', 'root', undefined,
  n('hero-section', 'section', undefined, n('hero', 'hero', { title: 'Big sale: up to 60% off' })),
  ...['deals', 'snacks', 'favourites', 'drinks', 'new'].map(productRow),
  n('trust-section', 'section', undefined, n('trust', 'banner', { text: 'x' })),
  n('newsletter-section', 'section', undefined, n('newsletter', 'newsletter-signup', { title: 'Get the weekly deals' })))

const ids: CollectionIds = Object.fromEntries([...MENU_COLLECTIONS, ...ROW_COLLECTIONS].map(c => [c.slug, { id: `id-${c.slug}`, slug: c.slug }]))

function all(node: Node): Node[] {
  return [node, ...(node.children ?? []).flatMap(all)]
}

describe('cantinho seed content', () => {
  it('turns the collections into smart collections: category when the tenant has it, else a tag', () => {
    const cats = [{ id: 'cat-bebidas', name: 'bebidas' }, { id: 'cat-cp', name: 'Cuidados pessoais' }]
    expect(collectionInput(MENU_COLLECTIONS[1], cats)).toMatchObject({ slug: 'bebidas', type: 'smart', published: true, rules: [{ field: 'category', value: 'cat-bebidas', include_subcategories: true }] })
    expect(collectionInput(ROW_COLLECTIONS.find(c => c.slug === 'mate')!, cats).rules).toEqual([{ field: 'tag', value: 'mate' }])
    expect(collectionInput(ROW_COLLECTIONS[0], cats).rules).toEqual([{ field: 'on_sale' }])
    expect(collectionInput(ROW_COLLECTIONS.find(c => c.slug === 'novidades')!, cats)).toMatchObject({ rules: [{ field: 'added_within_days', days: 30 }], sort: 'newest' })
    expect(collectionInput(MENU_COLLECTIONS.find(c => c.slug === 'farinhas')!, []).rules).toEqual([{ field: 'tag', value: 'farinhas' }])
  })

  it('translates preset texts, including rich text and menu items', () => {
    const out = translate(n('r', 'root', undefined,
      n('b', 'button', { label: 'See all' }),
      n('t', 'text', { content: [{ type: 'paragraph', children: [{ text: 'Opening hours: 10:00 – 18:00' }, { text: 'Help' }] }] }),
      n('m', 'navigation-menu', { items: [{ label: 'Wishlist', link: '/wishlist' }] })))
    expect(out.children![0].props!.label).toBe('Ver mais')
    expect(JSON.stringify(out.children![1].props)).toContain('Ajuda')
    expect(out.children![2].props!.items).toEqual([{ label: 'Favoritos', link: '/wishlist' }])
  })

  it('builds the home page: eight rows on the store collections, all ids unique', () => {
    const out = cantinhoDocument('template', 'home', home, ids)
    const nodes = all(out)
    expect(new Set(nodes.map(x => x.id)).size).toBe(nodes.length)
    const grids = nodes.filter(x => x.type === 'product-grid')
    expect(grids.map(g => g.props!.collection_id)).toEqual([
      'id-promocoes', 'id-snacks', 'id-queridinhos-do-brasil', 'id-mate', 'id-cuidado-e-beleza', 'id-skala', 'id-bebidas', 'id-novidades',
    ])
    expect(grids.every(g => g.props!.source === 'collection')).toBe(true)
    expect(nodes.find(x => x.id === 'hero')!.props).toMatchObject({ title: 'Super Promoção até 60% off', button_link: '/collections/promocoes' })
    expect(nodes.find(x => x.id === 'mate-link')!.props).toEqual({ label: 'Ver mais', link: '/collections/mate' })
  })

  it('fills header menu and footer contact and socials', () => {
    const header = cantinhoDocument('section_group', 'header', n('root', 'root', undefined,
      n('announcement', 'banner', { text: 'x' }), n('wishlist-link', 'navigation-menu', { items: [] }), n('categories', 'navigation-menu', { items: [] })), ids)
    const menu = all(header).find(x => x.id === 'categories')!.props!.items as { label: string; link: string }[]
    expect(menu).toHaveLength(10)
    expect(menu[0]).toEqual({ label: 'Mercearia', link: '/collections/mercearia' })
    const footer = cantinhoDocument('section_group', 'footer', n('root', 'root', undefined,
      n('footer-about-text', 'text', {}), n('footer-contact-text', 'text', {}), n('footer-menu', 'navigation-menu', {}), n('social', 'social-links', {})), ids)
    expect(JSON.stringify(footer)).toContain('Purmerplein 28A')
    expect((all(footer).find(x => x.id === 'social')!.props!.items as unknown[]).length).toBe(4)
  })
})
