import { describe, it, expect } from 'vitest'
import { PAGE_ROUTES } from '../../../routes'
import { navGroupsForRole, navTitle, roleLabel, HOME_LINK, GO_TO_SHORTCUTS, homeLink, sidebarActiveId, REPORT_SECTIONS } from '../navConfig'

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

  it('shows the assistant only to admin roles, in both designs', () => {
    expect(ids('customer_user')).not.toContain('assistant')
    for (const role of ['customer_owner', 'customer_admin', 'super_admin']) {
      expect(ids(role), role).toContain('assistant')
      expect(navGroupsForRole(role, 'depot').flatMap(g => g.items.map(i => i.id)), role).toContain('assistant')
    }
    expect(navGroupsForRole('customer_user', 'depot').flatMap(g => g.items.map(i => i.id))).not.toContain('assistant')
    expect(navTitle('assistant')).toBe('Assistant')
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

describe('depot navigation', () => {
  const depotIds = (role: string) => navGroupsForRole(role, 'depot').flatMap(g => g.items.map(i => i.id))

  it('reaches every page the classic sidebar reaches, reports through one link', () => {
    for (const role of ['customer_user', 'customer_owner', 'super_admin']) {
      const depot = new Set(depotIds(role))
      for (const id of ids(role)) {
        if (id.startsWith('reports-')) continue
        expect(depot.has(id), `${role}: ${id}`).toBe(true)
      }
      expect(depot.has('reports-overview')).toBe(true)
    }
  })

  it('lists each page once', () => {
    const all = depotIds('super_admin')
    expect(new Set(all).size).toBe(all.length)
  })

  it('keeps the sidebar short: one reports link instead of thirteen', () => {
    expect(depotIds('super_admin').filter(id => id.startsWith('reports-'))).toEqual(['reports-overview'])
    expect(depotIds('super_admin').length).toBeLessThan(ids('super_admin').length - 10)
  })

  it('puts setup pages in their own group, collapsed until needed', () => {
    const setup = navGroupsForRole('super_admin', 'depot').find(g => g.id === 'setup')
    expect(setup?.collapsedByDefault).toBe(true)
    expect(setup?.pinned).toBe(true)
    expect(setup?.items.map(i => i.id)).toEqual(
      expect.arrayContaining(['sales-channels', 'integrations', 'webhooks', 'sync-log', 'users', 'scheduled-jobs', 'settings']),
    )
  })

  it('groups daily warehouse work together', () => {
    const warehouse = navGroupsForRole('customer_user', 'depot').find(g => g.id === 'warehouse')
    expect(warehouse?.items.map(i => i.id)).toEqual(['picking', 'packing', 'stock-issues', 'inventory', 'warehouses'])
  })

  it('calls the home page Today', () => {
    expect(homeLink('depot').label).toBe('Today')
    expect(homeLink('classic')).toBe(HOME_LINK)
    expect(navTitle('dashboard', 'depot')).toBe('Today')
  })

  it('highlights the one reports link on every report page', () => {
    expect(sidebarActiveId('reports-customer-rfm', 'depot')).toBe('reports-overview')
    expect(sidebarActiveId('reports-customer-rfm', 'classic')).toBe('reports-customer-rfm')
    expect(sidebarActiveId('products', 'depot')).toBe('products')
  })
})

describe('REPORT_SECTIONS', () => {
  it('lists every report exactly once', () => {
    const reportIds = ids('super_admin').filter(id => id.startsWith('reports-'))
    const listed = REPORT_SECTIONS.flatMap(s => s.items.map(i => i.id))
    expect([...listed].sort()).toEqual([...reportIds].sort())
  })
})
