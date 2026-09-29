// Content for the Cantinho Brasileiro store (cantinhobrasileiro.nl), built
// on the "mercado" preset: Portuguese texts, the brand's green, and product
// rows pointed at its collections. Pure data + transforms (tested); the
// runner (../seed-cantinho.ts) does the API calls. Images: none -- the
// merchant uploads logo and banners in the editor.

export interface Node {
  id: string
  type: string
  props?: Record<string, unknown>
  style?: unknown
  children?: Node[]
}

export const STORE = {
  name: 'Cantinho Brasileiro',
  locale: 'pt-BR',
  whatsapp: 'https://wa.me/31640140482',
  instagram: 'https://instagram.com/cantinhobrasileiro.nl',
  maps: 'https://maps.google.com/?q=Purmerplein+28A,+1023BD+Amsterdam',
  email: 'mailto:cantinhobrasileironl@gmail.com',
}

/** The site's green (#007500) with a Brazilian-flag yellow accent. */
export const COLORS = {
  primary: '#007500', secondary: '#004d00', background: '#ffffff', surface: '#f3f8f2',
  text: '#1d2a1d', muted: '#667366', accent: '#ffcc29',
}

export interface CollectionSpec {
  slug: string
  name: string
  /** Category name to match (accent/case-insensitive); else the tag. */
  category?: string
  tag?: string
  rule?: 'on_sale' | 'new'
  sort: 'name_asc' | 'price_asc' | 'newest'
}

/** Menu categories first (in menu order), then the home-page rows. */
export const MENU_COLLECTIONS: CollectionSpec[] = [
  { slug: 'mercearia', name: 'Mercearia', category: 'Mercearia', sort: 'name_asc' },
  { slug: 'bebidas', name: 'Bebidas', category: 'Bebidas', sort: 'name_asc' },
  { slug: 'graos-e-cereais', name: 'Grãos e cereais', category: 'Grãos e cereais', sort: 'name_asc' },
  { slug: 'farinhas', name: 'Farinhas', category: 'Farinhas', sort: 'name_asc' },
  { slug: 'doces', name: 'Doces', category: 'Doces', sort: 'name_asc' },
  { slug: 'temperos', name: 'Temperos', category: 'Temperos', sort: 'name_asc' },
  { slug: 'congelados', name: 'Congelados', category: 'Congelados', sort: 'name_asc' },
  { slug: 'utilidades', name: 'Utilidades', category: 'Utilidades', sort: 'name_asc' },
  { slug: 'cuidados-pessoais', name: 'Cuidados Pessoais', category: 'Cuidados Pessoais', sort: 'name_asc' },
  { slug: 'mamae-e-bebe', name: 'Mamãe e Bebê', category: 'Mamãe e Bebê', sort: 'name_asc' },
]

export const ROW_COLLECTIONS: CollectionSpec[] = [
  { slug: 'promocoes', name: 'Super Promoção até 60% off', rule: 'on_sale', sort: 'price_asc' },
  { slug: 'snacks', name: 'Snacks', tag: 'snacks', sort: 'name_asc' },
  { slug: 'queridinhos-do-brasil', name: 'Os Queridinhos do Brasil 🇧🇷', tag: 'queridinhos', sort: 'name_asc' },
  { slug: 'mate', name: 'Mate', tag: 'mate', sort: 'name_asc' },
  { slug: 'cuidado-e-beleza', name: 'Cuidado e Beleza', category: 'Cuidados Pessoais', sort: 'name_asc' },
  { slug: 'skala', name: 'Skala', tag: 'skala', sort: 'name_asc' },
  { slug: 'novidades', name: 'Novidades', rule: 'new', sort: 'newest' },
  { slug: 'bebidas', name: 'Bebidas', category: 'Bebidas', sort: 'name_asc' },
]

const fold = (s: string) => s.normalize('NFD').replace(/\p{Diacritic}/gu, '').toLowerCase().trim()

/** The collection API body for a spec, given the tenant's categories. */
export function collectionInput(spec: CollectionSpec, categories: { id: string; name: string }[]) {
  const category = spec.category ? categories.find(c => fold(c.name) === fold(spec.category!)) : undefined
  const rules =
    spec.rule === 'on_sale' ? [{ field: 'on_sale' }]
    : spec.rule === 'new' ? [{ field: 'added_within_days', days: 30 }]
    : category ? [{ field: 'category', value: category.id, include_subcategories: true }]
    : [{ field: 'tag', value: spec.tag ?? fold(spec.name).replace(/[^a-z0-9]+/g, '-') }]
  return { name: spec.name, slug: spec.slug, type: 'smart', match: 'all', rules, sort: spec.sort, published: true }
}

/** Preset (English) content texts in Portuguese. */
export const PT: Record<string, string> = {
  'Shop now': 'Comprar agora', 'Back to the shop': 'Voltar para a loja', 'Home': 'Início', 'Search': 'Buscar',
  'My account': 'Minha conta', 'Privacy policy': 'Política de privacidade', 'Returns': 'Trocas e devoluções',
  'Shipping': 'Entrega', 'See all': 'Ver mais', 'Terms & conditions': 'Termos e condições', 'Wishlist': 'Favoritos',
  'Learn more': 'Saiba mais', 'Discover our latest products.': 'Conheça nossos produtos.',
  'Free shipping on orders over €50': 'Frete grátis em pedidos acima de €50',
  'News and offers, straight to your inbox.': 'Novidades e promoções direto no seu e-mail.',
  'Categories': 'Categorias', 'Reviews': 'Avaliações', 'Sign in': 'Entrar', 'Stay in the loop': 'Fique por dentro',
  'Thank you for your order': 'Obrigado pelo seu pedido!', 'Welcome to our store': 'Bem-vindo à nossa loja',
  'You may also like': 'Você também pode gostar', 'Checkout': 'Finalizar compra', 'Contact': 'Contato',
  'Featured products': 'Destaques', 'Follow us': 'Siga-nos', 'Help': 'Ajuda', 'Page not found': 'Página não encontrada',
  'Shop by collection': 'Compre por categoria', 'Your account': 'Minha conta', 'Your cart': 'Meu carrinho',
  'Your wishlist': 'Meus favoritos', "The page you're looking for doesn't exist.": 'A página que você procura não existe.',
}

const TEXT_PROPS = ['text', 'title', 'subtitle', 'label', 'button_label', 'link_label', 'placeholder', 'empty_message', 'home_label']

/** Every preset text in Portuguese (rich text too); unknown texts stay. */
export function translate(node: Node): Node {
  const props = { ...(node.props ?? {}) }
  for (const key of TEXT_PROPS) {
    if (typeof props[key] === 'string' && PT[props[key] as string]) props[key] = PT[props[key] as string]
  }
  if (Array.isArray(props.items)) {
    props.items = props.items.map(i => (i && typeof i === 'object' && 'label' in i && PT[(i as { label: string }).label] ? { ...i, label: PT[(i as { label: string }).label] } : i))
  }
  if (Array.isArray(props.content)) props.content = translateRich(props.content)
  return { ...node, props, children: node.children?.map(translate) }
}

function translateRich(content: unknown[]): unknown[] {
  return content.map(block => {
    if (!block || typeof block !== 'object') return block
    const b = block as { children?: { text?: string }[] }
    return b.children ? { ...b, children: b.children.map(c => (c.text && PT[c.text] ? { ...c, text: PT[c.text] } : c)) } : block
  })
}

const paragraphs = (...lines: string[]) => ({ content: lines.map(text => ({ type: 'paragraph', children: [{ text }] })) })

/** Sets props on the node with this id (deep). */
function patch(node: Node, id: string, props: Record<string, unknown>): Node {
  if (node.id === id) return { ...node, props: { ...(node.props ?? {}), ...props } }
  return { ...node, children: node.children?.map(c => patch(c, id, props)) }
}

/** A copy of a subtree with every id prefixed, so it can sit next to the original. */
function cloneAs(node: Node, from: string, to: string): Node {
  return { ...node, id: node.id.replace(from, to), children: node.children?.map(c => cloneAs(c, from, to)) }
}

/** Points a preset product row at a collection, with its title. */
function row(root: Node, id: string, title: string, collection: { id: string; slug: string }): Node {
  let r = patch(root, `${id}-title`, { text: title })
  r = patch(r, `${id}-link`, { label: 'Ver mais', link: `/collections/${collection.slug}` })
  return patch(r, `${id}-grid`, { source: 'collection', collection_id: collection.id, limit: 10 })
}

export type CollectionIds = Record<string, { id: string; slug: string }>

export function home(preset: Node, c: CollectionIds): Node {
  // Mercado has five rows; Cantinho has eight. Extra rows copy "favourites".
  const favourites = preset.children!.find(s => s.id === 'favourites')!
  const extra = ['mate', 'beleza', 'skala'].map(id => cloneAs(favourites, 'favourites', id))
  const children = preset.children!.flatMap(s => (s.id === 'favourites' ? [s, ...extra] : [s]))
  let r: Node = { ...preset, children }
  r = patch(r, 'hero', { title: 'Super Promoção até 60% off', subtitle: 'Com muito amor do Brasil direto para sua casa.', button_label: 'Ver promoções', button_link: '/collections/promocoes' })
  r = row(r, 'deals', 'Super Promoção até 60% off', c['promocoes'])
  r = row(r, 'snacks', 'Snacks', c['snacks'])
  r = row(r, 'favourites', 'Os Queridinhos do Brasil 🇧🇷', c['queridinhos-do-brasil'])
  r = row(r, 'mate', 'Mate', c['mate'])
  r = row(r, 'beleza', 'Cuidado e Beleza', c['cuidado-e-beleza'])
  r = row(r, 'skala', 'Skala', c['skala'])
  r = row(r, 'new', 'Novidades', c['novidades'])
  r = row(r, 'drinks', 'Bebidas', c['bebidas'])
  r = patch(r, 'trust', { text: 'Compra 100% segura · Entrega rápida · Atendimento em português' })
  return patch(r, 'newsletter', { title: 'Receba nossas ofertas', text: 'Novidades e promoções direto no seu e-mail.', button_label: '', consent_label: '' })
}

export function header(preset: Node, c: CollectionIds): Node {
  let r = patch(preset, 'announcement', {
    text: 'Pedido mínimo €30 · Ofertas exclusivas no nosso grupo de WhatsApp', link_label: 'Entrar no grupo', link: STORE.whatsapp,
  })
  r = patch(r, 'wishlist-link', { items: [{ label: 'Favoritos', link: '/wishlist' }] })
  return patch(r, 'categories', { items: MENU_COLLECTIONS.map(m => ({ label: m.name, link: `/collections/${c[m.slug]?.slug ?? m.slug}` })) })
}

export function footer(preset: Node): Node {
  let r = patch(preset, 'footer-about-text', paragraphs('A melhor loja de produtos brasileiros. Com muito amor do Brasil direto para sua casa.', '© Cantinho Brasileiro'))
  r = patch(r, 'footer-contact-text', paragraphs('cantinhobrasileironl@gmail.com', 'Horário: 10:00 – 18:00', 'Loja física: Purmerplein 28A – 1023BD Amsterdam'))
  r = patch(r, 'footer-menu', { items: [
    { label: 'Minha conta', link: '/account' }, { label: 'Favoritos', link: '/wishlist' },
    { label: 'Trocas e devoluções', link: '/pages/returns' }, { label: 'Política de privacidade', link: '/pages/privacy' },
    { label: 'Termos e condições', link: '/pages/terms' },
  ] })
  return patch(r, 'social', { items: [
    { network: 'whatsapp', link: STORE.whatsapp }, { network: 'instagram', link: STORE.instagram },
    { network: 'maps', link: STORE.maps }, { network: 'email', link: STORE.email },
  ] })
}

/** The Portuguese Cantinho version of one preset document. */
export function cantinhoDocument(kind: string, key: string, root: Node, c: CollectionIds): Node {
  const translated = translate(root)
  if (kind === 'template' && key === 'home') return home(translated, c)
  if (kind === 'section_group' && key === 'header') return header(translated, c)
  if (kind === 'section_group' && key === 'footer') return footer(translated)
  return translated
}
