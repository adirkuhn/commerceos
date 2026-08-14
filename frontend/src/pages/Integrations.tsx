import { useState } from 'react'
import { CheckCircle, XCircle, Plug, X } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useConnections } from '../integrations/hooks/useConnections'
import * as api from '../integrations/api'
import type { Connection, ImportJob, DeleteAllImportedProductsResult } from '../integrations/types'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { notifyProductsChanged } from '../pms/productsBus'
import { notifyNotificationsChanged } from '../notifications/notificationsBus'
import WooCommerceSyncSettingsPage from '../integrations/WooCommerceSyncSettingsPage'

const PROVIDER_COLORS: Record<string, string> = {
  woocommerce: 'from-purple-500 to-violet-600',
}

const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

// Currency codes saved before the backend started normalizing them
// (uppercase, trimmed) won't exact-match a <select> option otherwise —
// normalize on read too so existing connections aren't stuck unselected.
function normalizeCurrency(value: string | null | undefined): string {
  return (value ?? '').trim().toUpperCase()
}

const STATUS_ICON = {
  connected: CheckCircle,
  error: XCircle,
}

const STATUS_COLOR = {
  connected: 'text-green-400',
  error: 'text-red-400',
}

export default function Integrations() {
  const { token } = useAuth()
  const { connections, loading, error, refresh, disconnect } = useConnections(token ?? '')
  const { warehouses } = useWarehouses(token ?? '')
  const { settings: tenantSettings } = useTenantSettings(token ?? '')

  const [configuringConnection, setConfiguringConnection] = useState<Connection | null>(null)

  const [browsing, setBrowsing] = useState(false)
  const [connectingWoo, setConnectingWoo] = useState(false)
  const [connectMode, setConnectMode] = useState<'auto' | 'manual'>('auto')
  const [storeUrl, setStoreUrl] = useState('')
  const [consumerKey, setConsumerKey] = useState('')
  const [consumerSecret, setConsumerSecret] = useState('')
  const [connectError, setConnectError] = useState('')
  const [starting, setStarting] = useState(false)

  const [testingId, setTestingId] = useState<string | null>(null)
  const [testResult, setTestResult] = useState<{ id: string; ok: boolean; message: string } | null>(null)

  const [importingConnection, setImportingConnection] = useState<Connection | null>(null)
  const [importWarehouseId, setImportWarehouseId] = useState('')
  const [importCurrency, setImportCurrency] = useState('')
  const [importing, setImporting] = useState(false)
  const [importError, setImportError] = useState('')
  const [importJob, setImportJob] = useState<ImportJob | null>(null)

  const closeImport = () => {
    setImportingConnection(null)
    setImportWarehouseId('')
    setImportCurrency('')
    setImportError('')
    setImportJob(null)
  }

  const [deletingConnection, setDeletingConnection] = useState<Connection | null>(null)
  const [deleteConfirmText, setDeleteConfirmText] = useState('')
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState('')
  const [deleteResult, setDeleteResult] = useState<DeleteAllImportedProductsResult | null>(null)

  const closeDelete = () => {
    setDeletingConnection(null)
    setDeleteConfirmText('')
    setDeleteError('')
    setDeleteResult(null)
    refresh()
  }

  const handleDeleteImported = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token || !deletingConnection) return
    setDeleteError('')
    setDeleting(true)
    try {
      const result = await api.deleteAllImportedProducts(token, deletingConnection.id)
      setDeleteResult(result)
      notifyProductsChanged()
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Delete failed')
    } finally {
      setDeleting(false)
    }
  }

  // Enqueues a background job and returns immediately (202) instead of
  // blocking on the full sync, which for a real catalog can take minutes —
  // the job runs in cmd/scheduler; progress/completion shows up in the
  // notification bell, not here. The backend rejects a second import for
  // the same connection with 409 while one's already active, so this is
  // safe even if the button is clicked again in a new tab/session.
  const handleImport = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token || !importingConnection) return
    setImportError('')
    setImporting(true)
    try {
      const job = await api.importWooProducts(token, importingConnection.id, importWarehouseId, importCurrency)
      setImportJob(job)
      notifyNotificationsChanged()
    } catch (err) {
      setImportError(err instanceof Error ? err.message : 'Import failed')
    } finally {
      setImporting(false)
    }
  }

  const handleTestConnection = async (id: string) => {
    if (!token) return
    setTestingId(id)
    setTestResult(null)
    try {
      await api.testConnection(token, id)
      setTestResult({ id, ok: true, message: 'Connection OK' })
    } catch (err) {
      setTestResult({ id, ok: false, message: err instanceof Error ? err.message : 'Test failed' })
    } finally {
      setTestingId(null)
      refresh()
    }
  }

  const closeBrowse = () => {
    setBrowsing(false)
    setConnectingWoo(false)
    setConnectMode('auto')
    setStoreUrl('')
    setConsumerKey('')
    setConsumerSecret('')
    setConnectError('')
  }

  const handleStartConnect = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token) return
    setConnectError('')
    setStarting(true)
    try {
      const { authorize_url } = await api.startWooConnect(token, storeUrl)
      window.location.href = authorize_url
    } catch (err) {
      setConnectError(err instanceof Error ? err.message : 'Failed to start connection')
      setStarting(false)
    }
  }

  const handleManualConnect = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token) return
    setConnectError('')
    setStarting(true)
    try {
      await api.manualConnectWooCommerce(token, { storeUrl, consumerKey, consumerSecret })
      refresh()
      closeBrowse()
    } catch (err) {
      setConnectError(err instanceof Error ? err.message : 'Failed to connect')
      setStarting(false)
    }
  }

  if (configuringConnection) {
    return <WooCommerceSyncSettingsPage connection={configuringConnection} onClose={() => setConfiguringConnection(null)} />
  }

  return (
    <div className="flex h-full overflow-hidden">
      <div className="flex-1 flex flex-col overflow-hidden p-5 gap-5">
        {connections.length === 0 && !loading ? (
          <div className="flex-1 flex flex-col items-center justify-center text-center">
            <div className="w-12 h-12 rounded-xl dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 flex items-center justify-center mb-3">
              <Plug size={20} className="dark:text-gray-500 text-gray-400" />
            </div>
            <p className="text-sm font-medium dark:text-gray-300 text-gray-700 mb-1">No integrations connected</p>
            <p className="text-xs dark:text-gray-500 text-gray-400 mb-4">Connect a store to start syncing orders and inventory</p>
            <button
              onClick={() => setBrowsing(true)}
              className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
            >
              Browse Integrations
            </button>
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Connected integrations</h2>
              <button
                onClick={() => setBrowsing(true)}
                className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
              >
                Browse Integrations
              </button>
            </div>
            {error && <p className="text-xs text-red-400">{error}</p>}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
              {connections.map(conn => {
                const Icon = STATUS_ICON[conn.status]
                return (
                  <div
                    key={conn.id}
                    className="text-left p-4 rounded-xl border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white"
                  >
                    <div className="flex items-start justify-between mb-3">
                      <div className={clsx(
                        'w-9 h-9 rounded-lg bg-gradient-to-br flex items-center justify-center text-white font-bold text-sm',
                        PROVIDER_COLORS[conn.provider],
                      )}>
                        {conn.provider[0].toUpperCase()}
                      </div>
                      <Icon size={14} className={STATUS_COLOR[conn.status]} />
                    </div>
                    <p className="text-sm font-semibold dark:text-gray-100 text-gray-900 capitalize">{conn.provider}</p>
                    <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5 truncate">{conn.store_url}</p>
                    {testResult && testResult.id === conn.id && (
                      <p className={clsx(
                        'mt-2 text-[10px] px-2 py-1 rounded-lg',
                        testResult.ok ? 'text-green-400 bg-green-500/10' : 'text-red-400 bg-red-500/10',
                      )}>
                        {testResult.message}
                      </p>
                    )}
                    <div className="mt-3 grid grid-cols-2 gap-2">
                      {conn.provider === 'woocommerce' && (
                        <button
                          onClick={() => setConfiguringConnection(conn)}
                          className="py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 transition-colors"
                        >
                          Configure
                        </button>
                      )}
                      <button
                        onClick={() => handleTestConnection(conn.id)}
                        disabled={testingId === conn.id}
                        className="py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 disabled:opacity-50 transition-colors"
                      >
                        {testingId === conn.id ? 'Testing…' : 'Test'}
                      </button>
                      <button
                        onClick={() => { setImportingConnection(conn); setImportCurrency(normalizeCurrency(conn.default_currency) || normalizeCurrency(tenantSettings?.default_currency)) }}
                        className="py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 transition-colors"
                      >
                        Import
                      </button>
                      <button
                        onClick={() => setDeletingConnection(conn)}
                        className="py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-red-400 transition-colors"
                      >
                        Delete imported
                      </button>
                      <button
                        onClick={() => disconnect(conn.id)}
                        className="py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-red-400 transition-colors"
                      >
                        Disconnect
                      </button>
                    </div>
                  </div>
                )
              })}
            </div>
          </>
        )}
      </div>

      {browsing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 dark:bg-black/60 bg-black/30"
            onClick={closeBrowse}
          />
          <div className="relative w-full max-w-sm dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Browse Integrations</h3>
              <button onClick={closeBrowse}>
                <X size={14} className="dark:text-gray-500 text-gray-400" />
              </button>
            </div>

            {!connectingWoo ? (
              <button
                onClick={() => setConnectingWoo(true)}
                className="w-full flex items-center gap-3 p-3 rounded-lg border dark:border-[#2D3748] border-gray-200 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors text-left"
              >
                <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-purple-500 to-violet-600 flex items-center justify-center text-white font-bold text-xs shrink-0">
                  W
                </div>
                <span className="text-sm dark:text-gray-200 text-gray-800">WooCommerce</span>
              </button>
            ) : (
              <>
                <div className="flex gap-1 mb-4 p-0.5 rounded-lg dark:bg-[#111827] bg-gray-100">
                  <button
                    type="button"
                    onClick={() => setConnectMode('auto')}
                    className={clsx(
                      'flex-1 py-1.5 rounded-md text-xs font-medium transition-colors',
                      connectMode === 'auto'
                        ? 'bg-blue-600 text-white'
                        : 'dark:text-gray-400 text-gray-500',
                    )}
                  >
                    Connect automatically
                  </button>
                  <button
                    type="button"
                    onClick={() => setConnectMode('manual')}
                    className={clsx(
                      'flex-1 py-1.5 rounded-md text-xs font-medium transition-colors',
                      connectMode === 'manual'
                        ? 'bg-blue-600 text-white'
                        : 'dark:text-gray-400 text-gray-500',
                    )}
                  >
                    Manual setup
                  </button>
                </div>

                {connectMode === 'auto' ? (
                  <form onSubmit={handleStartConnect} className="space-y-3">
                    <div>
                      <label className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                        Store URL
                      </label>
                      <input
                        type="url"
                        required
                        autoFocus
                        value={storeUrl}
                        onChange={e => setStoreUrl(e.target.value)}
                        placeholder="https://mystore.com"
                        className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                      />
                      <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                        Redirects to your store to approve access. Requires your store to reach this server.
                      </p>
                    </div>
                    {connectError && (
                      <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{connectError}</p>
                    )}
                    <button
                      type="submit"
                      disabled={starting}
                      className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                    >
                      {starting ? 'Connecting…' : 'Connect'}
                    </button>
                  </form>
                ) : (
                  <form onSubmit={handleManualConnect} className="space-y-3">
                    <div>
                      <label className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                        Store URL
                      </label>
                      <input
                        type="url"
                        required
                        autoFocus
                        value={storeUrl}
                        onChange={e => setStoreUrl(e.target.value)}
                        placeholder="https://mystore.com"
                        className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                      />
                    </div>
                    <div>
                      <label className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                        Consumer Key
                      </label>
                      <input
                        type="text"
                        required
                        value={consumerKey}
                        onChange={e => setConsumerKey(e.target.value)}
                        placeholder="ck_..."
                        className="w-full px-3 py-2 text-xs font-mono rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                      />
                    </div>
                    <div>
                      <label className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                        Consumer Secret
                      </label>
                      <input
                        type="password"
                        required
                        value={consumerSecret}
                        onChange={e => setConsumerSecret(e.target.value)}
                        placeholder="cs_..."
                        className="w-full px-3 py-2 text-xs font-mono rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                      />
                      <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                        Generate under WooCommerce → Settings → Advanced → REST API.
                      </p>
                    </div>
                    {connectError && (
                      <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{connectError}</p>
                    )}
                    <button
                      type="submit"
                      disabled={starting}
                      className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                    >
                      {starting ? 'Connecting…' : 'Connect'}
                    </button>
                  </form>
                )}
              </>
            )}
          </div>
        </div>
      )}

      {importingConnection && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={closeImport} />
          <div className="relative w-full max-w-sm dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Import products</h3>
              <button onClick={closeImport} aria-label="Close">
                <X size={14} className="dark:text-gray-500 text-gray-400" />
              </button>
            </div>

            {importJob ? (
              <div className="space-y-3">
                <p className="text-xs dark:text-gray-300 text-gray-700">
                  Import started — it's running in the background. You'll see a notification when it's done (check the bell icon, top right).
                </p>
                <button
                  onClick={closeImport}
                  className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
                >
                  Done
                </button>
              </div>
            ) : (
              <form onSubmit={handleImport} className="space-y-3">
                <p className="text-[10px] dark:text-gray-500 text-gray-400">
                  Imports every WooCommerce product not yet linked, along with its category. Existing categories are matched by name; missing ones are created.
                </p>
                <div>
                  <label htmlFor="import-warehouse-id" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Default warehouse ID
                  </label>
                  <select
                    id="import-warehouse-id"
                    required
                    autoFocus
                    value={importWarehouseId}
                    onChange={e => setImportWarehouseId(e.target.value)}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    <option value="" disabled>Select a warehouse…</option>
                    {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                  </select>
                  <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                    Imported stock quantities are attributed to this warehouse — WooCommerce has no warehouse concept.
                  </p>
                </div>
                <div>
                  <label htmlFor="import-currency" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Default currency
                  </label>
                  <select
                    id="import-currency"
                    required
                    value={importCurrency}
                    onChange={e => setImportCurrency(e.target.value)}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    <option value="" disabled>Select a currency…</option>
                    {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
                {importError && (
                  <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{importError}</p>
                )}
                <button
                  type="submit"
                  disabled={importing}
                  className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                >
                  {importing ? 'Importing…' : 'Start import'}
                </button>
              </form>
            )}
          </div>
        </div>
      )}

      {deletingConnection && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={deleteResult ? closeDelete : undefined} />
          <div className="relative w-full max-w-sm dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Delete imported products</h3>
              {deleteResult && (
                <button onClick={closeDelete} aria-label="Close">
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
                  onClick={closeDelete}
                  className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
                >
                  Done
                </button>
              </div>
            ) : (
              <form onSubmit={handleDeleteImported} className="space-y-3">
                <div className="dark:bg-red-500/10 bg-red-50 border dark:border-red-500/20 border-red-200 rounded-lg p-3">
                  <p className="text-xs text-red-400">
                    This permanently deletes every product imported from <strong>{deletingConnection.store_url}</strong> — including
                    their variants, SKUs, stock history, and images. Categories left empty afterward are also deleted. This cannot be undone.
                  </p>
                </div>
                <div>
                  <label htmlFor="delete-confirm" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Type DELETE to confirm
                  </label>
                  <input
                    id="delete-confirm"
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
                    onClick={closeDelete}
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
