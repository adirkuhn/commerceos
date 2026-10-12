import type { ElementType } from 'react'
import i18next from 'i18next'
import {
  LayoutDashboard, Package, Settings, Plug, Radio, Warehouse, History, Boxes, Truck, ShoppingBag,
  ShoppingCart, Users, AlertTriangle, PackageSearch, Clock, ScanBarcode, Tag, BarChart3, Ticket, Star,
  Webhook, MessageSquare, PackageCheck, UserCog, Layers, Mail, CalendarClock,
} from 'lucide-react'
import type { NavItem } from '../../types'
import type { Design } from '../../theme/designPreference'

export interface NavLink {
  id: NavItem
  label: string
  icon: ElementType
}

export interface NavGroup {
  id: string
  label: string
  items: NavLink[]
  // Long groups start closed unless one of their pages is open.
  collapsedByDefault?: boolean
  // Page titles for this group's items read "<group>: <item>" (e.g.
  // "Reports: Overview"), for items whose label alone is ambiguous.
  qualifyTitles?: boolean
  // Rendered as plain top-level links, without a group heading.
  flat?: boolean
  // Kept below the scrolling list (always on screen), above the account.
  pinned?: boolean
}

// Single source for navigation: the Sidebar renders it, the Topbar takes
// page titles from it and the command palette lists it -- so a page's name
// is the same everywhere.
export const HOME_LINK: NavLink = { id: 'dashboard', get label() { return i18next.t('layout:nav.dashboard') }, icon: LayoutDashboard }
const TODAY_LINK: NavLink = { id: 'dashboard', get label() { return i18next.t('layout:nav.today') }, icon: LayoutDashboard }

// Labels are getters so they follow a language switch.
function link(id: NavItem, label: () => string, icon: ElementType): NavLink {
  return { id, get label() { return label() }, icon }
}

function group(id: string, label: () => string, items: NavLink[], extra: Partial<NavGroup> = {}): NavGroup {
  return { id, get label() { return label() }, items, ...extra }
}

// Every sidebar page, once -- both designs arrange these same links.
const L = {
  salesOrders: link('sales-orders', () => i18next.t('layout:nav.salesOrders'), ShoppingCart),
  customers: link('customers', () => i18next.t('layout:nav.customers'), Users),
  salesChannels: link('sales-channels', () => i18next.t('layout:nav.salesChannels'), Radio),
  picking: link('picking', () => i18next.t('layout:nav.picking'), ScanBarcode),
  packing: link('packing', () => i18next.t('layout:nav.packing'), PackageCheck),
  stockIssues: link('stock-issues', () => i18next.t('layout:nav.stockIssues'), AlertTriangle),
  products: link('products', () => i18next.t('layout:nav.products'), Package),
  collections: link('collections', () => i18next.t('layout:nav.collections'), Layers),
  inventory: link('inventory', () => i18next.t('layout:nav.inventory'), Boxes),
  warehouses: link('warehouses', () => i18next.t('layout:nav.warehouses'), Warehouse),
  unmatchedSkus: link('unmatched-skus', () => i18next.t('layout:nav.unmatchedSkus'), PackageSearch),
  suppliers: link('suppliers', () => i18next.t('layout:nav.suppliers'), Truck),
  procurement: link('procurement', () => i18next.t('layout:nav.procurement'), ShoppingBag),
  purchasePlanner: link('purchase-planner', () => i18next.t('layout:nav.purchasePlanner'), CalendarClock),
  promotions: link('promotions', () => i18next.t('layout:nav.promotions'), Tag),
  coupons: link('coupons', () => i18next.t('layout:nav.coupons'), Ticket),
  reviews: link('reviews', () => i18next.t('layout:nav.reviews'), Star),
  messageCenter: link('message-center', () => i18next.t('layout:nav.messageCenter'), MessageSquare),
  newsletter: link('newsletter', () => i18next.t('layout:nav.newsletter'), Mail),
  integrations: link('integrations', () => i18next.t('layout:nav.integrations'), Plug),
  webhooks: link('webhooks', () => i18next.t('layout:nav.webhooks'), Webhook),
  syncLog: link('sync-log', () => i18next.t('layout:nav.syncLog'), History),
}

const R = {
  overview: link('reports-overview', () => i18next.t('layout:nav.reportsOverview'), BarChart3),
  salesTrends: link('reports-sales-trends', () => i18next.t('layout:nav.reportsSalesTrends'), BarChart3),
  productPerformance: link('reports-product-performance', () => i18next.t('layout:nav.reportsProductPerformance'), BarChart3),
  performanceMatrix: link('reports-product-performance-matrix', () => i18next.t('layout:nav.reportsPerformanceMatrix'), BarChart3),
  productLifecycle: link('reports-product-lifecycle', () => i18next.t('layout:nav.reportsProductLifecycle'), BarChart3),
  categoryPerformance: link('reports-category-performance', () => i18next.t('layout:nav.reportsCategoryPerformance'), BarChart3),
  slowMoving: link('reports-slow-moving-products', () => i18next.t('layout:nav.reportsSlowMoving'), BarChart3),
  stockoutRisk: link('reports-stockout-risk', () => i18next.t('layout:nav.reportsStockoutRisk'), BarChart3),
  inventorySales: link('reports-inventory-sales', () => i18next.t('layout:nav.reportsInventorySales'), BarChart3),
  customerPerformance: link('reports-customer-performance', () => i18next.t('layout:nav.reportsCustomerPerformance'), BarChart3),
  customerCohorts: link('reports-customer-cohorts', () => i18next.t('layout:nav.reportsCustomerCohorts'), BarChart3),
  customerRfm: link('reports-customer-rfm', () => i18next.t('layout:nav.reportsCustomerRfm'), BarChart3),
  frequentlyBoughtTogether: link('reports-frequently-bought-together', () => i18next.t('layout:nav.reportsFrequentlyBoughtTogether'), BarChart3),
}

const CLASSIC_GROUPS: NavGroup[] = [
  group('sell', () => i18next.t('layout:group.sell'), [L.salesOrders, L.customers, L.salesChannels]),
  group('fulfill', () => i18next.t('layout:group.fulfill'), [L.picking, L.packing, L.stockIssues]),
  group('catalog', () => i18next.t('layout:group.catalog'), [L.products, L.collections, L.inventory, L.warehouses, L.unmatchedSkus]),
  group('buy', () => i18next.t('layout:group.buy'), [L.suppliers, L.procurement, L.purchasePlanner]),
  group('marketing', () => i18next.t('layout:group.marketing'), [L.promotions, L.coupons, L.reviews, L.messageCenter, L.newsletter]),
  group('reports', () => i18next.t('layout:group.reports'), [
    R.overview, R.salesTrends, R.productPerformance, R.performanceMatrix, R.productLifecycle,
    R.categoryPerformance, R.slowMoving, R.stockoutRisk, R.inventorySales, R.customerPerformance,
    R.customerCohorts, R.customerRfm, R.frequentlyBoughtTogether,
  ], { collapsedByDefault: true, qualifyTitles: true }),
  group('connect', () => i18next.t('layout:group.connect'), [L.integrations, L.webhooks, L.syncLog]),
]

// Depot: grouped by the work, daily work first. Reports are one link (the
// report pages carry their own grouped list, see REPORT_SECTIONS) and
// pages you set up once sit in a collapsed Setup group at the bottom.
const REPORTS_LINK: NavLink = { id: 'reports-overview', get label() { return i18next.t('layout:group.reports') }, icon: BarChart3 }

const DEPOT_GROUPS: NavGroup[] = [
  group('orders', () => i18next.t('layout:group.orders'), [L.salesOrders, L.customers]),
  group('warehouse', () => i18next.t('layout:group.warehouse'), [L.picking, L.packing, L.stockIssues, L.inventory, L.warehouses]),
  group('catalog', () => i18next.t('layout:group.catalog'), [L.products, L.collections, L.unmatchedSkus]),
  group('buying', () => i18next.t('layout:group.buying'), [L.suppliers, L.procurement, L.purchasePlanner]),
  group('marketing', () => i18next.t('layout:group.marketing'), [L.promotions, L.coupons, L.reviews, L.messageCenter, L.newsletter]),
  group('insight', () => i18next.t('layout:group.insight'), [REPORTS_LINK], { flat: true }),
]

// The reports' own navigation in the depot design, grouped by what they
// are about.
export const REPORT_SECTIONS: NavGroup[] = [
  group('reportsSales', () => i18next.t('layout:group.reportsSales'), [R.overview, R.salesTrends]),
  group('reportsProducts', () => i18next.t('layout:group.reportsProducts'), [R.productPerformance, R.performanceMatrix, R.productLifecycle, R.categoryPerformance, R.slowMoving, R.frequentlyBoughtTogether]),
  group('reportsCustomers', () => i18next.t('layout:group.reportsCustomers'), [R.customerPerformance, R.customerCohorts, R.customerRfm]),
  group('reportsStock', () => i18next.t('layout:group.reportsStock'), [R.stockoutRisk, R.inventorySales]),
]

// Team management calls lm2's company-owner endpoints (team/api.ts);
// platform operations match the backend's RequireRole(RoleSuperAdmin).
const USERS_LINK: NavLink = { id: 'users', get label() { return i18next.t('layout:nav.users') }, icon: UserCog }
const SCHEDULED_JOBS_LINK: NavLink = { id: 'scheduled-jobs', get label() { return i18next.t('layout:nav.scheduledJobs') }, icon: Clock }
const SETTINGS_LINK: NavLink = { id: 'settings', get label() { return i18next.t('layout:nav.settings') }, icon: Settings }

// Pages reachable by URL but not listed in the sidebar (demo screens).
const UNLISTED_TITLES: Partial<Record<NavItem, () => string>> = {
  orders: () => i18next.t('layout:nav.orders'),
  shipping: () => i18next.t('layout:nav.shipping'),
}

export function navGroupsForRole(role?: string, design: Design = 'classic'): NavGroup[] {
  const admin: NavLink[] = [
    ...(role === 'customer_owner' || role === 'super_admin' ? [USERS_LINK] : []),
    ...(role === 'super_admin' ? [SCHEDULED_JOBS_LINK] : []),
    SETTINGS_LINK,
  ]
  if (design === 'depot') {
    return [...DEPOT_GROUPS, group('setup', () => i18next.t('layout:group.setup'), [L.salesChannels, L.integrations, L.webhooks, L.syncLog, ...admin], { collapsedByDefault: true, pinned: true })]
  }
  return [...CLASSIC_GROUPS, group('admin', () => i18next.t('layout:group.admin'), admin)]
}

export function homeLink(design: Design): NavLink {
  return design === 'depot' ? TODAY_LINK : HOME_LINK
}

// The sidebar link to highlight for the current page -- in the depot
// design every report page lights up the one Reports link.
export function sidebarActiveId(nav: NavItem, design: Design): NavItem {
  return design === 'depot' && nav.startsWith('reports-') ? REPORTS_LINK.id : nav
}

const ALL_GROUPS = navGroupsForRole('super_admin')

export function navTitle(nav: NavItem, design: Design = 'classic'): string {
  if (nav === HOME_LINK.id) return homeLink(design).label
  for (const g of ALL_GROUPS) {
    const found = g.items.find(i => i.id === nav)
    if (found) return g.qualifyTitles ? i18next.t('layout:ui.qualifiedTitle', { group: g.label, page: found.label }) : found.label
  }
  return UNLISTED_TITLES[nav]?.() ?? nav
}

// "g then <letter>" jumps (see shortcuts/useGoToShortcuts). Kept to the
// few pages used all day -- custom shortcuts should be rare.
export const GO_TO_SHORTCUTS: Readonly<Record<string, NavItem>> = {
  d: 'dashboard',
  o: 'sales-orders',
  c: 'customers',
  p: 'products',
  i: 'inventory',
}

export function shortcutFor(nav: NavItem): string | undefined {
  const letter = Object.keys(GO_TO_SHORTCUTS).find(k => GO_TO_SHORTCUTS[k] === nav)
  return letter ? `G ${letter.toUpperCase()}` : undefined
}

export function roleLabel(role: string | undefined): string {
  switch (role) {
    case 'customer_owner': return i18next.t('layout:role.customer_owner')
    case 'customer_admin': return i18next.t('layout:role.customer_admin')
    case 'customer_user': return i18next.t('layout:role.customer_user')
    case 'super_admin': return i18next.t('layout:role.super_admin')
    default: return i18next.t('layout:role.signedIn')
  }
}
