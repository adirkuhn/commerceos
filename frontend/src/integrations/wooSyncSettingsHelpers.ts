import i18next from 'i18next'
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
  { get label() { return i18next.t('integrations:interval.minutes', { count: 1 }) }, minutes: 1 },
  { get label() { return i18next.t('integrations:interval.minutes', { count: 5 }) }, minutes: 5 },
  { get label() { return i18next.t('integrations:interval.minutes', { count: 10 }) }, minutes: 10 },
  { get label() { return i18next.t('integrations:interval.minutes', { count: 15 }) }, minutes: 15 },
  { get label() { return i18next.t('integrations:interval.minutes', { count: 30 }) }, minutes: 30 },
  { get label() { return i18next.t('integrations:interval.hours', { count: 1 }) }, minutes: 60 },
  { get label() { return i18next.t('integrations:interval.hours', { count: 6 }) }, minutes: 360 },
  { get label() { return i18next.t('integrations:interval.hours', { count: 12 }) }, minutes: 720 },
  { get label() { return i18next.t('integrations:interval.hours', { count: 24 }) }, minutes: 1440 },
]

// runnableDirections: which of Import/Export columns actually work for
// this entity. Orders is import-only -- CommerceOS never pushes an order
// it created back to Woo.
export const ROWS: { entity: SyncEntityType; label: string; runnableDirections: SyncDirection[] }[] = [
  { entity: 'orders', get label() { return i18next.t('integrations:entity.orders') }, runnableDirections: ['import'] },
  { entity: 'orders_reconciliation', get label() { return i18next.t('integrations:entity.ordersReconciliation') }, runnableDirections: ['import'] },
  { entity: 'stock_reconciliation', get label() { return i18next.t('integrations:entity.stockReconciliation') }, runnableDirections: ['export'] },
  { entity: 'categories', get label() { return i18next.t('integrations:entity.categories') }, runnableDirections: [] },
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

export function directionLabel(direction: SyncDirection): string {
  return direction === 'import' ? i18next.t('integrations:direction.import') : i18next.t('integrations:direction.export')
}

export function runStatusLabel(status: SyncSchedule['last_run_status']): string {
  if (status === 'success') return i18next.t('integrations:schedule.statusSuccess')
  if (status === 'error') return i18next.t('integrations:schedule.statusError')
  return i18next.t('integrations:schedule.statusUnknown')
}
