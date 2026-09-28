import { describe, it, expect } from 'vitest'
import { resolveRoute } from '../route'

describe('resolveRoute', () => {
  it.each([
    ['/', 'template', 'home'],
    ['', 'template', 'home'],
    ['/products/red-shoes', 'template', 'product'],
    ['/collections/summer', 'template', 'collection'],
    ['/search', 'template', 'search'],
    ['/cart', 'template', 'cart'],
    ['/checkout', 'template', 'checkout'],
    ['/login', 'template', 'login'],
    ['/account', 'template', 'account'],
    ['/wishlist', 'template', 'wishlist'],
    ['/orders/CO-1001', 'template', 'order-confirmation'],
    ['/pages/about', 'page', 'about'],
  ])('%s', (path, kind, key) => {
    expect(resolveRoute(path)).toMatchObject({ kind, key, notFound: false })
  })

  it('passes the dynamic segment through', () => {
    expect(resolveRoute('/products/red-shoes').param).toBe('red-shoes')
  })

  it('sends anything else to the not-found template', () => {
    for (const p of ['/nope', '/pages/Bad_Slug', '/products', '/products/a/b']) {
      expect(resolveRoute(p), p).toEqual({ kind: 'template', key: 'not-found', notFound: true })
    }
  })
})
