import { useState, useEffect, useCallback } from 'react'
import { Link, useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { idFromPath } from '../routes'
import { ArrowLeft, ShoppingCart, Plus, Trash2, ChevronUp, ChevronDown, Printer, X, Edit3 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSalesOrders } from '../oms/hooks/useSalesOrders'
import { useSalesOrderDetail } from '../oms/hooks/useSalesOrderDetail'
import { useSalesOrderActivities } from '../oms/hooks/useSalesOrderActivities'
import type { SalesOrder, SalesOrderStatus, SalesOrderSource, SalesOrderDetail, CreateSalesOrderInput, CreateSalesOrderLineItemInput, Address, SalesOrderSortField, SortDirection } from '../oms/types'
import { PickingListPrintView } from './print/PickingListPrintView'
import { InvoicePrintView } from './print/InvoicePrintView'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { ProductSkuPicker } from './Procurement'
import type { SKU } from '../pms/types'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { useConnections } from '../integrations/hooks/useConnections'
import * as shippingApi from '../shipping/api'
import { AddressConfirmModal } from '../shipping/AddressConfirmModal'
import { LabelPreviewModal } from '../shipping/LabelPreviewModal'
import { ExistingShipmentConfirmModal } from '../shipping/ExistingShipmentConfirmModal'
import { loadShipmentLabelIntoTab, openShipmentLabel } from '../shipping/openLabel'
import { LabelOpenedNotice } from '../shipping/LabelOpenedNotice'
import type { RecipientAddress, Shipment, ShipmentTrackingEvent } from '../shipping/types'
import { CARRIERS } from '../shipping/types'
import { useConfirm } from '../components/common/useConfirm'

const PAGE_SIZE = 20
const STATUS_OPTIONS: SalesOrderStatus[] = ['pending', 'processing', 'on-hold', 'shipped', 'completed', 'cancelled', 'refunded', 'failed']

// SOURCE_FILTER_OPTIONS: 'manual' reads as "Internal" here -- POS/manual/
// bol/woocommerce is the wire vocabulary (see SalesOrderSource), Internal is
// the label staff actually recognize for a manually-created order.
const SOURCE_FILTER_OPTIONS: { value: SalesOrderSource; label: string }[] = [
  { value: 'pos', label: 'POS' },
  { value: 'woocommerce', label: 'WOO' },
  { value: 'bol', label: 'Bol' },
  { value: 'manual', label: 'Internal' },
]

const STATUS_COLORS: Record<SalesOrderStatus, string> = {
  pending: 'bg-gray-500/15 text-ink-4',
  processing: 'bg-accent/15 text-accent',
  'on-hold': 'bg-amber-500/15 text-warning',
  shipped: 'bg-indigo-500/15 text-indigo-400',
  completed: 'bg-green-500/15 text-success',
  cancelled: 'bg-gray-500/15 text-ink-4',
  refunded: 'bg-purple-500/15 text-purple-400',
  failed: 'bg-red-500/15 text-danger',
}

function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString()
}

function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || '—'
}

// Billing shown as "Same as shipping address" when the two addresses match
// on every field a customer would actually check -- not e.g. phone, which
// legitimately differs (billing contact vs. delivery contact) without the
// addresses being different.
function addressesEqual(a?: Address, b?: Address): boolean {
  if (!a || !b) return false
  const fields: (keyof Address)[] = ['first_name', 'last_name', 'company', 'address_1', 'address_2', 'city', 'state', 'postcode', 'country']
  return fields.every(f => (a[f] ?? '') === (b[f] ?? ''))
}

function SortableTh({ field, label, sortBy, sortDir, onSort, align = 'left' }: {
  field: SalesOrderSortField
  label: string
  sortBy: SalesOrderSortField
  sortDir: SortDirection
  onSort: (field: SalesOrderSortField) => void
  align?: 'left' | 'right'
}) {
  const active = field === sortBy
  const Icon = sortDir === 'asc' ? ChevronUp : ChevronDown
  return (
    <th
      className={clsx(
        'px-3 py-2.5 font-medium uppercase tracking-wider cursor-pointer select-none hover:text-ink-2 transition-colors',
        align === 'right' ? 'text-right' : 'text-left',
        active ? 'text-ink-2' : 'text-ink-4',
      )}
      onClick={() => onSort(field)}
    >
      <span className={clsx('inline-flex items-center gap-1', align === 'right' && 'flex-row-reverse')}>
        {label}
        {active && <Icon size={11} />}
      </span>
    </th>
  )
}

export default function SalesOrders() {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  // Filters/sort/page are seeded from the URL on first render and kept in
  // sync with it (see the effect below) -- reloading the page, or sharing
  // the link, reproduces the exact same view instead of resetting it.
  const [searchParams, setSearchParams] = useSearchParams()
  const [page, setPage] = useState(() => {
    const p = Number(searchParams.get('page'))
    return p > 0 ? p : 1
  })
  const [sortBy, setSortBy] = useState<SalesOrderSortField>(() => (searchParams.get('sort_by') as SalesOrderSortField | null) ?? 'occurred_at')
  const [sortDir, setSortDir] = useState<SortDirection>(() => (searchParams.get('sort_dir') as SortDirection | null) ?? 'desc')
  const [sourceFilter, setSourceFilter] = useState<SalesOrderSource[]>(() => searchParams.getAll('source') as SalesOrderSource[])
  const [statusFilter, setStatusFilter] = useState<SalesOrderStatus[]>(() => searchParams.getAll('status') as SalesOrderStatus[])
  // fulfilledFilter: '' = don't filter (all substatuses), otherwise
  // "true"/"false" -- a substatus nested under whatever status is selected
  // above (e.g. Processing + not yet fulfilled).
  const [fulfilledFilter, setFulfilledFilter] = useState<'' | 'true' | 'false'>(() => {
    const f = searchParams.get('fulfilled')
    return f === 'true' || f === 'false' ? f : ''
  })
  // customerId: not a filter button -- arrives via a link from the Customer
  // detail page's "Orders" stat (see Customers.tsx), same URL-seeding
  // convention as the other filters so page/sort changes don't drop it.
  const [customerId, setCustomerId] = useState<string | null>(() => searchParams.get('customer_id'))

  // The open order lives in the URL (/sales-orders/<uuid>, or
  // /sales-orders/new) -- read via location.pathname (not useParams) so it
  // reflects the URL on every render even when navigating here doesn't
  // remount the page. This used to be `useState(() => searchParams.get('id'))`,
  // a lazy initializer that only ran on mount: global search's
  // CommandPalette navigating straight to this order while already on this
  // page changed the URL but never re-ran the initializer, so the click
  // silently did nothing. See routes.ts's idFromPath doc comment.
  const location = useLocation()
  const navigate = useNavigate()
  const selectedId = idFromPath(location.pathname, '/sales-orders')

  const buildListParams = useCallback(() => {
    const params = new URLSearchParams()
    if (page !== 1) params.set('page', String(page))
    if (sortBy !== 'occurred_at') params.set('sort_by', sortBy)
    if (sortDir !== 'desc') params.set('sort_dir', sortDir)
    sourceFilter.forEach(s => params.append('source', s))
    statusFilter.forEach(s => params.append('status', s))
    if (fulfilledFilter) params.set('fulfilled', fulfilledFilter)
    if (customerId) params.set('customer_id', customerId)
    return params
  }, [page, sortBy, sortDir, sourceFilter, statusFilter, fulfilledFilter, customerId])

  useEffect(() => {
    // Don't touch the query string while viewing a record -- it now lives
    // on its own path (/sales-orders/<id>), not here.
    if (selectedId !== null) return
    setSearchParams(buildListParams(), { replace: true })
  }, [selectedId, buildListParams, setSearchParams])

  // Returns to the list, restoring whatever page/sort/filters were active
  // before opening the record -- captured here from current state rather
  // than relying on it having survived the route change.
  function closeDetail() {
    const qs = buildListParams().toString()
    navigate(`/sales-orders${qs ? `?${qs}` : ''}`)
  }

  const hasActiveFilters = sourceFilter.length > 0 || statusFilter.length > 0 || fulfilledFilter !== '' || customerId !== null

  function clearFilters() {
    setSourceFilter([])
    setStatusFilter([])
    setFulfilledFilter('')
    setCustomerId(null)
    setPage(1)
  }

  const { salesOrders, total, loading, error, create, refresh, bulkUpdate } = useSalesOrders(token ?? '', {
    page, perPage: PAGE_SIZE, sortBy, sortDir, source: sourceFilter, status: statusFilter,
    fulfilled: fulfilledFilter === '' ? undefined : fulfilledFilter === 'true',
    customerId: customerId ?? undefined,
  })
  const [checkedIds, setCheckedIds] = useState<Set<string>>(new Set())
  const [bulkStatus, setBulkStatus] = useState('')
  const [bulkFulfilled, setBulkFulfilled] = useState('')
  const [bulkApplying, setBulkApplying] = useState(false)
  const [bulkError, setBulkError] = useState('')

  const allOnPageChecked = salesOrders.length > 0 && salesOrders.every(o => checkedIds.has(o.id))

  function toggleAllOnPage() {
    setCheckedIds(prev => {
      const next = new Set(prev)
      if (allOnPageChecked) {
        salesOrders.forEach(o => next.delete(o.id))
      } else {
        salesOrders.forEach(o => next.add(o.id))
      }
      return next
    })
  }

  function toggleOne(id: string) {
    setCheckedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function applyBulkChange() {
    if (!bulkStatus && !bulkFulfilled) return
    const count = checkedIds.size
    if (!(await confirm({ title: `Apply this change to ${count} order${count === 1 ? '' : 's'}?`, confirmLabel: 'Apply change' }))) return
    setBulkApplying(true)
    setBulkError('')
    try {
      const result = await bulkUpdate({
        orderIds: Array.from(checkedIds),
        status: bulkStatus ? (bulkStatus as SalesOrderStatus) : undefined,
        fulfilled: bulkFulfilled ? bulkFulfilled === 'true' : undefined,
      })
      if (result.failed.length > 0) {
        // Leave the bar (and the still-checked ids) up so the error stays
        // visible -- clearing here would hide which orders failed.
        setBulkError(`${result.updated.length} updated, ${result.failed.length} failed: ${result.failed.map(f => f.error).join('; ')}`)
      } else {
        setCheckedIds(new Set())
        setBulkStatus('')
        setBulkFulfilled('')
      }
    } catch (err) {
      setBulkError(err instanceof Error ? err.message : 'Failed to apply bulk change')
    } finally {
      setBulkApplying(false)
    }
  }

  function toggleSourceFilter(source: SalesOrderSource) {
    setSourceFilter(prev => prev.includes(source) ? prev.filter(s => s !== source) : [...prev, source])
    setPage(1)
  }

  function toggleStatusFilter(status: SalesOrderStatus) {
    setStatusFilter(prev => prev.includes(status) ? prev.filter(s => s !== status) : [...prev, status])
    setPage(1)
  }

  function changeFulfilledFilter(value: '' | 'true' | 'false') {
    setFulfilledFilter(value)
    setPage(1)
  }

  function toggleSort(field: SalesOrderSortField) {
    if (field === sortBy) {
      setSortDir(d => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortBy(field)
      setSortDir('desc')
    }
    setPage(1)
  }

  if (selectedId === 'new') {
    return (
      <CreateSalesOrderPage
        onCreate={create}
        onClose={closeDetail}
      />
    )
  }
  if (selectedId !== null) {
    return <SalesOrderDetailPage salesOrderId={selectedId} onClose={() => { closeDetail(); refresh() }} />
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center justify-end gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <button
          onClick={() => navigate('/sales-orders/new')}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
        >
          <Plus size={13} /> New Sales Order
        </button>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4 whitespace-nowrap">{total} sales order{total === 1 ? '' : 's'}</span>
        <div className="flex items-center gap-1.5" role="group" aria-label="Filter by channel">
          {SOURCE_FILTER_OPTIONS.map(({ value, label }) => (
            <button
              key={value}
              onClick={() => toggleSourceFilter(value)}
              aria-pressed={sourceFilter.includes(value)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
                sourceFilter.includes(value)
                  ? 'bg-accent-solid text-white'
                  : 'bg-raised text-ink-3 hover:bg-hover',
              )}
            >
              {label}
            </button>
          ))}
        </div>
        {error && <span className="text-xs text-danger ml-auto">{error}</span>}
      </div>

      {/* Status + substatus filter strip */}
      <div className="flex items-center flex-wrap gap-3 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <div className="flex items-center flex-wrap gap-1.5" role="group" aria-label="Filter by status">
          {STATUS_OPTIONS.map(s => (
            <button
              key={s}
              onClick={() => toggleStatusFilter(s)}
              aria-pressed={statusFilter.includes(s)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-xs font-medium capitalize transition-colors',
                statusFilter.includes(s)
                  ? 'bg-accent-solid text-white'
                  : 'dark:bg-night-800 bg-gray-50 text-ink-3 hover:bg-hover',
              )}
            >
              {s}
            </button>
          ))}
        </div>
        <select
          aria-label="Filter by fulfilled substatus" value={fulfilledFilter}
          onChange={e => changeFulfilledFilter(e.target.value as '' | 'true' | 'false')}
          className="px-2 py-1.5 text-xs rounded-lg dark:bg-night-800 bg-gray-50 border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
        >
          <option value="">Any fulfillment</option>
          <option value="true">Only fulfilled</option>
          <option value="false">Only unfulfilled</option>
        </select>
        {customerId && (
          <span className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-accent/15 text-accent">
            Filtered to one customer
            <button type="button" onClick={() => setCustomerId(null)} aria-label="Clear customer filter" className="hover:opacity-70">
              <X size={12} />
            </button>
          </span>
        )}
        {hasActiveFilters && (
          <button
            type="button" onClick={clearFilters}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium text-ink-3 hover:text-ink transition-colors ml-auto"
          >
            <X size={12} /> Clear filters
          </button>
        )}
      </div>

      {/* Bulk action bar */}
      {checkedIds.size > 0 && (
        <div className="flex items-center gap-3 px-5 py-2.5 border-b border-line dark:bg-night-800 bg-accent/10 shrink-0">
          <span className="text-xs font-medium text-ink-2">{checkedIds.size} selected</span>
          <select
            aria-label="Bulk change status" value={bulkStatus} onChange={e => setBulkStatus(e.target.value)} disabled={bulkApplying}
            className="px-2 py-1 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60 disabled:opacity-50"
          >
            <option value="">Status…</option>
            {STATUS_OPTIONS.map(s => <option key={s} value={s}>{s}</option>)}
          </select>
          <select
            aria-label="Bulk change fulfilled" value={bulkFulfilled} onChange={e => setBulkFulfilled(e.target.value)} disabled={bulkApplying}
            className="px-2 py-1 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60 disabled:opacity-50"
          >
            <option value="">Fulfilled…</option>
            <option value="true">Fulfilled</option>
            <option value="false">Unfulfilled</option>
          </select>
          <button
            onClick={applyBulkChange}
            disabled={bulkApplying || (!bulkStatus && !bulkFulfilled)}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {bulkApplying ? 'Applying…' : 'Apply'}
          </button>
          <button
            onClick={() => { setCheckedIds(new Set()); setBulkStatus(''); setBulkFulfilled(''); setBulkError('') }}
            disabled={bulkApplying}
            className="text-xs font-medium text-ink-3 hover:text-ink-2 disabled:opacity-50"
          >
            Clear
          </button>
          {bulkError && <span className="text-xs text-danger ml-auto">{bulkError}</span>}
        </div>
      )}

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-3 py-2.5 w-8">
                <input
                  type="checkbox" aria-label="Select all orders on this page"
                  checked={allOnPageChecked} onChange={toggleAllOnPage}
                  className="rounded border dark:border-night-700 border-gray-300"
                />
              </th>
              <th className="px-2 py-2.5 w-8" />
              <SortableTh field="occurred_at" label="Order" sortBy={sortBy} sortDir={sortDir} onSort={toggleSort} />
              <SortableTh field="customer" label="Customer" sortBy={sortBy} sortDir={sortDir} onSort={toggleSort} />
              <SortableTh field="source" label="Source" sortBy={sortBy} sortDir={sortDir} onSort={toggleSort} />
              <SortableTh field="status" label="Status" sortBy={sortBy} sortDir={sortDir} onSort={toggleSort} />
              <SortableTh field="total" label="Total" sortBy={sortBy} sortDir={sortDir} onSort={toggleSort} align="right" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && salesOrders.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">No sales orders yet</td></tr>
            )}
            {salesOrders.map(o => (
              <tr
                key={o.id}
                onClick={() => navigate(`/sales-orders/${o.id}`)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
                  <input
                    type="checkbox" aria-label={`Select order #${o.number || o.external_order_id}`}
                    checked={checkedIds.has(o.id)} onChange={() => toggleOne(o.id)}
                    className="rounded border dark:border-night-700 border-gray-300"
                  />
                </td>
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
                    <ShoppingCart size={12} className="text-ink-4" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[200px]">
                  <p className="font-medium text-ink truncate">#{o.number || o.external_order_id}</p>
                  <p className="text-2xs text-ink-4 truncate">{new Date(o.occurred_at).toLocaleString()}</p>
                </td>
                <td className="px-3 py-2.5 text-ink-3 max-w-[220px] truncate">{customerName(o)}</td>
                <td className="px-3 py-2.5 text-ink-3">{o.source}</td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium', STATUS_COLORS[o.status])}>
                      {o.status}
                    </span>
                    <span className={clsx(
                      'px-2 py-0.5 rounded-full text-2xs font-medium',
                      o.fulfilled ? 'bg-green-500/15 text-success' : 'bg-gray-500/15 text-ink-4',
                    )}>
                      {o.fulfilled ? 'Fulfilled' : 'Unfulfilled'}
                    </span>
                    {(o.needs_stock_review || o.needs_sku_review) && (
                      <span className="px-2 py-0.5 rounded-full text-2xs font-medium bg-amber-500/15 text-warning">
                        Needs review
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-3 py-2.5 text-right font-medium text-ink">
                  {formatMoney(o.total_minor_units, o.currency_code)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
      {confirmDialog}
    </div>
  )
}

// ── Create Sales Order Page ──────────────────────────────────────────────────

interface DraftLineItem {
  key: string
  skuId: string
  label: string
  quantity: string
  price: string
  // null means "use the tenant's default tax class" -- same convention as
  // pms.SKU.TaxClassID.
  taxClassId: string | null
}

function emptyAddress(): Address {
  return { first_name: '', last_name: '', company: '', address_1: '', address_2: '', city: '', state: '', postcode: '', country: '', email: '', phone: '' }
}

function CreateSalesOrderPage({ onCreate, onClose }: { onCreate: (input: CreateSalesOrderInput) => Promise<SalesOrderDetail>; onClose: () => void }) {
  const { token } = useAuth()
  const { warehouses } = useWarehouses(token ?? '')
  const { classes: taxClasses } = useTaxClasses(token ?? '')
  const { rates: taxRates } = useTaxRates(token ?? '')
  const { settings: tenantSettings } = useTenantSettings(token ?? '')

  const [warehouseId, setWarehouseId] = useState('')
  const [currencyCode, setCurrencyCode] = useState('USD')
  const [status, setStatus] = useState<SalesOrderStatus>('pending')
  const [occurredAt, setOccurredAt] = useState(() => new Date().toISOString().slice(0, 16))
  const [customerNote, setCustomerNote] = useState('')
  const [billing, setBilling] = useState<Address>(emptyAddress())
  const [shipping, setShipping] = useState<Address>(emptyAddress())
  const [lineItems, setLineItems] = useState<DraftLineItem[]>([])
  const [pickerOpen, setPickerOpen] = useState(false)
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  function addSku(sku: SKU, label: string) {
    setLineItems(prev => [...prev, {
      key: `${sku.id}-${prev.length}`, skuId: sku.id, label,
      quantity: '1', price: (sku.price_minor_units / 100).toFixed(2),
      taxClassId: sku.tax_class_id ?? null,
    }])
    setPickerOpen(false)
  }

  function updateLineItem(key: string, field: 'quantity' | 'price', value: string) {
    setLineItems(prev => prev.map(li => (li.key === key ? { ...li, [field]: value } : li)))
  }

  function removeLineItem(key: string) {
    setLineItems(prev => prev.filter(li => li.key !== key))
  }

  const defaultTaxClassId = taxClasses.find(c => c.is_default)?.id ?? null

  // Per line: subtotal===total (no discount UI here), tax split out from it
  // using the tenant's prices_include_tax convention -- same math as the
  // product form's price preview, just applied per order line and summed.
  function lineTax(li: DraftLineItem): { minorUnits: number; taxMinorUnits: number } {
    const qty = parseInt(li.quantity, 10) || 0
    const price = Math.round(parseFloat(li.price || '0') * 100)
    const minorUnits = qty * price
    const taxClassId = li.taxClassId ?? defaultTaxClassId
    const rate = taxClassId && tenantSettings?.default_tax_country
      ? taxRates.find(r => r.tax_class_id === taxClassId && r.country === tenantSettings.default_tax_country)
      : undefined
    if (!rate) return { minorUnits, taxMinorUnits: 0 }
    const rateFraction = rate.rate_basis_points / 10000
    const taxMinorUnits = tenantSettings?.prices_include_tax
      ? minorUnits - Math.round(minorUnits / (1 + rateFraction))
      : Math.round(minorUnits * rateFraction)
    return { minorUnits, taxMinorUnits }
  }

  const totalMinorUnits = lineItems.reduce((sum, li) => {
    const { minorUnits, taxMinorUnits } = lineTax(li)
    // prices_include_tax: the line price already carries the tax, so the
    // order total is just the sum of line amounts. Exclusive prices: tax is
    // added on top of each line to reach the order total.
    return sum + minorUnits + (tenantSettings?.prices_include_tax ? 0 : taxMinorUnits)
  }, 0)
  const taxTotalMinorUnits = lineItems.reduce((sum, li) => sum + lineTax(li).taxMinorUnits, 0)

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    if (lineItems.length === 0) {
      setFormError('Add at least one line item')
      return
    }
    setSaving(true)
    try {
      const input: CreateSalesOrderInput = {
        warehouse_id: warehouseId, currency_code: currencyCode, status,
        occurred_at: new Date(occurredAt).toISOString(),
        customer_note: customerNote, billing, shipping,
        total_minor_units: totalMinorUnits,
        tax_total_minor_units: taxTotalMinorUnits,
        line_items: lineItems.map((li): CreateSalesOrderLineItemInput => {
          const qty = parseInt(li.quantity, 10) || 0
          const price = Math.round(parseFloat(li.price || '0') * 100)
          const { minorUnits, taxMinorUnits } = lineTax(li)
          return {
            sku_id: li.skuId, quantity: qty, price_minor_units: price,
            subtotal_minor_units: minorUnits, subtotal_tax_minor_units: taxMinorUnits,
            total_minor_units: minorUnits, total_tax_minor_units: taxMinorUnits,
          }
        }),
      }
      await onCreate(input)
      onClose()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to create sales order')
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
            <ArrowLeft size={11} /> Sales Order
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">New Sales Order</h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="create-sales-order-form" disabled={saving || !warehouseId}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Creating…' : 'Create Sales Order'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto p-6 space-y-5">
          <form id="create-sales-order-form" onSubmit={handleCreate} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <h3 className="text-xs font-semibold text-ink">General</h3>
              <div className="grid grid-cols-3 gap-3">
                <div>
                  <label htmlFor="so-warehouse" className={labelClass}>Warehouse</label>
                  <select id="so-warehouse" required value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
                    <option value="">Select…</option>
                    {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="so-status" className={labelClass}>Status</label>
                  <select id="so-status" value={status} onChange={e => setStatus(e.target.value as SalesOrderStatus)} className={inputClass}>
                    {STATUS_OPTIONS.map(s => <option key={s} value={s}>{s}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="so-currency" className={labelClass}>Currency</label>
                  <input id="so-currency" type="text" value={currencyCode} onChange={e => setCurrencyCode(e.target.value.toUpperCase())} className={inputClass} />
                </div>
                <div className="col-span-2">
                  <label htmlFor="so-occurred-at" className={labelClass}>Date</label>
                  <input id="so-occurred-at" type="datetime-local" value={occurredAt} onChange={e => setOccurredAt(e.target.value)} className={inputClass} />
                </div>
              </div>
              <div>
                <label htmlFor="so-note" className={labelClass}>Customer note</label>
                <input id="so-note" type="text" value={customerNote} onChange={e => setCustomerNote(e.target.value)} className={inputClass} />
              </div>
              {formError && (
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">Line items</h3>
                <button
                  type="button" onClick={() => setPickerOpen(v => !v)}
                  className="flex items-center gap-1 text-2xs font-medium text-accent hover:text-accent"
                >
                  <Plus size={12} /> Add product
                </button>
              </div>
              {pickerOpen && <ProductSkuPicker onSelect={addSku} onClose={() => setPickerOpen(false)} />}
              {lineItems.length === 0 && <p className="text-xs text-ink-4">No line items yet</p>}
              {lineItems.length > 0 && (
                <table className="w-full text-xs">
                  <thead>
                    <tr className="border-b border-line">
                      <th className="text-left py-1.5 font-medium text-ink-4">Product</th>
                      <th className="text-right py-1.5 font-medium text-ink-4 w-20">Qty</th>
                      <th className="text-right py-1.5 font-medium text-ink-4 w-24">Price</th>
                      <th className="w-8" />
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-line/60">
                    {lineItems.map(li => (
                      <tr key={li.key}>
                        <td className="py-1.5 text-ink">{li.label}</td>
                        <td className="py-1.5">
                          <input
                            type="number" min="1" value={li.quantity}
                            onChange={e => updateLineItem(li.key, 'quantity', e.target.value)}
                            className="w-16 px-2 py-1 text-xs text-right rounded bg-raised border border-line text-ink-2"
                          />
                        </td>
                        <td className="py-1.5">
                          <input
                            type="number" step="0.01" min="0" value={li.price}
                            onChange={e => updateLineItem(li.key, 'price', e.target.value)}
                            className="w-20 px-2 py-1 text-xs text-right rounded bg-raised border border-line text-ink-2"
                          />
                        </td>
                        <td className="py-1.5 text-right">
                          <button type="button" onClick={() => removeLineItem(li.key)} aria-label="Remove line item" title="Remove line item" className="p-1 -m-1 rounded hover:bg-hover">
                            <Trash2 size={12} className="text-ink-4 hover:text-danger" />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
              {lineItems.length > 0 && taxTotalMinorUnits > 0 && (
                <div className="flex justify-between text-xs text-ink-3 pt-2 border-t border-line">
                  <span>Tax</span>
                  <span>{formatMoney(taxTotalMinorUnits, currencyCode)}</span>
                </div>
              )}
              {lineItems.length > 0 && (
                <div className={clsx('flex justify-between text-xs font-semibold text-ink', taxTotalMinorUnits > 0 ? 'pt-1' : 'pt-2 border-t border-line')}>
                  <span>Total</span>
                  <span>{formatMoney(totalMinorUnits, currencyCode)}</span>
                </div>
              )}
            </div>

            <CreateAddressFields title="Billing address" prefix="new-billing" value={billing} onChange={setBilling} showPhone />
            <CreateAddressFields title="Shipping address" prefix="new-shipping" value={shipping} onChange={setShipping} />
          </form>
        </div>
      </div>
    </div>
  )
}

function CreateAddressFields({ title, prefix, value, onChange, showPhone }: { title: string; prefix: string; value: Address; onChange: (a: Address) => void; showPhone?: boolean }) {
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
          <label htmlFor={id('first-name')} className={labelClass}>{label('first name')}</label>
          <input id={id('first-name')} type="text" value={value.first_name ?? ''} onChange={set('first_name')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('last-name')} className={labelClass}>{label('last name')}</label>
          <input id={id('last-name')} type="text" value={value.last_name ?? ''} onChange={set('last_name')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('city')} className={labelClass}>{label('city')}</label>
          <input id={id('city')} type="text" value={value.city ?? ''} onChange={set('city')} className={inputClass} />
        </div>
        <div>
          <label htmlFor={id('country')} className={labelClass}>{label('country')}</label>
          <input id={id('country')} type="text" value={value.country ?? ''} onChange={set('country')} className={inputClass} />
        </div>
        {showPhone && (
          <div>
            <label htmlFor={id('phone')} className={labelClass}>{label('phone')}</label>
            <input id={id('phone')} type="tel" value={value.phone ?? ''} onChange={set('phone')} className={inputClass} />
          </div>
        )}
      </div>
    </div>
  )
}

// ── Sales Order Detail Page ──────────────────────────────────────────────────

function SalesOrderDetailPage({ salesOrderId, onClose }: { salesOrderId: string; onClose: () => void }) {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { detail, loading, error, update, remove } = useSalesOrderDetail(token ?? '', salesOrderId)
  const { activities, refresh: refreshActivities } = useSalesOrderActivities(token ?? '', salesOrderId)
  const [savingStatus, setSavingStatus] = useState(false)
  const [actionError, setActionError] = useState('')
  const [deleting, setDeleting] = useState(false)
  const [editingNote, setEditingNote] = useState(false)
  const [noteDraft, setNoteDraft] = useState('')
  const [savingNote, setSavingNote] = useState(false)

  function startEditingNote() {
    setNoteDraft(detail?.sales_order.customer_note ?? '')
    setEditingNote(true)
  }

  async function saveNote() {
    setSavingNote(true)
    setActionError('')
    try {
      await update({ customer_note: noteDraft })
      setEditingNote(false)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to save note')
    } finally {
      setSavingNote(false)
    }
  }
  // Full-screen overlay in place of the normal detail chrome, same pattern
  // as this page's own list<->detail<->create switch -- this app has no
  // router, so a one-off view is just a conditional render, not a route.
  const [printMode, setPrintMode] = useState<'picking-list' | 'invoice' | null>(null)

  // "Create label" -- a second, independent entry point for creating a
  // MyParcel shipment, alongside the existing one on the Picking page
  // (Picking.tsx's own handlePrintLabel/AddressConfirmModal usage).
  // Unlike Picking's flow, this one exports the order via MyParcel's real
  // Orders API first (ExportMyParcelOrderUseCase) -- confirmed address in
  // hand from the same modal -- before creating the actual shipment.
  const { connections } = useConnections(token ?? '')
  const myParcelConnection = connections.find(c => c.provider === 'myparcel')
  // Label creation is a 3-step flow: 'address' (AddressConfirmModal, no API
  // call) -> 'preview' (LabelPreviewModal, still no API call) -> Print
  // label actually creates the shipment (see handlePrintLabel). Same shape
  // as Picking.tsx's identical flow.
  const [labelStep, setLabelStep] = useState<'address' | 'preview' | null>(null)
  const [pendingRecipient, setPendingRecipient] = useState<RecipientAddress | null>(null)
  const [pendingCarrier, setPendingCarrier] = useState(1)
  const [pendingPackageType, setPendingPackageType] = useState(1)
  const [labelSize, setLabelSize] = useState<'A4' | 'A6'>('A4')
  const [createdShipmentId, setCreatedShipmentId] = useState<string | null>(null)
  // See Picking.tsx's identical labelTab doc.
  const [labelTab, setLabelTab] = useState<Window | null>(null)
  // See Picking.tsx's identical existingShipment doc.
  const [existingShipment, setExistingShipment] = useState<Shipment | null>(null)
  // Every shipment ever created for this order -- normally one, but
  // nothing stops a second after a failed/lost first attempt (see
  // ShipmentRepository.FindBySalesOrderID's own doc comment). Each gets
  // its own tracking timeline below.
  const [shipments, setShipments] = useState<Shipment[]>([])
  const loadShipments = useCallback(() => {
    if (!token || !detail) return
    shippingApi.listShipmentsForOrder(token, detail.sales_order.id).then(setShipments).catch(() => setShipments([]))
  }, [token, detail])
  useEffect(() => { loadShipments() }, [loadShipments])
  // auto_print_enabled: once a label exists, open the print view right
  // away instead of requiring a manual click -- still the OS print dialog,
  // not silent printing (see ShippingConnectionSettings.AutoPrintEnabled's
  // doc). Fetched once the connection is known so the "Create label"
  // button's own click handler doesn't need to await it.
  const [autoPrintEnabled, setAutoPrintEnabled] = useState(false)
  useEffect(() => {
    if (!token || !myParcelConnection) return
    shippingApi.getShippingSettings(token, myParcelConnection.id)
      .then(result => {
        setAutoPrintEnabled(result.settings?.auto_print_enabled ?? false)
        setLabelSize(result.settings?.default_label_size ?? 'A4')
      })
      .catch(() => setAutoPrintEnabled(false))
  }, [token, myParcelConnection])

  // See Picking.tsx's identical handleCreateLabelClick doc.
  async function handleCreateLabelClick() {
    if (!token || !detail || !myParcelConnection) return
    const shipments = await shippingApi.listShipmentsForOrder(token, detail.sales_order.id)
    // A cancelled shipment (wrong carrier, "Delete & create new") no longer
    // blocks creating a fresh one -- same rule CreateShipmentUseCase's own
    // duplicate-guard applies server-side.
    const active = shipments.find(s => !s.cancelled_at)
    if (active) {
      setExistingShipment(active)
      return
    }
    setLabelStep('address')
  }

  // See Picking.tsx's identical handleOpenLabel doc.
  async function handleOpenLabel(shipmentId: string) {
    if (!token) return
    try {
      setLabelTab(await openShipmentLabel(token, shipmentId, labelSize))
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to load label')
    }
  }

  // The actual shipment-creation call -- moved here from the old
  // AddressConfirmModal.onConfirm. Staff have already seen this exact data
  // laid out label-style in LabelPreviewModal and explicitly clicked "Print
  // label"; this is what that click does, then opens the label in a new
  // tab immediately (see Picking.tsx's identical handlePrintLabel doc on
  // why the blank tab opens synchronously, before either await).
  async function handlePrintLabel() {
    if (!token || !detail || !myParcelConnection || !pendingRecipient) return
    const input = { carrier: pendingCarrier, recipient: pendingRecipient }
    const tab = window.open('', '_blank')
    let shipment
    try {
      await shippingApi.exportMyParcelOrder(token, myParcelConnection.id, detail.sales_order.id, input)
      shipment = await shippingApi.createShipment(token, {
        ...input, package_type: pendingPackageType, connection_id: myParcelConnection.id, sales_order_id: detail.sales_order.id,
      })
    } catch (err) {
      tab?.close()
      throw err
    }
    setLabelStep(null)
    setCreatedShipmentId(shipment.id)
    try {
      setLabelTab(await loadShipmentLabelIntoTab(tab, token, shipment.id, labelSize))
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to load label')
    }
    refreshActivities()
    loadShipments()
  }

  if (printMode && detail) {
    return printMode === 'picking-list' ? (
      <PickingListPrintView order={detail.sales_order} lineItems={detail.line_items} onClose={() => setPrintMode(null)} />
    ) : (
      <InvoicePrintView order={detail.sales_order} lineItems={detail.line_items} onClose={() => setPrintMode(null)} />
    )
  }

  async function handleStatusChange(status: SalesOrderStatus) {
    setSavingStatus(true)
    setActionError('')
    try {
      await update({ status })
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to update status')
    } finally {
      setSavingStatus(false)
    }
  }

  async function handleDelete() {
    if (!detail) return
    if (!(await confirm({ title: `Delete sales order #${detail.sales_order.number || detail.sales_order.external_order_id}?`, message: 'This cannot be undone.', confirmLabel: 'Delete order', destructive: true }))) return
    setDeleting(true)
    setActionError('')
    try {
      await remove()
      onClose()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to delete')
      setDeleting(false)
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Sales Order
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight flex items-center gap-2">
            {detail ? `#${detail.sales_order.number || detail.sales_order.external_order_id}` : 'Sales Order'}
            {detail?.sales_order.needs_stock_review && (
              <span className="px-2 py-0.5 rounded-full text-2xs font-medium bg-amber-500/15 text-warning">Stock issue</span>
            )}
            {detail?.sales_order.needs_sku_review && (
              <span className="px-2 py-0.5 rounded-full text-2xs font-medium bg-amber-500/15 text-warning">Unmatched SKU</span>
            )}
          </h1>
        </div>
        {detail && (
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPrintMode('picking-list')}
              title="Print picking list"
              className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
            >
              <Printer size={13} /> Picking list
            </button>
            <button
              onClick={() => setPrintMode('invoice')}
              title="Print invoice"
              className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
            >
              <Printer size={13} /> Invoice
            </button>
            {myParcelConnection && !createdShipmentId && (
              <button
                onClick={handleCreateLabelClick}
                title="Create a MyParcel shipping label for this order"
                className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
              >
                Create label
              </button>
            )}
            {createdShipmentId && (
              <button
                onClick={() => createdShipmentId && handleOpenLabel(createdShipmentId)}
                title="Print shipping label"
                className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-line bg-raised text-xs font-medium text-ink-2 hover:bg-hover transition-colors"
              >
                {/* auto_print_enabled already opened the print view once on
                    creation -- this button is then for re-printing (paper
                    jam, dismissed dialog, etc), not the first print. */}
                <Printer size={13} /> {autoPrintEnabled ? 'Retry print' : 'Print label'}
              </button>
            )}
            <select
              value={detail.sales_order.status}
              disabled={savingStatus}
              onChange={e => handleStatusChange(e.target.value as SalesOrderStatus)}
              className={clsx(
                'px-2.5 py-1.5 rounded-lg text-xs font-medium border border-line outline-none focus-visible:ring-2 focus-visible:ring-focus/60 cursor-pointer disabled:opacity-50',
                'bg-raised text-ink',
              )}
            >
              {STATUS_OPTIONS.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
            <button
              onClick={handleDelete}
              disabled={deleting}
              title="Delete sales order"
              className="w-8 h-8 flex items-center justify-center rounded-lg border border-line hover:bg-hover disabled:opacity-50 transition-colors"
            >
              <Trash2 size={14} className="text-ink-3" />
            </button>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-5xl mx-auto p-6 space-y-5">
          {loading && <p className="text-xs text-ink-4">Loading…</p>}
          {(error || actionError) && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error || actionError}</p>}

          {detail && (
            <div className="grid grid-cols-1 lg:grid-cols-[1fr_320px] gap-5 items-start">
              {/* Main column */}
              <div className="space-y-5 min-w-0">
                <div className="rounded-xl border border-line p-4 space-y-3">
                  <h3 className="text-xs font-semibold text-ink">Line items</h3>
                  <table className="w-full text-xs">
                    <thead>
                      <tr className="border-b border-line">
                        <th className="text-left py-1.5 font-medium text-ink-4">Product</th>
                        <th className="text-right py-1.5 font-medium text-ink-4">Qty</th>
                        <th className="text-right py-1.5 font-medium text-ink-4">Price</th>
                        <th className="text-right py-1.5 font-medium text-ink-4">Total</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-line/60">
                      {detail.line_items.map(li => (
                        <tr key={li.id}>
                          <td className="py-1.5">
                            <p className="text-ink">{li.name}</p>
                            {li.sku_code && (
                              <p className="text-2xs text-ink-4">{li.sku_code}</p>
                            )}
                            {li.barcode && (
                              <p className="text-2xs text-ink-4">{li.barcode}</p>
                            )}
                            {li.sold_expiry_date && (
                              <p className="text-2xs text-ink-4" title="Reported by WooCommerce at checkout -- may not match the batch actually allocated">
                                Sold exp. {formatDate(li.sold_expiry_date)}
                              </p>
                            )}
                            {!li.sku_id && (
                              <p className="text-2xs text-warning">Not linked to a product yet</p>
                            )}
                          </td>
                          <td className="py-1.5 text-right text-ink-2">{li.quantity}</td>
                          <td className="py-1.5 text-right text-ink-2">
                            {formatMoney(li.price_minor_units, detail.sales_order.currency_code)}
                          </td>
                          <td className="py-1.5 text-right font-medium text-ink">
                            {formatMoney(li.total_minor_units, detail.sales_order.currency_code)}
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>

                  <div className="pt-2 border-t border-line space-y-1 text-xs">
                    <div className="flex justify-between text-ink-3">
                      <span>Discount</span>
                      <span>{formatMoney(detail.sales_order.discount_total_minor_units, detail.sales_order.currency_code)}</span>
                    </div>
                    <div className="flex justify-between text-ink-3">
                      <span>Shipping</span>
                      <span>{formatMoney(detail.sales_order.shipping_total_minor_units, detail.sales_order.currency_code)}</span>
                    </div>
                    <div className="flex justify-between text-ink-3">
                      <span>Tax</span>
                      <span>{formatMoney(detail.sales_order.tax_total_minor_units, detail.sales_order.currency_code)}</span>
                    </div>
                    <div className="flex justify-between font-semibold text-ink pt-1">
                      <span>Total</span>
                      <span>{formatMoney(detail.sales_order.total_minor_units, detail.sales_order.currency_code)}</span>
                    </div>
                  </div>
                </div>

                <div className="rounded-xl border border-line p-4 space-y-1 text-xs">
                  <h3 className="text-xs font-semibold text-ink mb-2">Payment</h3>
                  <p className="text-ink-3">Source: <span className="text-ink capitalize">{detail.sales_order.source}</span></p>
                  {(detail.sales_order.payment_method_title || detail.sales_order.payment_method) && (
                    <p className="text-ink-3">Payment: <span className="text-ink">{detail.sales_order.payment_method_title || detail.sales_order.payment_method}</span></p>
                  )}
                  {detail.sales_order.shipping_method_title && (
                    <p className="text-ink-3">Shipping method: <span className="text-ink">{detail.sales_order.shipping_method_title}</span></p>
                  )}
                </div>

                {shipments.length > 0 && token && (
                  <div className="rounded-xl border border-line p-4 space-y-3 text-xs">
                    <p className="font-medium text-ink-2">Shipments</p>
                    {shipments.map(s => <ShipmentTrackingSection key={s.id} token={token} shipment={s} onCancelled={loadShipments} />)}
                  </div>
                )}

                {activities.length > 0 && (
                  <div className="rounded-xl border border-line p-4 space-y-2 text-xs">
                    <p className="font-medium text-ink-2">Timeline</p>
                    {activities.map(a => (
                      <div key={a.id} className="flex justify-between gap-3 text-ink-3">
                        <span className="text-ink">{a.summary}</span>
                        <span className="shrink-0">{new Date(a.created_at).toLocaleString()}</span>
                      </div>
                    ))}
                  </div>
                )}
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
                      <label htmlFor="sales-order-note" className="sr-only">Notes</label>
                      <textarea
                        id="sales-order-note"
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
                  ) : detail.sales_order.customer_note ? (
                    <p className="text-ink whitespace-pre-wrap">{detail.sales_order.customer_note}</p>
                  ) : (
                    <p className="text-ink-4">No notes</p>
                  )}
                </div>

                <div className="rounded-xl border border-line p-4 space-y-3 text-xs">
                  <h3 className="text-xs font-semibold text-ink">Customer</h3>
                  <p>
                    {detail.sales_order.customer_id ? (
                      <Link to={`/customers/${detail.sales_order.customer_id}`} className="text-accent hover:underline">
                        {customerName(detail.sales_order)}
                      </Link>
                    ) : (
                      <span className="text-ink">{customerName(detail.sales_order)}</span>
                    )}
                  </p>

                  {detail.sales_order.billing?.email && (
                    <div>
                      <h4 className="text-2xs font-semibold uppercase tracking-wider text-ink-4 mb-1">Contact information</h4>
                      <p className="text-ink">{detail.sales_order.billing.email}</p>
                    </div>
                  )}

                  <div>
                    <h4 className="text-2xs font-semibold uppercase tracking-wider text-ink-4 mb-1">Shipping address</h4>
                    <AddressLines address={detail.sales_order.shipping} />
                  </div>

                  <div>
                    <h4 className="text-2xs font-semibold uppercase tracking-wider text-ink-4 mb-1">Billing address</h4>
                    {addressesEqual(detail.sales_order.billing, detail.sales_order.shipping) ? (
                      <p className="text-ink-4">Same as shipping address</p>
                    ) : (
                      <AddressLines address={detail.sales_order.billing} />
                    )}
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>

      {labelStep === 'address' && detail && (
        <AddressConfirmModal
          token={token ?? ''}
          initialAddress={detail.sales_order.shipping}
          fallbackEmail={detail.sales_order.billing?.email}
          initialRecipient={pendingRecipient ?? undefined}
          initialCarrier={pendingCarrier}
          initialPackageType={pendingPackageType}
          onNext={({ carrier, packageType, recipient }) => { setPendingCarrier(carrier); setPendingPackageType(packageType); setPendingRecipient(recipient); setLabelStep('preview') }}
          onClose={() => setLabelStep(null)}
        />
      )}
      {labelStep === 'preview' && pendingRecipient && (
        <LabelPreviewModal
          recipient={pendingRecipient}
          carrier={pendingCarrier}
          packageType={pendingPackageType}
          labelSize={labelSize}
          onLabelSizeChange={setLabelSize}
          onBack={() => setLabelStep('address')}
          onPrint={handlePrintLabel}
          onClose={() => setLabelStep(null)}
        />
      )}
      {labelTab && (
        <LabelOpenedNotice onPrint={() => labelTab.print()} onDismiss={() => setLabelTab(null)} />
      )}
      {existingShipment && token && (
        <ExistingShipmentConfirmModal
          token={token}
          shipment={existingShipment}
          onCancel={() => setExistingShipment(null)}
          onReprint={() => {
            setCreatedShipmentId(existingShipment.id)
            handleOpenLabel(existingShipment.id)
            setExistingShipment(null)
          }}
          onCreateNew={() => {
            setExistingShipment(null)
            setLabelStep('address')
          }}
        />
      )}
      {confirmDialog}
    </div>
  )
}

// One shipment's carrier/barcode header plus its full tracking timeline
// (newest first) and a "Refresh from provider" button -- fetches the
// shipment's current status right now instead of waiting for the next
// webhook/reconciliation tick, and saves it as a new timeline entry (see
// RefreshShipmentTrackingUseCase).
function ShipmentTrackingSection({ token, shipment, onCancelled }: { token: string; shipment: Shipment; onCancelled: () => void }) {
  const { confirm, confirmDialog } = useConfirm()
  const [events, setEvents] = useState<ShipmentTrackingEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState('')
  const [cancelling, setCancelling] = useState(false)
  const [cancelMessage, setCancelMessage] = useState('')

  const load = useCallback(() => {
    setLoading(true)
    shippingApi.listShipmentTrackingEvents(token, shipment.id)
      .then(setEvents)
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load tracking'))
      .finally(() => setLoading(false))
  }, [token, shipment.id])
  useEffect(() => { load() }, [load])

  async function handleRefresh() {
    setRefreshing(true)
    setError('')
    try {
      const event = await shippingApi.refreshShipmentTracking(token, shipment.id)
      setEvents(prev => [event, ...prev])
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to fetch from provider')
    } finally {
      setRefreshing(false)
    }
  }

  // Wrong carrier picked -- attempts MyParcel's real delete, then always
  // marks the shipment cancelled locally regardless of that outcome (see
  // cancelShipment's doc comment). onCancelled() re-fetches the parent's
  // shipment list so this card picks up cancelled_at/cancel_note and
  // "Create label" up top stops treating this as the active shipment.
  async function handleCancel() {
    if (!(await confirm({ title: 'Cancel this shipment?', message: 'This cannot be undone.', confirmLabel: 'Cancel shipment', destructive: true }))) return
    setCancelling(true)
    setError('')
    try {
      const result = await shippingApi.cancelShipment(token, shipment.id)
      setCancelMessage(result.message)
      onCancelled()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to cancel shipment')
    } finally {
      setCancelling(false)
    }
  }

  const carrierLabel = CARRIERS.find(c => c.id === shipment.carrier)?.label ?? `Carrier ${shipment.carrier}`

  return (
    <div className="border border-line rounded-lg p-3 space-y-2">
      <div className="flex items-center justify-between gap-3">
        <div>
          <span className="text-ink font-medium">{carrierLabel}</span>
          <span className="text-ink-4">
            {' · '}
            {shipment.barcode ? (
              shipment.tracking_url ? (
                <a href={shipment.tracking_url} target="_blank" rel="noreferrer" className="text-accent hover:underline">
                  {shipment.barcode}
                </a>
              ) : (
                shipment.barcode
              )
            ) : (
              'no barcode yet'
            )}
          </span>
          {shipment.cancelled_at && (
            <span className="ml-2 text-2xs px-1.5 py-0.5 rounded-full bg-red-50 dark:bg-red-500/10 text-danger">Cancelled</span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={handleRefresh} disabled={refreshing}
            className="px-2 py-1 rounded bg-fill text-ink-3 hover:text-ink-2 disabled:opacity-50 transition-colors"
          >
            {refreshing ? 'Fetching…' : 'Refresh from provider'}
          </button>
          {!shipment.cancelled_at && (
            <button
              type="button" onClick={handleCancel} disabled={cancelling}
              title="Wrong carrier? Cancel this shipment so a corrected one can be created"
              aria-label="Delete"
              className="w-7 h-7 flex items-center justify-center rounded bg-fill hover:bg-hover disabled:opacity-50 transition-colors"
            >
              {cancelling ? (
                <span className="text-2xs text-danger">…</span>
              ) : (
                <Trash2 size={13} className="text-ink-4 hover:text-danger" />
              )}
            </button>
          )}
        </div>
      </div>

      {(shipment.cancel_note || cancelMessage) && <p className="text-warning">{shipment.cancel_note || cancelMessage}</p>}
      {error && <p className="text-danger">{error}</p>}

      {loading ? (
        <p className="text-ink-4">Loading tracking…</p>
      ) : events.length === 0 ? (
        <p className="text-ink-4">No tracking updates yet.</p>
      ) : (
        <div className="space-y-1">
          {events.map(e => (
            <div key={e.id} className="flex justify-between gap-3 text-ink-3">
              <span className="text-ink">{e.status_label}</span>
              <span className="shrink-0">{new Date(e.occurred_at).toLocaleString()} · {e.source}</span>
            </div>
          ))}
        </div>
      )}
      {confirmDialog}
    </div>
  )
}

function AddressLines({ address }: { address?: { first_name?: string; last_name?: string; company?: string; address_1?: string; address_2?: string; city?: string; state?: string; postcode?: string; country?: string; email?: string; phone?: string } }) {
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
