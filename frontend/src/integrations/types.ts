export interface Connection {
  id: string
  provider: string
  store_url: string
  status: 'connected' | 'error'
  default_currency?: string | null
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
  deleted_products: number
  deleted_categories: number
  errors: string[]
}

export type SyncEntityType = 'products' | 'stock' | 'orders' | 'categories'
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

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
