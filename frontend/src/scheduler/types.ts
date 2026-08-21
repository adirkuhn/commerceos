export type LastRunStatus = '' | 'success' | 'error'

// Job is a global cron job (cost-price recalculation, expired sale-price/
// batch sweeps) -- one run touches every tenant, super_admin only. See
// backend domain/scheduler.Job.
export interface Job {
  job_key: string
  enabled: boolean
  interval_minutes: number
  last_run_at: string | null
  last_run_status: LastRunStatus
  last_run_error: string
  next_run_at: string
}

// SyncScheduleRow is one tenant's WooCommerce sync schedule, enriched with
// its connection's store URL for display -- cross-tenant list, super_admin
// only. See backend domain/channels.SyncSchedule.
export interface SyncScheduleRow {
  id: string
  tenant_id: string
  connection_id: string
  connection_store_url: string
  entity_type: 'products' | 'stock' | 'orders' | 'categories' | 'orders_reconciliation'
  direction: 'import' | 'export'
  enabled: boolean
  interval_minutes: number
  last_run_at: string | null
  last_run_status: LastRunStatus
  last_run_error: string
  next_run_at: string | null
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
