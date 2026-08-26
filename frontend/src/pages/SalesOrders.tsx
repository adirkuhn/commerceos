import { useState } from 'react'
import { ArrowLeft, ShoppingCart, Plus, Trash2, ChevronUp, ChevronDown, Printer } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSalesOrders } from '../oms/hooks/useSalesOrders'
import { useSalesOrderDetail } from '../oms/hooks/useSalesOrderDetail'
import { useSalesOrderActivities } from '../oms/hooks/useSalesOrderActivities'
import type { SalesOrder, SalesOrderStatus, SalesOrderDetail, CreateSalesOrderInput, CreateSalesOrderLineItemInput, Address, SalesOrderSortField, SortDirection } from '../oms/types'
import { PickingListPrintView } from './print/PickingListPrintView'
import { InvoicePrintView } from './print/InvoicePrintView'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { ProductSkuPicker } from './Procurement'
import type { SKU } from '../pms/types'
import { useTaxClasses } from '../tax/hooks/useTaxClasses'
import { useTaxRates } from '../tax/hooks/useTaxRates'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'

const PAGE_SIZE = 20
const STATUS_OPTIONS: SalesOrderStatus[] = ['pending', 'processing', 'on-hold', 'completed', 'cancelled', 'refunded', 'failed']

const STATUS_COLORS: Record<SalesOrderStatus, string> = {
  pending: 'bg-gray-500/15 text-gray-400',
  processing: 'bg-blue-500/15 text-blue-400',
  'on-hold': 'bg-amber-500/15 text-amber-400',
  completed: 'bg-green-500/15 text-green-400',
  cancelled: 'bg-gray-500/15 text-gray-400',
  refunded: 'bg-purple-500/15 text-purple-400',
  failed: 'bg-red-500/15 text-red-400',
}

function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

function customerName(order: SalesOrder): string {
  const name = [order.billing?.first_name, order.billing?.last_name].filter(Boolean).join(' ')
  return name || order.billing?.email || '—'
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
        'px-3 py-2.5 font-medium uppercase tracking-wider cursor-pointer select-none hover:dark:text-gray-300 hover:text-gray-600 transition-colors',
        align === 'right' ? 'text-right' : 'text-left',
        active ? 'dark:text-gray-300 text-gray-600' : 'dark:text-gray-500 text-gray-400',
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
  const { token } = useAuth()
  const [page, setPage] = useState(1)
  const [sortBy, setSortBy] = useState<SalesOrderSortField>('occurred_at')
  const [sortDir, setSortDir] = useState<SortDirection>('desc')
  const [selectedId, setSelectedId] = useState<string | null | 'new'>(null)
  const { salesOrders, total, loading, error, create, refresh, bulkUpdate } = useSalesOrders(token ?? '', { page, perPage: PAGE_SIZE, sortBy, sortDir })
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
    if (!window.confirm(`Apply this change to ${count} order${count === 1 ? '' : 's'}?`)) return
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
        onClose={() => setSelectedId(null)}
      />
    )
  }
  if (selectedId !== null) {
    return <SalesOrderDetailPage salesOrderId={selectedId} onClose={() => { setSelectedId(null); refresh() }} />
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center justify-end gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <button
          onClick={() => setSelectedId('new')}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 transition-colors font-medium"
        >
          <Plus size={13} /> New Sales Order
        </button>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{total} sales order{total === 1 ? '' : 's'}</span>
        {error && <span className="text-xs text-red-400 ml-auto">{error}</span>}
      </div>

      {/* Bulk action bar */}
      {checkedIds.size > 0 && (
        <div className="flex items-center gap-3 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-blue-50 shrink-0">
          <span className="text-xs font-medium dark:text-gray-200 text-gray-700">{checkedIds.size} selected</span>
          <select
            aria-label="Bulk change status" value={bulkStatus} onChange={e => setBulkStatus(e.target.value)} disabled={bulkApplying}
            className="px-2 py-1 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none disabled:opacity-50"
          >
            <option value="">Status…</option>
            {STATUS_OPTIONS.map(s => <option key={s} value={s}>{s}</option>)}
          </select>
          <select
            aria-label="Bulk change fulfilled" value={bulkFulfilled} onChange={e => setBulkFulfilled(e.target.value)} disabled={bulkApplying}
            className="px-2 py-1 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none disabled:opacity-50"
          >
            <option value="">Fulfilled…</option>
            <option value="true">Fulfilled</option>
            <option value="false">Unfulfilled</option>
          </select>
          <button
            onClick={applyBulkChange}
            disabled={bulkApplying || (!bulkStatus && !bulkFulfilled)}
            className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {bulkApplying ? 'Applying…' : 'Apply'}
          </button>
          <button
            onClick={() => { setCheckedIds(new Set()); setBulkStatus(''); setBulkFulfilled(''); setBulkError('') }}
            disabled={bulkApplying}
            className="text-xs font-medium dark:text-gray-400 text-gray-500 hover:dark:text-gray-200 hover:text-gray-700 disabled:opacity-50"
          >
            Clear
          </button>
          {bulkError && <span className="text-xs text-red-400 ml-auto">{bulkError}</span>}
        </div>
      )}

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-3 py-2.5 w-8">
                <input
                  type="checkbox" aria-label="Select all orders on this page"
                  checked={allOnPageChecked} onChange={toggleAllOnPage}
                  className="rounded border dark:border-[#2D3748] border-gray-300"
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
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && salesOrders.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No sales orders yet</td></tr>
            )}
            {salesOrders.map(o => (
              <tr
                key={o.id}
                onClick={() => setSelectedId(o.id)}
                className="group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors cursor-pointer"
              >
                <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
                  <input
                    type="checkbox" aria-label={`Select order #${o.number || o.external_order_id}`}
                    checked={checkedIds.has(o.id)} onChange={() => toggleOne(o.id)}
                    className="rounded border dark:border-[#2D3748] border-gray-300"
                  />
                </td>
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-[#2D3748] bg-gray-200 flex items-center justify-center">
                    <ShoppingCart size={12} className="dark:text-gray-500 text-gray-400" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[200px]">
                  <p className="font-medium dark:text-gray-100 text-gray-900 truncate">#{o.number || o.external_order_id}</p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 truncate">{new Date(o.occurred_at).toLocaleString()}</p>
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500 max-w-[220px] truncate">{customerName(o)}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{o.source}</td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium', STATUS_COLORS[o.status])}>
                      {o.status}
                    </span>
                    <span className={clsx(
                      'px-2 py-0.5 rounded-full text-[10px] font-medium',
                      o.fulfilled ? 'bg-green-500/15 text-green-400' : 'bg-gray-500/15 text-gray-400',
                    )}>
                      {o.fulfilled ? 'Fulfilled' : 'Unfulfilled'}
                    </span>
                    {(o.needs_stock_review || o.needs_sku_review) && (
                      <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-amber-500/15 text-amber-500">
                        Needs review
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-3 py-2.5 text-right font-medium dark:text-gray-100 text-gray-900">
                  {formatMoney(o.total_minor_units, o.currency_code)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
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

  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500 transition-shadow'

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Sales Order
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">New Sales Order</h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-semibold dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="create-sales-order-form" disabled={saving || !warehouseId}
            className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Creating…' : 'Create Sales Order'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto p-6 space-y-5">
          <form id="create-sales-order-form" onSubmit={handleCreate} className="space-y-5">
            <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
              <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">General</h3>
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
                <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>
              )}
            </div>

            <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">Line items</h3>
                <button
                  type="button" onClick={() => setPickerOpen(v => !v)}
                  className="flex items-center gap-1 text-[11px] font-medium text-blue-500 hover:text-blue-600"
                >
                  <Plus size={12} /> Add product
                </button>
              </div>
              {pickerOpen && <ProductSkuPicker onSelect={addSku} onClose={() => setPickerOpen(false)} />}
              {lineItems.length === 0 && <p className="text-xs dark:text-gray-500 text-gray-400">No line items yet</p>}
              {lineItems.length > 0 && (
                <table className="w-full text-xs">
                  <thead>
                    <tr className="border-b dark:border-[#2D3748] border-gray-200">
                      <th className="text-left py-1.5 font-medium dark:text-gray-500 text-gray-400">Product</th>
                      <th className="text-right py-1.5 font-medium dark:text-gray-500 text-gray-400 w-20">Qty</th>
                      <th className="text-right py-1.5 font-medium dark:text-gray-500 text-gray-400 w-24">Price</th>
                      <th className="w-8" />
                    </tr>
                  </thead>
                  <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                    {lineItems.map(li => (
                      <tr key={li.key}>
                        <td className="py-1.5 dark:text-gray-200 text-gray-800">{li.label}</td>
                        <td className="py-1.5">
                          <input
                            type="number" min="1" value={li.quantity}
                            onChange={e => updateLineItem(li.key, 'quantity', e.target.value)}
                            className="w-16 px-2 py-1 text-xs text-right rounded dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700"
                          />
                        </td>
                        <td className="py-1.5">
                          <input
                            type="number" step="0.01" min="0" value={li.price}
                            onChange={e => updateLineItem(li.key, 'price', e.target.value)}
                            className="w-20 px-2 py-1 text-xs text-right rounded dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700"
                          />
                        </td>
                        <td className="py-1.5 text-right">
                          <button type="button" onClick={() => removeLineItem(li.key)}>
                            <Trash2 size={12} className="dark:text-gray-500 text-gray-400 hover:text-red-400" />
                          </button>
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              )}
              {lineItems.length > 0 && taxTotalMinorUnits > 0 && (
                <div className="flex justify-between text-xs dark:text-gray-400 text-gray-500 pt-2 border-t dark:border-[#2D3748] border-gray-200">
                  <span>Tax</span>
                  <span>{formatMoney(taxTotalMinorUnits, currencyCode)}</span>
                </div>
              )}
              {lineItems.length > 0 && (
                <div className={clsx('flex justify-between text-xs font-semibold dark:text-gray-100 text-gray-900', taxTotalMinorUnits > 0 ? 'pt-1' : 'pt-2 border-t dark:border-[#2D3748] border-gray-200')}>
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
  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500 transition-shadow'
  const set = (field: keyof Address) => (e: React.ChangeEvent<HTMLInputElement>) => onChange({ ...value, [field]: e.target.value })
  const id = (field: string) => `${prefix}-${field}`
  const label = (text: string) => `${title.replace(' address', '')} ${text}`

  return (
    <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
      <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">{title}</h3>
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
  const { token } = useAuth()
  const { detail, loading, error, update, remove } = useSalesOrderDetail(token ?? '', salesOrderId)
  const { activities } = useSalesOrderActivities(token ?? '', salesOrderId)
  const [savingStatus, setSavingStatus] = useState(false)
  const [actionError, setActionError] = useState('')
  const [deleting, setDeleting] = useState(false)
  // Full-screen overlay in place of the normal detail chrome, same pattern
  // as this page's own list<->detail<->create switch -- this app has no
  // router, so a one-off view is just a conditional render, not a route.
  const [printMode, setPrintMode] = useState<'picking-list' | 'invoice' | null>(null)

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
    if (!window.confirm(`Delete sales order #${detail.sales_order.number || detail.sales_order.external_order_id}? This cannot be undone.`)) return
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
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Sales Order
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight flex items-center gap-2">
            {detail ? `#${detail.sales_order.number || detail.sales_order.external_order_id}` : 'Sales Order'}
            {detail?.sales_order.needs_stock_review && (
              <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-amber-500/15 text-amber-500">Stock issue</span>
            )}
            {detail?.sales_order.needs_sku_review && (
              <span className="px-2 py-0.5 rounded-full text-[10px] font-medium bg-amber-500/15 text-amber-500">Unmatched SKU</span>
            )}
          </h1>
        </div>
        {detail && (
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPrintMode('picking-list')}
              title="Print picking list"
              className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-medium dark:text-gray-300 text-gray-700 dark:hover:bg-[#2D3748] hover:bg-gray-100 transition-colors"
            >
              <Printer size={13} /> Picking list
            </button>
            <button
              onClick={() => setPrintMode('invoice')}
              title="Print invoice"
              className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-medium dark:text-gray-300 text-gray-700 dark:hover:bg-[#2D3748] hover:bg-gray-100 transition-colors"
            >
              <Printer size={13} /> Invoice
            </button>
            <select
              value={detail.sales_order.status}
              disabled={savingStatus}
              onChange={e => handleStatusChange(e.target.value as SalesOrderStatus)}
              className={clsx(
                'px-2.5 py-1.5 rounded-lg text-xs font-medium border dark:border-[#2D3748] border-gray-200 outline-none cursor-pointer disabled:opacity-50',
                'dark:bg-[#1F2937] bg-white dark:text-gray-200 text-gray-800',
              )}
            >
              {STATUS_OPTIONS.map(s => <option key={s} value={s}>{s}</option>)}
            </select>
            <button
              onClick={handleDelete}
              disabled={deleting}
              title="Delete sales order"
              className="w-8 h-8 flex items-center justify-center rounded-lg border dark:border-[#2D3748] border-gray-200 dark:hover:bg-[#2D3748] hover:bg-gray-100 disabled:opacity-50 transition-colors"
            >
              <Trash2 size={14} className="dark:text-gray-400 text-gray-500" />
            </button>
          </div>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto p-6 space-y-5">
          {loading && <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
          {(error || actionError) && <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error || actionError}</p>}

          {detail && (
            <>
              <div className="grid grid-cols-2 gap-4">
                <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-1">
                  <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900 mb-2">Billing</h3>
                  <AddressLines address={detail.sales_order.billing} />
                </div>
                <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-1">
                  <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900 mb-2">Shipping</h3>
                  <AddressLines address={detail.sales_order.shipping} />
                </div>
              </div>

              <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
                <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">Line items</h3>
                <table className="w-full text-xs">
                  <thead>
                    <tr className="border-b dark:border-[#2D3748] border-gray-200">
                      <th className="text-left py-1.5 font-medium dark:text-gray-500 text-gray-400">Product</th>
                      <th className="text-right py-1.5 font-medium dark:text-gray-500 text-gray-400">Qty</th>
                      <th className="text-right py-1.5 font-medium dark:text-gray-500 text-gray-400">Price</th>
                      <th className="text-right py-1.5 font-medium dark:text-gray-500 text-gray-400">Total</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                    {detail.line_items.map(li => (
                      <tr key={li.id}>
                        <td className="py-1.5">
                          <p className="dark:text-gray-200 text-gray-800">{li.name}</p>
                          {li.sku_code && (
                            <p className="text-[10px] dark:text-gray-500 text-gray-400">{li.sku_code}</p>
                          )}
                          {li.barcode && (
                            <p className="text-[10px] dark:text-gray-500 text-gray-400">{li.barcode}</p>
                          )}
                          {!li.sku_id && (
                            <p className="text-[10px] text-amber-500">Not linked to a product yet</p>
                          )}
                        </td>
                        <td className="py-1.5 text-right dark:text-gray-300 text-gray-700">{li.quantity}</td>
                        <td className="py-1.5 text-right dark:text-gray-300 text-gray-700">
                          {formatMoney(li.price_minor_units, detail.sales_order.currency_code)}
                        </td>
                        <td className="py-1.5 text-right font-medium dark:text-gray-100 text-gray-900">
                          {formatMoney(li.total_minor_units, detail.sales_order.currency_code)}
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>

                <div className="pt-2 border-t dark:border-[#2D3748] border-gray-200 space-y-1 text-xs">
                  <div className="flex justify-between dark:text-gray-400 text-gray-500">
                    <span>Discount</span>
                    <span>{formatMoney(detail.sales_order.discount_total_minor_units, detail.sales_order.currency_code)}</span>
                  </div>
                  <div className="flex justify-between dark:text-gray-400 text-gray-500">
                    <span>Shipping</span>
                    <span>{formatMoney(detail.sales_order.shipping_total_minor_units, detail.sales_order.currency_code)}</span>
                  </div>
                  <div className="flex justify-between dark:text-gray-400 text-gray-500">
                    <span>Tax</span>
                    <span>{formatMoney(detail.sales_order.tax_total_minor_units, detail.sales_order.currency_code)}</span>
                  </div>
                  <div className="flex justify-between font-semibold dark:text-gray-100 text-gray-900 pt-1">
                    <span>Total</span>
                    <span>{formatMoney(detail.sales_order.total_minor_units, detail.sales_order.currency_code)}</span>
                  </div>
                </div>
              </div>

              <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-1 text-xs">
                <p className="dark:text-gray-400 text-gray-500">Source: <span className="dark:text-gray-200 text-gray-800 capitalize">{detail.sales_order.source}</span></p>
                {(detail.sales_order.payment_method_title || detail.sales_order.payment_method) && (
                  <p className="dark:text-gray-400 text-gray-500">Payment: <span className="dark:text-gray-200 text-gray-800">{detail.sales_order.payment_method_title || detail.sales_order.payment_method}</span></p>
                )}
                {detail.sales_order.shipping_method_title && (
                  <p className="dark:text-gray-400 text-gray-500">Shipping method: <span className="dark:text-gray-200 text-gray-800">{detail.sales_order.shipping_method_title}</span></p>
                )}
                {detail.sales_order.customer_note && (
                  <p className="dark:text-gray-400 text-gray-500">Note: <span className="dark:text-gray-200 text-gray-800">{detail.sales_order.customer_note}</span></p>
                )}
              </div>

              {activities.length > 0 && (
                <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-2 text-xs">
                  <p className="font-medium dark:text-gray-300 text-gray-700">Activity</p>
                  {activities.map(a => (
                    <div key={a.id} className="flex justify-between gap-3 dark:text-gray-400 text-gray-500">
                      <span className="dark:text-gray-200 text-gray-800">{a.summary}</span>
                      <span className="shrink-0">{new Date(a.created_at).toLocaleString()}</span>
                    </div>
                  ))}
                </div>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  )
}

function AddressLines({ address }: { address?: { first_name?: string; last_name?: string; company?: string; address_1?: string; address_2?: string; city?: string; state?: string; postcode?: string; country?: string; email?: string; phone?: string } }) {
  if (!address) return <p className="text-xs dark:text-gray-500 text-gray-400">—</p>
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

  if (lines.length === 0) return <p className="text-xs dark:text-gray-500 text-gray-400">—</p>

  return (
    <>
      {lines.map((line, i) => (
        <p key={i} className="text-xs dark:text-gray-300 text-gray-600">{line}</p>
      ))}
    </>
  )
}
