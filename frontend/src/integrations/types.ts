export interface WooTaxClass {
  slug: string
  label: string
}

export interface Connection {
  id: string
  provider: string
  store_url: string
  status: 'connected' | 'error'
  default_currency?: string | null
  default_warehouse_id?: string | null
  created_at: string
}

export interface CatalogEntry {
  provider: string
  name: string
}

export interface StartWooConnectResponse {
  authorize_url: string
}

export type ImportJobStatus = 'pending' | 'running' | 'completed' | 'failed'

// The import endpoint enqueues a background job and returns immediately
// instead of blocking on the full sync (which can take minutes for a real
// catalog) — see cmd/scheduler for the worker that actually runs it, and
// the notification center for how progress/completion is surfaced.
export interface ImportJob {
  id: string
  connection_id: string
  status: ImportJobStatus
  imported: number
  already_linked: number
  skipped_type: number
  errors: string[]
  error_message: string
  created_at: string
  started_at?: string | null
  finished_at?: string | null
}

export interface DeleteAllImportedProductsResult {
  deleted_sales_orders: number
  deleted_products: number
  deleted_categories: number
  errors: string[]
}

export interface FetchWooOrderResult {
  external_order_id: string
  sales_order_id?: string | null
  created: boolean
  updated: boolean
  unchanged: boolean
  errors: string[]
}

export type SyncEntityType = 'products' | 'stock' | 'orders' | 'categories' | 'orders_reconciliation'
export type SyncDirection = 'import' | 'export'

export interface SyncSchedule {
  entity_type: SyncEntityType
  direction: SyncDirection
  enabled: boolean
  interval_minutes: number
  default_warehouse_id?: string | null
  default_currency?: string | null
  last_run_at?: string | null
  last_run_status: '' | 'success' | 'error'
  last_run_error: string
}

export type ProductChannelListingSyncStatus = 'pending' | 'synced' | 'error'

export interface ProductChannelListing {
  id: string
  product_id: string
  connection_id: string
  provider: string
  external_id: string
  sync_status: ProductChannelListingSyncStatus
  last_synced_at?: string | null
  created_at: string
  updated_at: string
}

export type SyncLogDirection = 'push' | 'import'
export type SyncLogStatus = 'success' | 'error'

export interface FieldChange {
  from: string
  to: string
}

export interface SyncLogEntry {
  id: string
  listing_id: string
  product_id: string
  connection_id: string
  provider: string
  external_id: string
  direction: SyncLogDirection
  status: SyncLogStatus
  error_message: string
  fields: Record<string, string>
  changes: Record<string, FieldChange>
  synced_at: string
  // Set once staff acknowledge this failure from the "needs attention"
  // view without retrying it.
  dismissed_at?: string | null
  // Only populated by listAllSyncLogs (the tenant-wide history list) — the
  // per-listing endpoint's caller already knows the product.
  product_name?: string
  store_url?: string
}

export type AIProvider = 'openai' | 'anthropic' | 'gemini' | 'ollama'

export interface AIConnection {
  id: string
  provider: AIProvider
  base_url?: string | null
  model?: string | null
  status: 'connected' | 'error'
  created_at: string
}

export type ChannelType = 'woocommerce' | 'internal'
export type EnrollmentMode = 'all' | 'category' | 'tag' | 'manual'

export interface SalesChannel {
  id: string
  name: string
  type: ChannelType
  connection_id?: string | null
  enrollment_mode: EnrollmentMode
  enrollment_category_id?: string | null
  enrollment_tag?: string | null
  active: boolean
  created_at: string
  updated_at: string
}

export interface SalesChannelProduct {
  id: string
  name: string
  slug: string
  category_id?: string | null
  tags?: string[]
}

export interface SalesChannelProductEnrollment {
  product: SalesChannelProduct
  enrolled: boolean
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
