import { useEffect, useRef, useState } from 'react'
import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import * as api from './api'
import { CARRIERS, PACKAGE_TYPES } from './types'
import type { ShippingCarrierOverride } from './types'
import * as integrationsApi from '../integrations/api'
import ProviderWebhooksPanel from '../integrations/ProviderWebhooksPanel'
import { apiErrorText } from '../api/apiErrorText'
import { appEnv } from '../config/appEnv'

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
  const apiBase = (appEnv().VITE_API_BASE_URL as string) ?? ''
  return `${apiBase}/webhooks/v1/${tenantId}/myparcel`
}

// MyParcel has one credential (the API key) -- no OAuth redirect, no
// client_id/secret pair like Bol's client_credentials, so the connect step
// is a single field. Once connected (connectionId is set, either just now
// or passed in from an already-connected integration), this page also
// shows the global default carrier/package-type/label-format + per-carrier
// override rows -- "configure general configuration and settings global
// and per carrier" from the design.
// Brand name, label sizes and an example URL: not translated.
const PROVIDER_NAME = 'MyParcel'
const LABEL_SIZES = ['A4', 'A6'] as const
const TRACKING_TEMPLATE_EXAMPLE = 'https://example.com/track?code={barcode}&postcode={postcode}'

export default function MyParcelSyncSettingsPage({
  connectionId: initialConnectionId, onClose, onConnected, focusWebhooks,
}: { connectionId?: string; onClose: () => void; onConnected?: () => void; focusWebhooks?: boolean }) {
  const { token, user } = useAuth()
  const { t } = useTranslation(['shipping', 'common'])
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
  const [exportOrderToMyParcel, setExportOrderToMyParcel] = useState(false)
  const [overrides, setOverrides] = useState<ShippingCarrierOverride[]>([])
  const [settingsLoading, setSettingsLoading] = useState(false)
  const [settingsError, setSettingsError] = useState('')
  const [saving, setSaving] = useState(false)
  const [savedNote, setSavedNote] = useState('')

  const [webhookSecret, setWebhookSecret] = useState('')
  const [generatingSecret, setGeneratingSecret] = useState(false)
  const [webhookSecretError, setWebhookSecretError] = useState('')
  const [copiedField, setCopiedField] = useState<'url' | 'secret' | null>(null)

  const [registeringWebhook, setRegisteringWebhook] = useState(false)
  const [registerWebhookError, setRegisterWebhookError] = useState('')
  const [registerWebhookDone, setRegisterWebhookDone] = useState(false)
  // Bumped after a successful register so ProviderWebhooksPanel (its own
  // component, own useProviderWebhooks fetch) remounts and re-fetches --
  // RegisterMyParcelWebhookUseCase writes directly at MyParcel, not through
  // that hook, so without this the new subscription wouldn't show up in
  // its table until the next full page load.
  const [webhooksPanelKey, setWebhooksPanelKey] = useState(0)

  async function handleGenerateWebhookSecret() {
    setWebhookSecretError('')
    setGeneratingSecret(true)
    try {
      const result = await integrationsApi.setWebhookSecret(token ?? '', connectionId)
      setWebhookSecret(result.webhook_secret)
    } catch (err) {
      setWebhookSecretError(apiErrorText(err) ?? t('myparcel.secretFailed'))
    } finally {
      setGeneratingSecret(false)
    }
  }

  // The actual "make MyParcel send webhooks to CommerceOS" action --
  // generates a fresh signing secret (RegisterMyParcelWebhookUseCase
  // requires one) then registers the shipment_status_change callback with
  // MyParcel using it. Safe to re-run: MyParcel overwrites its existing
  // subscription of that hook type rather than erroring (see
  // domain/integrations.MyParcelClient.RegisterWebhook's doc comment), so
  // regenerating the secret and re-registering together always leave the
  // live subscription's token in sync with the secret CommerceOS has on file.
  async function handleRegisterWebhook() {
    setRegisterWebhookError('')
    setRegisteringWebhook(true)
    try {
      const secretResult = await integrationsApi.setWebhookSecret(token ?? '', connectionId)
      setWebhookSecret(secretResult.webhook_secret)
      await api.registerMyParcelWebhook(token ?? '', connectionId)
      setRegisterWebhookDone(true)
      setWebhooksPanelKey(k => k + 1)
    } catch (err) {
      setRegisterWebhookError(apiErrorText(err) ?? t('myparcel.registerFailed'))
    } finally {
      setRegisteringWebhook(false)
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
          setExportOrderToMyParcel(result.settings.export_order_to_myparcel ?? false)
        }
        setOverrides(result.carrier_overrides)
      })
      .catch(err => setSettingsError(apiErrorText(err) ?? t('myparcel.loadFailed')))
      .finally(() => setSettingsLoading(false))
  }, [connectionId, token])

  // Deep link from Integrations.tsx's "Register webhook" quick action --
  // scrolls straight to the Webhooks section. Does NOT auto-run the
  // registration itself -- that's a real outbound call to MyParcel
  // (overwrites whatever subscription of this hook type MyParcel already
  // has), so it stays behind the explicit "Register with MyParcel" click.
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
      setConnectError(apiErrorText(err) ?? t('myparcel.connectFailed'))
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
        export_order_to_myparcel: exportOrderToMyParcel,
      })
      setSavedNote(t('myparcel.saved'))
    } catch (err) {
      setSettingsError(apiErrorText(err) ?? t('myparcel.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  function overrideFor(carrier: number): ShippingCarrierOverride | undefined {
    return overrides.find(o => o.carrier === carrier)
  }

  async function handleOverrideChange(carrier: number, field: 'package_type' | 'signature_required' | 'only_recipient' | 'insure_by_default' | 'tracking_url_template', value: number | boolean | string | null) {
    if (!token || !connectionId) return
    const current = overrideFor(carrier)
    const input = {
      package_type: current?.package_type ?? null,
      signature_required: current?.signature_required ?? null,
      only_recipient: current?.only_recipient ?? null,
      insure_by_default: current?.insure_by_default ?? null,
      tracking_url_template: current?.tracking_url_template ?? null,
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
      setSettingsError(apiErrorText(err) ?? t('myparcel.overrideFailed'))
    }
  }

  const selectClass = 'px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink'

  return (
    <div className="flex flex-col">
      <div className="flex items-center gap-3 px-5 py-4 border-b border-line">
        <button onClick={onClose} aria-label={t('common:actions.back')} className="p-1.5 rounded-lg hover:bg-hover transition-colors">
          <ArrowLeft size={18} className="text-ink-3" />
        </button>
        <h2 className="text-sm font-semibold text-ink">{PROVIDER_NAME}</h2>
      </div>
      {/* The page's own <main> scrolls (App.tsx) -- no overflow/height
          wiring needed here, just flow naturally. */}
      <div className="p-5 max-w-2xl">
        {!connectionId ? (
          <form onSubmit={handleConnect} className="space-y-4">
            <p className="text-xs text-ink-4">
              {t('myparcel.intro')}
            </p>
            <div>
              <label htmlFor="myparcel-api-key" className="block text-xs font-medium text-ink-3 mb-1">{t('myparcel.apiKey')}</label>
              <input
                id="myparcel-api-key"
                type="password"
                value={apiKey}
                onChange={e => setApiKey(e.target.value)}
                required
                className="w-full px-3 py-2 rounded-lg text-sm bg-surface border border-line text-ink"
              />
            </div>
            {connectError && <p className="text-xs text-danger">{connectError}</p>}
            <button
              type="submit"
              disabled={connecting}
              className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
            >
              {connecting ? t('myparcel.connecting') : t('myparcel.connect')}
            </button>
          </form>
        ) : (
          <div className="space-y-6">
            {settingsLoading && <p className="text-xs text-ink-4">{t('myparcel.loadingSettings')}</p>}
            <form onSubmit={handleSaveSettings} className="space-y-4">
              <h3 className="text-xs font-semibold text-ink-2">{t('myparcel.globalDefaults')}</h3>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="default-carrier" className="block text-2xs font-medium text-ink-3 mb-1">{t('myparcel.defaultCarrier')}</label>
                  <select id="default-carrier" value={defaultCarrier} onChange={e => setDefaultCarrier(Number(e.target.value))} className={selectClass + ' w-full'}>
                    {CARRIERS.map(c => <option key={c.id} value={c.id}>{c.label}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="default-package-type" className="block text-2xs font-medium text-ink-3 mb-1">{t('myparcel.defaultPackageType')}</label>
                  <select id="default-package-type" value={defaultPackageType} onChange={e => setDefaultPackageType(Number(e.target.value))} className={selectClass + ' w-full'}>
                    {PACKAGE_TYPES.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="default-label-format" className="block text-2xs font-medium text-ink-3 mb-1">{t('myparcel.labelFormat')}</label>
                  <select id="default-label-format" value={defaultLabelFormat} onChange={e => setDefaultLabelFormat(e.target.value as 'pdf' | 'zpl')} className={selectClass + ' w-full'}>
                    <option value="pdf">PDF</option>
                    <option value="zpl">{t('myparcel.zpl')}</option>
                  </select>
                </div>
                <div>
                  <label htmlFor="default-label-size" className="block text-2xs font-medium text-ink-3 mb-1">{t('myparcel.labelSize')}</label>
                  <select id="default-label-size" value={defaultLabelSize} onChange={e => setDefaultLabelSize(e.target.value as 'A4' | 'A6')} className={selectClass + ' w-full'}>
                    {LABEL_SIZES.map(size => <option key={size} value={size}>{size}</option>)}
                  </select>
                  <p className="text-2xs text-ink-4 mt-1">{t('myparcel.labelSizeHint')}</p>
                </div>
              </div>
              <div className="flex items-center gap-4">
                <label className="flex items-center gap-2 text-xs text-ink-2">
                  <input type="checkbox" checked={defaultSignatureRequired} onChange={e => setDefaultSignatureRequired(e.target.checked)} />
                  {t('myparcel.signatureRequired')}
                </label>
                <label className="flex items-center gap-2 text-xs text-ink-2">
                  <input type="checkbox" checked={defaultOnlyRecipient} onChange={e => setDefaultOnlyRecipient(e.target.checked)} />
                  {t('myparcel.onlyRecipient')}
                </label>
              </div>
              <div>
                <label className="flex items-center gap-2 text-xs text-ink-2">
                  <input type="checkbox" checked={insureByDefault} onChange={e => setInsureByDefault(e.target.checked)} />
                  {t('myparcel.insureDefault')}
                </label>
                <p className="text-2xs text-ink-4 mt-1">
                  {t('myparcel.insureHint')}
                </p>
              </div>
              <h3 className="text-xs font-semibold text-ink-2 pt-2">{t('myparcel.account')}</h3>
              <p className="text-2xs text-ink-4 -mt-2">
                {t('myparcel.accountHint')}
              </p>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="myparcel-account-id" className="block text-2xs font-medium text-ink-3 mb-1">{t('myparcel.accountId')}</label>
                  <input
                    id="myparcel-account-id"
                    type="number"
                    value={myparcelAccountId}
                    onChange={e => setMyparcelAccountId(e.target.value === '' ? '' : Number(e.target.value))}
                    className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
                  />
                </div>
                <div>
                  <label htmlFor="myparcel-shop-id" className="block text-2xs font-medium text-ink-3 mb-1">{t('myparcel.shopId')}</label>
                  <input
                    id="myparcel-shop-id"
                    type="number"
                    value={myparcelShopId}
                    onChange={e => setMyparcelShopId(e.target.value === '' ? '' : Number(e.target.value))}
                    className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
                  />
                </div>
              </div>
              <div>
                <label className="flex items-center gap-2 text-xs text-ink-2">
                  <input type="checkbox" checked={exportOrderToMyParcel} onChange={e => setExportOrderToMyParcel(e.target.checked)} />
                  {t('myparcel.exportOrders')}
                </label>
                <p className="text-2xs text-ink-4 mt-1">
                  {t('myparcel.exportOrdersHint')}
                </p>
              </div>
              <label className="flex items-center gap-2 text-xs text-ink-2">
                <input type="checkbox" checked={autoPrintEnabled} onChange={e => setAutoPrintEnabled(e.target.checked)} />
                {t('myparcel.autoPrint')}
              </label>
              <p className="text-2xs text-ink-4 -mt-2">
                {t('myparcel.autoPrintHint')}
              </p>
              {settingsError && <p className="text-xs text-danger">{settingsError}</p>}
              {savedNote && <p className="text-xs text-success">{savedNote}</p>}
              <button
                type="submit"
                disabled={saving}
                className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
              >
                {saving ? t('myparcel.saving') : t('myparcel.saveDefaults')}
              </button>
            </form>

            <div>
              <h3 className="text-xs font-semibold text-ink-2 mb-2">{t('myparcel.overrides')}</h3>
              <div className="space-y-2">
                {CARRIERS.map(c => {
                  const o = overrideFor(c.id)
                  return (
                    <div key={c.id} className="flex flex-col gap-2 p-2.5 rounded-lg border border-line">
                      <div className="flex items-center gap-3">
                        <span className="text-xs text-ink w-40 shrink-0">{c.label}</span>
                        <select
                          aria-label={t('myparcel.overridePackageAria', { carrier: c.label })}
                          value={o?.package_type ?? ''}
                          onChange={e => handleOverrideChange(c.id, 'package_type', e.target.value === '' ? null : Number(e.target.value))}
                          className={selectClass}
                        >
                          <option value="">{t('myparcel.useDefault')}</option>
                          {PACKAGE_TYPES.map(p => <option key={p.id} value={p.id}>{p.label}</option>)}
                        </select>
                        <label className="flex items-center gap-1.5 text-2xs text-ink-3">
                          <input
                            type="checkbox"
                            aria-label={t('myparcel.overrideSignatureAria', { carrier: c.label })}
                            checked={o?.signature_required ?? false}
                            onChange={e => handleOverrideChange(c.id, 'signature_required', e.target.checked)}
                          />
                          {t('myparcel.signature')}
                        </label>
                        <label className="flex items-center gap-1.5 text-2xs text-ink-3">
                          <input
                            type="checkbox"
                            aria-label={t('myparcel.overrideInsureAria', { carrier: c.label })}
                            checked={o?.insure_by_default ?? false}
                            onChange={e => handleOverrideChange(c.id, 'insure_by_default', e.target.checked)}
                          />
                          {t('myparcel.insure')}
                        </label>
                      </div>
                      <div className="flex items-center gap-3">
                        <label className="text-2xs text-ink-4 w-40 shrink-0">{t('myparcel.trackingTemplate')}</label>
                        <input
                          type="text"
                          aria-label={t('myparcel.trackingTemplateAria', { carrier: c.label })}
                          defaultValue={o?.tracking_url_template ?? ''}
                          onBlur={e => {
                            const value = e.target.value.trim()
                            if (value === (o?.tracking_url_template ?? '')) return
                            handleOverrideChange(c.id, 'tracking_url_template', value === '' ? null : value)
                          }}
                          placeholder={TRACKING_TEMPLATE_EXAMPLE}
                          className="flex-1 px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink placeholder:text-ink-4"
                        />
                      </div>
                    </div>
                  )
                })}
              </div>
            </div>

            {canManageWebhooks(user?.role) && (
              <div ref={webhooksSectionRef}>
                <h3 className="text-xs font-semibold text-ink-2 mb-2">{t('myparcel.webhooks')}</h3>
                <div className="bg-raised rounded-xl border border-line p-4 space-y-3">
                  <div className="flex items-start justify-between gap-3">
                    <p className="text-xs text-ink-3">
                      {t('myparcel.webhooksIntro')}
                    </p>
                    <button
                      type="button"
                      onClick={handleRegisterWebhook}
                      disabled={registeringWebhook}
                      className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors shrink-0"
                    >
                      {registeringWebhook ? t('myparcel.registering') : t('myparcel.register')}
                    </button>
                  </div>
                  {registerWebhookError && (
                    <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{registerWebhookError}</p>
                  )}
                  {registerWebhookDone && !registerWebhookError && (
                    <p className="text-xs text-success bg-green-50 dark:bg-green-500/10 rounded-lg px-2 py-1.5">
                      {t('myparcel.registered')}
                    </p>
                  )}

                  <div>
                    <label className="block text-2xs text-ink-4 caps-label mb-1.5">
                      {t('myparcel.deliveryUrl')}
                    </label>
                    <div className="flex items-center gap-2">
                      <code className="flex-1 px-3 py-2 text-2xs rounded-lg bg-sunken border border-line text-ink-2 overflow-x-auto whitespace-nowrap">
                        {webhookURL(user?.tenant_id ?? '')}
                      </code>
                      <button
                        type="button"
                        onClick={() => handleCopy('url', webhookURL(user?.tenant_id ?? ''))}
                        className="px-2.5 py-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors shrink-0"
                      >
                        {copiedField === 'url' ? t('myparcel.copied') : t('myparcel.copy')}
                      </button>
                    </div>
                  </div>

                  {webhookSecret ? (
                    <div>
                      <label className="block text-2xs text-ink-4 caps-label mb-1.5">
                        {t('myparcel.secretOnce')}
                      </label>
                      <div className="flex items-center gap-2">
                        <code className="flex-1 px-3 py-2 text-2xs font-mono rounded-lg bg-sunken border border-line text-ink-2 overflow-x-auto whitespace-nowrap">
                          {webhookSecret}
                        </code>
                        <button
                          type="button"
                          onClick={() => handleCopy('secret', webhookSecret)}
                          className="px-2.5 py-2 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors shrink-0"
                        >
                          {copiedField === 'secret' ? t('myparcel.copied') : t('myparcel.copy')}
                        </button>
                      </div>
                      <p className="mt-1.5 text-2xs text-warning">
                        {t('myparcel.secretWarning')}
                      </p>
                    </div>
                  ) : (
                    <p className="text-xs text-ink-4">
                      {t('myparcel.noSecret')}
                    </p>
                  )}

                  {webhookSecretError && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{webhookSecretError}</p>}

                  <button
                    type="button"
                    onClick={handleGenerateWebhookSecret}
                    disabled={generatingSecret}
                    className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
                  >
                    {generatingSecret ? t('myparcel.generating') : webhookSecret ? t('myparcel.regenerate') : t('myparcel.generate')}
                  </button>
                </div>

                <div className="mt-3">
                  <ProviderWebhooksPanel key={webhooksPanelKey} token={token ?? ''} connectionId={connectionId} />
                </div>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  )
}
