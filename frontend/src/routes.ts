import type { ComponentType } from 'react'
import type { NavItem } from './types'

import Dashboard from './pages/Dashboard'
import Products from './pages/Products'
import Inventory from './pages/Inventory'
import Warehouses from './pages/Warehouses'
import Orders from './pages/Orders'
import Picking from './pages/Picking'
import Integrations from './pages/Integrations'
import SalesChannels from './pages/SalesChannels'
import SyncLog from './pages/SyncLog'
import Users from './pages/Users'
import Suppliers from './pages/Suppliers'
import Procurement from './pages/Procurement'
import Customers from './pages/Customers'
import SalesOrders from './pages/SalesOrders'
import StockIssues from './pages/StockIssues'
import UnmatchedSkus from './pages/UnmatchedSkus'
import Shipping from './pages/Shipping'
import Reports from './pages/Reports'
import Settings from './pages/Settings'

export interface PageRoute {
  nav: NavItem
  path: string
  Component: ComponentType
}

// Single source of truth for "which URL shows which page" -- App.tsx turns
// this into <Route> elements, AppContext derives the active nav item from
// the current URL instead of tracking it as separate state (that separate
// state is what used to get lost on refresh).
export const PAGE_ROUTES: PageRoute[] = [
  { nav: 'dashboard', path: '/', Component: Dashboard },
  { nav: 'sales-orders', path: '/sales-orders', Component: SalesOrders },
  { nav: 'stock-issues', path: '/stock-issues', Component: StockIssues },
  { nav: 'unmatched-skus', path: '/unmatched-skus', Component: UnmatchedSkus },
  { nav: 'customers', path: '/customers', Component: Customers },
  { nav: 'products', path: '/products', Component: Products },
  { nav: 'inventory', path: '/inventory', Component: Inventory },
  { nav: 'warehouses', path: '/warehouses', Component: Warehouses },
  { nav: 'orders', path: '/orders', Component: Orders },
  { nav: 'picking', path: '/picking', Component: Picking },
  { nav: 'suppliers', path: '/suppliers', Component: Suppliers },
  { nav: 'procurement', path: '/procurement', Component: Procurement },
  { nav: 'integrations', path: '/integrations', Component: Integrations },
  { nav: 'sales-channels', path: '/sales-channels', Component: SalesChannels },
  { nav: 'sync-log', path: '/sync-log', Component: SyncLog },
  { nav: 'shipping', path: '/shipping', Component: Shipping },
  { nav: 'reports', path: '/reports', Component: Reports },
  { nav: 'users', path: '/users', Component: Users },
  { nav: 'settings', path: '/settings', Component: Settings },
]

const NAV_TO_PATH = Object.fromEntries(
  PAGE_ROUTES.map(r => [r.nav, r.path]),
) as Record<NavItem, string>

const PATH_TO_NAV = Object.fromEntries(
  PAGE_ROUTES.map(r => [r.path, r.nav]),
) as Record<string, NavItem>

export function navItemToPath(nav: NavItem): string {
  return NAV_TO_PATH[nav]
}

export function pathToNavItem(pathname: string): NavItem {
  return PATH_TO_NAV[pathname] ?? 'dashboard'
}
