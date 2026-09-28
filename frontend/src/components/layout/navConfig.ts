import type { ElementType } from 'react'
import {
  LayoutDashboard, Package, Settings, Plug, Radio, Warehouse, History, Boxes, Truck, ShoppingBag,
  ShoppingCart, Users, AlertTriangle, PackageSearch, Clock, ScanBarcode, Tag, BarChart3, Ticket, Star,
  Webhook, MessageSquare, PackageCheck, UserCog, Layers,
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
export const HOME_LINK: NavLink = { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard }

const GROUPS: NavGroup[] = [
  {
    id: 'sell', label: 'Sell', items: [
      { id: 'sales-orders', label: 'Sales Orders', icon: ShoppingCart },
      { id: 'customers', label: 'Customers', icon: Users },
      { id: 'sales-channels', label: 'Sales Channels', icon: Radio },
    ],
  },
  {
    id: 'fulfill', label: 'Fulfill', items: [
      { id: 'picking', label: 'Picking', icon: ScanBarcode },
      { id: 'packing', label: 'Packing', icon: PackageCheck },
      { id: 'stock-issues', label: 'Stock Issues', icon: AlertTriangle },
    ],
  },
  {
    id: 'catalog', label: 'Catalog', items: [
      { id: 'products', label: 'Products', icon: Package },
      { id: 'collections', label: 'Collections', icon: Layers },
      { id: 'inventory', label: 'Inventory', icon: Boxes },
      { id: 'warehouses', label: 'Warehouses', icon: Warehouse },
      { id: 'unmatched-skus', label: 'Unmatched SKUs', icon: PackageSearch },
    ],
  },
  {
    id: 'buy', label: 'Buy', items: [
      { id: 'suppliers', label: 'Suppliers', icon: Truck },
      { id: 'procurement', label: 'Procurement', icon: ShoppingBag },
    ],
  },
  {
    id: 'marketing', label: 'Marketing', items: [
      { id: 'promotions', label: 'Promotions', icon: Tag },
      { id: 'coupons', label: 'Coupons', icon: Ticket },
      { id: 'reviews', label: 'Reviews', icon: Star },
      { id: 'message-center', label: 'Message Center', icon: MessageSquare },
    ],
  },
  {
    id: 'reports', label: 'Reports', collapsedByDefault: true, qualifyTitles: true, items: [
      { id: 'reports-overview', label: 'Overview', icon: BarChart3 },
      { id: 'reports-sales-trends', label: 'Sales Trends', icon: BarChart3 },
      { id: 'reports-product-performance', label: 'Product Performance', icon: BarChart3 },
      { id: 'reports-product-performance-matrix', label: 'Performance Matrix', icon: BarChart3 },
      { id: 'reports-product-lifecycle', label: 'Product Lifecycle', icon: BarChart3 },
      { id: 'reports-category-performance', label: 'Category Performance', icon: BarChart3 },
      { id: 'reports-slow-moving-products', label: 'Slow-Moving Products', icon: BarChart3 },
      { id: 'reports-stockout-risk', label: 'Stockout Risk', icon: BarChart3 },
      { id: 'reports-inventory-sales', label: 'Inventory + Sales', icon: BarChart3 },
      { id: 'reports-customer-performance', label: 'Customer Performance', icon: BarChart3 },
      { id: 'reports-customer-cohorts', label: 'Customer Cohorts', icon: BarChart3 },
      { id: 'reports-customer-rfm', label: 'Customer Segments (RFM)', icon: BarChart3 },
      { id: 'reports-frequently-bought-together', label: 'Frequently Bought Together', icon: BarChart3 },
    ],
  },
  {
    id: 'connect', label: 'Connect', items: [
      { id: 'integrations', label: 'Integrations', icon: Plug },
      { id: 'webhooks', label: 'Webhooks', icon: Webhook },
      { id: 'sync-log', label: 'Sync History', icon: History },
    ],
  },
]

// Team management calls lm2's company-owner endpoints (team/api.ts);
// platform operations match the backend's RequireRole(RoleSuperAdmin).
const USERS_LINK: NavLink = { id: 'users', label: 'Users & Roles', icon: UserCog }
const SCHEDULED_JOBS_LINK: NavLink = { id: 'scheduled-jobs', label: 'Scheduled Jobs', icon: Clock }
const SETTINGS_LINK: NavLink = { id: 'settings', label: 'Settings', icon: Settings }

// Pages reachable by URL but not listed in the sidebar (demo screens).
const UNLISTED_TITLES: Partial<Record<NavItem, string>> = {
  orders: 'Orders',
  shipping: 'Shipping',
}

export function navGroupsForRole(role?: string): NavGroup[] {
  const admin: NavLink[] = [
    ...(role === 'customer_owner' || role === 'super_admin' ? [USERS_LINK] : []),
    ...(role === 'super_admin' ? [SCHEDULED_JOBS_LINK] : []),
    SETTINGS_LINK,
  ]
  return [...GROUPS, { id: 'admin', label: 'Admin', items: admin }]
}

const ALL_GROUPS = navGroupsForRole('super_admin')

export function navTitle(nav: NavItem): string {
  if (nav === HOME_LINK.id) return HOME_LINK.label
  for (const group of ALL_GROUPS) {
    const link = group.items.find(i => i.id === nav)
    if (link) return group.qualifyTitles ? `${group.label}: ${link.label}` : link.label
  }
  return UNLISTED_TITLES[nav] ?? nav
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

const ROLE_LABELS: Record<string, string> = {
  customer_owner: 'Owner',
  customer_admin: 'Admin',
  customer_user: 'User',
  super_admin: 'Platform admin',
}

export function roleLabel(role: string | undefined): string {
  return (role && ROLE_LABELS[role]) || 'Signed in'
}
