import { useEffect, useState } from 'react'
import { CheckCircle, XCircle, Plug, X, Webhook, ChevronRight } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useApp } from '../context/AppContext'
import { useConnections } from '../integrations/hooks/useConnections'
import { useWebhookFallbackEvents } from '../integrations/hooks/useWebhookFallbackEvents'
import { WebhookFallbackEventsTable } from '../integrations/WebhookFallbackEventsTable'
import { WhatsAppTestModeModal } from '../integrations/WhatsAppTestModeModal'
import { WhatsAppAIParsingModal } from '../integrations/WhatsAppAIParsingModal'
import { WhatsAppCredentialsModal } from '../integrations/WhatsAppCredentialsModal'
import { WhatsAppWebhookModal } from '../integrations/WhatsAppWebhookModal'
import * as api from '../integrations/api'
import type { Connection, ImportJob, DeleteAllImportedProductsResult } from '../integrations/types'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { notifyProductsChanged } from '../pms/productsBus'
import { notifyNotificationsChanged } from '../notifications/notificationsBus'
import WooCommerceSyncSettingsPage from '../integrations/WooCommerceSyncSettingsPage'
import BolSyncSettingsPage from '../integrations/BolSyncSettingsPage'
import MyParcelSyncSettingsPage from '../shipping/MyParcelSyncSettingsPage'
import { useIntegrationsRoute, type ConnectionPanel } from '../integrations/hooks/useIntegrationsRoute'
import { useInitOnce } from '../routing/useInitOnce'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'

// Same tier as WooCommerceSyncSettingsPage's webhook-secret tab -- this
// table exposes raw (if opaque) third-party payloads and a manual
// republish action, security/ops-adjacent rather than day-to-day
// integration use, see that file's canManageWebhooks doc comment.
function canManageWebhooks(role: string | undefined): boolean {
  return role === 'customer_owner' || role === 'customer_admin' || role === 'super_admin'
}

const PROVIDER_COLORS: Record<string, string> = {
  woocommerce: 'from-purple-500 to-violet-600',
  bol: 'from-blue-500 to-cyan-600',
  myparcel: 'from-orange-500 to-red-600',
  whatsapp: 'from-green-500 to-emerald-600',
  stripe: 'from-indigo-500 to-purple-600',
  revolut: 'from-slate-700 to-black',
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
  connected: 'text-success',
  error: 'text-danger',
}

interface PaymentWebhookSetupStepProps {
  providerLabel: string
  dashboardHint: string
  webhookUrl: string
  secret: string
  onSecretChange: (value: string) => void
  onSubmit: (e: React.FormEvent) => void
  saving: boolean
  error: string
  saved: boolean
  onDone: () => void
}

// Reversed from WhatsApp's connect step (there we generate a secret and
// the merchant pastes our URL+secret into Evolution API) -- Stripe/Revolut
// generate their OWN signing secret when the merchant registers this URL
// as a webhook endpoint in their own dashboard, so this step shows the URL
// to register there, then collects THEIR secret to paste back into us.
function PaymentWebhookSetupStep({
  providerLabel, dashboardHint, webhookUrl, secret, onSecretChange, onSubmit, saving, error, saved, onDone,
}: PaymentWebhookSetupStepProps) {
  return (
    <div className="space-y-3">
      <p className="text-xs text-ink-2">
        Connected! Register this URL as a webhook endpoint in {dashboardHint}, then paste the signing secret it gives you below.
      </p>
      <p className="text-2xs font-mono break-all bg-fill rounded-lg px-2 py-1.5 text-ink-2">
        {webhookUrl}
      </p>
      {saved ? (
        <>
          <p className="text-xs text-success bg-green-50 dark:bg-green-500/10 rounded-lg px-2 py-1.5">Webhook secret saved.</p>
          <button
            onClick={onDone}
            className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
          >
            Done
          </button>
        </>
      ) : (
        <form onSubmit={onSubmit} className="space-y-3">
          <div>
            <label htmlFor="payment-webhook-secret" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
              {providerLabel} webhook signing secret
            </label>
            <input
              id="payment-webhook-secret"
              type="password"
              required
              autoFocus
              value={secret}
              onChange={e => onSecretChange(e.target.value)}
              className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          {error && (
            <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>
          )}
          <div className="grid grid-cols-2 gap-2">
            <button
              type="button"
              onClick={onDone}
              className="py-2 rounded-lg bg-fill text-xs font-medium text-ink-2 transition-colors"
            >
              Skip for now
            </button>
            <button
              type="submit"
              disabled={saving}
              className="py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {saving ? 'Saving…' : 'Save'}
            </button>
          </div>
        </form>
      )}
    </div>
  )
}

export default function Integrations() {
  const { token, user } = useAuth()
  const { connections, loading, error, refresh, disconnect } = useConnections(token ?? '')
  const { warehouses } = useWarehouses(token ?? '')
  const { settings: tenantSettings } = useTenantSettings(token ?? '')
  const showWebhookAdmin = canManageWebhooks(user?.role)
  const { setActiveNav } = useApp()
  const fallbackEvents = useWebhookFallbackEvents(showWebhookAdmin ? token ?? '' : '')

  // Which page/modal is open -- a connection's settings page or one of its
  // modals, the "Add integration" catalog, a connect form -- comes from the
  // URL (see useIntegrationsRoute), so a refresh keeps it open and Back
  // closes it. Form field values stay local.
  const route = useIntegrationsRoute()
  const { view } = route
  const routeConnectionId = view.kind === 'connection' ? view.connectionId : null
  const routeConnection = routeConnectionId ? connections.find(c => c.id === routeConnectionId) ?? null : null
  const panelFor = (panel: ConnectionPanel, providers?: string[]): Connection | null =>
    view.kind === 'connection' && view.panel === panel && routeConnection && (!providers || providers.includes(routeConnection.provider))
      ? routeConnection
      : null
  const openPanel = (conn: Connection, panel: ConnectionPanel) => route.openPanel(conn.id, panel)

  const configuringConnection = panelFor('settings', ['woocommerce'])
  const configuringWhatsAppTestMode = panelFor('test-mode')
  const configuringWhatsAppAIParsing = panelFor('ai-parsing')
  const configuringWhatsAppCredentials = panelFor('credentials')
  const configuringWhatsAppWebhook = panelFor('whatsapp-webhook')

  const browsing = view.kind === 'browse' || view.kind === 'connect'
  const connectingWoo = view.kind === 'connect' && view.provider === 'woocommerce'
  const bolConnection: Connection | 'new' | null =
    view.kind === 'new' && view.provider === 'bol' ? 'new' : panelFor('settings', ['bol'])
  const myParcelConnection: Connection | 'new' | null =
    view.kind === 'new' && view.provider === 'myparcel' ? 'new' : panelFor('settings', ['myparcel']) ?? panelFor('webhooks', ['myparcel'])
  const focusMyParcelWebhooks = view.kind === 'connection' && view.panel === 'webhooks'
  const [connectMode, setConnectMode] = useState<'auto' | 'manual'>('auto')
  const [storeUrl, setStoreUrl] = useState('')
  const [consumerKey, setConsumerKey] = useState('')
  const [consumerSecret, setConsumerSecret] = useState('')
  const [connectError, setConnectError] = useState('')
  const [starting, setStarting] = useState(false)

  const connectingWhatsapp = view.kind === 'connect' && view.provider === 'whatsapp'
  const [waBaseUrl, setWaBaseUrl] = useState('')
  const [waInstanceName, setWaInstanceName] = useState('')
  const [waApiKey, setWaApiKey] = useState('')
  const [waWebhookUrl, setWaWebhookUrl] = useState('')

  const connectingStripe = view.kind === 'connect' && view.provider === 'stripe'
  const [stripeSecretKey, setStripeSecretKey] = useState('')
  const [stripePublishableKey, setStripePublishableKey] = useState('')
  const connectingRevolut = view.kind === 'connect' && view.provider === 'revolut'
  const [revolutApiKey, setRevolutApiKey] = useState('')
  const connectingEmail = view.kind === 'connect' && view.provider === 'email'
  const [emailHost, setEmailHost] = useState('')
  const [emailPort, setEmailPort] = useState('587')
  const [emailUsername, setEmailUsername] = useState('')
  const [emailPassword, setEmailPassword] = useState('')
  const [emailFrom, setEmailFrom] = useState('')
  // Set once connectStripe/connectRevolut succeeds -- switches the form to
  // the "register this URL, then paste back the secret it gives you" step.
  // Reversed from WhatsApp's flow (there we generate the secret) since
  // Stripe/Revolut generate their own when a webhook endpoint is registered
  // in their dashboard.
  const [paymentWebhook, setPaymentWebhook] = useState<{ connectionId: string; url: string } | null>(null)
  const [paymentWebhookSecret, setPaymentWebhookSecret] = useState('')
  const [savingPaymentWebhookSecret, setSavingPaymentWebhookSecret] = useState(false)
  const [paymentWebhookSecretError, setPaymentWebhookSecretError] = useState('')
  const [paymentWebhookSecretSaved, setPaymentWebhookSecretSaved] = useState(false)

  const [testingId, setTestingId] = useState<string | null>(null)
  const [testResult, setTestResult] = useState<{ id: string; ok: boolean; message: string } | null>(null)

  const importingConnection = panelFor('import')
  const [importWarehouseId, setImportWarehouseId] = useState('')
  const [importCurrency, setImportCurrency] = useState('')
  const [importing, setImporting] = useState(false)
  const [importError, setImportError] = useState('')
  const [importJob, setImportJob] = useState<ImportJob | null>(null)

  // Default the import currency once per opened import modal (also after a
  // refresh on its URL, once the connection has loaded).
  useInitOnce(importingConnection ? `import:${importingConnection.id}` : null, !!importingConnection, () => {
    setImportWarehouseId('')
    setImportError('')
    setImportJob(null)
    setImportCurrency(normalizeCurrency(importingConnection?.default_currency) || normalizeCurrency(tenantSettings?.default_currency))
  })
  const closeImport = route.close

  const deletingConnection = panelFor('delete-products')
  const [deleteConfirmText, setDeleteConfirmText] = useState('')
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState('')
  const [deleteResult, setDeleteResult] = useState<DeleteAllImportedProductsResult | null>(null)

  const resetDelete = () => {
    setDeleteConfirmText('')
    setDeleteError('')
    setDeleteResult(null)
  }
  // Also on Back, not only via the close buttons.
  useEffect(() => { if (!deletingConnection) resetDelete() }, [deletingConnection])
  const closeDelete = () => {
    route.close()
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

  // Clears every connect form -- runs whenever the catalog closes, whether
  // via its close buttons or browser Back.
  const resetConnectForms = () => {
    setConnectMode('auto')
    setStoreUrl('')
    setConsumerKey('')
    setConsumerSecret('')
    setConnectError('')
    setWaBaseUrl('')
    setWaInstanceName('')
    setWaApiKey('')
    setWaWebhookUrl('')
    setStripeSecretKey('')
    setStripePublishableKey('')
    setRevolutApiKey('')
    setEmailHost('')
    setEmailPort('587')
    setEmailUsername('')
    setEmailPassword('')
    setEmailFrom('')
    setPaymentWebhook(null)
    setPaymentWebhookSecret('')
    setSavingPaymentWebhookSecret(false)
    setPaymentWebhookSecretError('')
    setPaymentWebhookSecretSaved(false)
  }
  useEffect(() => { if (!browsing) resetConnectForms() }, [browsing])
  const closeBrowse = route.close

  const handleConnectWhatsapp = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token) return
    setConnectError('')
    setStarting(true)
    try {
      const { webhook_url } = await api.connectWhatsapp(token, { baseUrl: waBaseUrl, instanceName: waInstanceName, apiKey: waApiKey })
      setWaWebhookUrl(webhook_url)
      refresh()
    } catch (err) {
      setConnectError(err instanceof Error ? err.message : 'Failed to connect')
    } finally {
      setStarting(false)
    }
  }

  const handleConnectStripe = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token) return
    setConnectError('')
    setStarting(true)
    try {
      const { connection, webhook_url } = await api.connectStripe(token, stripeSecretKey, stripePublishableKey)
      setPaymentWebhook({ connectionId: connection.id, url: webhook_url })
      refresh()
    } catch (err) {
      setConnectError(err instanceof Error ? err.message : 'Failed to connect')
    } finally {
      setStarting(false)
    }
  }

  const handleConnectRevolut = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token) return
    setConnectError('')
    setStarting(true)
    try {
      const { connection, webhook_url } = await api.connectRevolut(token, revolutApiKey)
      setPaymentWebhook({ connectionId: connection.id, url: webhook_url })
      refresh()
    } catch (err) {
      setConnectError(err instanceof Error ? err.message : 'Failed to connect')
    } finally {
      setStarting(false)
    }
  }

  const handleConnectEmail = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token) return
    setConnectError('')
    setStarting(true)
    try {
      await api.connectEmail(token, { host: emailHost, port: emailPort, username: emailUsername, password: emailPassword, from: emailFrom })
      closeBrowse()
      refresh()
    } catch (err) {
      setConnectError(err instanceof Error ? err.message : 'Failed to connect')
    } finally {
      setStarting(false)
    }
  }

  const handleSavePaymentWebhookSecret = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!token || !paymentWebhook) return
    setPaymentWebhookSecretError('')
    setSavingPaymentWebhookSecret(true)
    try {
      await api.setWebhookSecret(token, paymentWebhook.connectionId, paymentWebhookSecret)
      setPaymentWebhookSecretSaved(true)
    } catch (err) {
      setPaymentWebhookSecretError(err instanceof Error ? err.message : 'Failed to save webhook secret')
    } finally {
      setSavingPaymentWebhookSecret(false)
    }
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

  if (routeConnectionId && !routeConnection) {
    return loading ? <RecordLoading /> : <RecordNotFound label="Connection" onBack={route.close} />
  }

  if (configuringConnection) {
    return <WooCommerceSyncSettingsPage connection={configuringConnection} onClose={route.close} />
  }

  if (bolConnection) {
    return (
      <BolSyncSettingsPage
        connectionId={bolConnection === 'new' ? undefined : bolConnection.id}
        initialConnection={bolConnection === 'new' ? undefined : bolConnection}
        onClose={route.close}
        onConnected={() => {
          route.close()
          refresh()
        }}
      />
    )
  }

  if (myParcelConnection) {
    return (
      <MyParcelSyncSettingsPage
        connectionId={myParcelConnection === 'new' ? undefined : myParcelConnection.id}
        focusWebhooks={focusMyParcelWebhooks}
        onClose={route.close}
        onConnected={() => {
          route.close()
          refresh()
        }}
      />
    )
  }

  return (
    <div className="flex h-full overflow-hidden">
      <div className="flex-1 flex flex-col overflow-hidden p-5 gap-5">
        {showWebhookAdmin && (
          <button
            onClick={() => setActiveNav('webhooks')}
            className="flex items-center justify-between px-4 py-3 rounded-xl bg-raised border border-line hover:border-blue-500 transition-colors text-left"
          >
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 rounded-lg dark:bg-blue-600/15 bg-blue-50 flex items-center justify-center shrink-0">
                <Webhook size={15} className="text-accent" />
              </div>
              <div>
                <p className="text-xs font-semibold text-ink">Webhooks</p>
                <p className="text-2xs text-ink-4">Fan-out rules and every active provider subscription, in one place</p>
              </div>
            </div>
            <ChevronRight size={16} className="text-ink-4 shrink-0" />
          </button>
        )}

        {connections.length === 0 && !loading ? (
          <div className="flex-1 flex flex-col items-center justify-center text-center">
            <div className="w-12 h-12 rounded-xl bg-raised border border-line flex items-center justify-center mb-3">
              <Plug size={20} className="text-ink-4" />
            </div>
            <p className="text-sm font-medium text-ink-2 mb-1">No integrations connected</p>
            <p className="text-xs text-ink-4 mb-4">Connect a store to start syncing orders and inventory</p>
            <button
              onClick={route.openBrowse}
              className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
            >
              Browse Integrations
            </button>
          </div>
        ) : (
          <>
            <div className="flex items-center justify-between">
              <h2 className="text-sm font-semibold text-ink">Connected integrations</h2>
              <button
                onClick={route.openBrowse}
                className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
              >
                Browse Integrations
              </button>
            </div>
            {error && <p className="text-xs text-danger">{error}</p>}
            <div className="grid grid-cols-2 lg:grid-cols-4 gap-3">
              {connections.map(conn => {
                const Icon = STATUS_ICON[conn.status]
                return (
                  <div
                    key={conn.id}
                    className="text-left p-4 rounded-xl border border-line bg-raised"
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
                    <p className="text-sm font-semibold text-ink capitalize">{conn.provider}</p>
                    <p className="text-2xs text-ink-4 mt-0.5 truncate">{conn.store_url}</p>
                    {conn.provider === 'whatsapp' && conn.whatsapp_test_mode_enabled && (
                      <p className="mt-2 text-2xs px-2 py-1 rounded-lg text-warning bg-amber-500/10">
                        Test mode on — all messages go to {conn.whatsapp_test_phone_e164}
                      </p>
                    )}
                    {conn.provider === 'whatsapp' && conn.whatsapp_ai_parsing_enabled && (
                      <p className="mt-2 text-2xs px-2 py-1 rounded-lg text-accent bg-blue-500/10">
                        AI-assisted parsing on
                      </p>
                    )}
                    {testResult && testResult.id === conn.id && (
                      <p className={clsx(
                        'mt-2 text-2xs px-2 py-1 rounded-lg',
                        testResult.ok ? 'text-success bg-green-500/10' : 'text-danger bg-red-500/10',
                      )}>
                        {testResult.message}
                      </p>
                    )}
                    <div className="mt-3 grid grid-cols-2 gap-2">
                      {conn.provider === 'woocommerce' && (
                        <button
                          onClick={() => openPanel(conn, 'settings')}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                        >
                          Configure
                        </button>
                      )}
                      {conn.provider === 'myparcel' && (
                        <button
                          onClick={() => openPanel(conn, 'settings')}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                        >
                          Configure
                        </button>
                      )}
                      {conn.provider === 'myparcel' && (
                        <button
                          onClick={() => openPanel(conn, 'webhooks')}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                        >
                          Register webhook
                        </button>
                      )}
                      {conn.provider === 'bol' && (
                        <button
                          onClick={() => openPanel(conn, 'settings')}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                        >
                          Configure
                        </button>
                      )}
                      {conn.provider === 'whatsapp' && (
                        <button
                          onClick={() => openPanel(conn, 'test-mode')}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                        >
                          Test mode
                        </button>
                      )}
                      {conn.provider === 'whatsapp' && (
                        <button
                          onClick={() => openPanel(conn, 'ai-parsing')}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                        >
                          AI parsing
                        </button>
                      )}
                      {conn.provider === 'whatsapp' && (
                        <button
                          onClick={() => openPanel(conn, 'credentials')}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                        >
                          Edit credentials
                        </button>
                      )}
                      {conn.provider === 'whatsapp' && (
                        <button
                          onClick={() => openPanel(conn, 'whatsapp-webhook')}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                        >
                          Webhook
                        </button>
                      )}
                      {conn.provider === 'woocommerce' && (
                        <button
                          onClick={() => handleTestConnection(conn.id)}
                          disabled={testingId === conn.id}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent disabled:opacity-50 transition-colors"
                        >
                          {testingId === conn.id ? 'Testing…' : 'Test'}
                        </button>
                      )}
                      {conn.provider === 'woocommerce' && (
                        <button
                          onClick={() => openPanel(conn, 'import')}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                        >
                          Import
                        </button>
                      )}
                      {conn.provider === 'woocommerce' && (
                        <button
                          onClick={() => openPanel(conn, 'delete-products')}
                          className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-danger transition-colors"
                        >
                          Delete imported
                        </button>
                      )}
                      <button
                        onClick={() => disconnect(conn.id)}
                        className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-danger transition-colors"
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

        {showWebhookAdmin && <WebhookFallbackEventsTable {...fallbackEvents} />}
      </div>

      {configuringWhatsAppTestMode && (
        <WhatsAppTestModeModal
          connection={configuringWhatsAppTestMode}
          onClose={route.close}
          onSaved={() => refresh()}
        />
      )}

      {configuringWhatsAppAIParsing && (
        <WhatsAppAIParsingModal
          connection={configuringWhatsAppAIParsing}
          onClose={route.close}
          onSaved={() => refresh()}
        />
      )}

      {configuringWhatsAppCredentials && (
        <WhatsAppCredentialsModal
          connection={configuringWhatsAppCredentials}
          onClose={route.close}
          onSaved={() => refresh()}
        />
      )}

      {configuringWhatsAppWebhook && (
        <WhatsAppWebhookModal
          connection={configuringWhatsAppWebhook}
          tenantId={user?.tenant_id ?? ''}
          onClose={route.close}
        />
      )}

      {browsing && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div
            className="absolute inset-0 dark:bg-black/60 bg-black/30"
            onClick={closeBrowse}
          />
          <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">Browse Integrations</h3>
              <button onClick={closeBrowse}>
                <X size={14} className="text-ink-4" />
              </button>
            </div>

            {!connectingWoo && !connectingWhatsapp && !connectingStripe && !connectingRevolut && !connectingEmail ? (
              <div className="space-y-2">
                <button
                  onClick={() => route.openConnect('woocommerce')}
                  className="w-full flex items-center gap-3 p-3 rounded-lg border border-line hover:bg-hover transition-colors text-left"
                >
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-purple-500 to-violet-600 flex items-center justify-center text-white font-bold text-xs shrink-0">
                    W
                  </div>
                  <span className="text-sm text-ink">WooCommerce</span>
                </button>
                <button
                  onClick={() => route.openNew('bol')}
                  className="w-full flex items-center gap-3 p-3 rounded-lg border border-line hover:bg-hover transition-colors text-left"
                >
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-blue-500 to-cyan-600 flex items-center justify-center text-white font-bold text-xs shrink-0">
                    B
                  </div>
                  <span className="text-sm text-ink">Bol.com</span>
                </button>
                <button
                  onClick={() => route.openNew('myparcel')}
                  className="w-full flex items-center gap-3 p-3 rounded-lg border border-line hover:bg-hover transition-colors text-left"
                >
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-orange-500 to-red-600 flex items-center justify-center text-white font-bold text-xs shrink-0">
                    M
                  </div>
                  <span className="text-sm text-ink">MyParcel</span>
                </button>
                <button
                  onClick={() => route.openConnect('whatsapp')}
                  className="w-full flex items-center gap-3 p-3 rounded-lg border border-line hover:bg-hover transition-colors text-left"
                >
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-green-500 to-emerald-600 flex items-center justify-center text-white font-bold text-xs shrink-0">
                    WA
                  </div>
                  <span className="text-sm text-ink">WhatsApp</span>
                </button>
                <button
                  onClick={() => route.openConnect('stripe')}
                  className="w-full flex items-center gap-3 p-3 rounded-lg border border-line hover:bg-hover transition-colors text-left"
                >
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-indigo-500 to-purple-600 flex items-center justify-center text-white font-bold text-xs shrink-0">
                    S
                  </div>
                  <span className="text-sm text-ink">Stripe</span>
                </button>
                <button
                  onClick={() => route.openConnect('revolut')}
                  className="w-full flex items-center gap-3 p-3 rounded-lg border border-line hover:bg-hover transition-colors text-left"
                >
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-slate-700 to-black flex items-center justify-center text-white font-bold text-xs shrink-0">
                    R
                  </div>
                  <span className="text-sm text-ink">Revolut</span>
                </button>
                <button
                  onClick={() => route.openConnect('email')}
                  className="w-full flex items-center gap-3 p-3 rounded-lg border border-line hover:bg-hover transition-colors text-left"
                >
                  <div className="w-8 h-8 rounded-lg bg-gradient-to-br from-teal-500 to-cyan-600 flex items-center justify-center text-white font-bold text-xs shrink-0">
                    E
                  </div>
                  <span className="text-sm text-ink">Email</span>
                </button>
              </div>
            ) : connectingWhatsapp ? (
              waWebhookUrl ? (
                <div className="space-y-3">
                  <p className="text-xs text-ink-2">
                    Connected! Paste this URL into Evolution API's own webhook settings (messages.upsert event) — it's shown only once.
                  </p>
                  <p className="text-2xs font-mono break-all bg-fill rounded-lg px-2 py-1.5 text-ink-2">
                    {waWebhookUrl}
                  </p>
                  <button
                    onClick={closeBrowse}
                    className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
                  >
                    Done
                  </button>
                </div>
              ) : (
                <form onSubmit={handleConnectWhatsapp} className="space-y-3">
                  <div>
                    <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                      Evolution API base URL
                    </label>
                    <input
                      type="url"
                      required
                      autoFocus
                      value={waBaseUrl}
                      onChange={e => setWaBaseUrl(e.target.value)}
                      placeholder="https://evolution.example.com"
                      className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                    />
                    <p className="mt-1.5 text-2xs text-ink-4">
                      Your own self-hosted Evolution API instance (unofficial WhatsApp gateway) — not an official WhatsApp Business API.
                    </p>
                  </div>
                  <div>
                    <label htmlFor="wa-instance-name" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                      Instance name
                    </label>
                    <input
                      id="wa-instance-name"
                      type="text"
                      required
                      value={waInstanceName}
                      onChange={e => setWaInstanceName(e.target.value)}
                      className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                  <div>
                    <label htmlFor="wa-api-key" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                      API key
                    </label>
                    <input
                      id="wa-api-key"
                      type="password"
                      required
                      value={waApiKey}
                      onChange={e => setWaApiKey(e.target.value)}
                      className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                    />
                  </div>
                  {connectError && (
                    <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{connectError}</p>
                  )}
                  <button
                    type="submit"
                    disabled={starting}
                    className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                  >
                    {starting ? 'Connecting…' : 'Connect'}
                  </button>
                </form>
              )
            ) : connectingStripe ? (
              paymentWebhook ? (
                <PaymentWebhookSetupStep
                  providerLabel="Stripe"
                  dashboardHint="Stripe Dashboard → Developers → Webhooks"
                  webhookUrl={paymentWebhook.url}
                  secret={paymentWebhookSecret}
                  onSecretChange={setPaymentWebhookSecret}
                  onSubmit={handleSavePaymentWebhookSecret}
                  saving={savingPaymentWebhookSecret}
                  error={paymentWebhookSecretError}
                  saved={paymentWebhookSecretSaved}
                  onDone={closeBrowse}
                />
              ) : (
              <form onSubmit={handleConnectStripe} className="space-y-3">
                <div>
                  <label htmlFor="stripe-secret-key" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Secret key
                  </label>
                  <input
                    id="stripe-secret-key"
                    type="password"
                    required
                    autoFocus
                    value={stripeSecretKey}
                    onChange={e => setStripeSecretKey(e.target.value)}
                    placeholder="sk_live_..."
                    className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  <p className="mt-1.5 text-2xs text-ink-4">
                    From your Stripe Dashboard → Developers → API keys.
                  </p>
                </div>
                <div>
                  <label htmlFor="stripe-publishable-key" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Publishable key
                  </label>
                  <input
                    id="stripe-publishable-key"
                    type="text"
                    value={stripePublishableKey}
                    onChange={e => setStripePublishableKey(e.target.value)}
                    placeholder="pk_live_..."
                    className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  <p className="mt-1.5 text-2xs text-ink-4">
                    Optional — required only to offer an embedded card form at checkout instead of a hosted payment link.
                  </p>
                </div>
                {connectError && (
                  <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{connectError}</p>
                )}
                <button
                  type="submit"
                  disabled={starting}
                  className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                >
                  {starting ? 'Connecting…' : 'Connect'}
                </button>
              </form>
              )
            ) : connectingRevolut ? (
              paymentWebhook ? (
                <PaymentWebhookSetupStep
                  providerLabel="Revolut"
                  dashboardHint="Revolut Business → Merchant API → Webhooks"
                  webhookUrl={paymentWebhook.url}
                  secret={paymentWebhookSecret}
                  onSecretChange={setPaymentWebhookSecret}
                  onSubmit={handleSavePaymentWebhookSecret}
                  saving={savingPaymentWebhookSecret}
                  error={paymentWebhookSecretError}
                  saved={paymentWebhookSecretSaved}
                  onDone={closeBrowse}
                />
              ) : (
              <form onSubmit={handleConnectRevolut} className="space-y-3">
                <div>
                  <label htmlFor="revolut-api-key" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Merchant API secret key
                  </label>
                  <input
                    id="revolut-api-key"
                    type="password"
                    required
                    autoFocus
                    value={revolutApiKey}
                    onChange={e => setRevolutApiKey(e.target.value)}
                    className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  <p className="mt-1.5 text-2xs text-ink-4">
                    From your Revolut Business account → Merchant API settings.
                  </p>
                </div>
                {connectError && (
                  <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{connectError}</p>
                )}
                <button
                  type="submit"
                  disabled={starting}
                  className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                >
                  {starting ? 'Connecting…' : 'Connect'}
                </button>
              </form>
              )
            ) : connectingEmail ? (
              <form onSubmit={handleConnectEmail} className="space-y-3">
                <div>
                  <label htmlFor="email-host" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    SMTP host
                  </label>
                  <input
                    id="email-host"
                    type="text"
                    required
                    autoFocus
                    value={emailHost}
                    onChange={e => setEmailHost(e.target.value)}
                    placeholder="smtp.example.com"
                    className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label htmlFor="email-port" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Port
                  </label>
                  <input
                    id="email-port"
                    type="text"
                    required
                    value={emailPort}
                    onChange={e => setEmailPort(e.target.value)}
                    placeholder="587"
                    className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label htmlFor="email-username" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Username (optional)
                  </label>
                  <input
                    id="email-username"
                    type="text"
                    value={emailUsername}
                    onChange={e => setEmailUsername(e.target.value)}
                    className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
                <div>
                  <label htmlFor="email-password" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Password (optional)
                  </label>
                  <input
                    id="email-password"
                    type="password"
                    value={emailPassword}
                    onChange={e => setEmailPassword(e.target.value)}
                    className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  <p className="mt-1.5 text-2xs text-ink-4">
                    Leave username/password blank for a relay that needs no auth.
                  </p>
                </div>
                <div>
                  <label htmlFor="email-from" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    From address
                  </label>
                  <input
                    id="email-from"
                    type="email"
                    required
                    value={emailFrom}
                    onChange={e => setEmailFrom(e.target.value)}
                    placeholder="orders@mystore.com"
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
                {connectError && (
                  <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{connectError}</p>
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
              <>
                <div className="flex gap-1 mb-4 p-0.5 rounded-lg bg-fill">
                  <button
                    type="button"
                    onClick={() => setConnectMode('auto')}
                    className={clsx(
                      'flex-1 py-1.5 rounded-md text-xs font-medium transition-colors',
                      connectMode === 'auto'
                        ? 'bg-blue-600 text-white'
                        : 'text-ink-3',
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
                        : 'text-ink-3',
                    )}
                  >
                    Manual setup
                  </button>
                </div>

                {connectMode === 'auto' ? (
                  <form onSubmit={handleStartConnect} className="space-y-3">
                    <div>
                      <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                        Store URL
                      </label>
                      <input
                        type="url"
                        required
                        autoFocus
                        value={storeUrl}
                        onChange={e => setStoreUrl(e.target.value)}
                        placeholder="https://mystore.com"
                        className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                      />
                      <p className="mt-1.5 text-2xs text-ink-4">
                        Redirects to your store to approve access. Requires your store to reach this server.
                      </p>
                    </div>
                    {connectError && (
                      <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{connectError}</p>
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
                      <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                        Store URL
                      </label>
                      <input
                        type="url"
                        required
                        autoFocus
                        value={storeUrl}
                        onChange={e => setStoreUrl(e.target.value)}
                        placeholder="https://mystore.com"
                        className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                      />
                    </div>
                    <div>
                      <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                        Consumer Key
                      </label>
                      <input
                        type="text"
                        required
                        value={consumerKey}
                        onChange={e => setConsumerKey(e.target.value)}
                        placeholder="ck_..."
                        className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                      />
                    </div>
                    <div>
                      <label className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                        Consumer Secret
                      </label>
                      <input
                        type="password"
                        required
                        value={consumerSecret}
                        onChange={e => setConsumerSecret(e.target.value)}
                        placeholder="cs_..."
                        className="w-full px-3 py-2 text-xs font-mono rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                      />
                      <p className="mt-1.5 text-2xs text-ink-4">
                        Generate under WooCommerce → Settings → Advanced → REST API.
                      </p>
                    </div>
                    {connectError && (
                      <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{connectError}</p>
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
          <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">Import products</h3>
              <button onClick={closeImport} aria-label="Close">
                <X size={14} className="text-ink-4" />
              </button>
            </div>

            {importJob ? (
              <div className="space-y-3">
                <p className="text-xs text-ink-2">
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
                <p className="text-2xs text-ink-4">
                  Imports every WooCommerce product not yet linked, along with its category. Existing categories are matched by name; missing ones are created.
                </p>
                <div>
                  <label htmlFor="import-warehouse-id" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Default warehouse ID
                  </label>
                  <select
                    id="import-warehouse-id"
                    required
                    autoFocus
                    value={importWarehouseId}
                    onChange={e => setImportWarehouseId(e.target.value)}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    <option value="" disabled>Select a warehouse…</option>
                    {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                  </select>
                  <p className="mt-1.5 text-2xs text-ink-4">
                    Imported stock quantities are attributed to this warehouse — WooCommerce has no warehouse concept.
                  </p>
                </div>
                <div>
                  <label htmlFor="import-currency" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Default currency
                  </label>
                  <select
                    id="import-currency"
                    required
                    value={importCurrency}
                    onChange={e => setImportCurrency(e.target.value)}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    <option value="" disabled>Select a currency…</option>
                    {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
                {importError && (
                  <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{importError}</p>
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
          <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">Delete imported products</h3>
              {deleteResult && (
                <button onClick={closeDelete} aria-label="Close">
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
                  onClick={closeDelete}
                  className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
                >
                  Done
                </button>
              </div>
            ) : (
              <form onSubmit={handleDeleteImported} className="space-y-3">
                <div className="dark:bg-red-500/10 bg-red-50 border dark:border-red-500/20 border-red-200 rounded-lg p-3">
                  <p className="text-xs text-danger">
                    This permanently deletes every product imported from <strong>{deletingConnection.store_url}</strong> — including
                    their variants, SKUs, stock history, and images. Categories left empty afterward are also deleted. This cannot be undone.
                  </p>
                </div>
                <div>
                  <label htmlFor="delete-confirm" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Type DELETE to confirm
                  </label>
                  <input
                    id="delete-confirm"
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
                    onClick={closeDelete}
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
        </div>
      )}
    </div>
  )
}

