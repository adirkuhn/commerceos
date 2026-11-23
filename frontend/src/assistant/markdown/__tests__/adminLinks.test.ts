import { describe, expect, it } from 'vitest'
import { isAdminPath } from '../adminLinks'

describe('isAdminPath', () => {
  it.each(['/sales-orders/abc', '/quotes/q-1', '/products/p-1', '/customers/c-1', '/suppliers/s-1', '/procurement/po-1', '/reports/sales-trends', '/settings/tax?x=1'])(
    'allows %s, a page of the admin', path => {
      expect(isAdminPath(path)).toBe(true)
    },
  )

  it.each(['/', '//evil.example', 'https://evil.example', 'javascript:alert(1)', '/nope/1', 'quotes/1', '', '/\\evil.example'])(
    'refuses %s', path => {
      expect(isAdminPath(path)).toBe(false)
    },
  )
})
