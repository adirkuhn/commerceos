import type { ElementType } from 'react'
import i18next from 'i18next'
import {
  LayoutDashboard, Package, Settings, Plug, Radio, Warehouse, History, Boxes, Truck, ShoppingBag,
  ShoppingCart, Users, AlertTriangle, PackageSearch, Clock, ScanBarcode, Tag, BarChart3, Ticket, Star,
  Webhook, MessageSquare, PackageCheck, UserCog, Layers, Mail,
} from 'lucide-react'
import type { NavItem } from '../../types'

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
}

// Single source for navigation: the Sidebar renders it, the Topbar takes
// page titles from it and the command palette lists it -- so a page's name
// is the same everywhere.
export const HOME_LINK: NavLink = { id: 'dashboard', get label() { return i18next.t('layout:nav.dashboard') }, icon: LayoutDashboard }

const GROUPS: NavGroup[] = [
  {
    id: 'sell', get label() { return i18next.t('layout:group.sell') }, items: [
      { id: 'sales-orders', get label() { return i18next.t('layout:nav.salesOrders') }, icon: ShoppingCart },
      { id: 'customers', get label() { return i18next.t('layout:nav.customers') }, icon: Users },
      { id: 'sales-channels', get label() { return i18next.t('layout:nav.salesChannels') }, icon: Radio },
    ],
  },
  {
    id: 'fulfill', get label() { return i18next.t('layout:group.fulfill') }, items: [
      { id: 'picking', get label() { return i18next.t('layout:nav.picking') }, icon: ScanBarcode },
      { id: 'packing', get label() { return i18next.t('layout:nav.packing') }, icon: PackageCheck },
      { id: 'stock-issues', get label() { return i18next.t('layout:nav.stockIssues') }, icon: AlertTriangle },
    ],
  },
  {
    id: 'catalog', get label() { return i18next.t('layout:group.catalog') }, items: [
      { id: 'products', get label() { return i18next.t('layout:nav.products') }, icon: Package },
      { id: 'collections', get label() { return i18next.t('layout:nav.collections') }, icon: Layers },
      { id: 'inventory', get label() { return i18next.t('layout:nav.inventory') }, icon: Boxes },
      { id: 'warehouses', get label() { return i18next.t('layout:nav.warehouses') }, icon: Warehouse },
      { id: 'unmatched-skus', get label() { return i18next.t('layout:nav.unmatchedSkus') }, icon: PackageSearch },
    ],
  },
  {
    id: 'buy', get label() { return i18next.t('layout:group.buy') }, items: [
      { id: 'suppliers', get label() { return i18next.t('layout:nav.suppliers') }, icon: Truck },
      { id: 'procurement', get label() { return i18next.t('layout:nav.procurement') }, icon: ShoppingBag },
    ],
  },
  {
    id: 'marketing', get label() { return i18next.t('layout:group.marketing') }, items: [
      { id: 'promotions', get label() { return i18next.t('layout:nav.promotions') }, icon: Tag },
      { id: 'coupons', get label() { return i18next.t('layout:nav.coupons') }, icon: Ticket },
      { id: 'reviews', get label() { return i18next.t('layout:nav.reviews') }, icon: Star },
      { id: 'message-center', get label() { return i18next.t('layout:nav.messageCenter') }, icon: MessageSquare },
      { id: 'newsletter', get label() { return i18next.t('layout:nav.newsletter') }, icon: Mail },
    ],
  },
  {
    id: 'reports', get label() { return i18next.t('layout:group.reports') }, collapsedByDefault: true, qualifyTitles: true, items: [
      { id: 'reports-overview', get label() { return i18next.t('layout:nav.reportsOverview') }, icon: BarChart3 },
      { id: 'reports-sales-trends', get label() { return i18next.t('layout:nav.reportsSalesTrends') }, icon: BarChart3 },
      { id: 'reports-product-performance', get label() { return i18next.t('layout:nav.reportsProductPerformance') }, icon: BarChart3 },
      { id: 'reports-product-performance-matrix', get label() { return i18next.t('layout:nav.reportsPerformanceMatrix') }, icon: BarChart3 },
      { id: 'reports-product-lifecycle', get label() { return i18next.t('layout:nav.reportsProductLifecycle') }, icon: BarChart3 },
      { id: 'reports-category-performance', get label() { return i18next.t('layout:nav.reportsCategoryPerformance') }, icon: BarChart3 },
      { id: 'reports-slow-moving-products', get label() { return i18next.t('layout:nav.reportsSlowMoving') }, icon: BarChart3 },
      { id: 'reports-stockout-risk', get label() { return i18next.t('layout:nav.reportsStockoutRisk') }, icon: BarChart3 },
      { id: 'reports-inventory-sales', get label() { return i18next.t('layout:nav.reportsInventorySales') }, icon: BarChart3 },
      { id: 'reports-customer-performance', get label() { return i18next.t('layout:nav.reportsCustomerPerformance') }, icon: BarChart3 },
      { id: 'reports-customer-cohorts', get label() { return i18next.t('layout:nav.reportsCustomerCohorts') }, icon: BarChart3 },
      { id: 'reports-customer-rfm', get label() { return i18next.t('layout:nav.reportsCustomerRfm') }, icon: BarChart3 },
      { id: 'reports-frequently-bought-together', get label() { return i18next.t('layout:nav.reportsFrequentlyBoughtTogether') }, icon: BarChart3 },
    ],
  },
  {
    id: 'connect', get label() { return i18next.t('layout:group.connect') }, items: [
      { id: 'integrations', get label() { return i18next.t('layout:nav.integrations') }, icon: Plug },
      { id: 'webhooks', get label() { return i18next.t('layout:nav.webhooks') }, icon: Webhook },
      { id: 'sync-log', get label() { return i18next.t('layout:nav.syncLog') }, icon: History },
    ],
  },
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

export function navGroupsForRole(role?: string): NavGroup[] {
  const admin: NavLink[] = [
    ...(role === 'customer_owner' || role === 'super_admin' ? [USERS_LINK] : []),
    ...(role === 'super_admin' ? [SCHEDULED_JOBS_LINK] : []),
    SETTINGS_LINK,
  ]
  return [...GROUPS, { id: 'admin', get label() { return i18next.t('layout:group.admin') }, items: admin }]
}

const ALL_GROUPS = navGroupsForRole('super_admin')

export function navTitle(nav: NavItem): string {
  if (nav === HOME_LINK.id) return HOME_LINK.label
  for (const group of ALL_GROUPS) {
    const link = group.items.find(i => i.id === nav)
    if (link) return group.qualifyTitles ? i18next.t('layout:ui.qualifiedTitle', { group: group.label, page: link.label }) : link.label
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
