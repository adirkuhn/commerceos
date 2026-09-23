import { describe, it, expect } from 'vitest'
import { PAGE_ROUTES, SUB_PAGE_ROUTES, navItemToPath, pathToNavItem, idFromPath } from './routes'

describe('SUB_PAGE_ROUTES', () => {
  it('gives every page except the dashboard a splat sub-route, so each page can own per-resource URLs', () => {
    const navs = SUB_PAGE_ROUTES.map(r => r.nav)
    expect(navs).not.toContain('dashboard')
    expect(navs).toHaveLength(PAGE_ROUTES.length - 1)
  })

  it('each sub-route path is the base route path plus /*', () => {
    for (const sub of SUB_PAGE_ROUTES) {
      const base = PAGE_ROUTES.find(r => r.nav === sub.nav)
      expect(sub.path).toBe(`${base?.path}/*`)
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
    expect(pathToNavItem('/picking/ps-1')).toBe('picking')
  })

  it('resolves a nested sub-path to the most specific owning page', () => {
    expect(pathToNavItem('/settings/tax/classes')).toBe('settings')
    expect(pathToNavItem('/reports/customer-rfm')).toBe('reports-customer-rfm')
    expect(pathToNavItem('/reports/customer-rfm/segment-x')).toBe('reports-customer-rfm')
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
