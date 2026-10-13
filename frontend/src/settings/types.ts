export interface TenantSettings {
  tenant_id: string
  default_currency: string
  timezone: string
  date_format: string
  language: string
  prices_include_tax: boolean
  default_tax_country: string
  inventory_sync_rules_enabled: boolean
  batch_expiry_threshold_days: number
  // Tenant-wide fallback target markup (basis points, no upper bound, e.g.
  // 3000 = 30% markup) used to compute a SKU's recommended price when its
  // category has no override (see pms.Category.target_markup_basis_points).
  // 0 on a never-saved tenant means "unset".
  default_target_markup_basis_points: number
  // Minimum confidence (0-1, the fraction of orders containing a cart item
  // that also contain the candidate) a cross-sell recommendation must
  // clear before storefronts see it flagged as bundle-worthy. 0 on a
  // never-saved tenant means "unset" -- the backend falls back to a
  // documented default (0.30) in that case. Optional here (rather than
  // matching the backend's `required`) so existing test fixtures across
  // the frontend that predate this field don't all need updating just to
  // keep constructing a valid TenantSettings literal -- a real API
  // response always includes it.
  bundle_confidence_threshold?: number
  // Minimum number of orders an itemset must appear in before it counts as
  // a real cross-sell pattern rather than coincidence (FP-Growth's own
  // minSupport). 0 on a never-saved tenant means "unset" -- falls back to
  // a documented default (2). Optional here for the same reason as
  // bundle_confidence_threshold above.
  recommendations_min_support?: number
  // How far back (days) the backend looks for order history when mining
  // cross-sell itemsets. 0 on a never-saved tenant means "unset" -- falls
  // back to a documented default (180). Optional here for the same reason
  // as bundle_confidence_threshold above.
  recommendations_lookback_days?: number
  // How the picking module works (see backend domain/settings.PickingMode).
  // Optional for the same fixture reason as bundle_confidence_threshold --
  // a real response always includes it; missing means scan.
  picking_mode?: PickingMode
  // Daily stock-close-to-expiry report (see backend
  // domain/settings.ExpiryAlert). Optional for the same fixture reason as
  // bundle_confidence_threshold -- missing means off / no window.
  expiry_alert_enabled?: boolean
  expiry_alert_windows?: ExpiryAlertWindow[]
  // A product is low on stock when its total is above 0 and at or below
  // this (a product's own low_stock_threshold overrides it); 0 = never low.
  // Optional for the same fixture reason as bundle_confidence_threshold --
  // a real response always includes it.
  low_stock_threshold?: number
  updated_at: string
}

export interface TenantSettingsInput {
  default_currency?: string
  timezone?: string
  date_format?: string
  language?: string
  prices_include_tax?: boolean
  default_tax_country?: string
  inventory_sync_rules_enabled?: boolean
  batch_expiry_threshold_days?: number
  default_target_markup_basis_points?: number
  bundle_confidence_threshold?: number
  recommendations_min_support?: number
  recommendations_lookback_days?: number
  picking_mode?: PickingMode
  expiry_alert_enabled?: boolean
  expiry_alert_windows?: ExpiryAlertWindow[]
  low_stock_threshold?: number
}

// What a never-saved tenant reads as (backend
// settings.DefaultLowStockThreshold).
export const DEFAULT_LOW_STOCK_THRESHOLD = 10

// Days before expiry a stock report covers. Reports never overlap.
export const EXPIRY_ALERT_WINDOWS = [90, 30, 15, 7] as const
export type ExpiryAlertWindow = typeof EXPIRY_ALERT_WINDOWS[number]

// scan: pickers scan each item into a session. list: pickers see the
// order's items and mark the whole order picked in one action.
export type PickingMode = 'scan' | 'list'

export interface ApiKey {
  id: string
  name: string
  description: string
  created_at: string
  // Null for an active key.
  revoked_at: string | null
  // Null means this key is tenant-wide (e.g. the WooCommerce plugin's).
  // Set means the key is scoped to one internal storefront channel.
  sales_channel_id: string | null
}

export interface ApiKeyCreated extends ApiKey {
  // Plaintext key, present only in the response to createApiKey -- shown
  // exactly once, never retrievable again.
  key: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
