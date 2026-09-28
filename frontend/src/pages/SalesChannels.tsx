import { useState } from 'react'
import { Radio, X, Code2, Paintbrush } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useSalesChannelRoute } from '../integrations/hooks/useSalesChannelRoute'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import { useInitOnce } from '../routing/useInitOnce'
import { useSalesChannels } from '../integrations/hooks/useSalesChannels'
import { useConnections } from '../integrations/hooks/useConnections'
import { useCategories } from '../pms/hooks/useCategories'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import * as integrationsApi from '../integrations/api'
import type { SalesChannel, EnrollmentMode, ChannelType } from '../integrations/types'
import type { SalesChannelPayload, UpdateSalesChannelPayload } from '../integrations/api'
import { useExchangeRates } from '../fx/hooks/useExchangeRates'
import { CURRENCY_OPTIONS } from '../settings/currencyOptions'
import { Dialog } from '../components/common/Dialog'
import { CONNECTION_REQUIRED_TYPES, ENROLLMENT_LABELS, HEADLESS_API_ENDPOINTS, INTERNAL_TYPE, PAYMENT_CAPABLE_TYPES, TYPE_LABELS, WAREHOUSE_REQUIRED_TYPES, apiBaseURL, capitalize } from '../integrations/salesChannels/salesChannelConstants'
import { ChannelFormState, emptyForm, toForm } from '../integrations/salesChannels/channelForm'
import StorefrontModeField from '../integrations/salesChannels/StorefrontModeField'
import { ThemesPanel } from '../themes/ThemesPanel'
import { ChannelProductsPanel } from '../integrations/salesChannels/ChannelProductsPanel'

export default function SalesChannels() {
  const { token } = useAuth()
  const { channels, loading, error, create, update, remove } = useSalesChannels(token ?? '')
  const { connections, error: connectionsError, refresh: refreshConnections } = useConnections(token ?? '')
  const { categories } = useCategories(token ?? '')
  const { warehouses } = useWarehouses(token ?? '')
  const { rates: exchangeRates } = useExchangeRates(token ?? '')

  // Which view is open (manager, create/edit form, API docs) comes from the
  // URL -- see useSalesChannelRoute -- so it survives a refresh and Back.
  const route = useSalesChannelRoute()
  const { view } = route
  const routeChannelId = 'channelId' in view ? view.channelId : null
  const routeChannel = routeChannelId ? channels.find(c => c.id === routeChannelId) ?? null : null
  const managingChannel = view.kind === 'manage' ? routeChannel : null
  const themesChannel = view.kind === 'themes' ? routeChannel : null
  const editingChannel = view.kind === 'edit' ? routeChannel : null
  const creating = view.kind === 'create'
  const apiDocsChannel = view.kind === 'api' ? routeChannel : null

  const [form, setForm] = useState<ChannelFormState>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  const [deletingChannel, setDeletingChannel] = useState<SalesChannel | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState('')

  const [copiedAuthHeader, setCopiedAuthHeader] = useState(false)
  async function copyAuthHeader(channelId: string) {
    try {
      await navigator.clipboard.writeText(`Authorization: Bearer ${channelId}`)
      setCopiedAuthHeader(true)
      setTimeout(() => setCopiedAuthHeader(false), 1500)
    } catch {
      // Clipboard access can be denied (permissions, insecure context) --
      // the header is already shown selectable on screen, so this is a
      // convenience failure, not a blocker; no error surfaced for it.
    }
  }

  const openCreate = route.openCreate
  const openEdit = (channel: SalesChannel) => route.openEdit(channel.id)
  const closeForm = route.close

  // Fill the form once per opened form URL -- also on a refresh, where the
  // channel and connections arrive after the first render. An edit waits
  // for the channel's connection to be loaded (or for the load to fail):
  // toForm reads the connection's default location, and saving a form
  // built without it would clear that location on the connection.
  const formKey = creating ? 'new' : editingChannel ? `edit:${editingChannel.id}` : null
  const editReady = !!editingChannel && (
    !editingChannel.connection_id || connections.some(c => c.id === editingChannel.connection_id) || !!connectionsError
  )
  const formReady = useInitOnce(formKey, creating || editReady, () => {
    setForm(editingChannel ? toForm(editingChannel, connections) : emptyForm())
    setFormError('')
  })

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      const channelType: ChannelType = editingChannel ? editingChannel.type : form.channelType
      const shared = {
        name: form.name,
        connectionId: form.channelType === INTERNAL_TYPE ? null : form.connectionId || null,
        enrollmentMode: form.enrollmentMode,
        enrollmentCategoryId: form.enrollmentMode === 'category' ? form.enrollmentCategoryId || null : null,
        enrollmentTag: form.enrollmentMode === 'tag' ? form.enrollmentTag || null : null,
        // Discount/quantity tiers are edited from the Products tab now
        // (see ChannelProductsPanel's PricingSection), not this form --
        // an edit here must still resend the channel's current values
        // unchanged (this is a full-state update, not a sparse patch;
        // sending null/[] would silently wipe them). A brand-new channel
        // has neither yet.
        priceAdjustmentBasisPoints: editingChannel ? (editingChannel.price_adjustment_basis_points ?? null) : null,
        quantityBreaks: editingChannel ? (editingChannel.quantity_breaks ?? []) : [],
        // Only sent for the types that actually need it (see
        // WAREHOUSE_REQUIRED_TYPES's doc comment) -- omitted (not null) for
        // every other type, since it has no meaning there.
        defaultWarehouseId: WAREHOUSE_REQUIRED_TYPES.includes(channelType) ? (form.channelWarehouseId || null) : undefined,
        storefrontMode: channelType === INTERNAL_TYPE ? form.storefrontMode : undefined,
        signInUrl: channelType === INTERNAL_TYPE ? form.signInUrl.trim() : undefined,
      }
      if (editingChannel) {
        await update(editingChannel.id, { ...shared, active: form.active } satisfies UpdateSalesChannelPayload)
      } else {
        await create({ ...shared, type: channelType, currencyCode: form.currencyCode || undefined } satisfies SalesChannelPayload)
      }
      if (shared.connectionId) {
        const connection = connections.find(c => c.id === shared.connectionId)
        await integrationsApi.updateConnectionSettings(token ?? '', shared.connectionId, connection?.default_currency ?? '', form.defaultWarehouseId || null)
        refreshConnections()
      }
      if (editingChannel && PAYMENT_CAPABLE_TYPES.includes(channelType)) {
        await integrationsApi.setSalesChannelPaymentMethod(token ?? '', editingChannel.id, form.paymentConnectionId || null)
      }
      closeForm()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save channel')
    } finally {
      setSaving(false)
    }
  }

  const handleDelete = async () => {
    if (!deletingChannel) return
    setDeleteError('')
    setDeleting(true)
    try {
      await remove(deletingChannel.id)
      setDeletingChannel(null)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Failed to delete channel')
    } finally {
      setDeleting(false)
    }
  }

  const connectionLabel = (connectionId?: string | null) => {
    const conn = connections.find(c => c.id === connectionId)
    return conn ? `${capitalize(conn.provider)} — ${conn.store_url}` : 'No connection'
  }

  if (routeChannelId && !routeChannel) {
    return loading ? <RecordLoading /> : <RecordNotFound label="Sales channel" onBack={route.close} />
  }

  if (themesChannel) {
    return <ThemesPanel channel={themesChannel} onClose={route.close} />
  }

  if (managingChannel) {
    return (
      <ChannelProductsPanel
        channel={managingChannel}
        onClose={route.close}
        update={update}
        categories={categories}
      />
    )
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden p-5 gap-5">
      {channels.length === 0 && !loading ? (
        <div className="flex-1 flex flex-col items-center justify-center text-center">
          <div className="w-12 h-12 rounded-xl bg-raised border border-line flex items-center justify-center mb-3">
            <Radio size={20} className="text-ink-4" />
          </div>
          <p className="text-sm font-medium text-ink-2 mb-1">No sales channels yet</p>
          <p className="text-xs text-ink-4 mb-4">Add a channel to control which products sync where</p>
          <button
            onClick={openCreate}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
          >
            Add sales channel
          </button>
        </div>
      ) : (
        <>
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold text-ink">Sales channels</h2>
            <button
              onClick={openCreate}
              className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
            >
              Add sales channel
            </button>
          </div>
          {error && <p className="text-xs text-danger">{error}</p>}
          <div className="grid grid-cols-2 lg:grid-cols-3 gap-3">
            {channels.map(channel => (
              <div
                key={channel.id}
                className="text-left p-4 rounded-xl border border-line bg-raised"
              >
                <div className="flex items-start justify-between mb-3">
                  <p className="text-sm font-semibold text-ink">{channel.name}</p>
                  <span className={clsx('w-2 h-2 rounded-full mt-1.5', channel.active ? 'bg-green-400' : 'bg-gray-500')} />
                </div>
                <p className="text-2xs text-ink-4 truncate">{connectionLabel(channel.connection_id)}</p>
                <div className="mt-2 flex flex-wrap gap-1.5">
                  <span className="inline-block text-2xs px-2 py-0.5 rounded-full bg-fill text-ink-3">
                    {ENROLLMENT_LABELS[channel.enrollment_mode]}
                  </span>
                  {!!channel.price_adjustment_basis_points && (
                    <span className="inline-block text-2xs px-2 py-0.5 rounded-full bg-amber-500/10 text-warning">
                      {(channel.price_adjustment_basis_points / 100).toString()}% off
                    </span>
                  )}
                </div>
                {channel.type === INTERNAL_TYPE && (
                  <div className="mt-2 flex items-center gap-3">
                    <button
                      onClick={() => route.openApiDocs(channel.id)}
                      className="flex items-center gap-1 text-2xs text-accent hover:text-accent transition-colors"
                    >
                      <Code2 size={11} /> API docs
                    </button>
                    <button
                      onClick={() => route.openThemes(channel.id)}
                      className="flex items-center gap-1 text-2xs text-accent hover:text-accent transition-colors"
                    >
                      <Paintbrush size={11} /> Themes
                    </button>
                  </div>
                )}
                <div className="mt-3 grid grid-cols-3 gap-2">
                  <button
                    onClick={() => route.openManage(channel.id)}
                    className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                  >
                    Products
                  </button>
                  <button
                    onClick={() => openEdit(channel)}
                    className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-accent transition-colors"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => setDeletingChannel(channel)}
                    className="py-1.5 rounded-lg bg-fill text-2xs text-ink-3 hover:text-danger transition-colors"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      {formReady && (
        <Dialog onClose={closeForm} className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={closeForm} />
          <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold text-ink">
                {editingChannel ? 'Edit sales channel' : 'Add sales channel'}
              </h3>
              <button onClick={closeForm} aria-label="Close">
                <X size={14} className="text-ink-4" />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-3">
              <div>
                <label htmlFor="channel-name" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  Name
                </label>
                <input
                  id="channel-name"
                  type="text"
                  required
                  autoFocus
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                />
              </div>
              {!editingChannel && (
                <div>
                  <label htmlFor="channel-type" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Type
                  </label>
                  <select
                    id="channel-type"
                    value={form.channelType}
                    onChange={e => setForm(f => ({ ...f, channelType: e.target.value as ChannelType, connectionId: '', defaultWarehouseId: '', channelWarehouseId: '' }))}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  >
                    {(Object.keys(TYPE_LABELS) as ChannelType[]).map(t => (
                      <option key={t} value={t}>{TYPE_LABELS[t]}</option>
                    ))}
                  </select>
                  {form.channelType === INTERNAL_TYPE && (
                    <p className="mt-1.5 text-2xs text-ink-4">
                      A storefront CommerceOS itself hosts checkout for — no external connection needed.
                    </p>
                  )}
                </div>
              )}
              {!editingChannel && form.channelType === INTERNAL_TYPE && (
                <div>
                  <label htmlFor="channel-currency" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Currency
                  </label>
                  <select
                    id="channel-currency"
                    value={form.currencyCode}
                    onChange={e => setForm(f => ({ ...f, currencyCode: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  >
                    <option value="">Default (tenant's own currency)</option>
                    {CURRENCY_OPTIONS.map(c => <option key={c.value} value={c.value}>{c.label}</option>)}
                  </select>
                  <p className="mt-1.5 text-2xs text-ink-4">
                    Fixed for this channel once created.
                    {form.currencyCode && !exchangeRates.some(r => r.currency_code === form.currencyCode)
                      ? ` No exchange rate configured for ${form.currencyCode} yet -- add one under Settings > Localization, or prices won't convert.`
                      : " Prices convert via Settings > Localization's exchange rate unless a product has a manual override for it."}
                  </p>
                </div>
              )}
              {(editingChannel ? CONNECTION_REQUIRED_TYPES.includes(editingChannel.type) : form.channelType !== INTERNAL_TYPE) && (
                <div>
                  <label htmlFor="channel-connection" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Connection
                  </label>
                  <select
                    id="channel-connection"
                    required
                    value={form.connectionId}
                    onChange={e => {
                      const connectionId = e.target.value
                      const defaultWarehouseId = connections.find(c => c.id === connectionId)?.default_warehouse_id ?? ''
                      setForm(f => ({ ...f, connectionId, defaultWarehouseId }))
                    }}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  >
                    <option value="" disabled>Select a connection…</option>
                    {connections
                      .filter(c => c.provider === (editingChannel ? editingChannel.type : form.channelType))
                      .map(c => <option key={c.id} value={c.id}>{capitalize(c.provider)} — {c.store_url}</option>)}
                  </select>
                </div>
              )}
              {form.connectionId && (
                <div>
                  <label htmlFor="channel-default-warehouse" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Default location
                  </label>
                  <select
                    id="channel-default-warehouse"
                    value={form.defaultWarehouseId}
                    onChange={e => setForm(f => ({ ...f, defaultWarehouseId: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  >
                    <option value="">No preference</option>
                    {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                  </select>
                  <p className="mt-1.5 text-2xs text-ink-4">
                    Which warehouse this channel's orders and stock sync are attributed to.
                  </p>
                </div>
              )}
              {WAREHOUSE_REQUIRED_TYPES.includes(editingChannel ? editingChannel.type : form.channelType) && (
                <div>
                  <label htmlFor="channel-fulfillment-warehouse" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Fulfillment warehouse
                  </label>
                  <select
                    id="channel-fulfillment-warehouse"
                    required
                    value={form.channelWarehouseId}
                    onChange={e => setForm(f => ({ ...f, channelWarehouseId: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  >
                    <option value="" disabled>Select a warehouse…</option>
                    {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                  </select>
                  <p className="mt-1.5 text-2xs text-ink-4">
                    Required -- every order placed through this channel decrements stock from and ships from this warehouse.
                  </p>
                </div>
              )}
              {(editingChannel ? editingChannel.type : form.channelType) === INTERNAL_TYPE && (
                <StorefrontModeField
                  value={form.storefrontMode} onChange={storefrontMode => setForm(f => ({ ...f, storefrontMode }))}
                  signInUrl={form.signInUrl} onSignInUrlChange={signInUrl => setForm(f => ({ ...f, signInUrl }))}
                />
              )}
              <div>
                <label htmlFor="channel-enrollment-mode" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                  Default enrollment
                </label>
                <select
                  id="channel-enrollment-mode"
                  value={form.enrollmentMode}
                  onChange={e => setForm(f => ({ ...f, enrollmentMode: e.target.value as EnrollmentMode }))}
                  className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                >
                  {(Object.keys(ENROLLMENT_LABELS) as EnrollmentMode[]).map(mode => (
                    <option key={mode} value={mode}>{ENROLLMENT_LABELS[mode]}</option>
                  ))}
                </select>
                <p className="mt-1.5 text-2xs text-ink-4">
                  Which products belong on this channel by default. Individual products can still be added or removed later.
                </p>
              </div>
              {form.enrollmentMode === 'category' && (
                <div>
                  <label htmlFor="channel-enrollment-category" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Category
                  </label>
                  <select
                    id="channel-enrollment-category"
                    required
                    value={form.enrollmentCategoryId}
                    onChange={e => setForm(f => ({ ...f, enrollmentCategoryId: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  >
                    <option value="" disabled>Select a category…</option>
                    {categories.map(cat => <option key={cat.id} value={cat.id}>{cat.name}</option>)}
                  </select>
                </div>
              )}
              {form.enrollmentMode === 'tag' && (
                <div>
                  <label htmlFor="channel-enrollment-tag" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Tag
                  </label>
                  <input
                    id="channel-enrollment-tag"
                    type="text"
                    required
                    value={form.enrollmentTag}
                    onChange={e => setForm(f => ({ ...f, enrollmentTag: e.target.value }))}
                    placeholder="summer"
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  />
                </div>
              )}
              {editingChannel && PAYMENT_CAPABLE_TYPES.includes(editingChannel.type) && (
                <div>
                  <label htmlFor="channel-payment-method" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
                    Payment method
                  </label>
                  <select
                    id="channel-payment-method"
                    value={form.paymentConnectionId}
                    onChange={e => setForm(f => ({ ...f, paymentConnectionId: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
                  >
                    <option value="">Disabled — no payment link sent</option>
                    {connections
                      .filter(c => (c.provider === 'stripe' || c.provider === 'revolut') && c.status === 'connected')
                      .map(c => <option key={c.id} value={c.id}>{capitalize(c.provider)}</option>)}
                  </select>
                  <p className="mt-1.5 text-2xs text-ink-4">
                    Which connected payment integration an order's payment link is generated through. Connect Stripe or Revolut from the Integrations page first.
                  </p>
                </div>
              )}
              {editingChannel && (
                <div className="flex items-center justify-between">
                  <span className="text-2xs text-ink-4 uppercase tracking-wider">Active</span>
                  <label className="relative flex items-center shrink-0 cursor-pointer">
                    <input
                      type="checkbox"
                      role="switch"
                      aria-label="Active"
                      checked={form.active}
                      onChange={e => setForm(f => ({ ...f, active: e.target.checked }))}
                      className="sr-only peer"
                    />
                    <div className="w-8 h-4 dark:bg-night-700 bg-gray-300 rounded-full peer peer-checked:bg-accent-solid transition-colors" />
                    <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                  </label>
                </div>
              )}
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
              <button
                type="submit"
                disabled={saving}
                className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
              >
                {saving ? 'Saving…' : editingChannel ? 'Save changes' : 'Add channel'}
              </button>
            </form>
          </div>
        </Dialog>
      )}

      {deletingChannel && (
        <Dialog onClose={() => setDeletingChannel(null)} className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={() => setDeletingChannel(null)} />
          <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
            <h3 className="text-sm font-semibold text-ink mb-3">Delete sales channel</h3>
            <p className="text-xs text-ink-2 mb-4">
              Delete <strong>{deletingChannel.name}</strong>? Products already synced to it stay listed there — this only removes
              the enrollment rule.
            </p>
            {deleteError && (
              <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5 mb-3">{deleteError}</p>
            )}
            <div className="grid grid-cols-2 gap-2">
              <button
                onClick={() => setDeletingChannel(null)}
                className="py-2 rounded-lg bg-fill text-xs font-medium text-ink-2 transition-colors"
              >
                Cancel
              </button>
              <button
                onClick={handleDelete}
                disabled={deleting}
                className="py-2 rounded-lg bg-red-600 text-white text-xs font-medium hover:bg-red-700 disabled:opacity-50 transition-colors"
              >
                {deleting ? 'Deleting…' : 'Delete'}
              </button>
            </div>
          </div>
        </Dialog>
      )}

      {apiDocsChannel && (
        <Dialog onClose={route.close} className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={route.close} />
          <div className="relative w-full max-w-lg max-h-[85vh] flex flex-col bg-raised rounded-xl shadow-2xl border border-line p-5">
            <div className="flex items-center justify-between mb-1">
              <h3 className="text-sm font-semibold text-ink">{apiDocsChannel.name} — headless API</h3>
              <button onClick={route.close} aria-label="Close">
                <X size={14} className="text-ink-4" />
              </button>
            </div>
            <p className="text-2xs text-ink-4 mb-3">
              No storefront app is attached to this channel -- it's a plain REST API any frontend can call directly.
            </p>
            <div className="shrink-0 mb-3 p-2.5 rounded-lg bg-sunken border border-line space-y-1.5">
              <p className="text-2xs text-ink-4 uppercase tracking-wider">Base URL</p>
              <code className="block text-xs text-ink break-all">{apiBaseURL()}</code>
              <p className="text-2xs text-ink-4 uppercase tracking-wider pt-1">Auth (every request)</p>
              <div className="flex items-center gap-1.5">
                <code className="flex-1 text-xs text-ink break-all">Authorization: Bearer {apiDocsChannel.id}</code>
                <button
                  onClick={() => copyAuthHeader(apiDocsChannel.id)}
                  className="shrink-0 text-2xs text-accent hover:text-accent transition-colors"
                >
                  {copiedAuthHeader ? 'Copied' : 'Copy'}
                </button>
              </div>
              <p className="text-2xs text-ink-4">
                Public/anonymous-shopper privilege only -- this channel's own id doubles as its bearer key, same trust
                model as a Stripe publishable key.
              </p>
            </div>
            <div className="flex-1 overflow-y-auto -mx-5 px-5">
              {HEADLESS_API_ENDPOINTS.map(ep => (
                <div key={`${ep.method} ${ep.path}`} className="flex items-start gap-2 py-1.5 border-b border-line last:border-0">
                  <span
                    className={clsx(
                      'shrink-0 w-14 text-center text-2xs font-bold uppercase tracking-wide rounded px-1 py-0.5',
                      ep.method === 'GET' && 'bg-accent/10 text-accent',
                      ep.method === 'POST' && 'bg-green-500/10 text-success',
                      ep.method === 'PUT' && 'bg-amber-500/10 text-warning',
                      ep.method === 'DELETE' && 'bg-red-500/10 text-danger',
                    )}
                  >
                    {ep.method}
                  </span>
                  <div className="min-w-0">
                    <code className="text-2xs text-ink-2 break-all">{ep.path}</code>
                    <p className="text-2xs text-ink-4">{ep.summary}</p>
                  </div>
                </div>
              ))}
            </div>
            <p className="shrink-0 pt-3 text-2xs text-ink-4">
              Full request/response contracts: backend/api/storefront/openapi.yaml.
            </p>
          </div>
        </Dialog>
      )}
    </div>
  )
}
