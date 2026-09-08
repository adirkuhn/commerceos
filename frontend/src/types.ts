export type NavItem =
  | 'dashboard'
  | 'products'
  | 'promotions'
  | 'coupons'
  | 'inventory'
  | 'warehouses'
  | 'orders'
  | 'picking'
  | 'suppliers'
  | 'procurement'
  | 'customers'
  | 'sales-orders'
  | 'stock-issues'
  | 'unmatched-skus'
  | 'integrations'
  | 'webhooks'
  | 'sales-channels'
  | 'sync-log'
  | 'shipping'
  | 'reports-overview'
  | 'reports-product-performance'
  | 'reports-product-performance-matrix'
  | 'reports-customer-performance'
  | 'reports-category-performance'
  | 'reports-slow-moving-products'
  | 'reports-inventory-sales'
  | 'reports-sales-trends'
  | 'reports-product-lifecycle'
  | 'reports-customer-cohorts'
  | 'reports-customer-rfm'
  | 'reports-frequently-bought-together'
  | 'reports-stockout-risk'
  | 'users'
  | 'settings'
  | 'scheduled-jobs'

export type ThemeMode = 'dark' | 'light'

export interface Product {
  id: string
  name: string
  sku: string
  barcode: string
  category: string
  supplier: string
  price: number
  cost: number
  margin: number
  status: 'active' | 'inactive' | 'draft'
  stock: number
  image?: string
}

export interface StockItem {
  id: string
  product: string
  sku: string
  location: string
  qty: number
  reserved: number
  available: number
  lot: string
  expiry: string
  status: 'ok' | 'expiring' | 'expired' | 'low'
}

export interface Order {
  id: string
  channel: 'shopify' | 'woocommerce' | 'bol' | 'lightspeed' | 'manual'
  customer: string
  address: string
  status: 'new' | 'reserved' | 'picking' | 'packed' | 'shipped' | 'delivered' | 'cancelled'
  total: number
  paymentStatus: 'paid' | 'pending' | 'failed'
  fulfillmentStatus: 'unfulfilled' | 'partial' | 'fulfilled'
  date: string
  items: number
}

export interface Alert {
  id: string
  type: 'critical' | 'warning' | 'info'
  message: string
  time: string
}
