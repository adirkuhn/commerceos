import { useState, useEffect } from 'react'
import { ArrowLeft, X, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useSyncSchedules } from './hooks/useSyncSchedules'
import * as api from './api'
import type { Connection, SyncSchedule, SyncEntityType, SyncDirection, DeleteAllImportedProductsResult, WooTaxClass, FetchWooOrderResult } from './types'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { notifyProductsChanged } from '../pms/productsBus'
import ProviderWebhooksPanel from './ProviderWebhooksPanel'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useExternalTaxClassLinks } from '../tax/hooks/useExternalTaxClassLinks'
import type { ExternalTaxClassLink } from '../tax/types'
import { useCategories } from '../pms/hooks/useCategories'
import { useExternalCategoryLinks } from './hooks/useExternalCategoryLinks'
import type { WooCategory, ExternalCategoryLink } from './types'
import { useQueryEnum } from '../routing/useQueryState'
import { Dialog } from '../components/common/Dialog'

const WOO_PROVIDER = 'woocommerce'

const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

// Currency codes saved before the backend started normalizing them
// (uppercase, trimmed) won't exact-match a <select> option otherwise —
// normalize on read too so existing connections/schedules aren't stuck
// unselected.
function normalizeCurrency(value: string | null | undefined): string {
  return (value ?? '').trim().toUpperCase()
}

// The scheduler polls for due schedules every 60s (see cmd/scheduler's
// pollInterval), so 1 minute is the shortest interval that's meaningful --
// anything sub-minute would just run every poll tick anyway.
const INTERVAL_OPTIONS: { label: string; minutes: number }[] = [
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
const ROWS: { entity: SyncEntityType; label: string; runnableDirections: SyncDirection[] }[] = [
  { entity: 'products', label: 'Products', runnableDirections: ['import', 'export'] },
  { entity: 'stock', label: 'Stock levels', runnableDirections: ['import', 'export'] },
  { entity: 'orders', label: 'Orders', runnableDirections: ['import'] },
  { entity: 'orders_reconciliation', label: 'Order reconciliation', runnableDirections: ['import'] },
  { entity: 'categories', label: 'Categories', runnableDirections: [] },
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

const TABS = ['sync', 'taxes', 'categories', 'webhooks'] as const
type Tab = typeof TABS[number]

// Webhook secret provisioning is a security-sensitive credential, not a
// day-to-day sync setting -- gated to the same tier as tenant-management-
// adjacent pages (Users & Roles, see Sidebar.tsx's OWNER_NAV_ITEMS),
// broadened to include customer_admin per CLAUDE.md's role table ("full
// access to own tenant, minus tenant management" -- this is integration
// management, not tenant management). The rest of this page has no
// client-side role gate at all (relies on the backend's fine-grained RBAC
// alone), so this is a deliberately stricter addition, not a match of
// existing behavior.
function canManageWebhooks(role: string | undefined): boolean {
  return role === 'customer_owner' || role === 'customer_admin' || role === 'super_admin'
}

export default function WooCommerceSyncSettingsPage({ connection, onClose }: { connection: Connection; onClose: () => void }) {
  const { token, user } = useAuth()
  const { schedules, loading, error: loadError, save } = useSyncSchedules(token ?? '', connection.id)
  const { warehouses } = useWarehouses(token ?? '')
  const { settings: tenantSettings, loading: tenantSettingsLoading } = useTenantSettings(token ?? '')

  // In the URL (?tab=) so a refresh keeps the tab open.
  const [tab, setTab] = useQueryEnum<Tab>('tab', TABS, 'sync')

  const [cells, setCells] = useState<Record<string, CellState>>({})
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')

  // Connection has no currency of its own yet — fall back to the
  // tenant-wide Localization default rather than leaving the field blank.
  const [storeCurrency, setStoreCurrency] = useState('')
  const [storeWarehouse, setStoreWarehouse] = useState('')
  const [savingCurrency, setSavingCurrency] = useState(false)
  const [currencyError, setCurrencyError] = useState('')
  const [currencySaved, setCurrencySaved] = useState(false)

  async function handleSaveCurrency() {
    if (!token) return
    setSavingCurrency(true)
    setCurrencyError('')
    setCurrencySaved(false)
    try {
      await api.updateConnectionSettings(token, connection.id, storeCurrency, storeWarehouse || null)
      setCurrencySaved(true)
    } catch (err) {
      setCurrencyError(err instanceof Error ? err.message : 'Failed to save currency')
    } finally {
      setSavingCurrency(false)
    }
  }

  const [fetchOrderNumber, setFetchOrderNumber] = useState('')
  const [fetchingOrder, setFetchingOrder] = useState(false)
  const [fetchOrderError, setFetchOrderError] = useState('')
  const [fetchOrderResult, setFetchOrderResult] = useState<FetchWooOrderResult | null>(null)

  async function handleFetchOrder(e: React.FormEvent) {
    e.preventDefault()
    if (!token || !fetchOrderNumber.trim()) return
    setFetchingOrder(true)
    setFetchOrderError('')
    setFetchOrderResult(null)
    try {
      const result = await api.fetchWooOrder(token, connection.id, fetchOrderNumber.trim())
      setFetchOrderResult(result)
    } catch (err) {
      setFetchOrderError(err instanceof Error ? err.message : 'Failed to fetch order')
    } finally {
      setFetchingOrder(false)
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
      notifyProductsChanged()
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
    // A tenant with exactly one warehouse has nothing to choose between --
    // preselect it rather than making them pick the only option every time.
    const effectiveDefaultWarehouse = connection.default_warehouse_id || (warehouses.length === 1 ? warehouses[0].id : '')
    const next: Record<string, CellState> = {}
    for (const row of ROWS) {
      for (const direction of DIRECTIONS) {
        const key = cellKey(row.entity, direction)
        const existing = schedules.find(s => s.entity_type === row.entity && s.direction === direction)
        next[key] = existing
          ? {
              enabled: existing.enabled,
              intervalMinutes: existing.interval_minutes,
              defaultWarehouseId: existing.default_warehouse_id || effectiveDefaultWarehouse,
              defaultCurrency: normalizeCurrency(existing.default_currency) || effectiveDefaultCurrency,
              lastRunAt: existing.last_run_at,
              lastRunStatus: existing.last_run_status,
              lastRunError: existing.last_run_error,
            }
          : {
              ...defaultCell(row.entity === 'products' ? effectiveDefaultCurrency : ''),
              defaultWarehouseId: effectiveDefaultWarehouse,
              // Reconciliation re-scans a fixed window regardless of
              // interval -- daily is the sensible default, no reason to
              // start it at the generic 60-minute default like every
              // other row.
              ...(row.entity === 'orders_reconciliation' ? { intervalMinutes: 1440 } : {}),
            }
      }
    }
    setCells(next)
    setStoreCurrency(effectiveDefaultCurrency)
    setStoreWarehouse(effectiveDefaultWarehouse)
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
          const runnable = row.runnableDirections.includes(direction)
          return {
            entity_type: row.entity,
            direction,
            enabled: runnable ? c.enabled : false,
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

  const inputClass = 'px-2 py-1 text-2xs rounded bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Integrations
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">{connection.store_url}</h1>
        </div>
        {tab === 'sync' && (
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
        )}
      </div>

      <div className="flex gap-1 px-6 pt-3 border-b border-line bg-sunken shrink-0">
        {([
          { id: 'sync', label: 'Sync' },
          { id: 'taxes', label: 'Taxes' },
          { id: 'categories', label: 'Categories' },
          ...(canManageWebhooks(user?.role) ? [{ id: 'webhooks', label: 'Webhooks' }] as { id: Tab; label: string }[] : []),
        ] as { id: Tab; label: string }[]).map(t => (
          <button aria-pressed={tab === t.id}
            key={t.id}
            onClick={() => setTab(t.id)}
            className={clsx(
              'px-3 py-2 text-xs font-medium border-b-2 -mb-px transition-colors',
              tab === t.id
                ? 'border-accent text-accent'
                : 'border-transparent text-ink-4 hover:text-ink-2',
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      {tab === 'taxes' && <WooTaxesTab token={token ?? ''} connectionId={connection.id} />}

      {tab === 'categories' && <WooCategoriesTab token={token ?? ''} connectionId={connection.id} />}

      {tab === 'webhooks' && canManageWebhooks(user?.role) && <WebhooksTab token={token ?? ''} connectionId={connection.id} tenantId={user?.tenant_id ?? ''} provider={connection.provider} />}

      {tab === 'sync' && (
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-3xl space-y-4">
          <div className="bg-raised rounded-xl border border-line p-4">
            <h2 className="text-sm font-semibold text-ink mb-1">Store defaults</h2>
            <p className="text-xs text-ink-4 mb-3">
              WooCommerce's product endpoint doesn't return a currency (it's a store-level setting) — set it once here, reused by
              both manual imports and the scheduled products/import job. Default location seeds each sync schedule row below
              until that row picks its own.
            </p>
            <div className="flex items-end gap-2">
              <div className="flex-1 max-w-[160px]">
                <label htmlFor="store-default-currency" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  Default currency
                </label>
                <select
                  id="store-default-currency"
                  value={storeCurrency}
                  onChange={e => { setStoreCurrency(e.target.value); setCurrencySaved(false) }}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                >
                  <option value="" disabled>Select a currency…</option>
                  {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                </select>
              </div>
              <div className="flex-1 max-w-[160px]">
                <label htmlFor="store-default-location" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  Default location
                </label>
                <select
                  id="store-default-location"
                  value={storeWarehouse}
                  onChange={e => { setStoreWarehouse(e.target.value); setCurrencySaved(false) }}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                >
                  <option value="">No preference</option>
                  {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                </select>
              </div>
              <button
                onClick={handleSaveCurrency}
                disabled={savingCurrency}
                className="px-3 py-2 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
              >
                {savingCurrency ? 'Saving…' : 'Save currency'}
              </button>
            </div>
            {currencyError && <p className="text-2xs text-danger mt-1.5">{currencyError}</p>}
            {currencySaved && <p className="text-2xs text-success mt-1.5">Saved.</p>}
          </div>

          <div className="bg-raised rounded-xl border border-line p-4">
            <h2 className="text-sm font-semibold text-ink mb-1">Fetch a specific order</h2>
            <p className="text-xs text-ink-4 mb-3">
              Import (or update) one known order right now by its WooCommerce order number, without waiting for the
              next scheduled sync -- useful for backfilling an order you already know is missing.
            </p>
            <form onSubmit={handleFetchOrder} className="flex items-end gap-2">
              <div className="flex-1 max-w-[200px]">
                <label htmlFor="fetch-order-number" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  Woo order number
                </label>
                <input
                  id="fetch-order-number"
                  value={fetchOrderNumber}
                  onChange={e => { setFetchOrderNumber(e.target.value); setFetchOrderResult(null); setFetchOrderError('') }}
                  placeholder="e.g. 1234"
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                />
              </div>
              <button
                type="submit"
                disabled={fetchingOrder || !fetchOrderNumber.trim()}
                className="px-3 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
              >
                {fetchingOrder ? 'Fetching…' : 'Fetch order'}
              </button>
            </form>
            {fetchOrderError && <p className="text-2xs text-danger mt-2">{fetchOrderError}</p>}
            {fetchOrderResult && (
              <div className="mt-2 text-2xs space-y-1">
                {fetchOrderResult.errors.length > 0 ? (
                  fetchOrderResult.errors.map((e, i) => <p key={i} className="text-danger">{e}</p>)
                ) : (
                  <p className="text-success">
                    Order {fetchOrderResult.external_order_id}{' '}
                    {fetchOrderResult.created ? 'imported' : fetchOrderResult.updated ? 'updated' : 'already up to date'}.
                  </p>
                )}
              </div>
            )}
          </div>

          <div>
            <h2 className="text-sm font-semibold text-ink mb-1">Sync schedule</h2>
            <p className="text-xs text-ink-4">
              Configure how often each type of data syncs between CommerceOS and this WooCommerce store.
            </p>
          </div>

          {loadError && <p className="text-xs text-danger">{loadError}</p>}

          <div className="bg-raised rounded-xl border border-line divide-y divide-line">
            {ROWS.map(row => (
              <div key={row.entity} className="p-4">
                <p className="text-xs font-semibold text-ink mb-3">{row.label}</p>
                {row.runnableDirections.length === 0 && (
                  <p className="text-2xs text-ink-4 mb-2">Coming soon — not supported yet.</p>
                )}
                {row.entity === 'orders' && (
                  <p className="text-2xs text-ink-4 mb-2">
                    Import only — CommerceOS never pushes orders back to Woo. The first run always does a full import;
                    every run after that is incremental.
                  </p>
                )}
                {row.entity === 'orders_reconciliation' && (
                  <p className="text-2xs text-ink-4 mb-2">
                    Re-checks the last 5 days of Woo orders on every run, independent of the Orders schedule above — catches
                    anything that schedule ever missed. Already-imported orders are skipped, so this is safe to run daily.
                  </p>
                )}
                <div className="grid grid-cols-2 gap-4">
                  {DIRECTIONS.map(direction => {
                    const key = cellKey(row.entity, direction)
                    const cell = cells[key] ?? defaultCell()
                    const runnable = row.runnableDirections.includes(direction)
                    const needsWarehouse = row.entity === 'products' || row.entity === 'stock' || row.entity === 'orders' || row.entity === 'orders_reconciliation'
                    const needsCurrency = row.entity === 'products'
                    const showExtraFields = runnable && direction === 'import' && cell.enabled && (needsWarehouse || needsCurrency)
                    return (
                      <div key={direction} className="space-y-2">
                        <div className="flex items-center justify-between">
                          <span className="text-2xs font-medium text-ink-3 uppercase tracking-wider capitalize">
                            {direction}
                          </span>
                          <label className={clsx('relative flex items-center shrink-0', runnable ? 'cursor-pointer' : 'cursor-not-allowed opacity-40')}>
                            <input
                              type="checkbox"
                              role="switch"
                              aria-label={`${row.label} ${direction}`}
                              checked={cell.enabled}
                              disabled={!runnable}
                              onChange={e => updateCell(row.entity, direction, { enabled: e.target.checked })}
                              className="sr-only peer"
                            />
                            <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-accent-solid transition-colors" />
                            <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                          </label>
                        </div>
                        <select
                          aria-label={`${row.label} ${direction} interval`}
                          value={cell.intervalMinutes}
                          disabled={!runnable}
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
                          <p className="text-2xs text-ink-4">
                            Last run: {new Date(cell.lastRunAt).toLocaleString()} ({cell.lastRunStatus || 'unknown'})
                          </p>
                        )}
                        {cell.lastRunStatus === 'error' && cell.lastRunError && (
                          <p className="text-2xs text-danger">{cell.lastRunError}</p>
                        )}
                      </div>
                    )
                  })}
                </div>
              </div>
            ))}
          </div>

          {saveError && <p className="text-xs text-danger">{saveError}</p>}

          <div className="dark:bg-red-500/5 bg-red-50 border dark:border-red-500/20 border-red-200 rounded-xl p-4">
            <h2 className="text-sm font-semibold text-danger mb-1">Danger zone</h2>
            <p className="text-xs text-ink-4 mb-3">
              Permanently delete every product this connection has ever imported — variants, SKUs, stock history, images, and
              any category left empty afterward.
            </p>
            <button
              onClick={() => setShowDeleteModal(true)}
              className="px-3 py-1.5 rounded-lg bg-raised border border-line text-xs font-medium text-danger hover:bg-red-100 hover:dark:bg-red-500/10 transition-colors"
            >
              Delete imported products
            </button>
          </div>
        </div>
      </div>
      )}

      {showDeleteModal && (
        <Dialog onClose={closeDeleteModal} className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={deleteResult ? closeDeleteModal : undefined} />
          <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">Delete imported products</h3>
              {deleteResult && (
                <button onClick={closeDeleteModal} aria-label="Close">
                  <X size={14} className="text-ink-4" />
                </button>
              )}
            </div>

            {deleteResult ? (
              <div className="space-y-3">
                <p className="text-xs text-ink-2">
                  Deleted{deleteResult.deleted_sales_orders > 0 && (
                    <> <strong>{deleteResult.deleted_sales_orders}</strong> sales order{deleteResult.deleted_sales_orders === 1 ? '' : 's'},</>
                  )} <strong>{deleteResult.deleted_products}</strong> product{deleteResult.deleted_products === 1 ? '' : 's'} and{' '}
                  <strong>{deleteResult.deleted_categories}</strong> now-empty categor{deleteResult.deleted_categories === 1 ? 'y' : 'ies'}.
                </p>
                {(deleteResult.errors ?? []).length > 0 && (
                  <div className="max-h-32 overflow-y-auto space-y-1">
                    {(deleteResult.errors ?? []).map((e, i) => (
                      <p key={i} className="text-2xs text-danger bg-red-50 dark:bg-red-500/10 rounded px-2 py-1">{e}</p>
                    ))}
                  </div>
                )}
                <button
                  onClick={closeDeleteModal}
                  className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
                >
                  Done
                </button>
              </div>
            ) : (
              <form onSubmit={handleDeleteImported} className="space-y-3">
                <div className="dark:bg-red-500/10 bg-red-50 border dark:border-red-500/20 border-red-200 rounded-lg p-3">
                  <p className="text-xs text-danger">
                    This permanently deletes every product imported from <strong>{connection.store_url}</strong> — including their
                    variants, SKUs, stock history, and images. Categories left empty afterward are also deleted. This cannot be undone.
                  </p>
                </div>
                <div>
                  <label htmlFor="woo-delete-confirm" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Type DELETE to confirm
                  </label>
                  <input
                    id="woo-delete-confirm"
                    type="text"
                    autoFocus
                    autoComplete="off"
                    value={deleteConfirmText}
                    onChange={e => setDeleteConfirmText(e.target.value)}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-red-500"
                  />
                </div>
                {deleteError && (
                  <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{deleteError}</p>
                )}
                <div className="grid grid-cols-2 gap-2">
                  <button
                    type="button"
                    onClick={closeDeleteModal}
                    className="py-2 rounded-lg bg-fill text-xs font-medium text-ink-2 transition-colors"
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
        </Dialog>
      )}
    </div>
  )
}

function WooTaxesTab({ token, connectionId }: { token: string; connectionId: string }) {
  const { classes } = useTaxClasses(token)
  const { links, error: linksError, upsert: upsertLink, remove: removeLink } = useExternalTaxClassLinks(token)
  const wooLinks = links.filter(l => l.provider === WOO_PROVIDER)

  const inputClass = 'px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

  const [rowError, setRowError] = useState('')
  const [newSlug, setNewSlug] = useState('')
  const [newClassId, setNewClassId] = useState('')

  const [wooTaxes, setWooTaxes] = useState<WooTaxClass[]>([])
  const [wooTaxesLoading, setWooTaxesLoading] = useState(true)
  const [wooTaxesError, setWooTaxesError] = useState('')

  useEffect(() => {
    if (!token || !connectionId) return
    let cancelled = false
    setWooTaxesLoading(true)
    setWooTaxesError('')
    api.listWooTaxes(token, connectionId)
      .then(result => { if (!cancelled) setWooTaxes(result) })
      .catch(err => { if (!cancelled) setWooTaxesError(err instanceof Error ? err.message : 'Failed to load WooCommerce taxes') })
      .finally(() => { if (!cancelled) setWooTaxesLoading(false) })
    return () => { cancelled = true }
  }, [token, connectionId])

  async function handleRepoint(link: ExternalTaxClassLink, taxClassId: string) {
    setRowError('')
    try {
      await upsertLink({ provider: link.provider, external_slug: link.external_slug, tax_class_id: taxClassId })
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to update tax link')
    }
  }

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    setRowError('')
    try {
      await upsertLink({ provider: WOO_PROVIDER, external_slug: newSlug, tax_class_id: newClassId })
      setNewSlug('')
      setNewClassId('')
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to add tax link')
    }
  }

  async function handleRemove(id: string) {
    setRowError('')
    try {
      await removeLink(id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to delete tax link')
    }
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="max-w-3xl space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-ink mb-1">WooCommerce taxes</h2>
          <p className="text-xs text-ink-4">
            Every WooCommerce tax class seen on import shows up here, linked to a CommerceOS tax class. Repoint or remove a link,
            or add one manually for a Woo tax class that hasn't been imported yet.
          </p>
        </div>

        {linksError && <p className="text-xs text-danger">{linksError}</p>}
        {wooTaxesError && <p className="text-xs text-danger">{wooTaxesError}</p>}
        {rowError && <p className="text-xs text-danger">{rowError}</p>}

        <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
          {wooLinks.length === 0 && <p className="text-xs text-ink-4">No WooCommerce taxes yet — run an import to pull them in, or add one below.</p>}
          {wooLinks.length > 0 && (
            <div className="space-y-1.5">
              {wooLinks.map(link => (
                <div key={link.id} className="flex items-center gap-2 text-xs text-ink-2">
                  <span className="flex-1 truncate">{link.external_slug || 'Standard'}</span>
                  <select
                    aria-label="Linked tax class"
                    value={link.tax_class_id}
                    onChange={e => handleRepoint(link, e.target.value)}
                    className={clsx(inputClass, 'w-40')}
                  >
                    {classes.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
                  </select>
                  <button
                    onClick={() => handleRemove(link.id)}
                    className="text-2xs text-ink-4 hover:text-danger transition-colors flex items-center gap-1"
                  >
                    <Trash2 size={11} /> Remove link
                  </button>
                </div>
              ))}
            </div>
          )}
          <form onSubmit={handleAdd} className="pt-3 border-t border-line flex items-end gap-2">
            <div>
              <label htmlFor="woo-tax-slug" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Woo tax class
              </label>
              <select
                id="woo-tax-slug"
                aria-label="Woo tax class"
                required
                disabled={wooTaxesLoading || wooTaxes.length === 0}
                value={newSlug}
                onChange={e => setNewSlug(e.target.value)}
                className={clsx(inputClass, 'w-32 disabled:opacity-40')}
              >
                <option value="" disabled>{wooTaxesLoading ? 'Loading…' : 'Select…'}</option>
                {wooTaxes.map(t => <option key={t.slug} value={t.slug}>{t.label}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor="woo-tax-class" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Tax class
              </label>
              <select
                id="woo-tax-class"
                aria-label="Tax class"
                required
                value={newClassId}
                onChange={e => setNewClassId(e.target.value)}
                className={clsx(inputClass, 'w-40')}
              >
                <option value="" disabled>Select…</option>
                {classes.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <button
              type="submit"
              className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
            >
              Add link
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}

function WooCategoriesTab({ token, connectionId }: { token: string; connectionId: string }) {
  const { categories } = useCategories(token)
  const { links, error: linksError, upsert: upsertLink, remove: removeLink } = useExternalCategoryLinks(token, connectionId)

  const inputClass = 'px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

  const [rowError, setRowError] = useState('')
  const [newCategoryId, setNewCategoryId] = useState('')
  const [newExternalId, setNewExternalId] = useState('')

  const [wooCategories, setWooCategories] = useState<WooCategory[]>([])
  const [wooCategoriesLoading, setWooCategoriesLoading] = useState(true)
  const [wooCategoriesError, setWooCategoriesError] = useState('')

  useEffect(() => {
    if (!token || !connectionId) return
    let cancelled = false
    setWooCategoriesLoading(true)
    setWooCategoriesError('')
    api.listWooCategories(token, connectionId)
      .then(result => { if (!cancelled) setWooCategories(result) })
      .catch(err => { if (!cancelled) setWooCategoriesError(err instanceof Error ? err.message : 'Failed to load WooCommerce categories') })
      .finally(() => { if (!cancelled) setWooCategoriesLoading(false) })
    return () => { cancelled = true }
  }, [token, connectionId])

  function wooCategoryLabel(externalId: string): string {
    const match = wooCategories.find(c => c.external_id === externalId)
    return match ? match.name : externalId
  }
  function categoryLabel(categoryId: string): string {
    const match = categories.find(c => c.id === categoryId)
    return match ? match.name : categoryId
  }

  async function handleRepoint(link: ExternalCategoryLink, externalId: string) {
    setRowError('')
    try {
      await upsertLink(externalId, link.category_id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to update category link')
    }
  }

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    setRowError('')
    try {
      await upsertLink(newExternalId, newCategoryId)
      setNewCategoryId('')
      setNewExternalId('')
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to add category link')
    }
  }

  async function handleRemove(id: string) {
    setRowError('')
    try {
      await removeLink(id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to delete category link')
    }
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="max-w-3xl space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-ink mb-1">WooCommerce categories</h2>
          <p className="text-xs text-ink-4">
            A CommerceOS category is auto-matched to a WooCommerce category with the same name the first time a
            product in it is pushed. Repoint or remove a link, or add one manually for a category that hasn't
            matched automatically.
          </p>
        </div>

        {linksError && <p className="text-xs text-danger">{linksError}</p>}
        {wooCategoriesError && <p className="text-xs text-danger">{wooCategoriesError}</p>}
        {rowError && <p className="text-xs text-danger">{rowError}</p>}

        <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
          {links.length === 0 && <p className="text-xs text-ink-4">No category links yet — push a product to auto-match, or add one below.</p>}
          {links.length > 0 && (
            <div className="space-y-1.5">
              {links.map(link => (
                <div key={link.id} className="flex items-center gap-2 text-xs text-ink-2">
                  <span className="flex-1 truncate">{categoryLabel(link.category_id)}</span>
                  <select
                    aria-label="Linked Woo category"
                    value={link.external_id}
                    onChange={e => handleRepoint(link, e.target.value)}
                    className={clsx(inputClass, 'w-40')}
                  >
                    <option value={link.external_id}>{wooCategoryLabel(link.external_id)}</option>
                    {wooCategories.filter(c => c.external_id !== link.external_id).map(c => (
                      <option key={c.external_id} value={c.external_id}>{c.name}</option>
                    ))}
                  </select>
                  <button
                    onClick={() => handleRemove(link.id)}
                    className="text-2xs text-ink-4 hover:text-danger transition-colors flex items-center gap-1"
                  >
                    <Trash2 size={11} /> Remove link
                  </button>
                </div>
              ))}
            </div>
          )}
          <form onSubmit={handleAdd} className="pt-3 border-t border-line flex items-end gap-2">
            <div>
              <label htmlFor="commerceos-category" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Category
              </label>
              <select
                id="commerceos-category"
                aria-label="Category"
                required
                value={newCategoryId}
                onChange={e => setNewCategoryId(e.target.value)}
                className={clsx(inputClass, 'w-40')}
              >
                <option value="" disabled>Select…</option>
                {categories.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
              </select>
            </div>
            <div>
              <label htmlFor="woo-category" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Woo category
              </label>
              <select
                id="woo-category"
                aria-label="Woo category"
                required
                disabled={wooCategoriesLoading || wooCategories.length === 0}
                value={newExternalId}
                onChange={e => setNewExternalId(e.target.value)}
                className={clsx(inputClass, 'w-32 disabled:opacity-40')}
              >
                <option value="" disabled>{wooCategoriesLoading ? 'Loading…' : 'Select…'}</option>
                {wooCategories.map(c => <option key={c.external_id} value={c.external_id}>{c.name}</option>)}
              </select>
            </div>
            <button
              type="submit"
              className="px-3 py-1.5 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
            >
              Add link
            </button>
          </form>
        </div>
      </div>
    </div>
  )
}

// The webhook receiver itself lives at a URL outside VITE_API_BASE_URL's
// per-domain convention (see ../api.ts's base()) -- it's the single
// unauthenticated route, mounted at /webhooks/v1/{tenant_id}/{provider}
// on the API's root router, not under /integrations. Built here rather
// than exported from api.ts since it's pure display, no request involved.
function webhookURL(tenantId: string, provider: string): string {
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string) ?? ''
  return `${apiBase}/webhooks/v1/${tenantId}/${provider}`
}

function WebhooksTab({ token, connectionId, tenantId, provider }: { token: string; connectionId: string; tenantId: string; provider: string }) {
  const [secret, setSecret] = useState('')
  const [generating, setGenerating] = useState(false)
  const [error, setError] = useState('')
  const [copiedField, setCopiedField] = useState<'url' | 'secret' | null>(null)

  async function handleGenerate() {
    setError('')
    setGenerating(true)
    try {
      const result = await api.setWebhookSecret(token, connectionId)
      setSecret(result.webhook_secret)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to generate webhook secret')
    } finally {
      setGenerating(false)
    }
  }

  async function handleCopy(field: 'url' | 'secret', value: string) {
    try {
      await navigator.clipboard.writeText(value)
      setCopiedField(field)
      setTimeout(() => setCopiedField(prev => (prev === field ? null : prev)), 1500)
    } catch {
      // Clipboard access can be denied (permissions, insecure context) --
      // the value is already shown selectable on screen, so this is a
      // convenience failure, not a blocker; no error surfaced for it.
    }
  }

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="max-w-3xl space-y-4">
        <div>
          <h2 className="text-sm font-semibold text-ink mb-1">Webhook signature secret</h2>
          <p className="text-xs text-ink-4">
            Generate a secret, then configure it as this webhook's "Secret" when creating it under WooCommerce → Settings →
            Advanced → Webhooks. Regenerating replaces the old value immediately -- update WooCommerce's webhook config with
            the new one, or deliveries will start failing signature verification.
          </p>
        </div>

        <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
          <div>
            <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              Delivery URL
            </label>
            <div className="flex items-center gap-2">
              <code className="flex-1 px-3 py-2 text-2xs rounded-lg bg-sunken border border-line text-ink-2 overflow-x-auto whitespace-nowrap">
                {webhookURL(tenantId, provider)}
              </code>
              <button
                type="button"
                onClick={() => handleCopy('url', webhookURL(tenantId, provider))}
                className="px-2.5 py-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors shrink-0"
              >
                {copiedField === 'url' ? 'Copied' : 'Copy'}
              </button>
            </div>
          </div>

          {secret ? (
            <div>
              <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                Secret (shown once -- copy it now)
              </label>
              <div className="flex items-center gap-2">
                <code className="flex-1 px-3 py-2 text-2xs font-mono rounded-lg bg-sunken border border-line text-ink-2 overflow-x-auto whitespace-nowrap">
                  {secret}
                </code>
                <button
                  type="button"
                  onClick={() => handleCopy('secret', secret)}
                  className="px-2.5 py-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors shrink-0"
                >
                  {copiedField === 'secret' ? 'Copied' : 'Copy'}
                </button>
              </div>
              <p className="mt-1.5 text-2xs text-warning">
                This won't be shown again -- if you lose it, generate a new one and update WooCommerce.
              </p>
            </div>
          ) : (
            <p className="text-xs text-ink-4">No secret generated yet -- webhooks from this connection can't be verified until you generate one.</p>
          )}

          {error && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>}

          <button
            type="button"
            onClick={handleGenerate}
            disabled={generating}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {generating ? 'Generating…' : secret ? 'Regenerate secret' : 'Generate secret'}
          </button>
        </div>

        <ProviderWebhooksPanel token={token} connectionId={connectionId} />
      </div>
    </div>
  )
}

