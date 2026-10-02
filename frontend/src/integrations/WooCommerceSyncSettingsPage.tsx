import { Trans, useTranslation } from 'react-i18next'
import { BackfillOrderPaymentsCard } from './BackfillOrderPaymentsCard'
import { useState } from 'react'
import { ArrowLeft, X } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useSyncSchedules } from './hooks/useSyncSchedules'
import * as api from './api'
import type { Connection, SyncSchedule, SyncEntityType, SyncDirection, DeleteAllImportedProductsResult, FetchWooOrderResult } from './types'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { notifyProductsChanged } from '../pms/productsBus'
import { useQueryEnum } from '../routing/useQueryState'
import { Dialog } from '../components/common/Dialog'
import { WooTaxesTab } from './WooTaxesTab'
import { WooCategoriesTab } from './WooCategoriesTab'
import { WebhooksTab } from './WooWebhooksTab'
import { CURRENCIES, CellState, DIRECTIONS, INTERVAL_OPTIONS, ROWS, cellKey, defaultCell, directionLabel, normalizeCurrency, runStatusLabel } from './wooSyncSettingsHelpers'
import { useDateFormat } from '../format/DateFormatProvider'

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
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['integrations', 'common'])
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
      setCurrencyError(err instanceof Error ? err.message : t('woo.currencySaveFailed'))
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
      setFetchOrderError(err instanceof Error ? err.message : t('woo.fetchOrderFailed'))
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
      setDeleteError(err instanceof Error ? err.message : t('deleteImported.failed'))
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
      setSaveError(err instanceof Error ? err.message : t('schedule.saveFailed'))
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
            <ArrowLeft size={11} /> {t('title')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">{connection.store_url}</h1>
        </div>
        {tab === 'sync' && (
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? t('common:actions.saving') : t('common:actions.save')}
          </button>
        )}
      </div>

      <div className="flex gap-1 px-6 pt-3 border-b border-line bg-sunken shrink-0">
        {([
          { id: 'sync', label: t('woo.tabs.sync') },
          { id: 'taxes', label: t('woo.tabs.taxes') },
          { id: 'categories', label: t('woo.tabs.categories') },
          ...(canManageWebhooks(user?.role) ? [{ id: 'webhooks', label: t('woo.tabs.webhooks') }] as { id: Tab; label: string }[] : []),
        ] as { id: Tab; label: string }[]).map(tb => (
          <button aria-pressed={tab === tb.id}
            key={tb.id}
            onClick={() => setTab(tb.id)}
            className={clsx(
              'px-3 py-2 text-xs font-medium border-b-2 -mb-px transition-colors',
              tab === tb.id
                ? 'border-accent text-accent'
                : 'border-transparent text-ink-4 hover:text-ink-2',
            )}
          >
            {tb.label}
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
            <h2 className="text-sm font-semibold text-ink mb-1">{t('woo.storeDefaults')}</h2>
            <p className="text-xs text-ink-4 mb-3">
              {t('woo.storeDefaultsIntro')}
            </p>
            <div className="flex items-end gap-2">
              <div className="flex-1 max-w-[160px]">
                <label htmlFor="store-default-currency" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  {t('import.currency')}
                </label>
                <select
                  id="store-default-currency"
                  value={storeCurrency}
                  onChange={e => { setStoreCurrency(e.target.value); setCurrencySaved(false) }}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                >
                  <option value="" disabled>{t('import.selectCurrency')}</option>
                  {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                </select>
              </div>
              <div className="flex-1 max-w-[160px]">
                <label htmlFor="store-default-location" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  {t('woo.defaultLocation')}
                </label>
                <select
                  id="store-default-location"
                  value={storeWarehouse}
                  onChange={e => { setStoreWarehouse(e.target.value); setCurrencySaved(false) }}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                >
                  <option value="">{t('woo.noPreference')}</option>
                  {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                </select>
              </div>
              <button
                onClick={handleSaveCurrency}
                disabled={savingCurrency}
                className="px-3 py-2 rounded-lg bg-fill text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
              >
                {savingCurrency ? t('common:actions.saving') : t('woo.saveCurrency')}
              </button>
            </div>
            {currencyError && <p className="text-2xs text-danger mt-1.5">{currencyError}</p>}
            {currencySaved && <p className="text-2xs text-success mt-1.5">{t('woo.saved')}</p>}
          </div>

          <div className="bg-raised rounded-xl border border-line p-4">
            <h2 className="text-sm font-semibold text-ink mb-1">{t('woo.fetchOrderTitle')}</h2>
            <p className="text-xs text-ink-4 mb-3">
              {t('woo.fetchOrderIntro')}
            </p>
            <form onSubmit={handleFetchOrder} className="flex items-end gap-2">
              <div className="flex-1 max-w-[200px]">
                <label htmlFor="fetch-order-number" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  {t('woo.orderNumber')}
                </label>
                <input
                  id="fetch-order-number"
                  value={fetchOrderNumber}
                  onChange={e => { setFetchOrderNumber(e.target.value); setFetchOrderResult(null); setFetchOrderError('') }}
                  placeholder={t('woo.orderNumberPlaceholder')}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                />
              </div>
              <button
                type="submit"
                disabled={fetchingOrder || !fetchOrderNumber.trim()}
                className="px-3 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
              >
                {fetchingOrder ? t('bol.fetching') : t('woo.fetchOrder')}
              </button>
            </form>
            {fetchOrderError && <p className="text-2xs text-danger mt-2">{fetchOrderError}</p>}
            {fetchOrderResult && (
              <div className="mt-2 text-2xs space-y-1">
                {fetchOrderResult.errors.length > 0 ? (
                  fetchOrderResult.errors.map((e, i) => <p key={i} className="text-danger">{e}</p>)
                ) : (
                  <p className="text-success">
                    {fetchOrderResult.created
                      ? t('woo.orderImported', { id: fetchOrderResult.external_order_id })
                      : fetchOrderResult.updated
                        ? t('woo.orderUpdated', { id: fetchOrderResult.external_order_id })
                        : t('woo.orderUpToDate', { id: fetchOrderResult.external_order_id })}
                  </p>
                )}
              </div>
            )}
          </div>

          {canManageWebhooks(user?.role) && <BackfillOrderPaymentsCard token={token ?? ''} connectionId={connection.id} />}

          <div>
            <h2 className="text-sm font-semibold text-ink mb-1">{t('schedule.title')}</h2>
            <p className="text-xs text-ink-4">
              {t('woo.scheduleIntro')}
            </p>
          </div>

          {loadError && <p className="text-xs text-danger">{loadError}</p>}

          <div className="bg-raised rounded-xl border border-line divide-y divide-line">
            {ROWS.map(row => (
              <div key={row.entity} className="p-4">
                <p className="text-xs font-semibold text-ink mb-3">{row.label}</p>
                {row.runnableDirections.length === 0 && (
                  <p className="text-2xs text-ink-4 mb-2">{t('woo.comingSoon')}</p>
                )}
                {row.entity === 'orders' && (
                  <p className="text-2xs text-ink-4 mb-2">
                    {t('woo.ordersHint')}
                  </p>
                )}
                {row.entity === 'orders_reconciliation' && (
                  <p className="text-2xs text-ink-4 mb-2">
                    {t('woo.reconciliationHint')}
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
                            {directionLabel(direction)}
                          </span>
                          <label className={clsx('relative flex items-center shrink-0', runnable ? 'cursor-pointer' : 'cursor-not-allowed opacity-40')}>
                            <input
                              type="checkbox"
                              role="switch"
                              aria-label={t('schedule.switchAria', { entity: row.label, direction: directionLabel(direction) })}
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
                          aria-label={t('schedule.intervalAria', { entity: row.label, direction: directionLabel(direction) })}
                          value={cell.intervalMinutes}
                          disabled={!runnable}
                          onChange={e => updateCell(row.entity, direction, { intervalMinutes: Number(e.target.value) })}
                          className={clsx(inputClass, 'w-full disabled:opacity-40 disabled:cursor-not-allowed')}
                        >
                          {INTERVAL_OPTIONS.map(o => <option key={o.minutes} value={o.minutes}>{o.label}</option>)}
                        </select>
                        {showExtraFields && needsWarehouse && (
                          <select
                            aria-label={t('import.warehouse')}
                            value={cell.defaultWarehouseId}
                            onChange={e => updateCell(row.entity, direction, { defaultWarehouseId: e.target.value })}
                            className={clsx(inputClass, 'w-full')}
                          >
                            <option value="" disabled>{t('import.selectWarehouse')}</option>
                            {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                          </select>
                        )}
                        {showExtraFields && needsCurrency && (
                          <select
                            aria-label={t('woo.currencyAria')}
                            value={cell.defaultCurrency}
                            onChange={e => updateCell(row.entity, direction, { defaultCurrency: e.target.value })}
                            className={clsx(inputClass, 'w-full')}
                          >
                            <option value="" disabled>{t('import.selectCurrency')}</option>
                            {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                          </select>
                        )}
                        {cell.lastRunAt && (
                          <p className="text-2xs text-ink-4">
                            {t('schedule.lastRun', { date: formatDateTime(cell.lastRunAt), status: runStatusLabel(cell.lastRunStatus) })}
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
            <h2 className="text-sm font-semibold text-danger mb-1">{t('woo.dangerZone')}</h2>
            <p className="text-xs text-ink-4 mb-3">
              {t('woo.dangerZoneBody')}
            </p>
            <button
              onClick={() => setShowDeleteModal(true)}
              className="px-3 py-1.5 rounded-lg bg-raised border border-line text-xs font-medium text-danger hover:bg-red-100 hover:dark:bg-red-500/10 transition-colors"
            >
              {t('deleteImported.title')}
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
              <h3 className="text-sm font-semibold text-ink">{t('deleteImported.title')}</h3>
              {deleteResult && (
                <button onClick={closeDeleteModal} aria-label={t('common:actions.close')}>
                  <X size={14} className="text-ink-4" />
                </button>
              )}
            </div>

            {deleteResult ? (
              <div className="space-y-3">
                <p className="text-xs text-ink-2">
                  {t('deleteImported.deleted')}{deleteResult.deleted_sales_orders > 0 && (
                    <> <Trans t={t} i18nKey="deleteImported.salesOrders" count={deleteResult.deleted_sales_orders} components={{ strong: <strong /> }} />,</>
                  )} <Trans t={t} i18nKey="deleteImported.products" count={deleteResult.deleted_products} components={{ strong: <strong /> }} />{' '}
                  {t('deleteImported.and')}{' '}
                  <Trans t={t} i18nKey="deleteImported.categories" count={deleteResult.deleted_categories} components={{ strong: <strong /> }} />.
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
                  {t('done')}
                </button>
              </div>
            ) : (
              <form onSubmit={handleDeleteImported} className="space-y-3">
                <div className="dark:bg-red-500/10 bg-red-50 border dark:border-red-500/20 border-red-200 rounded-lg p-3">
                  <p className="text-xs text-danger">
                    <Trans t={t} i18nKey="deleteImported.warning" values={{ store: connection.store_url }} components={{ strong: <strong /> }} />
                  </p>
                </div>
                <div>
                  <label htmlFor="woo-delete-confirm" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    {t('deleteImported.typeToConfirm')}
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
                    {t('common:actions.cancel')}
                  </button>
                  <button
                    type="submit"
                    disabled={deleting || deleteConfirmText !== 'DELETE'}
                    className="py-2 rounded-lg bg-red-600 text-white text-xs font-medium hover:bg-red-700 disabled:opacity-50 transition-colors"
                  >
                    {deleting ? t('deleteImported.deleting') : t('common:actions.delete')}
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

