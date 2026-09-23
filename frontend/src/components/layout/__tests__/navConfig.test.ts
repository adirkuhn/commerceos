import { describe, it, expect } from 'vitest'
import { PAGE_ROUTES } from '../../../routes'
import { navGroupsForRole, navTitle, roleLabel, HOME_LINK, GO_TO_SHORTCUTS } from '../navConfig'

function ids(role?: string) {
  return navGroupsForRole(role).flatMap(g => g.items.map(i => i.id))
}

describe('navTitle', () => {
  it('gives every routed page a human title, never its internal id', () => {
    for (const { nav } of PAGE_ROUTES) {
      expect(navTitle(nav), nav).not.toBe(nav)
      expect(navTitle(nav), nav).not.toMatch(/^[a-z-]+$/)
    }
  })

  it('qualifies report pages with the Reports group', () => {
    expect(navTitle('reports-overview')).toBe('Reports: Overview')
    expect(navTitle('reports-sales-trends')).toBe('Reports: Sales Trends')
  })

  it('uses the sidebar label for regular pages', () => {
    expect(navTitle('sales-orders')).toBe('Sales Orders')
    expect(navTitle('dashboard')).toBe(HOME_LINK.label)
  })
})

describe('navGroupsForRole', () => {
  it('lists each page once', () => {
    const all = ids('super_admin')
    expect(new Set(all).size).toBe(all.length)
  })

  it('shows team management only to owners and super admins', () => {
    expect(ids('customer_user')).not.toContain('users')
    expect(ids('customer_owner')).toContain('users')
    expect(ids('super_admin')).toContain('users')
  })

  it('shows platform operations only to super admins', () => {
    expect(ids('customer_owner')).not.toContain('scheduled-jobs')
    expect(ids('super_admin')).toContain('scheduled-jobs')
  })

  it('names list pages with plural nouns', () => {
    const labels = navGroupsForRole('super_admin').flatMap(g => g.items.map(i => i.label))
    expect(labels).toContain('Sales Orders')
    expect(labels).toContain('Customers')
    expect(labels).not.toContain('Sales Order')
    expect(labels).not.toContain('Customer')
  })
})

describe('roleLabel', () => {
  it('turns lm2 role claims into words people use', () => {
    expect(roleLabel('customer_owner')).toBe('Owner')
    expect(roleLabel('customer_admin')).toBe('Admin')
    expect(roleLabel('customer_user')).toBe('User')
    expect(roleLabel('super_admin')).toBe('Platform admin')
  })

  it('falls back to a generic label for unknown roles', () => {
    expect(roleLabel(undefined)).toBe('Signed in')
    expect(roleLabel('something_new')).toBe('Signed in')
  })
})

describe('GO_TO_SHORTCUTS', () => {
  it('keeps custom shortcuts few, and only for pages everyone can reach', () => {
    const targets = Object.values(GO_TO_SHORTCUTS)
    expect(targets.length).toBeLessThanOrEqual(6)
    const reachable = [HOME_LINK.id, ...navGroupsForRole('customer_user').flatMap(g => g.items.map(i => i.id))]
    for (const t of targets) expect(reachable).toContain(t)
  })
})
