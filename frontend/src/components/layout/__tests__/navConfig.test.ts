import { describe, it, expect } from 'vitest'
import { PAGE_ROUTES } from '../../../routes'
import { navGroupsForRole, navTitle, roleLabel, HOME_LINK, GO_TO_SHORTCUTS, sidebarActiveId, REPORT_SECTIONS } from '../navConfig'

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

  it('calls the home page Today', () => {
    expect(HOME_LINK.label).toBe('Today')
    expect(navTitle('dashboard')).toBe('Today')
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

  it('shows the assistant only to admin roles', () => {
    expect(ids('customer_user')).not.toContain('assistant')
    for (const role of ['customer_owner', 'customer_admin', 'super_admin']) {
      expect(ids(role), role).toContain('assistant')
    }
    expect(navTitle('assistant')).toBe('Assistant')
  })

  it('reaches every routed page from the sidebar or the reports list', () => {
    const reachable = new Set([HOME_LINK.id, ...ids('super_admin'), ...REPORT_SECTIONS.flatMap(s => s.items.map(i => i.id))])
    const unlisted = ['orders', 'shipping']
    for (const { nav } of PAGE_ROUTES) {
      if (unlisted.includes(nav)) continue
      expect(reachable.has(nav), nav).toBe(true)
    }
  })

  it('keeps the sidebar short: one reports link for every report', () => {
    expect(ids('super_admin').filter(id => id.startsWith('reports-'))).toEqual(['reports-overview'])
  })

  it('puts setup pages in their own group, collapsed until needed', () => {
    const setup = navGroupsForRole('super_admin').find(g => g.id === 'setup')
    expect(setup?.collapsedByDefault).toBe(true)
    expect(setup?.pinned).toBe(true)
    expect(setup?.items.map(i => i.id)).toEqual(
      expect.arrayContaining(['sales-channels', 'integrations', 'webhooks', 'sync-log', 'users', 'scheduled-jobs', 'settings']),
    )
  })

  it('groups daily warehouse work together', () => {
    const warehouse = navGroupsForRole('customer_user').find(g => g.id === 'warehouse')
    expect(warehouse?.items.map(i => i.id)).toEqual(['picking', 'packing', 'stock-issues', 'inventory', 'warehouses'])
  })

  it('highlights the one reports link on every report page', () => {
    expect(sidebarActiveId('reports-customer-rfm')).toBe('reports-overview')
    expect(sidebarActiveId('products')).toBe('products')
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

describe('REPORT_SECTIONS', () => {
  it('lists every report exactly once', () => {
    const reportIds = PAGE_ROUTES.map(r => r.nav).filter(id => id.startsWith('reports-'))
    const listed = REPORT_SECTIONS.flatMap(s => s.items.map(i => i.id))
    expect([...listed].sort()).toEqual([...reportIds].sort())
  })
})
