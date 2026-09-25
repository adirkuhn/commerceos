import { useState, useEffect } from 'react'
import { Link } from 'react-router-dom'
import { Edit3, ArrowLeft } from 'lucide-react'
import { WhatsAppConversationPanel } from './WhatsAppConversationPanel'
import type { Customer, Address } from './types'
import * as omsApi from '../oms/api'
import type { SalesOrder } from '../oms/types'
import { formatMoney } from '../format/money'
import { addressesEqual } from '../oms/salesOrderDisplay'
import { fullName } from '../format/personName'

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

// ── Customer Detail Page ─────────────────────────────────────────────────────

export function CustomerDetailPage({ customer, loading, token, onEdit, onClose, onSaveNote }: {
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
