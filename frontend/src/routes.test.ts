import { describe, it, expect } from 'vitest'
import { PAGE_ROUTES, DETAIL_PAGE_ROUTES, navItemToPath, pathToNavItem, idFromPath } from './routes'

describe('DETAIL_PAGE_ROUTES', () => {
  it('adds a /:id sub-route for each entity that supports a per-record deep link', () => {
    const navs = DETAIL_PAGE_ROUTES.map(r => r.nav).sort()
    expect(navs).toEqual(['customers', 'products', 'sales-orders', 'suppliers'])
  })

  it('each detail route path is the base route path plus /:id', () => {
    for (const detail of DETAIL_PAGE_ROUTES) {
      const base = PAGE_ROUTES.find(r => r.nav === detail.nav)
      expect(detail.path).toBe(`${base?.path}/:id`)
    }
  })
})

describe('navItemToPath', () => {
  it('still returns the plain base path (unaffected by detail routes)', () => {
    expect(navItemToPath('products')).toBe('/products')
    expect(navItemToPath('sales-orders')).toBe('/sales-orders')
  })
})

describe('pathToNavItem', () => {
  it('resolves an exact base path', () => {
    expect(pathToNavItem('/products')).toBe('products')
  })

  it('resolves a per-record detail path back to the owning nav item, so the sidebar stays highlighted', () => {
    expect(pathToNavItem('/products/prod-1')).toBe('products')
    expect(pathToNavItem('/sales-orders/so-1')).toBe('sales-orders')
  })

  it('falls back to dashboard for an unknown path', () => {
    expect(pathToNavItem('/nope')).toBe('dashboard')
  })
})

describe('idFromPath', () => {
  it('returns null on the bare base path', () => {
    expect(idFromPath('/products', '/products')).toBeNull()
  })

  it('returns the id segment after the base path', () => {
    expect(idFromPath('/products/prod-1', '/products')).toBe('prod-1')
    expect(idFromPath('/sales-orders/new', '/sales-orders')).toBe('new')
  })

  it('URL-decodes the id segment', () => {
    expect(idFromPath('/products/a%20b', '/products')).toBe('a b')
  })

  it('returns null for an unrelated path', () => {
    expect(idFromPath('/suppliers/sup-1', '/products')).toBeNull()
  })
})
