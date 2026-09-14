import { useState, useEffect } from 'react'
import {
  LayoutDashboard, Package, Settings, Plug, Radio, Warehouse, History, Boxes, Truck, ShoppingBag,
  ShoppingCart, Users, AlertTriangle, PackageSearch, Clock, ScanBarcode, Tag, BarChart3, Ticket, Star,
  ChevronRight, ChevronDown, Zap, Webhook, MessageSquare,
} from 'lucide-react'
import { useApp } from '../../context/AppContext'
import { useAuth } from '../../auth/AuthContext'
import { useProducts } from '../../pms/hooks/useProducts'
import type { NavItem } from '../../types'
import clsx from 'clsx'

interface NavEntry {
  id: NavItem
  label: string
  icon: React.ElementType
  badge?: string | number
  badgeColor?: string
}

// A section header row -- purely a visual divider/label in the nav list,
// never itself clickable/active (see isSection).
interface NavSection {
  section: string
}

// A submenu row -- a parent label that expands/collapses to reveal its own
// child NavEntry rows (used for Reports, where every report gets its own
// page under one collapsible group instead of a single flat item).
interface NavSubmenu {
  label: string
  icon: React.ElementType
  children: NavEntry[]
}

type NavRow = NavEntry | NavSection | NavSubmenu

function isSection(row: NavRow): row is NavSection {
  return 'section' in row
}

function isSubmenu(row: NavRow): row is NavSubmenu {
  return 'children' in row
}

const REPORTS_SUBMENU: NavSubmenu = {
  label: 'Reports',
  icon: BarChart3,
  children: [
    { id: 'reports-overview', label: 'Overview', icon: BarChart3 },
    { id: 'reports-product-performance', label: 'Product Performance', icon: BarChart3 },
    { id: 'reports-product-performance-matrix', label: 'Performance Matrix', icon: BarChart3 },
    { id: 'reports-customer-performance', label: 'Customer Performance', icon: BarChart3 },
    { id: 'reports-category-performance', label: 'Category Performance', icon: BarChart3 },
    { id: 'reports-slow-moving-products', label: 'Slow-Moving Products', icon: BarChart3 },
    { id: 'reports-inventory-sales', label: 'Inventory + Sales', icon: BarChart3 },
    { id: 'reports-sales-trends', label: 'Sales Trends', icon: BarChart3 },
    { id: 'reports-product-lifecycle', label: 'Product Lifecycle', icon: BarChart3 },
    { id: 'reports-customer-cohorts', label: 'Customer Cohorts', icon: BarChart3 },
    { id: 'reports-customer-rfm', label: 'Customer Segments (RFM)', icon: BarChart3 },
    { id: 'reports-frequently-bought-together', label: 'Frequently Bought Together', icon: BarChart3 },
    { id: 'reports-stockout-risk', label: 'Stockout Risk', icon: BarChart3 },
  ],
}

const NAV_ITEMS: NavRow[] = [
  { id: 'dashboard', label: 'Dashboard', icon: LayoutDashboard },
  { id: 'sales-orders', label: 'Sales Order', icon: ShoppingCart },
  { id: 'stock-issues', label: 'Stock Issues', icon: AlertTriangle },
  { id: 'unmatched-skus', label: 'Unmatched SKUs', icon: PackageSearch },
  { id: 'customers', label: 'Customer', icon: Users },
  { id: 'products', label: 'Products', icon: Package },
  { id: 'warehouses', label: 'Warehouses', icon: Warehouse },
  { id: 'inventory', label: 'Inventory', icon: Boxes },
  // { id: 'orders', label: 'Orders', icon: ShoppingCart, badge: '38', badgeColor: 'blue' },
  { id: 'picking', label: 'Picking & Packing', icon: ScanBarcode },
  { id: 'suppliers', label: 'Suppliers', icon: Truck },
  { id: 'procurement', label: 'Procurement', icon: ShoppingBag },
  { id: 'integrations', label: 'Integrations', icon: Plug, badge: '1', badgeColor: 'red' },
  { id: 'webhooks', label: 'Webhooks', icon: Webhook },
  { id: 'sales-channels', label: 'Sales Channels', icon: Radio },
  { id: 'sync-log', label: 'Sync History', icon: History },
  { id: 'message-center', label: 'Message Center', icon: MessageSquare },
  // { id: 'shipping', label: 'Shipping', icon: Truck },
  // Intelligence & Recommendation roadmap (Phase 0) -- promotions,
  // reports, and future BI/recommendation screens all group here rather
  // than scattering as flat top-level items.
  { section: 'Intelligence' },
  { id: 'promotions', label: 'Promotions', icon: Tag },
  { id: 'coupons', label: 'Coupons', icon: Ticket },
  { id: 'reviews', label: 'Reviews', icon: Star },
  REPORTS_SUBMENU,
  { id: 'settings', label: 'Settings', icon: Settings },
]

// OWNER_NAV_ITEMS -- team/user management calls lm2's company-owner (or
// super_admin, e.g. akantio staff managing their own company for support)
// endpoints (see team/api.ts), so the nav entry itself is hidden from
// anyone else, same reasoning as SUPER_ADMIN_NAV_ITEMS.
const OWNER_NAV_ITEMS: NavRow[] = [
  { id: 'users', label: 'Users & Roles', icon: Users },
]

// SUPER_ADMIN_NAV_ITEMS are cross-tenant platform-operational pages --
// visible only to the super_admin role (matches the backend's
// middleware.RequireRole(RoleSuperAdmin) gate on their APIs), never shown
// to a regular tenant user.
const SUPER_ADMIN_NAV_ITEMS: NavRow[] = [
  { id: 'scheduled-jobs', label: 'Scheduled Jobs', icon: Clock },
]

const BADGE_COLORS: Record<string, string> = {
  red: 'bg-red-500/20 text-red-400',
  blue: 'bg-blue-500/20 text-blue-400',
  amber: 'bg-amber-500/20 text-amber-400',
  default: 'bg-gray-700 text-gray-400',
}

export default function Sidebar() {
  const { activeNav, setActiveNav } = useApp()
  const { user, logout, token } = useAuth()
  // Only page 1 is fetched here — the badge needs the server-side total,
  // not the current page's item count, now that products are paginated.
  const { total } = useProducts(token ?? '', { page: 1, perPage: 1 })

  const reportsChildActive = REPORTS_SUBMENU.children.some(c => c.id === activeNav)
  const [reportsExpanded, setReportsExpanded] = useState(reportsChildActive)
  useEffect(() => { if (reportsChildActive) setReportsExpanded(true) }, [reportsChildActive])

  const navItems: NavRow[] = [
    ...NAV_ITEMS,
    ...(user?.role === 'customer_owner' || user?.role === 'super_admin' ? OWNER_NAV_ITEMS : []),
    ...(user?.role === 'super_admin' ? SUPER_ADMIN_NAV_ITEMS : []),
  ].map(row =>
    !isSection(row) && !isSubmenu(row) && row.id === 'products' && total > 0
      ? { ...row, badge: total }
      : row,
  )

  return (
    <aside className="w-56 shrink-0 flex flex-col h-full bg-[#111827] border-r border-[#1F2937] dark:bg-[#111827] dark:border-[#1F2937] bg-white border-gray-200">
      {/* Logo */}
      <div className="h-14 flex items-center px-4 border-b border-[#1F2937] dark:border-[#1F2937] border-gray-200 shrink-0">
        <div className="flex items-center gap-2">
          <div className="w-7 h-7 rounded-lg bg-blue-600 flex items-center justify-center">
            <Zap size={14} className="text-white" />
          </div>
          <span className="font-semibold text-sm dark:text-white text-gray-900 tracking-tight">CommerceOS</span>
        </div>
      </div>

      {/* Nav */}
      <nav className="flex-1 overflow-y-auto scrollbar-none py-3 px-2">
        {navItems.map(item => {
          if (isSection(item)) {
            return (
              <p
                key={`section-${item.section}`}
                className="px-2.5 pt-3 pb-1.5 text-[10px] font-semibold uppercase tracking-wider dark:text-gray-600 text-gray-400"
              >
                {item.section}
              </p>
            )
          }
          if (isSubmenu(item)) {
            const ParentIcon = item.icon
            return (
              <div key={`submenu-${item.label}`}>
                <button
                  onClick={() => setReportsExpanded(prev => !prev)}
                  className={clsx(
                    'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left mb-0.5 transition-all duration-150',
                    reportsChildActive
                      ? 'text-blue-400'
                      : 'dark:text-gray-400 text-gray-600 hover:dark:bg-[#1F2937] hover:bg-gray-100 hover:dark:text-gray-200 hover:text-gray-800',
                  )}
                >
                  <ParentIcon size={16} className={clsx('shrink-0', reportsChildActive ? 'text-blue-400' : '')} />
                  <span className="text-xs font-medium flex-1 truncate">{item.label}</span>
                  {reportsExpanded ? <ChevronDown size={12} className="shrink-0" /> : <ChevronRight size={12} className="shrink-0" />}
                </button>
                {reportsExpanded && (
                  <div className="ml-3.5 pl-2.5 border-l dark:border-[#2D3748] border-gray-200 space-y-0.5 mb-0.5">
                    {item.children.map(child => {
                      const childActive = activeNav === child.id
                      return (
                        <button
                          key={child.id}
                          onClick={() => setActiveNav(child.id)}
                          className={clsx(
                            'w-full flex items-center px-2.5 py-1.5 rounded-lg text-left group transition-all duration-150',
                            childActive
                              ? 'bg-blue-600/15 text-blue-400'
                              : 'dark:text-gray-400 text-gray-600 hover:dark:bg-[#1F2937] hover:bg-gray-100 hover:dark:text-gray-200 hover:text-gray-800',
                          )}
                        >
                          <span className="text-xs font-medium flex-1 truncate">{child.label}</span>
                        </button>
                      )
                    })}
                  </div>
                )}
              </div>
            )
          }
          const active = activeNav === item.id
          const Icon = item.icon
          return (
            <button
              key={item.id}
              onClick={() => setActiveNav(item.id)}
              className={clsx(
                'w-full flex items-center gap-2.5 px-2.5 py-2 rounded-lg text-left mb-0.5 group transition-all duration-150',
                active
                  ? 'bg-blue-600/15 text-blue-400'
                  : 'dark:text-gray-400 text-gray-600 hover:dark:bg-[#1F2937] hover:bg-gray-100 hover:dark:text-gray-200 hover:text-gray-800',
              )}
            >
              <Icon size={16} className={clsx('shrink-0', active ? 'text-blue-400' : '')} />
              <span className="text-xs font-medium flex-1 truncate">{item.label}</span>
              {item.badge && (
                <span className={clsx(
                  'text-[10px] font-medium px-1.5 py-0.5 rounded-full',
                  item.badgeColor ? BADGE_COLORS[item.badgeColor] : BADGE_COLORS.default,
                )}>
                  {item.badge}
                </span>
              )}
              {active && <ChevronRight size={12} className="text-blue-400 shrink-0" />}
            </button>
          )
        })}
      </nav>

      {/* Footer */}
      <div className="p-3 border-t border-[#1F2937] dark:border-[#1F2937] border-gray-200 shrink-0">
        <button
          onClick={logout}
          title="Sign out"
          className="w-full flex items-center gap-2.5 px-2 py-1.5 rounded-lg hover:dark:bg-[#1F2937] hover:bg-gray-100 cursor-pointer text-left"
        >
          <div className="w-6 h-6 rounded-full bg-gradient-to-br from-blue-500 to-purple-600 shrink-0" />
          <div className="flex-1 min-w-0">
            <p className="text-xs font-medium dark:text-gray-200 text-gray-800 truncate">{user?.role ?? 'Signed in'}</p>
            <p className="text-[10px] dark:text-gray-500 text-gray-400 truncate">Sign out</p>
          </div>
        </button>
      </div>
    </aside>
  )
}
