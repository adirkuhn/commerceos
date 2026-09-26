import { describe, it, expect } from 'vitest'
import { isTenantAdmin } from '../roles'

describe('isTenantAdmin', () => {
  it('is true for owners, admins and super admins only', () => {
    expect(isTenantAdmin('customer_owner')).toBe(true)
    expect(isTenantAdmin('customer_admin')).toBe(true)
    expect(isTenantAdmin('super_admin')).toBe(true)
    expect(isTenantAdmin('customer_user')).toBe(false)
    expect(isTenantAdmin(undefined)).toBe(false)
  })
})
