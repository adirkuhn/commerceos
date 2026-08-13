export type NavItem =
  | 'dashboard'
  | 'products'
  | 'inventory'
  | 'warehouses'
  | 'orders'
  | 'picking'
  | 'suppliers'
  | 'procurement'
  | 'customers'
  | 'sales-orders'
  | 'integrations'
  | 'sales-channels'
  | 'sync-log'
  | 'shipping'
  | 'reports'
  | 'users'
  | 'settings'

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
