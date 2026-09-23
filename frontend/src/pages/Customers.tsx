import { useState, useEffect } from 'react'
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { Plus, Search, User, Trash2, Edit3, ArrowLeft } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useCustomers } from '../customers/hooks/useCustomers'
import { WhatsAppConversationPanel } from '../customers/WhatsAppConversationPanel'
import type { Customer, CustomerInput, Address } from '../customers/types'
import { idFromPath } from '../routes'
import * as omsApi from '../oms/api'
import type { SalesOrder } from '../oms/types'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { useStackNavigation } from '../routing/useStackNavigation'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import { useConfirm } from '../components/common/useConfirm'

const PAGE_SIZE = 20

function fullName(c: Customer): string {
  return [c.first_name, c.last_name].filter(Boolean).join(' ') || c.email
}

// The Notes editor on the detail view only changes customer_note, but
// update() is a full replace (see UpdateCustomerUseCase) -- round-trip every
// other field unchanged rather than adding a second, note-only update path.
function customerToInput(c: Customer): CustomerInput {
  return {
    email: c.email, first_name: c.first_name, last_name: c.last_name, username: c.username, role: c.role,
    billing: c.billing, shipping: c.shipping, is_paying_customer: c.is_paying_customer, avatar_url: c.avatar_url,
    customer_note: c.customer_note,
  }
}

function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

// "3 days ago" style formatting for the "Last order placed" stat -- no
// shared relative-time utility exists elsewhere in the app to reuse.
function formatRelativeTime(date: Date): string {
  const diffSec = Math.round((Date.now() - date.getTime()) / 1000)
  const diffMin = Math.round(diffSec / 60)
  const diffHour = Math.round(diffMin / 60)
  const diffDay = Math.round(diffHour / 24)
  if (diffSec < 60) return 'Just now'
  if (diffMin < 60) return `${diffMin} minute${diffMin === 1 ? '' : 's'} ago`
  if (diffHour < 24) return `${diffHour} hour${diffHour === 1 ? '' : 's'} ago`
  if (diffDay < 30) return `${diffDay} day${diffDay === 1 ? '' : 's'} ago`
  const diffMonth = Math.round(diffDay / 30)
  if (diffMonth < 12) return `${diffMonth} month${diffMonth === 1 ? '' : 's'} ago`
  const diffYear = Math.round(diffMonth / 12)
  return `${diffYear} year${diffYear === 1 ? '' : 's'} ago`
}

// Billing shown as "Same as shipping address" when the two addresses match
// on every field a customer would actually check.
function addressesEqual(a?: Address, b?: Address): boolean {
  if (!a || !b) return false
  const fields: (keyof Address)[] = ['first_name', 'last_name', 'company', 'address_1', 'address_2', 'city', 'state', 'postcode', 'country']
  return fields.every(f => (a[f] ?? '') === (b[f] ?? ''))
}

function AddressLines({ address }: { address?: Address }) {
  if (!address) return <p className="text-xs text-ink-4">—</p>
  const name = [address.first_name, address.last_name].filter(Boolean).join(' ')
  const lines = [
    name,
    address.company,
    address.address_1,
    address.address_2,
    [address.city, address.state, address.postcode].filter(Boolean).join(', '),
    address.country,
    address.email,
    address.phone,
  ].filter(Boolean)

  if (lines.length === 0) return <p className="text-xs text-ink-4">—</p>

  return (
    <>
      {lines.map((line, i) => (
        <p key={i} className="text-xs text-ink-2">{line}</p>
      ))}
    </>
  )
}

interface CustomerTimelineEntry {
  id: string
  label: string
  date: Date
}

// Customers have no structured activity log of their own (unlike sales
// orders' SalesOrderActivity) -- this derives a reasonable timeline from
// data already on hand: each order placed, plus the customer's own creation.
function buildCustomerTimeline(customer: Customer, orders: SalesOrder[]): CustomerTimelineEntry[] {
  const entries: CustomerTimelineEntry[] = orders.map(o => ({
    id: `order-${o.id}`,
    label: `Placed order #${o.number || o.external_order_id} (${formatMoney(o.total_minor_units, o.currency_code)})`,
    date: new Date(o.occurred_at),
  }))
  const createdDate = new Date(customer.date_created)
  if (!Number.isNaN(createdDate.getTime())) {
    entries.push({ id: 'customer-created', label: 'Customer created', date: createdDate })
  }
  return entries.sort((a, b) => b.date.getTime() - a.date.getTime())
}

export default function Customers() {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { customers, loading, error, create, update, remove } = useCustomers(token ?? '')

  // Search and page live in the URL (?q=, ?page=) -- refresh and Back keep them.
  const [searchParam, setSearch] = useQueryString('q', { resets: ['page'] })
  const search = searchParam ?? ''
  const [page, setPage] = useQueryPage()
  // The URL is the source of truth for "which customer is open"
  // (/customers/<uuid>, or /customers/new) -- survives a refresh, and is how
  // global search (Cmd+K) deep-links straight to a specific customer. Same
  // convention as Products.tsx's editingId (see its comment for why this
  // reads location.pathname on every render instead of useParams).
  //
  // /customers/<id> alone opens the read-only detail view (Amount
  // spent/Orders, Timeline, Notes, addresses); ?edit=1 on top of that opens
  // the edit form.
  const location = useLocation()
  const navigate = useNavigate()
  const [searchParams] = useSearchParams()
  const openId = idFromPath(location.pathname, '/customers')
  const isEditing = searchParams.get('edit') === '1'
  const { open, exitTo, exitToOpener } = useStackNavigation()
  const openEditor = (id: string) => open(`/customers/${id}?edit=1`)
  // Back to the exact list URL (search/page) it was opened from.
  const closeEditor = () => exitToOpener('/customers')
  const [deleteError, setDeleteError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
  }

  const filtered = customers.filter(c =>
    fullName(c).toLowerCase().includes(search.toLowerCase()) || c.email.toLowerCase().includes(search.toLowerCase()),
  )
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(c: Customer) {
    if (!(await confirm({ title: `Delete customer "${fullName(c)}"?`, message: 'This cannot be undone.', confirmLabel: 'Delete customer', destructive: true }))) return
    setDeleteError('')
    try {
      await remove(c.id)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  if (openId === 'new') {
    return <CustomerEditPage customer={null} onCreate={create} onUpdate={update} onClose={closeEditor} />
  }
  if (openId !== null) {
    const existing = customers.find(c => c.id === openId) ?? null
    if (isEditing) {
      if (!existing) {
        return loading ? <RecordLoading /> : <RecordNotFound label="Customer" onBack={closeEditor} />
      }
      return (
        <CustomerEditPage
          customer={existing}
          onCreate={create}
          onUpdate={update}
          onClose={() => exitTo(`/customers/${openId}`)}
        />
      )
    }
    return (
      <CustomerDetailPage
        customer={existing}
        loading={loading}
        token={token ?? ''}
        onEdit={() => openEditor(openId)}
        onClose={closeEditor}
        onSaveNote={note => update(openId, { ...customerToInput(existing!), customer_note: note })}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => handleSearchChange(e.target.value)}
            placeholder="Search by name or email..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <div className="flex items-center gap-2 ml-auto">
          <button
            onClick={() => navigate('/customers/new')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
          >
            <Plus size={13} /> Add customer
          </button>
        </div>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{filtered.length} customer{filtered.length === 1 ? '' : 's'}</span>
        {(error || deleteError) && <span className="text-xs text-danger ml-auto">{error || deleteError}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Customer</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Location</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 w-16" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={5} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={5} className="px-4 py-6 text-center text-ink-4">No customers yet</td></tr>
            )}
            {paged.map(c => (
              <tr
                key={c.id}
                onClick={() => navigate(`/customers/${c.id}`)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
                    <User size={12} className="text-ink-4" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[280px]">
                  <p className="font-medium text-ink truncate">{fullName(c)}</p>
                  <p className="text-2xs text-ink-4 truncate">{c.email}</p>
                </td>
                <td className="px-3 py-2.5 text-ink-3">
                  {[c.billing?.city, c.billing?.country].filter(Boolean).join(', ') || '—'}
                </td>
                <td className="px-3 py-2.5">
                  <span className={clsx(
                    'px-2 py-0.5 rounded-full text-2xs font-medium',
                    c.is_paying_customer ? 'bg-green-500/15 text-success' : 'bg-gray-500/15 text-ink-4',
                  )}>
                    {c.is_paying_customer ? 'Paying' : 'No orders yet'}
                  </span>
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label="Edit"
                      onClick={e => { e.stopPropagation(); openEditor(c.id) }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title="Edit"
                    >
                      <Edit3 size={12} className="text-ink-3" />
                    </button>
                    <button aria-label="Delete"
                      onClick={e => { e.stopPropagation(); handleDelete(c) }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={12} className="text-ink-3 hover:text-danger" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={filtered.length} onPageChange={setPage} />
      {confirmDialog}
    </div>
  )
}

// ── Customer Detail Page ─────────────────────────────────────────────────────

function CustomerDetailPage({ customer, loading, token, onEdit, onClose, onSaveNote }: {
  customer: Customer | null
  loading: boolean
  token: string
  onEdit: () => void
  onClose: () => void
  onSaveNote: (note: string) => Promise<Customer>
}) {
  const [orders, setOrders] = useState<SalesOrder[]>([])
  const [ordersTotal, setOrdersTotal] = useState(0)
  const [ordersError, setOrdersError] = useState('')

  // Fetched directly (not via useSalesOrders) so this can wait for
  // `customer` to actually be loaded -- a deep link (search/refresh) mounts
  // this before useCustomers' list resolves, and firing an unfiltered
  // "list every order" call in that gap would be wrong AND wasteful.
  useEffect(() => {
    if (!token || !customer) return
    setOrdersError('')
    omsApi.listSalesOrders(token, { customerId: customer.id, perPage: 200, sortBy: 'occurred_at', sortDir: 'desc' })
      .then(result => { setOrders(result.salesOrders); setOrdersTotal(result.total) })
      .catch(err => {
        setOrders([])
        setOrdersTotal(0)
        setOrdersError(err instanceof Error ? err.message : 'Failed to load orders')
      })
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token, customer?.id])

  const [editingNote, setEditingNote] = useState(false)
  const [noteDraft, setNoteDraft] = useState('')
  const [savingNote, setSavingNote] = useState(false)
  const [noteError, setNoteError] = useState('')

  function startEditingNote() {
    setNoteDraft(customer?.customer_note ?? '')
    setEditingNote(true)
  }

  async function saveNote() {
    setSavingNote(true)
    setNoteError('')
    try {
      await onSaveNote(noteDraft)
      setEditingNote(false)
    } catch (err) {
      setNoteError(err instanceof Error ? err.message : 'Failed to save note')
    } finally {
      setSavingNote(false)
    }
  }

  // Amount spent sums only the orders actually fetched (capped at 200,
  // ListSalesOrders' documented per_page maximum) --
  // accurate for the overwhelming majority of customers; ordersTotal (the
  // server's real count) is used for the Orders stat itself, which stays
  // correct regardless.
  const currency = orders[0]?.currency_code ?? 'USD'
  const amountSpentMinorUnits = orders.reduce((sum, o) => sum + o.total_minor_units, 0)
  const timeline = customer ? buildCustomerTimeline(customer, orders) : []

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Customers
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {customer ? fullName(customer) : 'Customer'}
          </h1>
        </div>
        {customer && (
          <button
            onClick={onEdit}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
          >
            <Edit3 size={13} /> Edit
          </button>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-5xl mx-auto p-6 space-y-5">
          {loading && !customer && <p className="text-xs text-ink-4">Loading…</p>}
          {!loading && !customer && <p className="text-xs text-ink-4">Customer not found</p>}
          {noteError && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{noteError}</p>}
          {ordersError && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{ordersError}</p>}

          {customer && (
            <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-5 items-start">
              {/* Main column */}
              <div className="space-y-5 min-w-0">
                <div className="grid grid-cols-3 gap-4">
                  <div className="rounded-xl border border-line p-4">
                    <p className="text-2xs font-semibold uppercase tracking-wider text-ink-4 mb-1">Amount spent</p>
                    <p className="text-lg font-semibold text-ink">{formatMoney(amountSpentMinorUnits, currency)}</p>
                  </div>
                  <Link
                    to={`/sales-orders?customer_id=${customer.id}`}
                    className="block rounded-xl border border-line p-4 hover:bg-hover/60 transition-colors"
                  >
                    <p className="text-2xs font-semibold uppercase tracking-wider text-ink-4 mb-1">Orders</p>
                    <p className="text-lg font-semibold text-accent">{ordersTotal}</p>
                  </Link>
                  <div className="rounded-xl border border-line p-4">
                    <p className="text-2xs font-semibold uppercase tracking-wider text-ink-4 mb-1">Last order placed</p>
                    <p className="text-sm font-medium text-ink">
                      {orders[0] ? formatRelativeTime(new Date(orders[0].occurred_at)) : 'No orders yet'}
                    </p>
                  </div>
                </div>

                <div className="rounded-xl border border-line p-4 space-y-2 text-xs">
                  <p className="font-medium text-ink-2">Timeline</p>
                  {timeline.length === 0 ? (
                    <p className="text-ink-4">No activity yet</p>
                  ) : timeline.map(entry => (
                    <div key={entry.id} className="flex justify-between gap-3 text-ink-3">
                      <span className="text-ink">{entry.label}</span>
                      <span className="shrink-0">{entry.date.toLocaleString()}</span>
                    </div>
                  ))}
                </div>

                <WhatsAppConversationPanel customerId={customer.id} />
              </div>

              {/* Aside column */}
              <div className="space-y-5 min-w-0">
                <div className="rounded-xl border border-line p-4 space-y-2 text-xs">
                  <div className="flex items-center justify-between">
                    <h3 className="text-xs font-semibold text-ink">Notes</h3>
                    {!editingNote && (
                      <button
                        onClick={startEditingNote}
                        title="Edit notes"
                        aria-label="Edit notes"
                        className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      >
                        <Edit3 size={12} className="text-ink-3" />
                      </button>
                    )}
                  </div>
                  {editingNote ? (
                    <div className="space-y-2">
                      <label htmlFor="customer-note" className="sr-only">Notes</label>
                      <textarea
                        id="customer-note"
                        value={noteDraft}
                        onChange={e => setNoteDraft(e.target.value)}
                        rows={3}
                        className="w-full px-2 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
                      />
                      <div className="flex items-center gap-2">
                        <button
                          onClick={saveNote}
                          disabled={savingNote}
                          className="px-2.5 py-1 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors disabled:opacity-50"
                        >
                          Save note
                        </button>
                        <button
                          onClick={() => setEditingNote(false)}
                          disabled={savingNote}
                          className="px-2.5 py-1 rounded-lg border border-line text-xs font-medium text-ink-2 hover:bg-hover transition-colors disabled:opacity-50"
                        >
                          Cancel
                        </button>
                      </div>
                    </div>
                  ) : customer.customer_note ? (
                    <p className="text-ink whitespace-pre-wrap">{customer.customer_note}</p>
                  ) : (
                    <p className="text-ink-4">No notes</p>
                  )}
                </div>

                <div className="rounded-xl border border-line p-4 space-y-3 text-xs">
                  <h3 className="text-xs font-semibold text-ink">Contact information</h3>
                  <p className="text-ink">{customer.email}</p>
                  <div>
                    <h4 className="text-2xs font-semibold uppercase tracking-wider text-ink-4 mb-1">Shipping address</h4>
                    <AddressLines address={customer.shipping} />
                  </div>
                  <div>
                    <h4 className="text-2xs font-semibold uppercase tracking-wider text-ink-4 mb-1">Billing address</h4>
                    {addressesEqual(customer.billing, customer.shipping) ? (
                      <p className="text-ink-4">Same as shipping address</p>
                    ) : (
                      <AddressLines address={customer.billing} />
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  )
}

// ── Customer Edit Page ───────────────────────────────────────────────────────

interface CustomerEditPageProps {
  customer: Customer | null
  onCreate: (input: CustomerInput) => Promise<Customer>
  onUpdate: (id: string, input: CustomerInput) => Promise<Customer>
  onClose: () => void
}

function emptyAddress(): Address {
  return { first_name: '', last_name: '', company: '', address_1: '', address_2: '', city: '', state: '', postcode: '', country: '', email: '', phone: '' }
}

function CustomerEditPage({ customer, onCreate, onUpdate, onClose }: CustomerEditPageProps) {
  const [firstName, setFirstName] = useState(customer?.first_name ?? '')
  const [lastName, setLastName] = useState(customer?.last_name ?? '')
  const [email, setEmail] = useState(customer?.email ?? '')
  const [billing, setBilling] = useState<Address>({ ...emptyAddress(), ...customer?.billing })
  const [shipping, setShipping] = useState<Address>({ ...emptyAddress(), ...customer?.shipping })
  const [isPayingCustomer, setIsPayingCustomer] = useState(customer?.is_paying_customer ?? false)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  // A deep link (Cmd+K search, direct URL, hard refresh) mounts this page
  // before useCustomers' list fetch resolves, so the useState initializers
  // above run with customer still null. Once the list arrives and `customer`
  // is passed in for the first time, re-seed the form from it -- without
  // this, the fields stay blank forever since useState never re-runs.
  const [hydrated, setHydrated] = useState(false)
  if (!hydrated && customer) {
    setFirstName(customer.first_name ?? '')
    setLastName(customer.last_name ?? '')
    setEmail(customer.email ?? '')
    setBilling({ ...emptyAddress(), ...customer.billing })
    setShipping({ ...emptyAddress(), ...customer.shipping })
    setIsPayingCustomer(customer.is_paying_customer ?? false)
    setHydrated(true)
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      const input: CustomerInput = {
        email, first_name: firstName, last_name: lastName, billing, shipping, is_paying_customer: isPayingCustomer,
      }
      if (customer) {
        await onUpdate(customer.id, input)
      } else {
        await onCreate(input)
      }
      onClose()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save customer')
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Customer
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {customer ? [firstName, lastName].filter(Boolean).join(' ') || 'Edit Customer' : 'New Customer'}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="customer-form" disabled={saving || !email.trim()}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : customer ? 'Save' : 'Create customer'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-2xl mx-auto p-6 space-y-5">
          <form id="customer-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">General</h3>
                <label className="flex items-center gap-1.5 text-2xs text-ink-3">
                  <input type="checkbox" checked={isPayingCustomer} onChange={e => setIsPayingCustomer(e.target.checked)} />
                  Paying customer
                </label>
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <label htmlFor="customer-first-name" className={labelClass}>First name</label>
                  <input id="customer-first-name" type="text" autoFocus value={firstName} onChange={e => setFirstName(e.target.value)} className={inputClass} />
                </div>
                <div>
                  <label htmlFor="customer-last-name" className={labelClass}>Last name</label>
                  <input id="customer-last-name" type="text" value={lastName} onChange={e => setLastName(e.target.value)} className={inputClass} />
                </div>
              </div>
              <div>
                <label htmlFor="customer-email" className={labelClass}>Email</label>
                <input id="customer-email" type="email" required value={email} onChange={e => setEmail(e.target.value)} className={inputClass} />
              </div>
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <AddressFields title="Billing address" prefix="billing" value={billing} onChange={setBilling} showEmailPhone />
            <AddressFields title="Shipping address" prefix="shipping" value={shipping} onChange={setShipping} />
          </form>
        </div>
      </div>
    </div>
  )
}

function AddressFields({ title, prefix, value, onChange, showEmailPhone }: { title: string; prefix: string; value: Address; onChange: (a: Address) => void; showEmailPhone?: boolean }) {
  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus transition-shadow'
  const set = (field: keyof Address) => (e: React.ChangeEvent<HTMLInputElement>) => onChange({ ...value, [field]: e.target.value })
  const id = (field: string) => `${prefix}-${field}`
  const label = (text: string) => `${title.replace(' address', '')} ${text}`

  return (
    <div className="rounded-xl border border-line p-4 space-y-3">
      <h3 className="text-xs font-semibold text-ink">{title}</h3>
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor={id('address-1')} className={labelClass}>{label('address line 1')}</label>
          <input id={id('address-1')} type="text" value={value.address_1 ?? ''} onChange={set('address_1')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('address-2')} className={labelClass}>{label('address line 2')}</label>
          <input id={id('address-2')} type="text" value={value.address_2 ?? ''} onChange={set('address_2')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('city')} className={labelClass}>{label('city')}</label>
          <input id={id('city')} type="text" value={value.city ?? ''} onChange={set('city')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('state')} className={labelClass}>{label('state')}</label>
          <input id={id('state')} type="text" value={value.state ?? ''} onChange={set('state')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('postcode')} className={labelClass}>{label('postcode')}</label>
          <input id={id('postcode')} type="text" value={value.postcode ?? ''} onChange={set('postcode')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('country')} className={labelClass}>{label('country')}</label>
          <input id={id('country')} type="text" value={value.country ?? ''} onChange={set('country')} className={inputClass} />
        </div>
        {showEmailPhone && (
          <div>
            <label htmlFor={id('phone')} className={labelClass}>{label('phone')}</label>
            <input id={id('phone')} type="tel" value={value.phone ?? ''} onChange={set('phone')} className={inputClass} />
          </div>
        )}
      </div>
    </div>
  )
}
