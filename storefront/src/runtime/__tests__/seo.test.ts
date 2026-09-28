import { describe, it, expect } from 'vitest'
import { breadcrumbJsonLd, canonicalURL, indexable, jsonLdScript, organizationJsonLd, productJsonLd } from '../seo'
import type { ProductDetail } from '../../api/storefront'
import type { ResolvedRoute } from '../route'

const route = (key: string, param?: string): ResolvedRoute => ({ kind: 'template', key, param, notFound: false })

const product = (skus: { price: number; sale?: number; stock: number }[]): ProductDetail => ({
  id: 'p1', name: 'Red shoes', slug: 'red-shoes', description: '<p>Comfy &amp; red</p>', category_id: null, tags: [],
  images: ['https://cdn.test/1.jpg'],
  variants: [{ id: 'v1', name: 'Default', skus: skus.map((s, i) => ({
    id: `s${i}`, sku_code: `RS-${i}`, price_minor_units: s.price, sale_price_minor_units: s.sale, currency_code: 'EUR', stock_quantity: s.stock,
  })) }],
})

describe('canonicalURL', () => {
  it('is the public URL of the page without query noise, keeping only the listing page', () => {
    expect(canonicalURL('https://shop.test/', 'k1', '/products/red-shoes', new URLSearchParams('utm_source=x'), route('product', 'red-shoes')))
      .toBe('https://shop.test/store/k1/products/red-shoes')
    expect(canonicalURL('https://shop.test', 'k1', '/collections/summer', new URLSearchParams('page=2&category=c1'), route('collection', 'summer')))
      .toBe('https://shop.test/store/k1/collections/summer?page=2')
    expect(canonicalURL('https://shop.test', 'k1', '/', new URLSearchParams(), route('home'))).toBe('https://shop.test/store/k1/')
  })

  it('is left out without a configured public origin', () => {
    expect(canonicalURL(undefined, 'k1', '/', new URLSearchParams(), route('home'))).toBeUndefined()
  })
})

describe('indexable', () => {
  it('keeps shopper-specific and utility pages out of search engines', () => {
    for (const key of ['cart', 'checkout', 'account', 'login', 'wishlist', 'order-confirmation', 'search', 'not-found']) {
      expect(indexable(route(key)), key).toBe(false)
    }
    expect(indexable({ ...route('newsletter-confirm'), builtin: 'newsletter-confirm' })).toBe(false)
    for (const key of ['home', 'product', 'collection']) expect(indexable(route(key)), key).toBe(true)
    expect(indexable({ kind: 'page', key: 'about', notFound: false })).toBe(true)
  })
})

describe('productJsonLd', () => {
  it('describes one purchasable offer with the price actually charged', () => {
    const ld = productJsonLd(product([{ price: 5000, sale: 4000, stock: 2 }]), 'https://shop.test/store/k1/products/red-shoes', { average: 4.5, count: 2 })
    expect(ld).toEqual({
      '@context': 'https://schema.org', '@type': 'Product', name: 'Red shoes', description: 'Comfy & red',
      image: ['https://cdn.test/1.jpg'], sku: 'RS-0', url: 'https://shop.test/store/k1/products/red-shoes',
      offers: { '@type': 'Offer', price: '40.00', priceCurrency: 'EUR', availability: 'https://schema.org/InStock', url: 'https://shop.test/store/k1/products/red-shoes' },
      aggregateRating: { '@type': 'AggregateRating', ratingValue: 4.5, reviewCount: 2 },
    })
  })

  it('summarises several variants as a price range and leaves out an empty rating', () => {
    const ld = productJsonLd(product([{ price: 5000, stock: 0 }, { price: 7000, stock: 0 }]), undefined, { average: 0, count: 0 })
    expect(ld.offers).toEqual({
      '@type': 'AggregateOffer', lowPrice: '50.00', highPrice: '70.00', offerCount: 2, priceCurrency: 'EUR', availability: 'https://schema.org/OutOfStock',
    })
    expect(ld).not.toHaveProperty('aggregateRating')
    expect(ld).not.toHaveProperty('url')
  })
})

describe('other structured data', () => {
  it('builds breadcrumbs and the organisation', () => {
    expect(breadcrumbJsonLd([{ name: 'Home', url: 'https://s/store/k/' }, { name: 'Summer', url: 'https://s/store/k/collections/summer' }])).toEqual({
      '@context': 'https://schema.org', '@type': 'BreadcrumbList', itemListElement: [
        { '@type': 'ListItem', position: 1, name: 'Home', item: 'https://s/store/k/' },
        { '@type': 'ListItem', position: 2, name: 'Summer', item: 'https://s/store/k/collections/summer' },
      ],
    })
    expect(organizationJsonLd('Acme', 'https://s/store/k/', 'https://cdn/logo.png')).toEqual({
      '@context': 'https://schema.org', '@type': 'Organization', name: 'Acme', url: 'https://s/store/k/', logo: 'https://cdn/logo.png',
    })
  })

  it('embeds JSON-LD so catalog text can never close the script tag', () => {
    const tag = jsonLdScript({ name: '</script><script>alert(1)</script>' })
    expect(tag).toBe('<script type="application/ld+json">{"name":"\\u003c/script>\\u003cscript>alert(1)\\u003c/script>"}</script>')
  })
})
