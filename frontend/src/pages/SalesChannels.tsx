import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { ArrowLeft, Radio, X, Search } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { themeEditorPath } from '../theme/routing'
import { useSalesChannels } from '../integrations/hooks/useSalesChannels'
import { useSalesChannelProducts } from '../integrations/hooks/useSalesChannelProducts'
import { useConnections } from '../integrations/hooks/useConnections'
import { useCategories } from '../pms/hooks/useCategories'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import * as integrationsApi from '../integrations/api'
import type { Connection, SalesChannel, EnrollmentMode, ChannelType } from '../integrations/types'
import type { SalesChannelPayload, UpdateSalesChannelPayload } from '../integrations/api'
import Pagination from '../components/common/Pagination'

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
  paymentConnectionId: string
  // Held as a percent string ("20" means 20%) for the input field --
  // converted to basis points only at submit time (see priceAdjustmentBasisPointsOrError).
  priceAdjustmentPercent: string
}

function emptyForm(): ChannelFormState {
  return {
    name: '', channelType: 'woocommerce', connectionId: '', enrollmentMode: 'all', enrollmentCategoryId: '',
    enrollmentTag: '', active: true, defaultWarehouseId: '', paymentConnectionId: '', priceAdjustmentPercent: '',
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
const CONNECTION_REQUIRED_TYPES: ChannelType[] = (Object.keys(TYPE_LABELS) as ChannelType[]).filter(t => t !== INTERNAL_TYPE)
const PAYMENT_CAPABLE_TYPES: ChannelType[] = ['whatsapp', INTERNAL_TYPE]

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
    paymentConnectionId: channel.payment_connection_id ?? '',
    priceAdjustmentPercent: channel.price_adjustment_basis_points ? (channel.price_adjustment_basis_points / 100).toString() : '',
  }
}

export default function SalesChannels() {
  const navigate = useNavigate()
  const { token } = useAuth()
  const { channels, loading, error, create, update, remove } = useSalesChannels(token ?? '')
  const { connections, refresh: refreshConnections } = useConnections(token ?? '')
  const { categories } = useCategories(token ?? '')
  const { warehouses } = useWarehouses(token ?? '')

  const [managingChannel, setManagingChannel] = useState<SalesChannel | null>(null)

  const [editingChannel, setEditingChannel] = useState<SalesChannel | null>(null)
  const [creating, setCreating] = useState(false)
  const [form, setForm] = useState<ChannelFormState>(emptyForm())
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  const [deletingChannel, setDeletingChannel] = useState<SalesChannel | null>(null)
  const [deleting, setDeleting] = useState(false)
  const [deleteError, setDeleteError] = useState('')

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
      const shared = {
        name: form.name,
        connectionId: form.channelType === INTERNAL_TYPE ? null : form.connectionId || null,
        enrollmentMode: form.enrollmentMode,
        enrollmentCategoryId: form.enrollmentMode === 'category' ? form.enrollmentCategoryId || null : null,
        enrollmentTag: form.enrollmentMode === 'tag' ? form.enrollmentTag || null : null,
        priceAdjustmentBasisPoints: priceAdjustmentBasisPointsOrError(form.priceAdjustmentPercent),
      }
      let channelType: ChannelType
      if (editingChannel) {
        await update(editingChannel.id, { ...shared, active: form.active } satisfies UpdateSalesChannelPayload)
        channelType = editingChannel.type
      } else {
        channelType = form.channelType
        await create({ ...shared, type: channelType } satisfies SalesChannelPayload)
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
                <div className={clsx('mt-3 grid gap-2', channel.type === INTERNAL_TYPE ? 'grid-cols-4' : 'grid-cols-3')}>
                  <button
                    onClick={() => setManagingChannel(channel)}
                    className="py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 transition-colors"
                  >
                    Products
                  </button>
                  {channel.type === INTERNAL_TYPE && (
                    <button
                      onClick={() => navigate(themeEditorPath(channel.id, 'home'))}
                      className="py-1.5 rounded-lg dark:bg-[#111827] bg-gray-100 text-[10px] dark:text-gray-400 text-gray-500 hover:text-blue-400 transition-colors"
                    >
                      Theme
                    </button>
                  )}
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
                    onChange={e => setForm(f => ({ ...f, channelType: e.target.value as ChannelType, connectionId: '', defaultWarehouseId: '' }))}
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
              {(editingChannel ? editingChannel.type : form.channelType) === INTERNAL_TYPE && (
                <div>
                  <label htmlFor="channel-price-adjustment" className="block text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5">
                    Discount off list price (%)
                  </label>
                  <input
                    id="channel-price-adjustment"
                    type="number" step={0.01}
                    value={form.priceAdjustmentPercent}
                    onChange={e => setForm(f => ({ ...f, priceAdjustmentPercent: e.target.value }))}
                    placeholder="0"
                    className="w-full px-3 py-2 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
                  />
                  <p className="mt-1.5 text-[10px] dark:text-gray-500 text-gray-400">
                    Applied to every product's price on this storefront -- e.g. 20 for a wholesale channel selling at 20% off retail.
                  </p>
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
