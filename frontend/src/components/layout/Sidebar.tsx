import {
  LayoutDashboard, Package, Settings, Plug, Radio, Warehouse, History, Boxes, Truck, ShoppingBag,
  ShoppingCart, Users, AlertTriangle, PackageSearch, Clock, ScanBarcode,
  // BarChart3,
  ChevronRight, Zap,
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

const NAV_ITEMS: NavEntry[] = [
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
  { id: 'sales-channels', label: 'Sales Channels', icon: Radio },
  { id: 'sync-log', label: 'Sync History', icon: History },
  // { id: 'shipping', label: 'Shipping', icon: Truck },
  // { id: 'reports', label: 'Reports', icon: BarChart3 },
  { id: 'settings', label: 'Settings', icon: Settings },
]

// OWNER_NAV_ITEMS -- team/user management calls lm2's company-owner (or
// super_admin, e.g. akantio staff managing their own company for support)
// endpoints (see team/api.ts), so the nav entry itself is hidden from
// anyone else, same reasoning as SUPER_ADMIN_NAV_ITEMS.
const OWNER_NAV_ITEMS: NavEntry[] = [
  { id: 'users', label: 'Users & Roles', icon: Users },
]

// SUPER_ADMIN_NAV_ITEMS are cross-tenant platform-operational pages --
// visible only to the super_admin role (matches the backend's
// middleware.RequireRole(RoleSuperAdmin) gate on their APIs), never shown
// to a regular tenant user.
const SUPER_ADMIN_NAV_ITEMS: NavEntry[] = [
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

  const navItems = [
    ...NAV_ITEMS,
    ...(user?.role === 'customer_owner' || user?.role === 'super_admin' ? OWNER_NAV_ITEMS : []),
    ...(user?.role === 'super_admin' ? SUPER_ADMIN_NAV_ITEMS : []),
  ].map(item =>
    item.id === 'products' && total > 0
      ? { ...item, badge: total }
      : item,
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
