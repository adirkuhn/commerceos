import { useEffect, useRef, useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import { useAuth } from '../auth/AuthContext'
import * as api from './api'
import { CARRIERS, PACKAGE_TYPES } from './types'
import type { ShippingCarrierOverride } from './types'
import * as integrationsApi from '../integrations/api'
import ProviderWebhooksPanel from '../integrations/ProviderWebhooksPanel'

// Same tier as WooCommerceSyncSettingsPage's webhook-secret tab -- exposes
// a signing secret + the ability to register/remove live provider
// subscriptions, security/ops-adjacent rather than day-to-day integration
// use.
function canManageWebhooks(role: string | undefined): boolean {
  return role === 'customer_owner' || role === 'customer_admin' || role === 'super_admin'
}

// Mirrors WooCommerceSyncSettingsPage's own webhookURL helper -- the
// webhook receiver lives at a URL outside VITE_API_BASE_URL's per-domain
// convention, the single unauthenticated route mounted at
// /webhooks/v1/{tenant_id}/{provider} on the API's root router.
function webhookURL(tenantId: string): string {
  const apiBase = (import.meta.env.VITE_API_BASE_URL as string) ?? ''
  return `${apiBase}/webhooks/v1/${tenantId}/myparcel`
}

// MyParcel has one credential (the API key) -- no OAuth redirect, no
// client_id/secret pair like Bol's client_credentials, so the connect step
// is a single field. Once connected (connectionId is set, either just now
// or passed in from an already-connected integration), this page also
// shows the global default carrier/package-type/label-format + per-carrier
// override rows -- "configure general configuration and settings global
// and per carrier" from the design.
export default function MyParcelSyncSettingsPage({
  connectionId: initialConnectionId, onClose, onConnected, focusWebhooks,
}: { connectionId?: string; onClose: () => void; onConnected?: () => void; focusWebhooks?: boolean }) {
  const { token, user } = useAuth()
  const [connectionId, setConnectionId] = useState(initialConnectionId ?? '')
  const webhooksSectionRef = useRef<HTMLDivElement>(null)
  const [apiKey, setApiKey] = useState('')
  const [connectError, setConnectError] = useState('')
  const [connecting, setConnecting] = useState(false)

  const [defaultCarrier, setDefaultCarrier] = useState(1)
  const [defaultPackageType, setDefaultPackageType] = useState(1)
  const [defaultLabelFormat, setDefaultLabelFormat] = useState<'pdf' | 'zpl'>('pdf')
  const [defaultLabelSize, setDefaultLabelSize] = useState<'A4' | 'A6'>('A4')
  const [defaultSignatureRequired, setDefaultSignatureRequired] = useState(false)
  const [defaultOnlyRecipient, setDefaultOnlyRecipient] = useState(false)
  const [insureByDefault, setInsureByDefault] = useState(false)
  const [myparcelAccountId, setMyparcelAccountId] = useState<number | ''>('')
  const [myparcelShopId, setMyparcelShopId] = useState<number | ''>('')
  const [autoPrintEnabled, setAutoPrintEnabled] = useState(false)
  const [overrides, setOverrides] = useState<ShippingCarrierOverride[]>([])
  const [settingsLoading, setSettingsLoading] = useState(false)
  const [settingsError, setSettingsError] = useState('')
  const [saving, setSaving] = useState(false)
  const [savedNote, setSavedNote] = useState('')

  const [webhookSecret, setWebhookSecret] = useState('')
  const [generatingSecret, setGeneratingSecret] = useState(false)
  const [webhookSecretError, setWebhookSecretError] = useState('')
  const [copiedField, setCopiedField] = useState<'url' | 'secret' | null>(null)

  async function handleGenerateWebhookSecret() {
    setWebhookSecretError('')
    setGeneratingSecret(true)
    try {
      const result = await integrationsApi.setWebhookSecret(token ?? '', connectionId)
      setWebhookSecret(result.webhook_secret)
    } catch (err) {
      setWebhookSecretError(err instanceof Error ? err.message : 'Failed to generate webhook secret')
    } finally {
      setGeneratingSecret(false)
    }
  }

  async function handleCopy(field: 'url' | 'secret', value: string) {
    try {
      await navigator.clipboard.writeText(value)
      setCopiedField(field)
      setTimeout(() => setCopiedField(prev => (prev === field ? null : prev)), 1500)
    } catch {
      // Clipboard access can be denied -- the value is already shown
      // selectable on screen, so this is a convenience failure, not a
      // blocker.
    }
  }

  useEffect(() => {
    if (!connectionId || !token) return
    setSettingsLoading(true)
    setSettingsError('')
    api.getShippingSettings(token, connectionId)
      .then(result => {
        if (result.settings) {
          setDefaultCarrier(result.settings.default_carrier)
          setDefaultPackageType(result.settings.default_package_type)
          setDefaultLabelFormat(result.settings.default_label_format)
          setDefaultLabelSize(result.settings.default_label_size ?? 'A4')
          setDefaultSignatureRequired(result.settings.default_signature_required)
          setDefaultOnlyRecipient(result.settings.default_only_recipient)
          setInsureByDefault(result.settings.insure_by_default)
          setMyparcelAccountId(result.settings.myparcel_account_id ?? '')
          setMyparcelShopId(result.settings.myparcel_shop_id ?? '')
          setAutoPrintEnabled(result.settings.auto_print_enabled)
        }
        setOverrides(result.carrier_overrides)
      })
      .catch(err => setSettingsError(err instanceof Error ? err.message : 'Failed to load settings'))
      .finally(() => setSettingsLoading(false))
  }, [connectionId, token])

  // Deep link from Integrations.tsx's "Register webhook" quick action --
  // jump straight to the Webhooks section instead of making the admin
  // scroll past carrier overrides to find it.
  useEffect(() => {
    if (focusWebhooks && !settingsLoading && connectionId) {
      // jsdom (unit tests) has no real layout engine -- scrollIntoView
      // throws "Not implemented" there. try/catch keeps that a no-op
      // instead of failing the render.
      try {
        webhooksSectionRef.current?.scrollIntoView({ block: 'start' })
      } catch {
        // no-op -- see comment above
      }
    }
  }, [focusWebhooks, settingsLoading, connectionId])

  async function handleConnect(e: React.FormEvent) {
    e.preventDefault()
    if (!token) return
    setConnectError('')
    setConnecting(true)
    try {
      const conn = await api.connectMyParcel(token, apiKey)
      setConnectionId(conn.connection_id)
      onConnected?.()
    } catch (err) {
      setConnectError(err instanceof Error ? err.message : 'Failed to connect')
    } finally {
      setConnecting(false)
    }
  }

  async function handleSaveSettings(e: React.FormEvent) {
    e.preventDefault()
    if (!token || !connectionId) return
    setSettingsError('')
    setSaving(true)
    setSavedNote('')
    try {
      await api.updateShippingSettings(token, connectionId, {
        default_carrier: defaultCarrier, default_package_type: defaultPackageType, default_label_format: defaultLabelFormat,
        default_label_size: defaultLabelSize,
        default_signature_required: defaultSignatureRequired, default_only_recipient: defaultOnlyRecipient,
        insure_by_default: insureByDefault,
        myparcel_account_id: myparcelAccountId === '' ? null : myparcelAccountId,
        myparcel_shop_id: myparcelShopId === '' ? null : myparcelShopId,
        auto_print_enabled: autoPrintEnabled,
      })
      setSavedNote('Saved')
    } catch (err) {
      setSettingsError(err instanceof Error ? err.message : 'Failed to save settings')
    } finally {
      setSaving(false)
    }
  }

  function overrideFor(carrier: number): ShippingCarrierOverride | undefined {
    return overrides.find(o => o.carrier === carrier)
  }

  async function handleOverrideChange(carrier: number, field: 'package_type' | 'signature_required' | 'only_recipient' | 'insure_by_default', value: number | boolean | null) {
    if (!token || !connectionId) return
    const current = overrideFor(carrier)
    const input = {
      package_type: current?.package_type ?? null,
      signature_required: current?.signature_required ?? null,
      only_recipient: current?.only_recipient ?? null,
      insure_by_default: current?.insure_by_default ?? null,
      [field]: value,
    }
    try {
      const updated = await api.upsertCarrierOverride(token, connectionId, carrier, input)
      setOverrides(prev => {
        const next = prev.filter(o => o.carrier !== carrier)
        next.push(updated)
        return next
      })
    } catch (err) {
      setSettingsError(err instanceof Error ? err.message : 'Failed to save carrier override')
    }
  }

  const selectClass = 'px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900'

  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="flex items-center gap-3 px-5 py-4 border-b dark:border-[#2D3748] border-gray-200">
        <button onClick={onClose} aria-label="Back" className="p-1.5 rounded-lg dark:hover:bg-[#2D3748] hover:bg-gray-100 transition-colors">
          <ArrowLeft size={18} className="dark:text-gray-400 text-gray-500" />
        </button>
        <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">MyParcel</h2>
      </div>
      <div className="flex-1 overflow-y-auto p-5 max-w-2xl">
        {!connectionId ? (
          <form onSubmit={handleConnect} className="space-y-4">
            <p className="text-xs dark:text-gray-500 text-gray-400">
              Paste your MyParcel API key. No redirect step -- MyParcel authenticates via a single API key.
            </p>
            <div>
              <label htmlFor="myparcel-api-key" className="block text-xs font-medium dark:text-gray-400 text-gray-500 mb-1">API key</label>
              <input
                id="myparcel-api-key"
                type="password"
                value={apiKey}
                onChange={e => setApiKey(e.target.value)}
                required
                className="w-full px-3 py-2 rounded-lg text-sm dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900"
              />
            </div>
            {connectError && <p className="text-xs text-red-400">{connectError}</p>}
            <button
              type="submit"
              disabled={connecting}
              className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
            >
              {connecting ? 'Connecting…' : 'Connect'}
            </button>
          </form>
        ) : (
          <div className="space-y-6">
            {settingsLoading && <p className="text-xs dark:text-gray-500 text-gray-400">Loading settings…</p>}
            <form onSubmit={handleSaveSettings} className="space-y-4">
              <h3 className="text-xs font-semibold dark:text-gray-300 text-gray-700">Global defaults</h3>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="default-carrier" className="block text-[11px] font-medium dark:text-gray-400 text-gray-500 mb-1">Default carrier</label>
                  <select id="default-carrier" value={defaultCarrier} onChange={e => setDefaultCarrier(Number(e.target.value))} className={selectClass + ' w-full'}>
                    {CARRIERS.map(c => <option key={c.id} value={c.id}>{c.label}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="default-package-type" className="block text-[11px] font-medium dark:text-gray-400 text-gray-500 mb-1">Default package type</label>
                  <select id="default-package-type" value={defaultPackageType} onChange={e => setDefaultPackageType(Number(e.target.value))} className={selectClass + ' w-full'}>
                    {PACKAGE_TYPES.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="default-label-format" className="block text-[11px] font-medium dark:text-gray-400 text-gray-500 mb-1">Label format</label>
                  <select id="default-label-format" value={defaultLabelFormat} onChange={e => setDefaultLabelFormat(e.target.value as 'pdf' | 'zpl')} className={selectClass + ' w-full'}>
                    <option value="pdf">PDF</option>
                    <option value="zpl">ZPL (thermal)</option>
                  </select>
                </div>
                <div>
                  <label htmlFor="default-label-size" className="block text-[11px] font-medium dark:text-gray-400 text-gray-500 mb-1">Label print size</label>
                  <select id="default-label-size" value={defaultLabelSize} onChange={e => setDefaultLabelSize(e.target.value as 'A4' | 'A6')} className={selectClass + ' w-full'}>
                    <option value="A4">A4</option>
                    <option value="A6">A6</option>
                  </select>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">Default only -- staff can pick a different size on each print.</p>
                </div>
              </div>
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 text-xs dark:text-gray-300 text-gray-700">
                  <input type="checkbox" checked={defaultSignatureRequired} onChange={e => setDefaultSignatureRequired(e.target.checked)} />
                  Signature required
                </label>
                <label className="flex items-center gap-2 text-xs dark:text-gray-300 text-gray-700">
                  <input type="checkbox" checked={defaultOnlyRecipient} onChange={e => setDefaultOnlyRecipient(e.target.checked)} />
                  Only recipient
                </label>
              </div>
              <div>
                <label className="flex items-center gap-2 text-xs dark:text-gray-300 text-gray-700">
                  <input type="checkbox" checked={insureByDefault} onChange={e => setInsureByDefault(e.target.checked)} />
                  Insure shipments by default
                </label>
                <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">
                  Insured amount is always the order's own value, computed automatically when the label is created -- not a fixed figure set here. Insuring a shipment forces "Signature required" and "Only recipient" -- required by MyParcel whenever insurance is set.
                </p>
              </div>
              <h3 className="text-xs font-semibold dark:text-gray-300 text-gray-700 pt-2">MyParcel fulfilment account</h3>
              <p className="text-[10px] dark:text-gray-500 text-gray-400 -mt-2">
                Required to export orders via MyParcel's Orders API before creating a label -- find these in your MyParcel account settings.
              </p>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="myparcel-account-id" className="block text-[11px] font-medium dark:text-gray-400 text-gray-500 mb-1">Account ID</label>
                  <input
                    id="myparcel-account-id"
                    type="number"
                    value={myparcelAccountId}
                    onChange={e => setMyparcelAccountId(e.target.value === '' ? '' : Number(e.target.value))}
                    className="w-full px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900"
                  />
                </div>
                <div>
                  <label htmlFor="myparcel-shop-id" className="block text-[11px] font-medium dark:text-gray-400 text-gray-500 mb-1">Shop ID</label>
                  <input
                    id="myparcel-shop-id"
                    type="number"
                    value={myparcelShopId}
                    onChange={e => setMyparcelShopId(e.target.value === '' ? '' : Number(e.target.value))}
                    className="w-full px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-100 text-gray-900"
                  />
                </div>
              </div>
              <label className="flex items-center gap-2 text-xs dark:text-gray-300 text-gray-700">
                <input type="checkbox" checked={autoPrintEnabled} onChange={e => setAutoPrintEnabled(e.target.checked)} />
                Auto-print labels
              </label>
              <p className="text-[10px] dark:text-gray-500 text-gray-400 -mt-2">
                Opens the print dialog automatically once a label is created, instead of requiring a manual click. You still pick the printer yourself -- this isn't silent printing.
              </p>
              {settingsError && <p className="text-xs text-red-400">{settingsError}</p>}
              {savedNote && <p className="text-xs text-green-400">{savedNote}</p>}
              <button
                type="submit"
                disabled={saving}
                className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {saving ? 'Saving…' : 'Save defaults'}
              </button>
            </form>

            <div>
              <h3 className="text-xs font-semibold dark:text-gray-300 text-gray-700 mb-2">Per-carrier overrides</h3>
              <div className="space-y-2">
                {CARRIERS.map(c => {
                  const o = overrideFor(c.id)
                  return (
                    <div key={c.id} className="flex items-center gap-3 p-2.5 rounded-lg border dark:border-[#2D3748] border-gray-200">
                      <span className="text-xs dark:text-gray-200 text-gray-800 w-40 shrink-0">{c.label}</span>
                      <select
                        aria-label={`${c.label} package type override`}
                        value={o?.package_type ?? ''}
                        onChange={e => handleOverrideChange(c.id, 'package_type', e.target.value === '' ? null : Number(e.target.value))}
                        className={selectClass}
                      >
                        <option value="">(use default)</option>
                        {PACKAGE_TYPES.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
                      </select>
                      <label className="flex items-center gap-1.5 text-[11px] dark:text-gray-400 text-gray-500">
                        <input
                          type="checkbox"
                          aria-label={`${c.label} signature required override`}
                          checked={o?.signature_required ?? false}
                          onChange={e => handleOverrideChange(c.id, 'signature_required', e.target.checked)}
                        />
                        Signature
                      </label>
                      <label className="flex items-center gap-1.5 text-[11px] dark:text-gray-400 text-gray-500">
                        <input
                          type="checkbox"
                          aria-label={`${c.label} insure by default override`}
                          checked={o?.insure_by_default ?? false}
                          onChange={e => handleOverrideChange(c.id, 'insure_by_default', e.target.checked)}
                        />
                        Insure
                      </label>
                    </div>
                  )
                })}
              </div>
            </div>

            {canManageWebhooks(user?.role) && (
              <div ref={webhooksSectionRef}>
                <h3 className="text-xs font-semibold dark:text-gray-300 text-gray-700 mb-2">Webhooks</h3>
                <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
                  <div>
                    <label className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                      Delivery URL
                    </label>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 px-3 py-2 text-[11px] rounded-lg dark:bg-[#111827] bg-gray-50 border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 overflow-x-auto whitespace-nowrap">
                        {webhookURL(user?.tenant_id ?? '')}
                      </code>
                      <button
                        type="button"
                        onClick={() => handleCopy('url', webhookURL(user?.tenant_id ?? ''))}
                        className="px-2.5 py-2 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 transition-colors shrink-0"
                      >
                        {copiedField === 'url' ? 'Copied' : 'Copy'}
                      </button>
                    </div>
                  </div>

                  {webhookSecret ? (
                    <div>
                      <label className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                        Secret (shown once -- copy it now)
                      </label>
                      <div className="flex items-center gap-2">
                        <code className="flex-1 px-3 py-2 text-[11px] font-mono rounded-lg dark:bg-[#111827] bg-gray-50 border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 overflow-x-auto whitespace-nowrap">
                          {webhookSecret}
                        </code>
                        <button
                          type="button"
                          onClick={() => handleCopy('secret', webhookSecret)}
                          className="px-2.5 py-2 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 transition-colors shrink-0"
                        >
                          {copiedField === 'secret' ? 'Copied' : 'Copy'}
                        </button>
                      </div>
                      <p className="mt-1.5 text-[10px] text-amber-500">
                        This won't be shown again -- generate a new one if you lose it.
                      </p>
                    </div>
                  ) : (
                    <p className="text-xs dark:text-gray-500 text-gray-400">
                      No secret generated yet -- Create a subscription below needs one first (MyParcel embeds it as a
                      ?token= query param, since MyParcel signs nothing).
                    </p>
                  )}

                  {webhookSecretError && <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{webhookSecretError}</p>}

                  <button
                    type="button"
                    onClick={handleGenerateWebhookSecret}
                    disabled={generatingSecret}
                    className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
                  >
                    {generatingSecret ? 'Generating…' : webhookSecret ? 'Regenerate secret' : 'Generate secret'}
                  </button>
                </div>

                <div className="mt-3">
                  <ProviderWebhooksPanel token={token ?? ''} connectionId={connectionId} />
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
