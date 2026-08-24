import { useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useSyncSchedules } from './hooks/useSyncSchedules'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import * as api from './api'
import type { SyncSchedule, SyncEntityType, SyncDirection } from './types'

// The scheduler polls for due schedules every 60s (see cmd/scheduler's
// pollInterval) -- same options WooCommerceSyncSettingsPage offers.
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

// Only three rows, unlike Woo's five -- Bol has no category import/export
// and no order push-back. bol_products/bol_stock are both export-only and
// both actually run the same backend operation (Bol has no stock-only
// endpoint, every push sends price+stock together) -- two rows exist only
// so the merchant can pick two separate intervals over it.
const ROWS: { entity: SyncEntityType; label: string; runnableDirections: SyncDirection[] }[] = [
  { entity: 'bol_orders', label: 'Orders', runnableDirections: ['import'] },
  { entity: 'bol_products', label: 'Products', runnableDirections: ['export'] },
  { entity: 'bol_stock', label: 'Stock levels', runnableDirections: ['export'] },
]

const DIRECTIONS: SyncDirection[] = ['import', 'export']

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
  connectionId: initialConnectionId, onClose, onConnected,
}: { connectionId?: string; onClose: () => void; onConnected: () => void }) {
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
      setConnectError(err instanceof Error ? err.message : 'Failed to connect')
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
          <button onClick={onClose} className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1">
            <ArrowLeft size={11} /> Integrations
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">Bol.com</h1>
        </div>
        {connectionId && (
          <button
            onClick={handleSave}
            disabled={saving}
            className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : 'Save'}
          </button>
        )}
      </div>

      <div className="flex-1 overflow-y-auto p-6">
        {!connectionId ? (
          <form onSubmit={handleConnect} className="max-w-lg space-y-4">
            <p className="text-xs dark:text-gray-500 text-gray-400">
              Paste the credentials from your Bol.com Retailer API portal. No redirect step -- Bol authenticates via client_credentials.
            </p>
            <div>
              <label htmlFor="bol-client-id" className="block text-xs font-medium dark:text-gray-400 text-gray-500 mb-1">Client ID</label>
              <input id="bol-client-id" value={clientId} onChange={e => setClientId(e.target.value)} required
                className="w-full px-3 py-2 rounded-lg text-sm dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900" />
            </div>
            <div>
              <label htmlFor="bol-client-secret" className="block text-xs font-medium dark:text-gray-400 text-gray-500 mb-1">Client Secret</label>
              <input id="bol-client-secret" type="password" value={clientSecret} onChange={e => setClientSecret(e.target.value)} required
                className="w-full px-3 py-2 rounded-lg text-sm dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900" />
            </div>
            <div>
              <label htmlFor="bol-economic-operator-id" className="block text-xs font-medium dark:text-gray-400 text-gray-500 mb-1">Economic Operator ID</label>
              <input id="bol-economic-operator-id" value={economicOperatorId} onChange={e => setEconomicOperatorId(e.target.value)} required
                className="w-full px-3 py-2 rounded-lg text-sm dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900" />
              <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">Your EU operator identifier, required on every offer.</p>
            </div>
            {connectError && <p className="text-xs text-red-400">{connectError}</p>}
            <button type="submit" disabled={connecting}
              className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors">
              {connecting ? 'Connecting…' : 'Connect'}
            </button>
          </form>
        ) : (
          <div className="max-w-3xl space-y-4">
            <div>
              <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Sync schedule</h2>
              <p className="text-xs dark:text-gray-500 text-gray-400">
                Configure how often each type of data syncs between CommerceOS and Bol.com. Products/stock only push what's
                enrolled in your Bol sales channel -- configure enrollment under Sales Channels.
              </p>
            </div>

            {loadError && <p className="text-xs text-red-400">{loadError}</p>}

            <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 divide-y dark:divide-[#2D3748] divide-gray-200">
              {ROWS.map(row => (
                <div key={row.entity} className="p-4">
                  <p className="text-xs font-semibold dark:text-gray-200 text-gray-800 mb-3">{row.label}</p>
                  {row.entity === 'bol_orders' && (
                    <p className="text-[10px] dark:text-gray-500 text-gray-400 mb-2">
                      Import only -- CommerceOS never pushes orders back to Bol. Polls currently open orders every run.
                    </p>
                  )}
                  {row.entity === 'bol_stock' && (
                    <p className="text-[10px] dark:text-gray-500 text-gray-400 mb-2">
                      Re-pushes every already-linked product's current price and stock -- Bol has no stock-only endpoint,
                      so this runs the same push as Products, just on its own interval.
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
                            <span className="text-[10px] font-medium dark:text-gray-400 text-gray-500 uppercase tracking-wider capitalize">
                              {direction}
                            </span>
                            <label className={clsx('relative flex items-center shrink-0', runnable ? 'cursor-pointer' : 'cursor-not-allowed opacity-40')}>
                              <input
                                type="checkbox" role="switch" aria-label={`${row.label} ${direction}`}
                                checked={cell.enabled} disabled={!runnable}
                                onChange={e => updateCell(row.entity, direction, { enabled: e.target.checked })}
                                className="sr-only peer"
                              />
                              <div className="w-8 h-4 dark:bg-[#2D3748] bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
                              <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                            </label>
                          </div>
                          <select
                            aria-label={`${row.label} ${direction} interval`}
                            value={cell.intervalMinutes} disabled={!runnable}
                            onChange={e => updateCell(row.entity, direction, { intervalMinutes: Number(e.target.value) })}
                            className={clsx(inputClass, 'w-full disabled:opacity-40 disabled:cursor-not-allowed')}
                          >
                            {INTERVAL_OPTIONS.map(o => <option key={o.minutes} value={o.minutes}>{o.label}</option>)}
                          </select>
                          {showExtraFields && (
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
          </div>
        )}
      </div>
    </div>
  )
}
