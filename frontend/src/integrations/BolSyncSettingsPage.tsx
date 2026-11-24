import { formatMinorUnitsAsInput, parseOptionalPriceToMinorUnits } from '../pms/productFormat'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useSyncSchedules } from './hooks/useSyncSchedules'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import * as api from './api'
import type { Connection, SyncSchedule, SyncEntityType, SyncDirection } from './types'
import { useDateFormat } from '../format/DateFormatProvider'
import { INTERVAL_OPTIONS, directionLabel, runStatusLabel } from './wooSyncSettingsHelpers'
import { apiErrorText } from '../api/apiErrorText'

// Only three rows, unlike Woo's five -- Bol has no category import/export
// and no order push-back. bol_products/bol_stock are both export-only and
// both actually run the same backend operation (Bol has no stock-only
// endpoint, every push sends price+stock together) -- two rows exist only
// so the merchant can pick two separate intervals over it.
const ROWS: { entity: SyncEntityType; label: string; runnableDirections: SyncDirection[] }[] = [
  { entity: 'bol_orders', get label() { return i18next.t('integrations:entity.orders') }, runnableDirections: ['import'] },
  { entity: 'bol_products', get label() { return i18next.t('integrations:entity.products') }, runnableDirections: ['export'] },
  { entity: 'bol_stock', get label() { return i18next.t('integrations:entity.stock') }, runnableDirections: ['export'] },
]

const DIRECTIONS: SyncDirection[] = ['import', 'export']

// Bol's real Fulfilment.deliveryCode enum (Retailer API v10) -- must match
// domain/integrations.ValidBolDeliveryCodes exactly, or a save here would
// pass client-side validation but get rejected by the backend.
const DELIVERY_CODE_OPTIONS: { value: string; label: string }[] = [
  { value: '3-5d', get label() { return i18next.t('integrations:bol.delivery.days35') } },
  { value: '1-2d', get label() { return i18next.t('integrations:bol.delivery.days', { range: '1-2' }) } },
  { value: '2-3d', get label() { return i18next.t('integrations:bol.delivery.days', { range: '2-3' }) } },
  { value: '4-8d', get label() { return i18next.t('integrations:bol.delivery.days', { range: '4-8' }) } },
  { value: '1-8d', get label() { return i18next.t('integrations:bol.delivery.days', { range: '1-8' }) } },
  { value: '24uurs-23', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '23:00' }) } },
  { value: '24uurs-22', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '22:00' }) } },
  { value: '24uurs-21', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '21:00' }) } },
  { value: '24uurs-20', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '20:00' }) } },
  { value: '24uurs-19', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '19:00' }) } },
  { value: '24uurs-18', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '18:00' }) } },
  { value: '24uurs-17', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '17:00' }) } },
  { value: '24uurs-16', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '16:00' }) } },
  { value: '24uurs-15', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '15:00' }) } },
  { value: '24uurs-14', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '14:00' }) } },
  { value: '24uurs-13', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '13:00' }) } },
  { value: '24uurs-12', get label() { return i18next.t('integrations:bol.delivery.nextDay', { time: '12:00' }) } },
  { value: 'MijnLeverbelofte', get label() { return i18next.t('integrations:bol.delivery.own') } },
  { value: 'VVB', get label() { return i18next.t('integrations:bol.delivery.vvb') } },
]

type CellState = {
  enabled: boolean
  intervalMinutes: number
  defaultWarehouseId: string
  lastRunAt?: string | null
  lastRunStatus: SyncSchedule['last_run_status']
  lastRunError: string
}

function cellKey(entity: SyncEntityType, direction: SyncDirection): string {
  return `${entity}:${direction}`
}

function defaultCell(): CellState {
  return { enabled: false, intervalMinutes: 60, defaultWarehouseId: '', lastRunStatus: '', lastRunError: '' }
}

// Bol's Retailer API uses client_credentials (client_id/client_secret
// pasted from Bol's own portal) -- unlike WooCommerce there's no OAuth
// redirect/callback flow, so connecting is just a form. connectionId
// mirrors MyParcelSyncSettingsPage's pattern: undefined for a new connect,
// set to show the sync schedule grid for an already-connected integration.
export default function BolSyncSettingsPage({
  connectionId: initialConnectionId, initialConnection, onClose, onConnected,
}: { connectionId?: string; initialConnection?: Connection; onClose: () => void; onConnected: () => void }) {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['integrations', 'common'])
  const { token } = useAuth()
  const [connectionId, setConnectionId] = useState(initialConnectionId ?? '')
  const [clientId, setClientId] = useState('')
  const [clientSecret, setClientSecret] = useState('')
  const [economicOperatorId, setEconomicOperatorId] = useState('')
  const [connectError, setConnectError] = useState('')
  const [connecting, setConnecting] = useState(false)

  const { schedules, loading, error: loadError, save } = useSyncSchedules(token ?? '', connectionId || null)
  const { warehouses } = useWarehouses(token ?? '')
  const [cells, setCells] = useState<Record<string, CellState>>({})
  const [hydrated, setHydrated] = useState(false)
  const [saving, setSaving] = useState(false)
  const [saveError, setSaveError] = useState('')

  // Configuration section (economic operator id / delivery code) -- a
  // separate save target from the schedule grid above (a different API
  // call, PATCH .../settings vs PUT .../sync-schedules), so it gets its
  // own hydration/saving/error state.
  const [configEconomicOperatorId, setConfigEconomicOperatorId] = useState('')
  const [configDeliveryCode, setConfigDeliveryCode] = useState(DELIVERY_CODE_OPTIONS[0].value)
  const [configShippingCost, setConfigShippingCost] = useState('')
  const [configHydrated, setConfigHydrated] = useState(false)
  const [configSaving, setConfigSaving] = useState(false)
  const [configSaveError, setConfigSaveError] = useState('')
  const [configSaved, setConfigSaved] = useState(false)
  const [fetchingOperators, setFetchingOperators] = useState(false)
  const [fetchOperatorsError, setFetchOperatorsError] = useState('')
  const [operatorChoices, setOperatorChoices] = useState<api.BolEconomicOperator[]>([])

  if (!configHydrated && initialConnection) {
    setConfigEconomicOperatorId(initialConnection.bol_economic_operator_id ?? '')
    setConfigDeliveryCode(initialConnection.bol_delivery_code || DELIVERY_CODE_OPTIONS[0].value)
    setConfigShippingCost(formatMinorUnitsAsInput(initialConnection.bol_shipping_cost_minor_units ?? 0))
    setConfigHydrated(true)
  }

  // Bol's API returns every registered economic operator (usually exactly
  // one) -- a single result auto-fills, more than one shows a picker so
  // the merchant chooses instead of guessing which applies here.
  async function handleFetchFromBol() {
    if (!token || !connectionId) return
    setFetchingOperators(true)
    setFetchOperatorsError('')
    setOperatorChoices([])
    try {
      const operators = await api.listBolEconomicOperators(token, connectionId)
      if (operators.length === 0) {
        setFetchOperatorsError(t('bol.noOperators'))
      } else {
        setOperatorChoices(operators)
      }
    } catch (err) {
      setFetchOperatorsError(apiErrorText(err) ?? t('bol.fetchFailed'))
    } finally {
      setFetchingOperators(false)
    }
  }

  async function handleSaveConfig() {
    if (!token || !connectionId) return
    setConfigSaving(true)
    setConfigSaveError('')
    setConfigSaved(false)
    try {
      const shippingCost = parseOptionalPriceToMinorUnits(configShippingCost.replace(',', '.'))
      await api.updateConnectionSettings(token, connectionId, '', null, configEconomicOperatorId || null, configDeliveryCode || null,
        shippingCost ?? undefined)
      setConfigSaved(true)
    } catch (err) {
      setConfigSaveError(apiErrorText(err) ?? t('bol.configSaveFailed'))
    } finally {
      setConfigSaving(false)
    }
  }

  const handleConnect = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token) return
    setConnectError('')
    setConnecting(true)
    try {
      const conn = await api.connectBol(token, { clientId, clientSecret, economicOperatorId })
      setConnectionId(conn.id)
      onConnected()
    } catch (err) {
      setConnectError(apiErrorText(err) ?? t('connect.failed'))
    } finally {
      setConnecting(false)
    }
  }

  if (!hydrated && connectionId && !loading) {
    const effectiveDefaultWarehouse = warehouses.length === 1 ? warehouses[0].id : ''
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
              lastRunAt: existing.last_run_at,
              lastRunStatus: existing.last_run_status,
              lastRunError: existing.last_run_error,
            }
          : { ...defaultCell(), defaultWarehouseId: effectiveDefaultWarehouse }
      }
    }
    setCells(next)
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
            default_currency: null,
            last_run_status: c.lastRunStatus,
            last_run_error: c.lastRunError,
          }
        }),
      )
      await save(payload)
    } catch (err) {
      setSaveError(apiErrorText(err) ?? t('schedule.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const inputClass = 'px-2 py-1 text-2xs rounded bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button onClick={onClose} className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1">
            <ArrowLeft size={11} /> {t('title')}
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">{'Bol.com'}</h1>
        </div>
        {connectionId && (
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? t('common:actions.saving') : t('common:actions.save')}
          </button>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        {!connectionId ? (
          <form onSubmit={handleConnect} className="max-w-lg space-y-4">
            <p className="text-xs text-ink-4">
              {t('bol.connectIntro')}
            </p>
            <div>
              <label htmlFor="bol-client-id" className="block text-xs font-medium text-ink-3 mb-1">{t('bol.clientId')}</label>
              <input id="bol-client-id" value={clientId} onChange={e => setClientId(e.target.value)} required
                className="w-full px-3 py-2 rounded-lg text-sm bg-surface border border-line text-ink" />
            </div>
            <div>
              <label htmlFor="bol-client-secret" className="block text-xs font-medium text-ink-3 mb-1">{t('bol.clientSecret')}</label>
              <input id="bol-client-secret" type="password" value={clientSecret} onChange={e => setClientSecret(e.target.value)} required
                className="w-full px-3 py-2 rounded-lg text-sm bg-surface border border-line text-ink" />
            </div>
            <div>
              <label htmlFor="bol-economic-operator-id" className="block text-xs font-medium text-ink-3 mb-1">{t('bol.operatorId')}</label>
              <input id="bol-economic-operator-id" value={economicOperatorId} onChange={e => setEconomicOperatorId(e.target.value)} required
                className="w-full px-3 py-2 rounded-lg text-sm bg-surface border border-line text-ink" />
              <p className="text-2xs text-ink-4 mt-1">{t('bol.operatorIdHint')}</p>
            </div>
            {connectError && <p className="text-xs text-danger">{connectError}</p>}
            <button type="submit" disabled={connecting}
              className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors">
              {connecting ? t('connect.connecting') : t('connect.submit')}
            </button>
          </form>
        ) : (
          <div className="max-w-3xl space-y-6">
            <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
              <div>
                <h2 className="text-sm font-semibold text-ink mb-1">{t('bol.configuration')}</h2>
                <p className="text-xs text-ink-4">
                  {t('bol.configurationHint')}
                </p>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label htmlFor="bol-config-economic-operator-id" className="block text-xs font-medium text-ink-3 mb-1">
                    {t('bol.operatorId')}
                  </label>
                  <div className="flex gap-2">
                    <input
                      id="bol-config-economic-operator-id" value={configEconomicOperatorId}
                      onChange={e => setConfigEconomicOperatorId(e.target.value)}
                      className={clsx(inputClass, 'flex-1')}
                    />
                    <button
                      type="button" onClick={handleFetchFromBol} disabled={fetchingOperators}
                      className="px-2 py-1 text-2xs rounded bg-fill text-ink-2 hover:opacity-80 disabled:opacity-50 transition-opacity shrink-0"
                    >
                      {fetchingOperators ? t('bol.fetching') : t('bol.fetch')}
                    </button>
                  </div>
                  <p className="text-2xs text-ink-4 mt-1">
                    {t('bol.operatorUuidHint')}
                  </p>
                  {fetchOperatorsError && <p className="text-2xs text-danger mt-1">{fetchOperatorsError}</p>}
                  {operatorChoices.length > 0 && (
                    <div className="mt-2">
                      <label htmlFor="bol-config-fetched-operators" className="block text-2xs text-ink-3 mb-1">
                        {t('bol.selectFromBol')}
                      </label>
                      <select
                        id="bol-config-fetched-operators"
                        value=""
                        onChange={e => { setConfigEconomicOperatorId(e.target.value); setOperatorChoices([]) }}
                        className={clsx(inputClass, 'w-full')}
                      >
                        <option value="" disabled>{t('bol.selectOperator')}</option>
                        {operatorChoices.map(op => (
                          <option key={op.id} value={op.id}>{op.name} ({op.status}) -- {op.id}</option>
                        ))}
                      </select>
                    </div>
                  )}
                </div>
                <div>
                  <label htmlFor="bol-config-delivery-code" className="block text-xs font-medium text-ink-3 mb-1">
                    {t('bol.deliveryPromise')}
                  </label>
                  <select
                    id="bol-config-delivery-code" value={configDeliveryCode}
                    onChange={e => setConfigDeliveryCode(e.target.value)}
                    className={clsx(inputClass, 'w-full')}
                  >
                    {DELIVERY_CODE_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="bol-config-shipping-cost" className="block text-xs font-medium text-ink-3 mb-1">
                    {t('bol.shippingCostPerParcel')}
                  </label>
                  <input
                    id="bol-config-shipping-cost" inputMode="decimal" value={configShippingCost}
                    onChange={e => setConfigShippingCost(e.target.value)}
                    className={clsx(inputClass, 'w-full')}
                  />
                  <p className="text-2xs text-ink-4 mt-1">{t('bol.shippingCostPerParcelHint')}</p>
                </div>
              </div>
              {configSaveError && <p className="text-xs text-danger">{configSaveError}</p>}
              <div className="flex items-center gap-2">
                <button
                  onClick={handleSaveConfig}
                  disabled={configSaving}
                  className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
                >
                  {configSaving ? t('common:actions.saving') : t('bol.saveConfiguration')}
                </button>
                {configSaved && <span className="text-2xs text-success">{t('common:status.saved')}</span>}
              </div>
            </div>

            <div>
              <h2 className="text-sm font-semibold text-ink mb-1">{t('schedule.title')}</h2>
              <p className="text-xs text-ink-4">
                {t('bol.scheduleIntro')}
              </p>
            </div>

            {loadError && <p className="text-xs text-danger">{loadError}</p>}

            <div className="bg-raised rounded-xl border border-line divide-y divide-line">
              {ROWS.map(row => (
                <div key={row.entity} className="p-4">
                  <p className="text-xs font-semibold text-ink mb-3">{row.label}</p>
                  {row.entity === 'bol_orders' && (
                    <p className="text-2xs text-ink-4 mb-2">
                      {t('bol.ordersHint')}
                    </p>
                  )}
                  {row.entity === 'bol_stock' && (
                    <p className="text-2xs text-ink-4 mb-2">
                      {t('bol.stockHint')}
                    </p>
                  )}
                  <div className="grid grid-cols-2 gap-4">
                    {DIRECTIONS.map(direction => {
                      const key = cellKey(row.entity, direction)
                      const cell = cells[key] ?? defaultCell()
                      const runnable = row.runnableDirections.includes(direction)
                      const needsWarehouse = row.entity === 'bol_orders'
                      const showExtraFields = runnable && direction === 'import' && cell.enabled && needsWarehouse
                      return (
                        <div key={direction} className="space-y-2">
                          <div className="flex items-center justify-between">
                            <span className="text-2xs font-medium text-ink-3 caps-label capitalize">
                              {directionLabel(direction)}
                            </span>
                            <label className={clsx('relative flex items-center shrink-0', runnable ? 'cursor-pointer' : 'cursor-not-allowed opacity-40')}>
                              <input
                                type="checkbox" role="switch" aria-label={t('schedule.switchAria', { entity: row.label, direction: directionLabel(direction) })}
                                checked={cell.enabled} disabled={!runnable}
                                onChange={e => updateCell(row.entity, direction, { enabled: e.target.checked })}
                                className="sr-only peer"
                              />
                              <div className="w-8 h-4 bg-line rounded-full peer peer-checked:bg-accent-solid transition-colors" />
                              <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                            </label>
                          </div>
                          <select
                            aria-label={t('schedule.intervalAria', { entity: row.label, direction: directionLabel(direction) })}
                            value={cell.intervalMinutes} disabled={!runnable}
                            onChange={e => updateCell(row.entity, direction, { intervalMinutes: Number(e.target.value) })}
                            className={clsx(inputClass, 'w-full disabled:opacity-40 disabled:cursor-not-allowed')}
                          >
                            {INTERVAL_OPTIONS.map(o => <option key={o.minutes} value={o.minutes}>{o.label}</option>)}
                          </select>
                          {showExtraFields && (
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
          </div>
        )}
      </div>
    </div>
  )
}
