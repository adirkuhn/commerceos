import type { ComponentType } from 'react'
import type { NavItem } from './types'

import Dashboard from './pages/Dashboard'
import Products from './pages/Products'
import Promotions from './pages/Promotions'
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
import ProfitabilityPage from './pages/reports/ProfitabilityPage'
import RevenueByChannelPage from './pages/reports/RevenueByChannelPage'
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
  { nav: 'reports-profitability', path: '/reports/profitability', Component: ProfitabilityPage },
  { nav: 'reports-revenue-by-channel', path: '/reports/revenue-by-channel', Component: RevenueByChannelPage },
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

export function pathToNavItem(pathname: string): NavItem {
  return PATH_TO_NAV[pathname] ?? 'dashboard'
}
