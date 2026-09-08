import type { ComponentType } from 'react'
import type { NavItem } from './types'

import Dashboard from './pages/Dashboard'
import Products from './pages/Products'
import Promotions from './pages/Promotions'
import Coupons from './pages/Coupons'
import Reviews from './pages/Reviews'
import Inventory from './pages/Inventory'
import Warehouses from './pages/Warehouses'
import Orders from './pages/Orders'
import Picking from './pages/Picking'
import Integrations from './pages/Integrations'
import Webhooks from './pages/Webhooks'
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
import Settings from './pages/Settings'
import ScheduledJobs from './pages/ScheduledJobs'
import OverviewPage from './pages/reports/OverviewPage'
import ProductPerformancePage from './pages/reports/ProductPerformancePage'
import ProductPerformanceMatrixPage from './pages/reports/ProductPerformanceMatrixPage'
import CustomerPerformancePage from './pages/reports/CustomerPerformancePage'
import CategoryPerformancePage from './pages/reports/CategoryPerformancePage'
import SlowMovingProductsPage from './pages/reports/SlowMovingProductsPage'
import InventorySalesPage from './pages/reports/InventorySalesPage'
import SalesTrendsPage from './pages/reports/SalesTrendsPage'
import ProductLifecyclePage from './pages/reports/ProductLifecyclePage'
import CustomerCohortsPage from './pages/reports/CustomerCohortsPage'
import CustomerRFMPage from './pages/reports/CustomerRFMPage'
import FrequentlyBoughtTogetherPage from './pages/reports/FrequentlyBoughtTogetherPage'
import StockoutRiskPage from './pages/reports/StockoutRiskPage'

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
  { nav: 'promotions', path: '/promotions', Component: Promotions },
  { nav: 'coupons', path: '/coupons', Component: Coupons },
  { nav: 'reviews', path: '/reviews', Component: Reviews },
  { nav: 'inventory', path: '/inventory', Component: Inventory },
  { nav: 'warehouses', path: '/warehouses', Component: Warehouses },
  { nav: 'orders', path: '/orders', Component: Orders },
  { nav: 'picking', path: '/picking', Component: Picking },
  { nav: 'suppliers', path: '/suppliers', Component: Suppliers },
  { nav: 'procurement', path: '/procurement', Component: Procurement },
  { nav: 'integrations', path: '/integrations', Component: Integrations },
  { nav: 'webhooks', path: '/webhooks', Component: Webhooks },
  { nav: 'sales-channels', path: '/sales-channels', Component: SalesChannels },
  { nav: 'sync-log', path: '/sync-log', Component: SyncLog },
  { nav: 'shipping', path: '/shipping', Component: Shipping },
  { nav: 'reports-overview', path: '/reports', Component: OverviewPage },
  { nav: 'reports-product-performance', path: '/reports/product-performance', Component: ProductPerformancePage },
  { nav: 'reports-product-performance-matrix', path: '/reports/product-performance-matrix', Component: ProductPerformanceMatrixPage },
  { nav: 'reports-customer-performance', path: '/reports/customer-performance', Component: CustomerPerformancePage },
  { nav: 'reports-category-performance', path: '/reports/category-performance', Component: CategoryPerformancePage },
  { nav: 'reports-slow-moving-products', path: '/reports/slow-moving-products', Component: SlowMovingProductsPage },
  { nav: 'reports-inventory-sales', path: '/reports/inventory-sales', Component: InventorySalesPage },
  { nav: 'reports-sales-trends', path: '/reports/sales-trends', Component: SalesTrendsPage },
  { nav: 'reports-product-lifecycle', path: '/reports/product-lifecycle', Component: ProductLifecyclePage },
  { nav: 'reports-customer-cohorts', path: '/reports/customer-cohorts', Component: CustomerCohortsPage },
  { nav: 'reports-customer-rfm', path: '/reports/customer-rfm', Component: CustomerRFMPage },
  { nav: 'reports-frequently-bought-together', path: '/reports/frequently-bought-together', Component: FrequentlyBoughtTogetherPage },
  { nav: 'reports-stockout-risk', path: '/reports/stockout-risk', Component: StockoutRiskPage },
  { nav: 'users', path: '/users', Component: Users },
  { nav: 'settings', path: '/settings', Component: Settings },
  { nav: 'scheduled-jobs', path: '/scheduled-jobs', Component: ScheduledJobs },
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

// A page not found here falls through to the "/*" catch-all, so it only
// needs to resolve exact base paths -- pathToNavItem below handles the
// "/products/<id>" case separately, without this map knowing about ids.
export function pathToNavItem(pathname: string): NavItem {
  if (PATH_TO_NAV[pathname]) return PATH_TO_NAV[pathname]
  // Strip one path segment (e.g. "/products/<id>" -> "/products") so a
  // per-record detail URL still highlights its owning nav item in the
  // sidebar instead of falling back to "dashboard".
  const base = '/' + pathname.split('/')[1]
  return PATH_TO_NAV[base] ?? 'dashboard'
}

// Entities that get a real per-record URL ("/products/<id>") in addition to
// their list route -- see DETAIL_PAGE_ROUTES.
const DETAIL_NAV_ITEMS: NavItem[] = ['products', 'suppliers', 'customers', 'sales-orders']

// One extra <Route> per detail-capable entity, same Component, path suffixed
// with "/:id" purely so react-router matches it instead of falling through
// to the "/*" catch-all -- the page itself reads the id back out of the URL
// with idFromPath, not useParams (see idFromPath's doc comment for why).
export const DETAIL_PAGE_ROUTES: PageRoute[] = PAGE_ROUTES
  .filter(r => DETAIL_NAV_ITEMS.includes(r.nav))
  .map(r => ({ ...r, path: `${r.path}/:id` }))

// Reads the record id out of a URL like "/products/<id>" given the entity's
// base path ("/products"). Deliberately NOT useParams: these pages are also
// rendered directly in tests without a wrapping <Route>, and more
// importantly this must reflect the current URL on every render even when
// navigating between the list and detail paths doesn't remount the page --
// useLocation (via useSearchParams/pathname) always does; a param captured
// once into local state does not, which was the root cause of search
// results silently failing to open when clicked while already on the page.
export function idFromPath(pathname: string, basePath: string): string | null {
  const prefix = `${basePath}/`
  if (!pathname.startsWith(prefix)) return null
  return decodeURIComponent(pathname.slice(prefix.length))
}
