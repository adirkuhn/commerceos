import type { ComponentType } from 'react'
import type { Product } from './catalog'
import type { PageSection } from './content'

export interface RouteDefinition {
  path: string
  element: ComponentType
}

export interface NavigationItem {
  label: string
  href: string
  position: 'header' | 'footer' | 'account'
}

export interface ProductTab {
  id: string
  label: string
  render: (product: Product) => ComponentType | null
}

export interface CheckoutExtension {
  id: string
  step: 'shipping' | 'payment' | 'review'
  render: ComponentType
}

export interface AccountExtension {
  id: string
  label: string
  path: string
  render: ComponentType
}

export interface DashboardWidget {
  id: string
  render: ComponentType
}

export interface PageSectionDefinition {
  type: string
  render: (section: PageSection) => ComponentType | null
}

export type HookName =
  | 'product_viewed' | 'cart_item_added' | 'cart_item_removed' | 'checkout_started' | 'order_placed'

export type HookHandler = (payload: unknown) => void

export interface EcommercePlugin {
  name: string
  routes?: RouteDefinition[]
  navigationItems?: NavigationItem[]
  productTabs?: ProductTab[]
  checkoutExtensions?: CheckoutExtension[]
  accountExtensions?: AccountExtension[]
  sections?: PageSectionDefinition[]
  dashboardWidgets?: DashboardWidget[]
  hooks?: Partial<Record<HookName, HookHandler>>
}
