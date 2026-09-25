import type { SyncSchedule, SyncEntityType, SyncDirection } from './types'

export const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

// Currency codes saved before the backend started normalizing them
// (uppercase, trimmed) won't exact-match a <select> option otherwise —
// normalize on read too so existing connections/schedules aren't stuck
// unselected.
export function normalizeCurrency(value: string | null | undefined): string {
  return (value ?? '').trim().toUpperCase()
}

// The scheduler polls for due schedules every 60s (see cmd/scheduler's
// pollInterval), so 1 minute is the shortest interval that's meaningful --
// anything sub-minute would just run every poll tick anyway.
export const INTERVAL_OPTIONS: { label: string; minutes: number }[] = [
  { label: '1 minute', minutes: 1 },
  { label: '5 minutes', minutes: 5 },
  { label: '10 minutes', minutes: 10 },
  { label: '15 minutes', minutes: 15 },
  { label: '30 minutes', minutes: 30 },
  { label: '1 hour', minutes: 60 },
  { label: '6 hours', minutes: 360 },
  { label: '12 hours', minutes: 720 },
  { label: '24 hours', minutes: 1440 },
]

// runnableDirections: which of Import/Export columns actually work for
// this entity. Orders is import-only -- CommerceOS never pushes an order
// it created back to Woo.
export const ROWS: { entity: SyncEntityType; label: string; runnableDirections: SyncDirection[] }[] = [
  { entity: 'products', label: 'Products', runnableDirections: ['import', 'export'] },
  { entity: 'stock', label: 'Stock levels', runnableDirections: ['import', 'export'] },
  { entity: 'orders', label: 'Orders', runnableDirections: ['import'] },
  { entity: 'orders_reconciliation', label: 'Order reconciliation', runnableDirections: ['import'] },
  { entity: 'categories', label: 'Categories', runnableDirections: [] },
]

export const DIRECTIONS: SyncDirection[] = ['import', 'export']

export type CellState = {
  enabled: boolean
  intervalMinutes: number
  defaultWarehouseId: string
  defaultCurrency: string
  lastRunAt?: string | null
  lastRunStatus: SyncSchedule['last_run_status']
  lastRunError: string
}

export function cellKey(entity: SyncEntityType, direction: SyncDirection): string {
  return `${entity}:${direction}`
}

export function defaultCell(seedCurrency = ''): CellState {
  return { enabled: false, intervalMinutes: 60, defaultWarehouseId: '', defaultCurrency: seedCurrency, lastRunStatus: '', lastRunError: '' }
}
