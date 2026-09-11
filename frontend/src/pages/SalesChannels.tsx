import { useState } from 'react'
import { ArrowLeft, Radio, X, Search, Code2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useSalesChannels } from '../integrations/hooks/useSalesChannels'
import { useSalesChannelProducts } from '../integrations/hooks/useSalesChannelProducts'
import { useConnections } from '../integrations/hooks/useConnections'
import { useCategories } from '../pms/hooks/useCategories'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import * as integrationsApi from '../integrations/api'
import type { Connection, SalesChannel, EnrollmentMode, ChannelType, QuantityBreak } from '../integrations/types'
import type { SalesChannelPayload, UpdateSalesChannelPayload } from '../integrations/api'
import Pagination from '../components/common/Pagination'
import { useExchangeRates } from '../fx/hooks/useExchangeRates'
import { CURRENCY_OPTIONS } from '../settings/currencyOptions'

const PER_PAGE = 20

const ENROLLMENT_LABELS: Record<EnrollmentMode, string> = {
  all: 'All products',
  category: 'By category',
  tag: 'By tag',
  manual: 'Manual only',
}

interface ChannelFormState {
  name: string
  channelType: ChannelType
  connectionId: string
  enrollmentMode: EnrollmentMode
  enrollmentCategoryId: string
  enrollmentTag: string
  active: boolean
  defaultWarehouseId: string
  // channelWarehouseId is SalesChannel.default_warehouse_id itself --
  // distinct from defaultWarehouseId above (Connection.default_warehouse_id,
  // which only routes a WooCommerce/Bol import's stock sync). See
  // WAREHOUSE_REQUIRED_TYPES's doc comment for which channel types need it.
  channelWarehouseId: string
  paymentConnectionId: string
  // "" means "use the tenant's default currency" -- only settable at
  // creation, see SalesChannelPayload.currencyCode's own doc comment.
  currencyCode: string
}

function emptyForm(): ChannelFormState {
  return {
    name: '', channelType: 'woocommerce', connectionId: '', enrollmentMode: 'all', enrollmentCategoryId: '',
    enrollmentTag: '', active: true, defaultWarehouseId: '', channelWarehouseId: '', paymentConnectionId: '',
    currencyCode: '',
  }
}

// priceAdjustmentBasisPointsOrError converts the form's percent string to
// basis points (e.g. "20" -> 2000), or throws a message-carrying Error for
// an out-of-range value -- caught by handleSubmit the same way every other
// save error is.
function priceAdjustmentBasisPointsOrError(percent: string): number | null {
  if (percent.trim() === '') return null
  const value = Math.round(parseFloat(percent) * 100)
  if (!Number.isFinite(value) || value < 0 || value > 10000) {
    throw new Error('Discount must be between 0 and 100%.')
  }
  return value
}

// quantityBreaksOrError mirrors priceAdjustmentBasisPointsOrError's
// convert-and-validate-at-submit convention for the repeatable tier rows
// -- a positive integer minimum quantity, a 0-100% discount, and no two
// rows sharing the same minimum (which one would even apply is undefined
// otherwise, same rule the backend enforces).
function quantityBreaksOrError(rows: { minQuantity: string; discountPercent: string }[]): QuantityBreak[] {
  const seen = new Set<number>()
  return rows.map(row => {
    const minQuantity = parseInt(row.minQuantity, 10)
    if (!Number.isFinite(minQuantity) || minQuantity <= 0) {
      throw new Error('Each quantity tier needs a minimum quantity of at least 1.')
    }
    if (seen.has(minQuantity)) {
      throw new Error(`Two quantity tiers both start at ${minQuantity} -- each minimum quantity must be unique.`)
    }
    seen.add(minQuantity)
    const discountBasisPoints = Math.round(parseFloat(row.discountPercent) * 100)
    if (!Number.isFinite(discountBasisPoints) || discountBasisPoints < 0 || discountBasisPoints > 10000) {
      throw new Error("Each quantity tier's discount must be between 0 and 100%.")
    }
    return { min_quantity: minQuantity, discount_basis_points: discountBasisPoints }
  })
}

function capitalize(s: string): string {
  return s.length === 0 ? s : s[0].toUpperCase() + s.slice(1)
}

const TYPE_LABELS: Record<ChannelType, string> = {
  woocommerce: 'WooCommerce',
  bol: 'Bol.com',
  whatsapp: 'WhatsApp',
  internal: 'Internal store',
}

// A built-in storefront has nothing external to connect to (see backend
// domain/channels.ChannelType.RequiresConnection) -- its own form fields
// never render, and it's the one type able to configure a payment method
// directly (WhatsApp is the other, for its chat checkout).
const INTERNAL_TYPE: ChannelType = 'internal'

// HEADLESS_API_ENDPOINTS is a reference snapshot of an internal channel's
// real commerce API -- backend/api/storefront/openapi.yaml is the actual
// source of truth (used for real request validation/codegen); this is
// just a legible summary for a developer wiring a headless frontend to
// this channel, not a substitute for that spec. Excludes /theme/* (the
// old Puck-editor-only routes -- irrelevant to a headless integration,
// see the "remove Freeform" commit). Keep in sync with openapi.yaml by
// hand if a storefront route is added/changed/removed.
const HEADLESS_API_ENDPOINTS: { method: string; path: string; summary: string }[] = [
  { method: 'GET', path: '/products', summary: "List the calling channel's enrolled catalog" },
  { method: 'GET', path: '/products/{id}', summary: "Fetch one enrolled product's full variant/SKU/image detail" },
  { method: 'GET', path: '/products/{id}/recommendations', summary: 'Products historically bought alongside this one' },
  { method: 'GET', path: '/products/{id}/reviews', summary: "List a product's reviews, with a rating summary" },
  { method: 'POST', path: '/products/{id}/reviews', summary: 'Submit a review for a product' },
  { method: 'GET', path: '/categories', summary: "The tenant's whole category tree, flat with parent_id" },
  { method: 'POST', path: '/recommendations/cart', summary: "Cross-sell candidates for the shopper's whole cart" },
  { method: 'POST', path: '/carts', summary: "Start a new guest cart on the calling key's sales channel" },
  { method: 'GET', path: '/carts/{token}', summary: "Fetch a cart's current line items and subtotal" },
  { method: 'PUT', path: '/carts/{token}/items', summary: "Replace a cart's whole line set" },
  { method: 'PUT', path: '/carts/{token}/coupon', summary: 'Attach a coupon code to the cart, or remove it' },
  { method: 'POST', path: '/carts/{token}/shipping-options', summary: 'Priced delivery options for a shipping address' },
  { method: 'POST', path: '/carts/{token}/payment-intent', summary: "Start an embedded Stripe Elements card payment for this cart's total" },
  { method: 'POST', path: '/carts/{token}/checkout', summary: "Place an order for a cart's current contents" },
  { method: 'POST', path: '/wishlists', summary: 'Start a new, empty wishlist on the calling channel' },
  { method: 'GET', path: '/wishlists/{token}', summary: 'Fetch a wishlist by its token' },
  { method: 'PUT', path: '/wishlists/{token}/items', summary: 'Add product_id to the wishlist, or remove it' },
  { method: 'POST', path: '/auth/magic-link', summary: 'Email a one-time sign-in link to the given address' },
  { method: 'POST', path: '/auth/session', summary: 'Exchange a magic-link token for a signed-in session' },
  { method: 'GET', path: '/account/{token}', summary: "Fetch the signed-in shopper's profile" },
  { method: 'DELETE', path: '/account/{token}', summary: 'End the session' },
  { method: 'GET', path: '/account/{token}/orders', summary: "The signed-in shopper's order history" },
  { method: 'GET', path: '/orders/lookup', summary: 'Guest order-status lookup by order number + billing email' },
]

function apiBaseURL(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/storefront/v1`
}

const CONNECTION_REQUIRED_TYPES: ChannelType[] = (Object.keys(TYPE_LABELS) as ChannelType[]).filter(t => t !== INTERNAL_TYPE)
const PAYMENT_CAPABLE_TYPES: ChannelType[] = ['whatsapp', INTERNAL_TYPE]
// The same two types create a SalesOrder directly (Woo/Bol only ever
// import one that already has a warehouse from the source system) -- so
// they're also the only ones that need SalesChannel.default_warehouse_id
// configured before checkout can succeed. Same set as PAYMENT_CAPABLE_TYPES
// today because both concerns track "does this channel sell directly," but
// kept as its own name since that's incidental, not a rule.
const WAREHOUSE_REQUIRED_TYPES: ChannelType[] = PAYMENT_CAPABLE_TYPES

// defaultWarehouseId lives on the linked Connection, not on the channel
// itself (SalesChannel is purely a product-enrollment rule on top of it --
// see its doc comment) -- so prefilling it means looking the connection up
// by the channel's connection_id.
function toForm(channel: SalesChannel, connections: Connection[]): ChannelFormState {
  const connection = connections.find(c => c.id === channel.connection_id)
  return {
    name: channel.name,
    channelType: channel.type,
    connectionId: channel.connection_id ?? '',
    enrollmentMode: channel.enrollment_mode,
    enrollmentCategoryId: channel.enrollment_category_id ?? '',
    enrollmentTag: channel.enrollment_tag ?? '',
    active: channel.active,
    defaultWarehouseId: connection?.default_warehouse_id ?? '',
    channelWarehouseId: channel.default_warehouse_id ?? '',
    paymentConnectionId: channel.payment_connection_id ?? '',
    currencyCode: channel.currency_code ?? '',
  }
}

export default function SalesChannels() {
  const { token } = useAuth()
  const { channels, loading, error, create, update, remove } = useSalesChannels(token ?? '')
  const { connections, refresh: refreshConnections } = useConnections(token ?? '')
  const { categories } = useCategories(token ?? '')
  const { warehouses } = useWarehouses(token ?? '')
  const { rates: exchangeRates } = useExchangeRates(token ?? '')

  const [managingChannel, setManagingChannel] = useState<SalesChannel | null>(null)

  const [editingChannel, setEditingChannel] = useState<SalesChannel | null>(null)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState<ChannelFormState>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  const [deletingChannel, setDeletingChannel] = useState<SalesChannel | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState('')

  const [apiDocsChannel, setApiDocsChannel] = useState<SalesChannel | null>(null)
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

  const openCreate = () => {
    setForm(emptyForm())
    setFormError('')
    setCreating(true)
  }

  const openEdit = (channel: SalesChannel) => {
    setForm(toForm(channel, connections))
    setFormError('')
    setEditingChannel(channel)
  }

  const closeForm = () => {
    setCreating(false)
    setEditingChannel(null)
    setFormError('')
  }

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

  if (managingChannel) {
    return (
      <ChannelProductsPanel
        channel={managingChannel}
        onChannelUpdated={setManagingChannel}
        onClose={() => setManagingChannel(null)}
        update={update}
        categories={categories}
      />
    )
  }

  return (
    <div className="flex-1 flex flex-col overflow-hidden p-5 gap-5">
      {channels.length === 0 && !loading ? (
        <div className="flex-1 flex flex-col items-center justify-center text-center">
          <div className="w-12 h-12 rounded-xl dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 flex items-center justify-center mb-3">
            <Radio size={20} className="dark:text-gray-500 text-gray-400" />
          </div>
          <p className="text-sm font-medium dark:text-gray-300 text-gray-700 mb-1">No sales channels yet</p>
          <p className="text-xs dark:text-gray-500 text-gray-400 mb-4">Add a channel to control which products sync where</p>
          <button
            onClick={openCreate}
            className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
          >
            Add sales channel
          </button>
        </div>
      ) : (
        <>
          <div className="flex items-center justify-between">
            <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Sales channels</h2>
            <button
              onClick={openCreate}
              className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
            >
              Add sales channel
            </button>
          </div>
          {error && <p className="text-xs text-red-400">{error}</p>}
          <div className="grid grid-cols-2 lg:grid-cols-3 gap-3">
            {channels.map(channel => (
              <div
                key={channel.id}
                className="text-left p-4 rounded-xl border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white"
              >
                <div className="flex items-start justify-between mb-3">
                  <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{channel.name}</p>
                  <span className={clsx('w-2 h-2 rounded-full mt-1.5', channel.active ? 'bg-green-400' : 'bg-gray-500')} />
                </div>
                <p className="text-[10px] dark:text-gray-500 text-gray-400 truncate">{connectionLabel(channel.connection_id)}</p>
                <div className="mt-2 flex flex-wrap gap-1.5">
                  <span className="inline-block text-[10px] px-2 py-0.5 rounded-full dark:bg-[#111827] bg-gray-100 dark:text-gray-400 text-gray-500">
                    {ENROLLMENT_LABELS[channel.enrollment_mode]}
                  </span>
                  {!!channel.price_adjustment_basis_points && (
                    <span className="inline-block text-[10px] px-2 py-0.5 rounded-full bg-amber-500/10 text-amber-500">
                      {(channel.price_adjustment_basis_points / 100).toString()}% off
                    </span>
                  )}
                </div>
                {channel.type === INTERNAL_TYPE && (
                  <button
                    onClick={() => setApiDocsChannel(channel)}
                    className="mt-2 flex items-center gap-1 text-[10px] text-blue-500 hover:text-blue-400 transition-colors"
                  >
                    <Code2 size={11} /> API docs
                  </button>
                )}
                <div className="mt-3 grid grid-cols-3 gap-2">
                  <button
                    onClick={() => setManagingChannel(channel)}
                    className="py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 transition-colors"
                  >
                    Products
                  </button>
                  <button
                    onClick={() => openEdit(channel)}
                    className="py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 transition-colors"
                  >
                    Edit
                  </button>
                  <button
                    onClick={() => setDeletingChannel(channel)}
                    className="py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-red-400 transition-colors"
                  >
                    Delete
                  </button>
                </div>
              </div>
            ))}
          </div>
        </>
      )}

      {(creating || editingChannel) && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={closeForm} />
          <div className="relative w-full max-w-sm dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
            <div className="flex items-center justify-between mb-4">
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">
                {editingChannel ? 'Edit sales channel' : 'Add sales channel'}
              </h3>
              <button onClick={closeForm} aria-label="Close">
                <X size={14} className="dark:text-gray-500 text-gray-400" />
              </button>
            </div>
            <form onSubmit={handleSubmit} className="space-y-3">
              <div>
                <label htmlFor="channel-name" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  Name
                </label>
                <input
                  id="channel-name"
                  type="text"
                  required
                  autoFocus
                  value={form.name}
                  onChange={e => setForm(f => ({ ...f, name: e.target.value }))}
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                />
              </div>
              {!editingChannel && (
                <div>
                  <label htmlFor="channel-type" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Type
                  </label>
                  <select
                    id="channel-type"
                    value={form.channelType}
                    onChange={e => setForm(f => ({ ...f, channelType: e.target.value as ChannelType, connectionId: '', defaultWarehouseId: '', channelWarehouseId: '' }))}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    {(Object.keys(TYPE_LABELS) as ChannelType[]).map(t => (
                      <option key={t} value={t}>{TYPE_LABELS[t]}</option>
                    ))}
                  </select>
                  {form.channelType === INTERNAL_TYPE && (
                    <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                      A storefront CommerceOS itself hosts checkout for — no external connection needed.
                    </p>
                  )}
                </div>
              )}
              {!editingChannel && form.channelType === INTERNAL_TYPE && (
                <div>
                  <label htmlFor="channel-currency" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Currency
                  </label>
                  <select
                    id="channel-currency"
                    value={form.currencyCode}
                    onChange={e => setForm(f => ({ ...f, currencyCode: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    <option value="">Default (tenant's own currency)</option>
                    {CURRENCY_OPTIONS.map(c => <option key={c.value} value={c.value}>{c.label}</option>)}
                  </select>
                  <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                    Fixed for this channel once created.
                    {form.currencyCode && !exchangeRates.some(r => r.currency_code === form.currencyCode)
                      ? ` No exchange rate configured for ${form.currencyCode} yet -- add one under Settings > Localization, or prices won't convert.`
                      : " Prices convert via Settings > Localization's exchange rate unless a product has a manual override for it."}
                  </p>
                </div>
              )}
              {(editingChannel ? CONNECTION_REQUIRED_TYPES.includes(editingChannel.type) : form.channelType !== INTERNAL_TYPE) && (
                <div>
                  <label htmlFor="channel-connection" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
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
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
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
                  <label htmlFor="channel-default-warehouse" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Default location
                  </label>
                  <select
                    id="channel-default-warehouse"
                    value={form.defaultWarehouseId}
                    onChange={e => setForm(f => ({ ...f, defaultWarehouseId: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    <option value="">No preference</option>
                    {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                  </select>
                  <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                    Which warehouse this channel's orders and stock sync are attributed to.
                  </p>
                </div>
              )}
              {WAREHOUSE_REQUIRED_TYPES.includes(editingChannel ? editingChannel.type : form.channelType) && (
                <div>
                  <label htmlFor="channel-fulfillment-warehouse" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Fulfillment warehouse
                  </label>
                  <select
                    id="channel-fulfillment-warehouse"
                    required
                    value={form.channelWarehouseId}
                    onChange={e => setForm(f => ({ ...f, channelWarehouseId: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    <option value="" disabled>Select a warehouse…</option>
                    {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                  </select>
                  <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                    Required -- every order placed through this channel decrements stock from and ships from this warehouse.
                  </p>
                </div>
              )}
              <div>
                <label htmlFor="channel-enrollment-mode" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                  Default enrollment
                </label>
                <select
                  id="channel-enrollment-mode"
                  value={form.enrollmentMode}
                  onChange={e => setForm(f => ({ ...f, enrollmentMode: e.target.value as EnrollmentMode }))}
                  className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                >
                  {(Object.keys(ENROLLMENT_LABELS) as EnrollmentMode[]).map(mode => (
                    <option key={mode} value={mode}>{ENROLLMENT_LABELS[mode]}</option>
                  ))}
                </select>
                <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                  Which products belong on this channel by default. Individual products can still be added or removed later.
                </p>
              </div>
              {form.enrollmentMode === 'category' && (
                <div>
                  <label htmlFor="channel-enrollment-category" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Category
                  </label>
                  <select
                    id="channel-enrollment-category"
                    required
                    value={form.enrollmentCategoryId}
                    onChange={e => setForm(f => ({ ...f, enrollmentCategoryId: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    <option value="" disabled>Select a category…</option>
                    {categories.map(cat => <option key={cat.id} value={cat.id}>{cat.name}</option>)}
                  </select>
                </div>
              )}
              {form.enrollmentMode === 'tag' && (
                <div>
                  <label htmlFor="channel-enrollment-tag" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Tag
                  </label>
                  <input
                    id="channel-enrollment-tag"
                    type="text"
                    required
                    value={form.enrollmentTag}
                    onChange={e => setForm(f => ({ ...f, enrollmentTag: e.target.value }))}
                    placeholder="summer"
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                </div>
              )}
              {editingChannel && PAYMENT_CAPABLE_TYPES.includes(editingChannel.type) && (
                <div>
                  <label htmlFor="channel-payment-method" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Payment method
                  </label>
                  <select
                    id="channel-payment-method"
                    value={form.paymentConnectionId}
                    onChange={e => setForm(f => ({ ...f, paymentConnectionId: e.target.value }))}
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  >
                    <option value="">Disabled — no payment link sent</option>
                    {connections
                      .filter(c => (c.provider === 'stripe' || c.provider === 'revolut') && c.status === 'connected')
                      .map(c => <option key={c.id} value={c.id}>{capitalize(c.provider)}</option>)}
                  </select>
                  <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                    Which connected payment integration an order's payment link is generated through. Connect Stripe or Revolut from the Integrations page first.
                  </p>
                </div>
              )}
              {editingChannel && (
                <div className="flex items-center justify-between">
                  <span className="text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider">Active</span>
                  <label className="relative flex items-center shrink-0 cursor-pointer">
                    <input
                      type="checkbox"
                      role="switch"
                      aria-label="Active"
                      checked={form.active}
                      onChange={e => setForm(f => ({ ...f, active: e.target.checked }))}
                      className="sr-only peer"
                    />
                    <div className="w-8 h-4 dark:bg-[#2D3748] bg-gray-300 rounded-full peer peer-checked:bg-blue-600 transition-colors" />
                    <div className="absolute left-0.5 top-0.5 w-3 h-3 bg-white rounded-full transition-all peer-checked:translate-x-4" />
                  </label>
                </div>
              )}
              {formError && (
                <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
              <button
                type="submit"
                disabled={saving}
                className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
              >
                {saving ? 'Saving…' : editingChannel ? 'Save changes' : 'Add channel'}
              </button>
            </form>
          </div>
        </div>
      )}

      {deletingChannel && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={() => setDeletingChannel(null)} />
          <div className="relative w-full max-w-sm dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-3">Delete sales channel</h3>
            <p className="text-xs dark:text-gray-300 text-gray-700 mb-4">
              Delete <strong>{deletingChannel.name}</strong>? Products already synced to it stay listed there — this only removes
              the enrollment rule.
            </p>
            {deleteError && (
              <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5 mb-3">{deleteError}</p>
            )}
            <div className="grid grid-cols-2 gap-2">
              <button
                onClick={() => setDeletingChannel(null)}
                className="py-2 rounded-lg dark:bg-[#111827] bg-gray-100 text-xs font-medium dark:text-gray-300 text-gray-600 transition-colors"
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
        </div>
      )}

      {apiDocsChannel && (
        <div className="fixed inset-0 z-50 flex items-center justify-center">
          <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={() => setApiDocsChannel(null)} />
          <div className="relative w-full max-w-lg max-h-[85vh] flex flex-col dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
            <div className="flex items-center justify-between mb-1">
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">{apiDocsChannel.name} — headless API</h3>
              <button onClick={() => setApiDocsChannel(null)} aria-label="Close">
                <X size={14} className="dark:text-gray-500 text-gray-400" />
              </button>
            </div>
            <p className="text-[10px] dark:text-gray-500 text-gray-400 mb-3">
              No storefront app is attached to this channel -- it's a plain REST API any frontend can call directly.
            </p>
            <div className="shrink-0 mb-3 p-2.5 rounded-lg dark:bg-[#111827] bg-gray-50 border dark:border-[#2D3748] border-gray-200 space-y-1.5">
              <p className="text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider">Base URL</p>
              <code className="block text-xs dark:text-gray-200 text-gray-800 break-all">{apiBaseURL()}</code>
              <p className="text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider pt-1">Auth (every request)</p>
              <div className="flex items-center gap-1.5">
                <code className="flex-1 text-xs dark:text-gray-200 text-gray-800 break-all">Authorization: Bearer {apiDocsChannel.id}</code>
                <button
                  onClick={() => copyAuthHeader(apiDocsChannel.id)}
                  className="shrink-0 text-[10px] text-blue-500 hover:text-blue-400 transition-colors"
                >
                  {copiedAuthHeader ? 'Copied' : 'Copy'}
                </button>
              </div>
              <p className="text-[10px] dark:text-gray-500 text-gray-400">
                Public/anonymous-shopper privilege only -- this channel's own id doubles as its bearer key, same trust
                model as a Stripe publishable key.
              </p>
            </div>
            <div className="flex-1 overflow-y-auto -mx-5 px-5">
              {HEADLESS_API_ENDPOINTS.map(ep => (
                <div key={`${ep.method} ${ep.path}`} className="flex items-start gap-2 py-1.5 border-b dark:border-[#2D3748] border-gray-100 last:border-0">
                  <span
                    className={clsx(
                      'shrink-0 w-14 text-center text-[9px] font-bold uppercase tracking-wide rounded px-1 py-0.5',
                      ep.method === 'GET' && 'bg-blue-500/10 text-blue-500',
                      ep.method === 'POST' && 'bg-green-500/10 text-green-500',
                      ep.method === 'PUT' && 'bg-amber-500/10 text-amber-500',
                      ep.method === 'DELETE' && 'bg-red-500/10 text-red-500',
                    )}
                  >
                    {ep.method}
                  </span>
                  <div className="min-w-0">
                    <code className="text-[11px] dark:text-gray-300 text-gray-700 break-all">{ep.path}</code>
                    <p className="text-[10px] dark:text-gray-500 text-gray-400">{ep.summary}</p>
                  </div>
                </div>
              ))}
            </div>
            <p className="shrink-0 pt-3 text-[10px] dark:text-gray-500 text-gray-400">
              Full request/response contracts: backend/api/storefront/openapi.yaml.
            </p>
          </div>
        </div>
      )}
    </div>
  )
}

interface PricingSectionProps {
  channel: SalesChannel
  update: (id: string, payload: UpdateSalesChannelPayload) => Promise<SalesChannel>
  onChannelUpdated: (channel: SalesChannel) => void
}

// PricingSection lives on the Products tab, not the create/edit channel
// form -- a discount/quantity tier is a pricing-on-this-catalog concern,
// same place a merchant already manages which products are even on this
// channel. Every save is a full-state update (see UpdateSalesChannelPayload
// itself, and applyRule's identical pattern above) -- name/enrollment/
// active are resent unchanged from the channel prop, only pricing fields
// come from this section's own local state.
function PricingSection({ channel, update, onChannelUpdated }: PricingSectionProps) {
  const [percent, setPercent] = useState(
    channel.price_adjustment_basis_points ? (channel.price_adjustment_basis_points / 100).toString() : '',
  )
  const [quantityBreaks, setQuantityBreaks] = useState(
    (channel.quantity_breaks ?? []).map(b => ({ minQuantity: b.min_quantity.toString(), discountPercent: (b.discount_basis_points / 100).toString() })),
  )
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const handleSave = async () => {
    setError('')
    setSaving(true)
    try {
      const updated = await update(channel.id, {
        name: channel.name,
        connectionId: channel.connection_id,
        enrollmentMode: channel.enrollment_mode,
        enrollmentCategoryId: channel.enrollment_category_id ?? null,
        enrollmentTag: channel.enrollment_tag ?? null,
        priceAdjustmentBasisPoints: priceAdjustmentBasisPointsOrError(percent),
        quantityBreaks: quantityBreaksOrError(quantityBreaks),
        active: channel.active,
      })
      onChannelUpdated(updated)
      setDirty(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save pricing')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="mx-5 mt-4 p-4 rounded-xl border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white shrink-0">
      <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900 mb-3">Pricing</h3>
      <div>
        <label htmlFor="pricing-discount" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
          Discount off list price (%)
        </label>
        <input
          id="pricing-discount"
          type="number" step={0.01}
          value={percent}
          onChange={e => { setPercent(e.target.value); setDirty(true) }}
          placeholder="0"
          className="w-full max-w-xs px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
        />
        <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
          Applied to every product's price on this storefront -- e.g. 20 for a wholesale channel selling at 20% off retail.
        </p>
      </div>
      <div className="mt-4">
        <div className="flex items-center justify-between mb-1.5">
          <label className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider">Quantity tiers</label>
          <button
            type="button"
            onClick={() => { setQuantityBreaks(b => [...b, { minQuantity: '', discountPercent: '' }]); setDirty(true) }}
            className="text-[10px] text-blue-500 hover:text-blue-400"
          >
            + Add tier
          </button>
        </div>
        {quantityBreaks.length === 0 && (
          <p className="text-[10px] dark:text-gray-500 text-gray-400">No quantity tiers -- every order uses the flat discount above.</p>
        )}
        <div className="space-y-2">
          {quantityBreaks.map((row, i) => (
            <div key={i} className="flex items-center gap-2">
              <input
                type="number" min={1} step={1}
                aria-label={`Quantity tier ${i + 1} minimum quantity`}
                value={row.minQuantity}
                onChange={e => {
                  setQuantityBreaks(qb => qb.map((r, ri) => (ri === i ? { ...r, minQuantity: e.target.value } : r)))
                  setDirty(true)
                }}
                placeholder="Qty"
                className="w-20 px-2 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
              />
              <span className="text-[10px] dark:text-gray-500 text-gray-400">units or more →</span>
              <input
                type="number" step={0.01}
                aria-label={`Quantity tier ${i + 1} discount percent`}
                value={row.discountPercent}
                onChange={e => {
                  setQuantityBreaks(qb => qb.map((r, ri) => (ri === i ? { ...r, discountPercent: e.target.value } : r)))
                  setDirty(true)
                }}
                placeholder="0"
                className="w-20 px-2 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
              />
              <span className="text-[10px] dark:text-gray-500 text-gray-400">% off</span>
              <button
                type="button"
                onClick={() => { setQuantityBreaks(qb => qb.filter((_, ri) => ri !== i)); setDirty(true) }}
                aria-label={`Remove quantity tier ${i + 1}`}
                className="ml-auto text-gray-400 hover:text-red-400"
              >
                <X size={12} />
              </button>
            </div>
          ))}
        </div>
        <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
          Buy this many or more of one SKU in a single cart line and get this bigger discount instead of the flat rate above.
        </p>
      </div>
      {error && <p className="mt-3 text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>}
      <button
        onClick={handleSave}
        disabled={saving || !dirty}
        className="mt-3 px-4 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
      >
        {saving ? 'Saving…' : 'Save pricing'}
      </button>
    </div>
  )
}

interface ChannelProductsPanelProps {
  channel: SalesChannel
  onChannelUpdated: (channel: SalesChannel) => void
  onClose: () => void
  update: (id: string, payload: UpdateSalesChannelPayload) => Promise<SalesChannel>
  categories: { id: string; name: string }[]
}

// The "add all" / "add by tag" / "add by category" bulk actions are just
// this channel's default enrollment rule changing -- evaluated live against
// the whole catalog, no per-product rows written. See SalesChannel's
// EnrollmentMode doc comment in the backend.
function ChannelProductsPanel({ channel, onChannelUpdated, onClose, update, categories }: ChannelProductsPanelProps) {
  const { token } = useAuth()
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState('')
  const { items, total, loading, error, setOverride } = useSalesChannelProducts(token ?? '', channel.id, { page, perPage: PER_PAGE, search })

  const [bulkCategoryId, setBulkCategoryId] = useState('')
  const [bulkTag, setBulkTag] = useState('')
  const [bulkSaving, setBulkSaving] = useState(false)
  const [bulkError, setBulkError] = useState('')

  const applyRule = async (mode: EnrollmentMode, categoryId: string | null, tag: string | null) => {
    setBulkError('')
    setBulkSaving(true)
    try {
      const updated = await update(channel.id, {
        name: channel.name,
        connectionId: channel.connection_id,
        enrollmentMode: mode,
        enrollmentCategoryId: categoryId,
        enrollmentTag: tag,
        priceAdjustmentBasisPoints: channel.price_adjustment_basis_points ?? null,
        quantityBreaks: channel.quantity_breaks ?? [],
        active: channel.active,
      })
      onChannelUpdated(updated)
    } catch (err) {
      setBulkError(err instanceof Error ? err.message : 'Failed to update enrollment rule')
    } finally {
      setBulkSaving(false)
    }
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Sales channels
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">{channel.name}</h1>
        </div>
        <span className="text-[10px] px-2.5 py-1 rounded-full dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-400 text-gray-500">
          {ENROLLMENT_LABELS[channel.enrollment_mode]}
        </span>
      </div>

      <div className="p-5 border-b dark:border-[#2D3748] border-gray-200 flex flex-wrap items-center gap-2 shrink-0">
        <button
          onClick={() => applyRule('all', null, null)}
          disabled={bulkSaving}
          className="px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-50 disabled:opacity-50 transition-colors"
        >
          Add all
        </button>
        <div className="flex items-center gap-1.5">
          <select
            aria-label="Bulk add by category"
            value={bulkCategoryId}
            onChange={e => setBulkCategoryId(e.target.value)}
            className="px-2 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
          >
            <option value="">By category…</option>
            {categories.map(cat => <option key={cat.id} value={cat.id}>{cat.name}</option>)}
          </select>
          <button
            onClick={() => bulkCategoryId && applyRule('category', bulkCategoryId, null)}
            disabled={bulkSaving || !bulkCategoryId}
            className="px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-50 disabled:opacity-50 transition-colors"
          >
            Apply
          </button>
        </div>
        <div className="flex items-center gap-1.5">
          <input
            type="text"
            aria-label="Bulk add by tag"
            value={bulkTag}
            onChange={e => setBulkTag(e.target.value)}
            placeholder="By tag…"
            className="w-28 px-2 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none"
          />
          <button
            onClick={() => bulkTag && applyRule('tag', null, bulkTag)}
            disabled={bulkSaving || !bulkTag}
            className="px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-50 disabled:opacity-50 transition-colors"
          >
            Apply
          </button>
        </div>
        <button
          onClick={() => applyRule('manual', null, null)}
          disabled={bulkSaving}
          className="px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs font-medium dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-50 disabled:opacity-50 transition-colors"
        >
          Manual only
        </button>
      </div>
      {bulkError && <p className="px-5 pt-2 text-xs text-red-400 shrink-0">{bulkError}</p>}

      {channel.type === INTERNAL_TYPE && (
        <PricingSection channel={channel} update={update} onChannelUpdated={onChannelUpdated} />
      )}

      <div className="px-5 pt-4 shrink-0">
        <div className="relative max-w-xs">
          <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            type="text"
            value={search}
            onChange={e => { setSearch(e.target.value); setPage(1) }}
            placeholder="Search products…"
            className="w-full pl-8 pr-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
      </div>

      {error && <p className="px-5 pt-3 text-xs text-red-400 shrink-0">{error}</p>}

      <div className="flex-1 overflow-y-auto px-5 py-4">
        {!loading && items.length === 0 ? (
          <p className="text-xs dark:text-gray-500 text-gray-400 text-center py-8">No products found.</p>
        ) : (
          <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 divide-y dark:divide-[#2D3748] divide-gray-200">
            {items.map(({ product, enrolled }) => (
              <div key={product.id} className="flex items-center justify-between px-4 py-2.5">
                <div className="min-w-0">
                  <p className="text-xs font-medium dark:text-gray-200 text-gray-800 truncate">{product.name}</p>
                  {(product.tags ?? []).length > 0 && (
                    <p className="text-[10px] dark:text-gray-500 text-gray-400 truncate">{(product.tags ?? []).join(', ')}</p>
                  )}
                </div>
                <button
                  onClick={() => setOverride(product.id, !enrolled)}
                  className={clsx(
                    'px-3 py-1 rounded-lg text-[10px] font-medium transition-colors shrink-0',
                    enrolled
                      ? 'bg-green-500/10 text-green-400 hover:bg-red-500/10 hover:text-red-400'
                      : 'dark:bg-[#111827] bg-gray-100 dark:text-gray-400 text-gray-500 hover:text-blue-400',
                  )}
                >
                  {enrolled ? 'Enrolled — remove' : 'Add'}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      <Pagination page={page} pageSize={PER_PAGE} totalItems={total} onPageChange={setPage} />
    </div>
  )
}
