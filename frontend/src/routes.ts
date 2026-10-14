import { createElement, type ComponentType } from 'react'
import type { NavItem } from './types'

import Home from './pages/Home'
import Products from './pages/Products'
import Promotions from './pages/Promotions'
import Coupons from './pages/Coupons'
import Collections from './pages/Collections'
import Reviews from './pages/Reviews'
import MessageCenter from './pages/MessageCenter'
import Newsletter from './pages/Newsletter'
import Inventory from './pages/Inventory'
import Warehouses from './pages/Warehouses'
import Orders from './pages/Orders'
import Picking from './pages/Picking'
import Packing from './pages/Packing'
import Integrations from './pages/Integrations'
import Webhooks from './pages/Webhooks'
import SalesChannels from './pages/SalesChannels'
import SyncLog from './pages/SyncLog'
import Users from './pages/Users'
import Suppliers from './pages/Suppliers'
import Procurement from './pages/Procurement'
import PurchasePlanner from './pages/PurchasePlanner'
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
import ShipmentsPage from './pages/reports/ShipmentsPage'
import ProductLifecyclePage from './pages/reports/ProductLifecyclePage'
import CustomerCohortsPage from './pages/reports/CustomerCohortsPage'
import CustomerRFMPage from './pages/reports/CustomerRFMPage'
import FrequentlyBoughtTogetherPage from './pages/reports/FrequentlyBoughtTogetherPage'
import StockoutRiskPage from './pages/reports/StockoutRiskPage'
import { ReportsShell } from './reports/ReportsShell'

export interface PageRoute {
  nav: NavItem
  path: string
  Component: ComponentType
}

// Single source of truth for "which URL shows which page" -- App.tsx turns
// this into <Route> elements, AppContext derives the active nav item from
// the current URL instead of tracking it as separate state (that separate
// state is what used to get lost on refresh).
// Report pages share one frame (the depot design's grouped report list).
// Wrapped once per page at module load, so the component identity is
// stable across renders.
function inReportsShell(Page: ComponentType): ComponentType {
  const Wrapped = () => createElement(ReportsShell, null, createElement(Page))
  Wrapped.displayName = `InReportsShell(${Page.displayName ?? Page.name})`
  return Wrapped
}

export const PAGE_ROUTES: PageRoute[] = [
  { nav: 'dashboard', path: '/', Component: Home },
  { nav: 'sales-orders', path: '/sales-orders', Component: SalesOrders },
  { nav: 'stock-issues', path: '/stock-issues', Component: StockIssues },
  { nav: 'unmatched-skus', path: '/unmatched-skus', Component: UnmatchedSkus },
  { nav: 'customers', path: '/customers', Component: Customers },
  { nav: 'products', path: '/products', Component: Products },
  { nav: 'collections', path: '/collections', Component: Collections },
  { nav: 'promotions', path: '/promotions', Component: Promotions },
  { nav: 'coupons', path: '/coupons', Component: Coupons },
  { nav: 'reviews', path: '/reviews', Component: Reviews },
  { nav: 'message-center', path: '/message-center', Component: MessageCenter },
  { nav: 'newsletter', path: '/newsletter', Component: Newsletter },
  { nav: 'inventory', path: '/inventory', Component: Inventory },
  { nav: 'warehouses', path: '/warehouses', Component: Warehouses },
  { nav: 'orders', path: '/orders', Component: Orders },
  { nav: 'picking', path: '/picking', Component: Picking },
  { nav: 'packing', path: '/packing', Component: Packing },
  { nav: 'suppliers', path: '/suppliers', Component: Suppliers },
  { nav: 'procurement', path: '/procurement', Component: Procurement },
  { nav: 'purchase-planner', path: '/purchase-planner', Component: PurchasePlanner },
  { nav: 'integrations', path: '/integrations', Component: Integrations },
  { nav: 'webhooks', path: '/webhooks', Component: Webhooks },
  { nav: 'sales-channels', path: '/sales-channels', Component: SalesChannels },
  { nav: 'sync-log', path: '/sync-log', Component: SyncLog },
  { nav: 'shipping', path: '/shipping', Component: Shipping },
  { nav: 'reports-overview', path: '/reports', Component: inReportsShell(OverviewPage) },
  { nav: 'reports-product-performance', path: '/reports/product-performance', Component: inReportsShell(ProductPerformancePage) },
  { nav: 'reports-product-performance-matrix', path: '/reports/product-performance-matrix', Component: inReportsShell(ProductPerformanceMatrixPage) },
  { nav: 'reports-customer-performance', path: '/reports/customer-performance', Component: inReportsShell(CustomerPerformancePage) },
  { nav: 'reports-category-performance', path: '/reports/category-performance', Component: inReportsShell(CategoryPerformancePage) },
  { nav: 'reports-slow-moving-products', path: '/reports/slow-moving-products', Component: inReportsShell(SlowMovingProductsPage) },
  { nav: 'reports-inventory-sales', path: '/reports/inventory-sales', Component: inReportsShell(InventorySalesPage) },
  { nav: 'reports-sales-trends', path: '/reports/sales-trends', Component: inReportsShell(SalesTrendsPage) },
  { nav: 'reports-product-lifecycle', path: '/reports/product-lifecycle', Component: inReportsShell(ProductLifecyclePage) },
  { nav: 'reports-customer-cohorts', path: '/reports/customer-cohorts', Component: inReportsShell(CustomerCohortsPage) },
  { nav: 'reports-customer-rfm', path: '/reports/customer-rfm', Component: inReportsShell(CustomerRFMPage) },
  { nav: 'reports-frequently-bought-together', path: '/reports/frequently-bought-together', Component: inReportsShell(FrequentlyBoughtTogetherPage) },
  { nav: 'reports-shipments', path: '/reports/shipments', Component: inReportsShell(ShipmentsPage) },
  { nav: 'reports-stockout-risk', path: '/reports/stockout-risk', Component: inReportsShell(StockoutRiskPage) },
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

// Longest base path first, so "/reports/customer-rfm/x" resolves to the
// RFM report rather than to the "/reports" overview.
const PATHS_BY_SPECIFICITY = PAGE_ROUTES
  .filter(r => r.path !== '/')
  .sort((a, b) => b.path.length - a.path.length)

// Maps any URL -- a page's base path or one of its sub-URLs
// ("/products/<id>", "/settings/tax") -- to the nav item that owns it, so
// the sidebar stays highlighted on per-record/per-section URLs.
export function pathToNavItem(pathname: string): NavItem {
  if (PATH_TO_NAV[pathname]) return PATH_TO_NAV[pathname]
  const owner = PATHS_BY_SPECIFICITY.find(r => pathname.startsWith(`${r.path}/`))
  return owner?.nav ?? 'dashboard'
}

// Every page owns the URLs under its base path ("/coupons/<id>",
// "/settings/<section>", "/picking/<sessionId>") so view state -- an open
// record, a section, a session -- lives in the URL and survives a refresh
// and Back. The page reads its sub-path itself (idFromPath); react-router
// only needs the splat so these URLs don't fall through to "Page not
// found". Static paths outrank splats in react-router's ranking, so
// "/reports/sales-trends" still matches its own page, not "/reports/*".
// The dashboard ("/") is excluded: "/*" would swallow every unknown URL.
export const SUB_PAGE_ROUTES: PageRoute[] = PAGE_ROUTES
  .filter(r => r.path !== '/')
  .map(r => ({ ...r, path: `${r.path}/*` }))

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

// All sub-path segments after `basePath`, decoded: "/sales-channels/c1/edit"
// -> ["c1", "edit"]. For pages whose sub-URLs have more than one level
// (record + view); idFromPath covers the single-id case.
export function subPathSegments(pathname: string, basePath: string): string[] {
  const prefix = `${basePath}/`
  if (!pathname.startsWith(prefix)) return []
  return pathname.slice(prefix.length).split('/').filter(Boolean).map(decodeURIComponent)
}
