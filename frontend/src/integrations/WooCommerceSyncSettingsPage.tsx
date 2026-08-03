import { useState } from 'react'
import { ArrowLeft, X } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useSyncSchedules } from './hooks/useSyncSchedules'
import * as api from './api'
import type { Connection, SyncSchedule, SyncEntityType, SyncDirection, DeleteAllImportedProductsResult } from './types'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'

const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

// Currency codes saved before the backend started normalizing them
// (uppercase, trimmed) won't exact-match a <select> option otherwise —
// normalize on read too so existing connections/schedules aren't stuck
// unselected.
function normalizeCurrency(value: string | null | undefined): string {
  return (value ?? '').trim().toUpperCase()
}

const INTERVAL_OPTIONS: { label: string; minutes: number }[] = [
  { label: '15 minutes', minutes: 15 },
  { label: '30 minutes', minutes: 30 },
  { label: '1 hour', minutes: 60 },
  { label: '6 hours', minutes: 360 },
  { label: '12 hours', minutes: 720 },
  { label: '24 hours', minutes: 1440 },
]

const ROWS: { entity: SyncEntityType; label: string; runnable: boolean }[] = [
  { entity: 'products', label: 'Products', runnable: true },
  { entity: 'stock', label: 'Stock levels', runnable: true },
  { entity: 'orders', label: 'Orders', runnable: false },
  { entity: 'categories', label: 'Categories', runnable: false },
]

const DIRECTIONS: SyncDirection[] = ['import', 'export']

type CellState = {
  enabled: boolean
  intervalMinutes: number
  defaultWarehouseId: string
  defaultCurrency: string
  lastRunAt?: string | null
  lastRunStatus: SyncSchedule['last_run_status']
  lastRunError: string
}

function cellKey(entity: SyncEntityType, direction: SyncDirection): string {
  return `${entity}:${direction}`
}

function defaultCell(seedCurrency = ''): CellState {
  return { enabled: false, intervalMinutes: 60, defaultWarehouseId: '', defaultCurrency: seedCurrency, lastRunStatus: '', lastRunError: '' }
}

export default function WooCommerceSyncSettingsPage({ connection, onClose }: { connection: Connection; onClose: () => void }) {
  const { token } = useAuth()
  const { schedules, loading, error: loadError, save } = useSyncSchedules(token ?? '', connection.id)
  const { warehouses } = useWarehouses(token ?? '')
  const { settings: tenantSettings, loading: tenantSettingsLoading } = useTenantSettings(token ?? '')

  const [cells, setCells] = useState<Record<string, CellState>>({})
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')

  // Connection has no currency of its own yet — fall back to the
  // tenant-wide Localization default rather than leaving the field blank.
  const [storeCurrency, setStoreCurrency] = useState('')
  const [savingCurrency, setSavingCurrency] = useState(false)
  const [currencyError, setCurrencyError] = useState('')
  const [currencySaved, setCurrencySaved] = useState(false)

  async function handleSaveCurrency() {
    if (!token) return
    setSavingCurrency(true)
    setCurrencyError('')
    setCurrencySaved(false)
    try {
      await api.updateConnectionSettings(token, connection.id, storeCurrency)
      setCurrencySaved(true)
    } catch (err) {
      setCurrencyError(err instanceof Error ? err.message : 'Failed to save currency')
    } finally {
      setSavingCurrency(false)
    }
  }

  const [showDeleteModal, setShowDeleteModal] = useState(false)
  const [deleteConfirmText, setDeleteConfirmText] = useState('')
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState('')
  const [deleteResult, setDeleteResult] = useState<DeleteAllImportedProductsResult | null>(null)

  const closeDeleteModal = () => {
    setShowDeleteModal(false)
    setDeleteConfirmText('')
    setDeleteError('')
    setDeleteResult(null)
  }

  async function handleDeleteImported(e: React.FormEvent) {
    e.preventDefault()
    if (!token) return
    setDeleteError('')
    setDeleting(true)
    try {
      const result = await api.deleteAllImportedProducts(token, connection.id)
      setDeleteResult(result)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Delete failed')
    } finally {
      setDeleting(false)
    }
  }

  // Sync local editable state from the fetched schedules once — runs once,
  // matching Products.tsx's ProductEditPage hydrate pattern. Waits on
  // tenant settings too, since the currency fallback below needs them.
  if (!hydrated && !loading && !tenantSettingsLoading) {
    const effectiveDefaultCurrency = normalizeCurrency(connection.default_currency) || normalizeCurrency(tenantSettings?.default_currency)
    const next: Record<string, CellState> = {}
    for (const row of ROWS) {
      for (const direction of DIRECTIONS) {
        const key = cellKey(row.entity, direction)
        const existing = schedules.find(s => s.entity_type === row.entity && s.direction === direction)
        next[key] = existing
          ? {
              enabled: existing.enabled,
              intervalMinutes: existing.interval_minutes,
              defaultWarehouseId: existing.default_warehouse_id ?? '',
              defaultCurrency: normalizeCurrency(existing.default_currency) || effectiveDefaultCurrency,
              lastRunAt: existing.last_run_at,
              lastRunStatus: existing.last_run_status,
              lastRunError: existing.last_run_error,
            }
          : defaultCell(row.entity === 'products' ? effectiveDefaultCurrency : '')
      }
    }
    setCells(next)
    setStoreCurrency(effectiveDefaultCurrency)
    setHydrated(true)
  }

  function updateCell(entity: SyncEntityType, direction: SyncDirection, patch: Partial<CellState>) {
    const key = cellKey(entity, direction)
    setCells(prev => ({ ...prev, [key]: { ...(prev[key] ?? defaultCell()), ...patch } }))
  }

  async function handleSave() {
    setSaving(true)
    setSaveError('')
    try {
      const payload: SyncSchedule[] = ROWS.flatMap(row =>
        DIRECTIONS.map(direction => {
          const c = cells[cellKey(row.entity, direction)] ?? defaultCell()
          return {
            entity_type: row.entity,
            direction,
            enabled: row.runnable ? c.enabled : false,
            interval_minutes: c.intervalMinutes,
            default_warehouse_id: c.defaultWarehouseId || null,
            default_currency: c.defaultCurrency || null,
            last_run_status: c.lastRunStatus,
            last_run_error: c.lastRunError,
          }
        }),
      )
      await save(payload)
    } catch (err) {
      setSaveError(err instanceof Error ? err.message : 'Failed to save sync schedules')
    } finally {
      setSaving(false)
    }
  }

  const inputClass = 'px-2 py-1 text-[10px] rounded dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Integrations
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">{connection.store_url}</h1>
        </div>
        <button
          onClick={handleSave}
          disabled={saving}
          className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
        >
          {saving ? 'Saving…' : 'Save'}
        </button>
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl space-y-4">
          <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4">
            <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Store defaults</h2>
            <p className="text-xs dark:text-gray-500 text-gray-400 mb-3">
              WooCommerce's product endpoint doesn't return a currency (it's a store-level setting) — set it once here, reused by
              both manual imports and the scheduled products/import job.
            </p>
            <div className="flex items-end gap-2">
              <div className="flex-1 max-w-[160px]">
                <label htmlFor="store-default-currency" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  Default currency
                </label>
                <select
                  id="store-default-currency"
                  value={storeCurrency}
                  onChange={e => { setStoreCurrency(e.target.value); setCurrencySaved(false) }}
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                >
                  <option value="" disabled>Select a currency…</option>
                  {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                </select>
              </div>
              <button
                onClick={handleSaveCurrency}
                disabled={savingCurrency}
                className="px-3 py-2 rounded-lg dark:bg-[#111827] bg-gray-100 text-xs font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-200 disabled:opacity-50 transition-colors"
              >
                {savingCurrency ? 'Saving…' : 'Save currency'}
              </button>
            </div>
            {currencyError && <p className="text-[10px] text-red-500 mt-1.5">{currencyError}</p>}
            {currencySaved && <p className="text-[10px] text-green-500 mt-1.5">Saved.</p>}
          </div>

          <div>
            <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Sync schedule</h2>
            <p className="text-xs dark:text-gray-500 text-gray-400">
              Configure how often each type of data syncs between CommerceOS and this WooCommerce store.
            </p>
          </div>

          {loadError && <p className="text-xs text-red-400">{loadError}</p>}

          <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 divide-y dark:divide-[#2D3748] divide-gray-200">
            {ROWS.map(row => (
              <div key={row.entity} className="p-4">
                <p className="text-xs font-semibold dark:text-gray-200 text-gray-800 mb-3">{row.label}</p>
                {!row.runnable && (
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 mb-2">Coming soon — not supported yet.</p>
                )}
                <div className="grid grid-cols-2 gap-4">
                  {DIRECTIONS.map(direction => {
                    const key = cellKey(row.entity, direction)
                    const cell = cells[key] ?? defaultCell()
                    const needsWarehouse = row.entity === 'products' || row.entity === 'stock'
                    const needsCurrency = row.entity === 'products'
                    const showExtraFields = row.runnable && direction === 'import' && cell.enabled && (needsWarehouse || needsCurrency)
                    return (
                      <div key={direction} className="space-y-2">
                        <div className="flex items-center justify-between">
                          <span className="text-[10px] font-medium dark:text-gray-400 text-gray-500 uppercase tracking-wider capitalize">
                            {direction}
                          </span>
                          <label className={clsx('relative flex items-center shrink-0', row.runnable ? 'cursor-pointer' : 'cursor-not-allowed opacity-40')}>
                            <input
                              type="checkbox"
                              role="switch"
                              aria-label={`${row.label} ${direction}`}
                              checked={cell.enabled}
                              disabled={!row.runnable}
                              onChange={e => updateCell(row.entity, direction, { enabled: e.target.checked })}
                              className="sr-only peer"
                            />
                            <div className="w-8 h-4 dark:bg-[#2D3748] bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
                            <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                          </label>
                        </div>
                        <select
                          aria-label={`${row.label} ${direction} interval`}
                          value={cell.intervalMinutes}
                          disabled={!row.runnable}
                          onChange={e => updateCell(row.entity, direction, { intervalMinutes: Number(e.target.value) })}
                          className={clsx(inputClass, 'w-full disabled:opacity-40 disabled:cursor-not-allowed')}
                        >
                          {INTERVAL_OPTIONS.map(o => <option key={o.minutes} value={o.minutes}>{o.label}</option>)}
                        </select>
                        {showExtraFields && needsWarehouse && (
                          <select
                            aria-label="Default warehouse ID"
                            value={cell.defaultWarehouseId}
                            onChange={e => updateCell(row.entity, direction, { defaultWarehouseId: e.target.value })}
                            className={clsx(inputClass, 'w-full')}
                          >
                            <option value="" disabled>Select a warehouse…</option>
                            {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                          </select>
                        )}
                        {showExtraFields && needsCurrency && (
                          <select
                            aria-label="Default currency (e.g. USD)"
                            value={cell.defaultCurrency}
                            onChange={e => updateCell(row.entity, direction, { defaultCurrency: e.target.value })}
                            className={clsx(inputClass, 'w-full')}
                          >
                            <option value="" disabled>Select a currency…</option>
                            {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                          </select>
                        )}
                        {cell.lastRunAt && (
                          <p className="text-[10px] dark:text-gray-500 text-gray-400">
                            Last run: {new Date(cell.lastRunAt).toLocaleString()} ({cell.lastRunStatus || 'unknown'})
                          </p>
                        )}
                        {cell.lastRunStatus === 'error' && cell.lastRunError && (
                          <p className="text-[10px] text-red-500">{cell.lastRunError}</p>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>

          {saveError && <p className="text-xs text-red-400">{saveError}</p>}

          <div className="dark:bg-red-500/5 bg-red-50 border dark:border-red-500/20 border-red-200 rounded-xl p-4">
            <h2 className="text-sm font-semibold text-red-400 mb-1">Danger zone</h2>
            <p className="text-xs dark:text-gray-500 text-gray-400 mb-3">
              Permanently delete every product this connection has ever imported — variants, SKUs, stock history, images, and
              any category left empty afterward.
            </p>
            <button
              onClick={() => setShowDeleteModal(true)}
              className="px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs font-medium text-red-400 hover:bg-red-100 hover:dark:bg-red-500/10 transition-colors"
            >
              Delete imported products
            </button>
          </div>
        </div>
      </div>

      {showDeleteModal && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={deleteResult ? closeDeleteModal : undefined} />
          <div className="relative w-full max-w-sm dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Delete imported products</h3>
              {deleteResult && (
                <button onClick={closeDeleteModal} aria-label="Close">
                  <X size={14} className="dark:text-gray-500 text-gray-400" />
                </button>
              )}
            </div>

            {deleteResult ? (
              <div className="space-y-3">
                <p className="text-xs dark:text-gray-300 text-gray-700">
                  Deleted <strong>{deleteResult.deleted_products}</strong> product{deleteResult.deleted_products === 1 ? '' : 's'} and{' '}
                  <strong>{deleteResult.deleted_categories}</strong> now-empty categor{deleteResult.deleted_categories === 1 ? 'y' : 'ies'}.
                </p>
                {(deleteResult.errors ?? []).length > 0 && (
                  <div className="max-h-32 overflow-y-auto space-y-1">
                    {(deleteResult.errors ?? []).map((e, i) => (
                      <p key={i} className="text-[10px] text-red-400 bg-red-50 dark:bg-red-500/10 rounded px-2 py-1">{e}</p>
                    ))}
                  </div>
                )}
                <button
                  onClick={closeDeleteModal}
                  className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
                >
                  Done
                </button>
              </div>
            ) : (
              <form onSubmit={handleDeleteImported} className="space-y-3">
                <div className="dark:bg-red-500/10 bg-red-50 border dark:border-red-500/20 border-red-200 rounded-lg p-3">
                  <p className="text-xs text-red-400">
                    This permanently deletes every product imported from <strong>{connection.store_url}</strong> — including their
                    variants, SKUs, stock history, and images. Categories left empty afterward are also deleted. This cannot be undone.
                  </p>
                </div>
                <div>
                  <label htmlFor="woo-delete-confirm" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Type DELETE to confirm
                  </label>
                  <input
                    id="woo-delete-confirm"
                    type="text"
                    autoFocus
                    autoComplete="off"
                    value={deleteConfirmText}
                    onChange={e => setDeleteConfirmText(e.target.value)}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-red-500"
                  />
                </div>
                {deleteError && (
                  <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{deleteError}</p>
                )}
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={closeDeleteModal}
                    className="py-2 rounded-lg dark:bg-[#111827] bg-gray-100 text-xs font-medium dark:text-gray-300 text-gray-600 transition-colors"
                  >
                    Cancel
                  </button>
                  <button
                    type="submit"
                    disabled={deleting || deleteConfirmText !== 'DELETE'}
                    className="py-2 rounded-lg bg-red-600 text-white text-xs font-medium hover:bg-red-700 disabled:opacity-50 transition-colors"
                  >
                    {deleting ? 'Deleting…' : 'Delete'}
                  </button>
                </div>
              </form>
            )}
          </div>
        </div>
      )}
    </div>
  )
}
