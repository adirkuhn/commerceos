import { useState, useEffect, useRef, Fragment } from 'react'
import { Plus, ShoppingBag, Trash2, Edit3, ArrowLeft, Search, X, Check, Upload, Sparkles, PackageCheck, FileText, AlertTriangle, Loader2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { usePurchaseOrders } from '../procurement/hooks/usePurchaseOrders'
import { usePurchaseOrderDetail } from '../procurement/hooks/usePurchaseOrderDetail'
import { useReceivingRecords } from '../procurement/hooks/useReceivingRecords'
import { useBookingDiscrepancies } from '../procurement/hooks/useBookingDiscrepancies'
import { pollParsingJob } from '../procurement/pollParsingJob'
import { useSuppliers } from '../suppliers/hooks/useSuppliers'
import type { Supplier } from '../suppliers/types'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { useProducts } from '../pms/hooks/useProducts'
import * as pmsApi from '../pms/api'
import * as procurementApi from '../procurement/api'
import * as inventoryApi from '../inventory/api'
import * as integrationsApi from '../integrations/api'
import type {
  PurchaseOrder, PurchaseOrderStatus, PurchaseOrderLineItem, PurchaseOrderDocument,
  CreatePurchaseOrderInput, UpdatePurchaseOrderInput, AddLineItemInput, UpdateLineItemInput,
  ReceivingRecord, ReceivingRecordDetail, ReceivingDiscrepancy, DiscrepancyType, LineItemType,
} from '../procurement/types'
import type { SKU } from '../pms/types'
import type { Warehouse } from '../inventory/types'

const STATUS_TABS: (PurchaseOrderStatus | 'all')[] = ['all', 'draft', 'sent', 'partially_received', 'received', 'needs_review', 'cancelled']

// Same list as Products.tsx's SKU currency select, for consistency.
const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

function todayDateString(): string {
  return new Date().toISOString().slice(0, 10)
}

const STATUS_STYLE: Record<PurchaseOrderStatus, string> = {
  draft: 'bg-gray-500/15 text-gray-400',
  sent: 'bg-blue-500/15 text-blue-400',
  partially_received: 'bg-amber-500/15 text-amber-400',
  received: 'bg-green-500/15 text-green-400',
  cancelled: 'bg-red-500/15 text-red-400',
  needs_review: 'bg-amber-500/20 text-amber-400',
}

function formatMoney(minorUnits: number, currencyCode: string): string {
  return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
}

// '' and 'product' both mean "an actual item" -- everything else is a
// charge/adjustment row that must never be forced through SKU matching.
const LINE_TYPE_LABELS: Record<Exclude<LineItemType, ''>, string> = {
  product: 'Product',
  shipping: 'Shipping',
  discount: 'Discount',
  credit_note: 'Credit Note',
  fee: 'Fee',
  other_charge: 'Other Charge',
}

function isChargeLineType(type: LineItemType): boolean {
  return type !== '' && type !== 'product'
}

// isReductionLineType mirrors backend LineItemType.IsReduction --
// unit_cost_minor_units is always entered as a non-negative amount (the
// "Unit cost" field has min="0"), so a discount/credit_note row's amount
// must be subtracted from the total here rather than relying on the stored
// value ever being negative.
function isReductionLineType(type: LineItemType): boolean {
  return type === 'discount' || type === 'credit_note'
}

function signedLineItemAmount(li: Pick<PurchaseOrderLineItem, 'line_type' | 'quantity' | 'unit_cost_minor_units'>): number {
  const amount = li.unit_cost_minor_units * li.quantity
  return isReductionLineType(li.line_type) ? -amount : amount
}

// Backend's exact wording for procurementDomain.ErrAIParsingNotConfigured
// (application/procurement/extract_line_items_from_document.go) -- swapped
// for a friendlier message pointing at the manual-entry fallback that's
// already available right below (the "+ Add Line Item" button never
// depends on a successful parse).
const AI_NOT_CONFIGURED_MESSAGE = 'AI document parsing is not configured'

function friendlyParsingError(errorMessage: string): string {
  if (errorMessage === AI_NOT_CONFIGURED_MESSAGE) {
    return 'No AI provider is configured, so this document couldn’t auto-parse into line items. The document is saved below — add line items manually, or connect an AI provider in Settings → Integrations.'
  }
  return errorMessage || 'Failed to parse document'
}

const RECEIVING_STATUS_STYLE: Record<ReceivingRecord['status'], string> = {
  pending_review: 'bg-amber-500/20 text-amber-400',
  applied: 'bg-green-500/15 text-green-400',
}

const DISCREPANCY_LABEL: Record<DiscrepancyType, string> = {
  quantity_mismatch: 'Quantity mismatch',
  missing_item: 'Missing from invoice',
  unexpected_item: 'Not on purchase order',
  expiry_mismatch: 'Expiry mismatch',
}

export default function Procurement() {
  const { token } = useAuth()
  const { purchaseOrders, loading, error, create, remove, refresh } = usePurchaseOrders(token ?? '')
  const { suppliers } = useSuppliers(token ?? '')
  const { warehouses } = useWarehouses(token ?? '')

  const [statusFilter, setStatusFilter] = useState<PurchaseOrderStatus | 'all'>('all')
  const [editingId, setEditingId] = useState<string | null | 'new'>(null)
  // Set alongside editingId right after a document-upload create -- the PO
  // page picks this up once (see LineItemsSection's initialParsingJobId) to
  // poll the still-in-flight parse and show a spinner instead of the
  // create flow blocking on it.
  const [pendingParsingJobId, setPendingParsingJobId] = useState<string | null>(null)
  const [deleteError, setDeleteError] = useState('')
  const [uploadModalOpen, setUploadModalOpen] = useState(false)
  const [showDiscrepancies, setShowDiscrepancies] = useState(false)

  const supplierName = (id?: string | null) => {
    if (!id) return null
    return suppliers.find(s => s.id === id)?.name ?? id
  }
  const warehouseName = (id?: string | null) => {
    if (!id) return null
    return warehouses.find(w => w.id === id)?.name ?? id
  }

  const filtered = purchaseOrders.filter(po => statusFilter === 'all' || po.status === statusFilter)

  async function handleDelete(po: PurchaseOrder) {
    if (!window.confirm(`Delete this purchase order? This cannot be undone.`)) return
    setDeleteError('')
    try {
      await remove(po.id)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  if (showDiscrepancies) {
    return <BookingDiscrepanciesPage token={token ?? ''} onClose={() => setShowDiscrepancies(false)} />
  }

  if (editingId !== null) {
    return (
      <PurchaseOrderEditPage
        purchaseOrderId={editingId === 'new' ? null : editingId}
        initialParsingJobId={editingId === 'new' ? null : pendingParsingJobId}
        onCreate={create}
        onClose={() => { setEditingId(null); setPendingParsingJobId(null); refresh() }}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div className="flex items-center gap-1 overflow-x-auto">
          {STATUS_TABS.map(s => (
            <button
              key={s}
              onClick={() => setStatusFilter(s)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-xs font-medium capitalize transition-colors whitespace-nowrap',
                statusFilter === s
                  ? 'dark:bg-[#1F2937] bg-gray-100 dark:text-gray-100 text-gray-800'
                  : 'dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
              )}
            >
              {s.replace('_', ' ')}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-2 ml-auto shrink-0">
          <button
            onClick={() => setShowDiscrepancies(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors font-medium"
          >
            <AlertTriangle size={13} /> Discrepancies
          </button>
          <button
            onClick={() => setUploadModalOpen(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors font-medium"
          >
            <Upload size={13} /> Upload Document
          </button>
          <button
            onClick={() => setEditingId('new')}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 transition-colors font-medium"
          >
            <Plus size={13} /> Add Purchase Order
          </button>
        </div>
      </div>

      {uploadModalOpen && (
        <CreateFromDocumentModal
          onClose={() => setUploadModalOpen(false)}
          onCreated={async (poId, jobId) => {
            setUploadModalOpen(false)
            await refresh()
            setPendingParsingJobId(jobId)
            setEditingId(poId)
          }}
        />
      )}

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{filtered.length} purchase order{filtered.length === 1 ? '' : 's'}</span>
        {(error || deleteError) && <span className="text-xs text-red-400 ml-auto">{error || deleteError}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Supplier</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Warehouse</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Expected</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Currency</th>
              <th className="px-3 py-2.5 text-right font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Total</th>
              <th className="px-3 py-2.5 w-16" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={8} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={8} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No purchase orders yet</td></tr>
            )}
            {filtered.map(po => (
              <tr
                key={po.id}
                onClick={() => setEditingId(po.id)}
                className="group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-[#2D3748] bg-gray-200 flex items-center justify-center">
                    <ShoppingBag size={12} className="dark:text-gray-500 text-gray-400" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[280px]">
                  {supplierName(po.supplier_id) ? (
                    <p className="font-medium dark:text-gray-100 text-gray-900 truncate">{supplierName(po.supplier_id)}</p>
                  ) : (
                    <p className="flex items-center gap-1 font-medium text-amber-400 truncate">
                      <Sparkles size={11} /> Missing supplier
                    </p>
                  )}
                  {po.is_potential_duplicate && (
                    <button
                      type="button"
                      onClick={e => { e.stopPropagation(); if (po.duplicate_of_purchase_order_id) setEditingId(po.duplicate_of_purchase_order_id) }}
                      className="flex items-center gap-1 mt-0.5 text-[10px] font-medium text-red-400 hover:underline"
                    >
                      <AlertTriangle size={10} /> Possible duplicate order
                    </button>
                  )}
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{warehouseName(po.warehouse_id) ?? '—'}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', STATUS_STYLE[po.status])}>
                    {po.status.replace('_', ' ')}
                  </span>
                </td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{po.expected_date ?? '—'}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{po.currency_code}</td>
                <td className="px-3 py-2.5 text-right dark:text-gray-200 text-gray-800 font-medium">
                  {formatMoney(po.total_minor_units, po.currency_code)}
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={e => { e.stopPropagation(); setEditingId(po.id) }}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Edit"
                    >
                      <Edit3 size={12} className="dark:text-gray-400 text-gray-500" />
                    </button>
                    <button
                      onClick={e => { e.stopPropagation(); handleDelete(po) }}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={12} className="dark:text-gray-400 text-gray-500 hover:text-red-400" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  )
}

// ── Booking discrepancies report ────────────────────────────────────────────

function formatDiscrepancyDate(iso?: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString()
}

type DiscrepancyTab = 'missing' | 'damaged-lost'

// Tenant-wide, derived on demand (not persisted) from every completed
// booking's line items vs scans -- same "not a real table" reasoning as
// the Inventory module's Expiry & Discards report.
function BookingDiscrepanciesPage({ token, onClose }: { token: string; onClose: () => void }) {
  const { report, loading, error } = useBookingDiscrepancies(token)
  const [tab, setTab] = useState<DiscrepancyTab>('missing')

  const TABS: { id: DiscrepancyTab; label: string }[] = [
    { id: 'missing', label: `Missing${report.missing_items.length ? ` (${report.missing_items.length})` : ''}` },
    { id: 'damaged-lost', label: `Damaged / Lost${report.damaged_lost.length ? ` (${report.damaged_lost.length})` : ''}` },
  ]

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Procurement
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">Booking Discrepancies</h1>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Every completed booking's under-scanned line items and damaged/lost stock</p>
        </div>
      </div>

      <div className="flex gap-2 px-6 pt-4 shrink-0">
        {TABS.map(t => (
          <button
            key={t.id}
            onClick={() => setTab(t.id)}
            className={clsx(
              'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              tab === t.id
                ? 'bg-blue-600 text-white'
                : 'dark:bg-[#1F2937] bg-white dark:text-gray-400 text-gray-500 hover:dark:bg-[#2D3748] hover:bg-gray-100',
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-red-400">{error}</p>}

        {tab === 'missing' && (
          <>
            {!error && loading && report.missing_items.length === 0 && <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
            {!error && !loading && report.missing_items.length === 0 && (
              <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">No missing items -- every completed booking was fully scanned.</p>
            )}
            {report.missing_items.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
                  <tr>
                    {['SKU', 'Product', 'Expected', 'Scanned', 'Missing', 'Completed'].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                  {report.missing_items.map(m => (
                    <tr key={m.line_item_id} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-2.5 font-mono dark:text-gray-300 text-gray-700">{m.sku_code ?? '—'}</td>
                      <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800">{m.product_name ?? m.raw_description}</td>
                      <td className="px-4 py-2.5 tabular-nums dark:text-gray-300 text-gray-700">{m.expected_qty}</td>
                      <td className="px-4 py-2.5 tabular-nums dark:text-gray-300 text-gray-700">{m.scanned_qty}</td>
                      <td className="px-4 py-2.5 tabular-nums text-red-400">{m.missing_qty}</td>
                      <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{formatDiscrepancyDate(m.completed_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}

        {tab === 'damaged-lost' && (
          <>
            {!error && loading && report.damaged_lost.length === 0 && <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
            {!error && !loading && report.damaged_lost.length === 0 && (
              <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">Nothing damaged or lost across any completed booking.</p>
            )}
            {report.damaged_lost.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
                  <tr>
                    {['SKU', 'Product', 'Type', 'Qty', 'Reason', 'Scanned by', 'Date'].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
                  {report.damaged_lost.map(d => (
                    <tr key={d.scan_id} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                      <td className="px-4 py-2.5 font-mono dark:text-gray-300 text-gray-700">{d.sku_code ?? '—'}</td>
                      <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800">{d.product_name ?? '—'}</td>
                      <td className="px-4 py-2.5 capitalize text-red-400">{d.action_type}</td>
                      <td className="px-4 py-2.5 tabular-nums dark:text-gray-300 text-gray-700">{d.quantity}</td>
                      <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{d.reason || '—'}</td>
                      <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{d.scanned_by || '—'}</td>
                      <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{formatDiscrepancyDate(d.scanned_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}
      </div>
    </div>
  )
}

// ── Create-from-document modal ──────────────────────────────────────────────

interface CreateFromDocumentModalProps {
  onClose: () => void
  onCreated: (purchaseOrderId: string, parsingJobId: string) => void
}

// Deliberately just a file picker -- supplier/warehouse/currency/expected
// date were removable friction: supplier and line items both come from the
// same document parse (no need to pick a supplier up front, the PO starts
// needs_review with a suggestion if one can't be matched), warehouse is
// never needed until receiving, and currency defaults from the tenant's
// own system-settings default.
function CreateFromDocumentModal({ onClose, onCreated }: CreateFromDocumentModalProps) {
  const { token } = useAuth()
  const { settings: tenantSettings } = useTenantSettings(token ?? '')
  const [file, setFile] = useState<File | null>(null)
  const [status, setStatus] = useState<'idle' | 'uploading'>('idle')
  const [formError, setFormError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!file) return
    setFormError('')
    setStatus('uploading')
    try {
      const job = await procurementApi.createPurchaseOrderFromDocument(token ?? '', {
        currency_code: tenantSettings?.default_currency || 'USD', file,
      })
      // Navigate immediately instead of blocking here on the parse --
      // it runs async in the background (cmd/scheduler) and can take a
      // few seconds; the PO page itself shows the parsing indicator and
      // picks up line items/supplier as soon as it finishes (see
      // LineItemsSection's initialParsingJobId), so there's no need to
      // freeze this modal waiting for it.
      onCreated(job.purchase_order_id, job.id)
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to upload document')
      setStatus('idle')
    }
  }

  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1.5'

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-md rounded-xl dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 shadow-xl">
        <div className="flex items-center justify-between px-5 py-4 border-b dark:border-[#2D3748] border-gray-200">
          <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Upload Document</h2>
          <button type="button" onClick={onClose} title="Close">
            <X size={16} className="dark:text-gray-500 text-gray-400" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          <p className="text-[11px] dark:text-gray-500 text-gray-400">
            Upload a supplier PDF, CSV, or XLSX. Supplier and line items are extracted and matched to products
            automatically — anything that can't be matched will need review.
          </p>
          <div>
            <label htmlFor="upload-file" className={labelClass}>Document</label>
            <input id="upload-file" type="file" accept=".pdf,.csv,.xlsx"
              disabled={status !== 'idle'}
              onChange={e => setFile(e.target.files?.[0] ?? null)}
              className="w-full text-xs dark:text-gray-300 text-gray-700 file:mr-3 file:px-3 file:py-1.5 file:rounded-lg file:border-0 file:bg-blue-600 file:text-white file:text-xs file:font-medium file:cursor-pointer disabled:opacity-50" />
          </div>
          {formError && <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>}
          <div className="flex items-center justify-end gap-2 pt-1">
            <button type="button" onClick={onClose}
              className="px-4 py-2 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-semibold dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors">
              Cancel
            </button>
            <button type="submit" disabled={status !== 'idle' || !file}
              className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors">
              {status === 'uploading' ? 'Uploading…' : 'Upload & Create'}
            </button>
          </div>
        </form>
      </div>
    </div>
  )
}

// ── Purchase Order Edit Page ────────────────────────────────────────────────

interface PurchaseOrderEditPageProps {
  purchaseOrderId: string | null
  // Set only right after a document-upload create -- lets LineItemsSection
  // pick up the still-in-flight parse and poll it instead of showing an
  // empty list until the user manually refreshes.
  initialParsingJobId?: string | null
  onCreate: (input: CreatePurchaseOrderInput) => Promise<PurchaseOrder>
  onClose: () => void
}

function PurchaseOrderEditPage({ purchaseOrderId, initialParsingJobId, onCreate, onClose }: PurchaseOrderEditPageProps) {
  const { token } = useAuth()
  const { suppliers } = useSuppliers(token ?? '')
  const { warehouses } = useWarehouses(token ?? '')
  const { settings: tenantSettings } = useTenantSettings(token ?? '')
  const [createdId, setCreatedId] = useState<string | null>(null)
  const effectiveId = purchaseOrderId ?? createdId
  const { detail, error: detailError, refresh: refreshDetail, updateHeader, addLineItem, updateLineItem, removeLineItem, deleteDocument } = usePurchaseOrderDetail(token ?? '', effectiveId)

  const po = detail?.purchase_order ?? null

  // AI-suggested supplier name matched against the existing supplier list --
  // when it matches, the dropdown pre-selects it and a Confirm control takes
  // over instead of the "create a new supplier" box (see MissingSupplierReview).
  const suggestedSupplierMatch = po?.suggested_supplier_name
    ? suppliers.find(s => normalizeSupplierName(s.name) === normalizeSupplierName(po.suggested_supplier_name!))
    : undefined

  const [supplierId, setSupplierId] = useState(po?.supplier_id ?? '')
  const [warehouseId, setWarehouseId] = useState(po?.warehouse_id ?? '')
  const [status, setStatus] = useState<PurchaseOrderStatus>(po?.status ?? 'draft')
  const [currencyCode, setCurrencyCode] = useState(po?.currency_code ?? 'USD')
  const [expectedDate, setExpectedDate] = useState(purchaseOrderId === null ? todayDateString() : (po?.expected_date ?? ''))
  const [notes, setNotes] = useState(po?.notes ?? '')
  const [saving, setSaving] = useState(false)
  const [formError, setFormError] = useState('')

  // Seeds the currency from the tenant's system-settings default for a
  // brand-new PO only -- an existing PO's own currency_code (set below)
  // always wins once `detail` loads.
  const currencyTouched = useRef(false)
  useEffect(() => {
    if (!po && !currencyTouched.current && tenantSettings?.default_currency) {
      setCurrencyCode(tenantSettings.default_currency)
    }
  }, [po, tenantSettings])

  // Header fields start empty for an existing PO because `detail` loads
  // asynchronously after mount. Seed the form once per PO id — a ref (not
  // state) so it doesn't retrigger this effect — so later refreshes (e.g.
  // after adding a line item) don't clobber in-progress header edits.
  const prefilledForId = useRef<string | null>(null)
  useEffect(() => {
    if (po && prefilledForId.current !== po.id) {
      prefilledForId.current = po.id
      setSupplierId(po.supplier_id ?? '')
      setWarehouseId(po.warehouse_id ?? '')
      setStatus(po.status)
      setCurrencyCode(po.currency_code)
      setExpectedDate(po.expected_date ?? '')
      setNotes(po.notes)
    }
  }, [po])

  const isNew = purchaseOrderId === null && createdId === null

  // Pre-selects the dropdown once the AI suggestion resolves to an existing
  // supplier -- separate from the field-seeding effect above because
  // `suppliers` loads on its own async timer, often after that effect's
  // one-shot-per-PO guard has already fired. Guarded the same way so it
  // doesn't fight a choice the user already made.
  const suggestedSupplierPreselectedFor = useRef<string | null>(null)
  useEffect(() => {
    if (!po || po.supplier_id || !suggestedSupplierMatch) return
    if (suggestedSupplierPreselectedFor.current === po.id) return
    suggestedSupplierPreselectedFor.current = po.id
    setSupplierId(suggestedSupplierMatch.id)
  }, [po, suggestedSupplierMatch])

  const [confirmingSuggestedSupplier, setConfirmingSuggestedSupplier] = useState(false)
  async function handleConfirmSuggestedSupplier() {
    if (!effectiveId || !suggestedSupplierMatch) return
    setFormError('')
    setConfirmingSuggestedSupplier(true)
    try {
      const updated = await procurementApi.createSupplierAndAssignToPurchaseOrder(token ?? '', effectiveId, { name: suggestedSupplierMatch.name })
      if (updated.supplier_id) setSupplierId(updated.supplier_id)
      await refreshDetail()
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to confirm supplier')
    } finally {
      setConfirmingSuggestedSupplier(false)
    }
  }

  async function handleSave(e: React.FormEvent) {
    e.preventDefault()
    setFormError('')
    setSaving(true)
    try {
      if (isNew) {
        const input: CreatePurchaseOrderInput = {
          supplier_id: supplierId, warehouse_id: warehouseId.trim() === '' ? null : warehouseId, currency_code: currencyCode,
          expected_date: expectedDate.trim() === '' ? null : expectedDate,
          notes,
        }
        const created = await onCreate(input)
        setCreatedId(created.id)
      } else if (effectiveId) {
        const input: UpdatePurchaseOrderInput = {
          supplier_id: supplierId, warehouse_id: warehouseId.trim() === '' ? null : warehouseId, status, currency_code: currencyCode,
          expected_date: expectedDate.trim() === '' ? null : expectedDate,
          notes,
        }
        await updateHeader(input)
      }
    } catch (err) {
      setFormError(err instanceof Error ? err.message : 'Failed to save purchase order')
    } finally {
      setSaving(false)
    }
  }

  // Called from the Receiving section's "Receive stock" form the first
  // time it needs a warehouse and the PO doesn't have one yet -- persists
  // it via the same updateHeader path the header Save button uses, and
  // mirrors it into the header select immediately so it's already correct
  // if the header form gets saved later in the same session.
  async function handleAssignWarehouse(newWarehouseId: string) {
    if (!effectiveId) return
    await updateHeader({
      supplier_id: supplierId, warehouse_id: newWarehouseId, status, currency_code: currencyCode,
      expected_date: expectedDate.trim() === '' ? null : expectedDate, notes,
    })
    setWarehouseId(newWarehouseId)
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
            <ArrowLeft size={11} /> Purchase Orders
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">
            {isNew ? 'New Purchase Order' : 'Edit Purchase Order'}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onClose}
            className="px-4 py-2 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-xs font-semibold dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 transition-colors"
          >
            Discard
          </button>
          <button
            type="submit" form="po-form" disabled={saving || !supplierId}
            className="px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : isNew ? 'Create Purchase Order' : 'Save'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto p-6 space-y-5">
          <form id="po-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">General</h3>
                {!isNew && (
                  <select
                    aria-label="Status"
                    value={status}
                    onChange={e => setStatus(e.target.value as PurchaseOrderStatus)}
                    className="px-2 py-1 text-[10px] rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none capitalize"
                  >
                    {(['draft', 'sent', 'partially_received', 'received', 'cancelled', 'needs_review'] as const).map(s => (
                      <option key={s} value={s} className="capitalize">{s.replace('_', ' ')}</option>
                    ))}
                  </select>
                )}
              </div>
              <div className="grid grid-cols-2 gap-3">
                <div>
                  <div className="flex items-center justify-between mb-1.5">
                    <label htmlFor="po-supplier" className={labelClass + ' mb-0'}>Supplier</label>
                    {suggestedSupplierMatch && !po?.supplier_id && (
                      <button
                        type="button" onClick={handleConfirmSuggestedSupplier} disabled={confirmingSuggestedSupplier}
                        className="text-[10px] font-semibold text-blue-400 hover:underline disabled:opacity-50"
                      >
                        {confirmingSuggestedSupplier ? 'Confirming…' : 'Confirm'}
                      </button>
                    )}
                  </div>
                  <select
                    id="po-supplier" required value={supplierId} onChange={e => setSupplierId(e.target.value)}
                    className={inputClass}
                  >
                    <option value="">Select supplier…</option>
                    {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
                  </select>
                  {suggestedSupplierMatch && !po?.supplier_id && (
                    <p className="text-[10px] text-blue-400 mt-1">Suggested from the uploaded document — confirm to assign.</p>
                  )}
                </div>
                <div>
                  <label htmlFor="po-warehouse" className={labelClass}>Warehouse (optional)</label>
                  <select
                    id="po-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)}
                    className={inputClass}
                  >
                    <option value="">Not set — assign when receiving</option>
                    {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="po-currency" className={labelClass}>Currency</label>
                  <select
                    id="po-currency" required value={currencyCode} onChange={e => { currencyTouched.current = true; setCurrencyCode(e.target.value) }}
                    className={inputClass}
                  >
                    {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
                  </select>
                </div>
                <div>
                  <label htmlFor="po-expected-date" className={labelClass}>Expected date</label>
                  <input
                    id="po-expected-date" type="date" value={expectedDate ?? ''} onChange={e => setExpectedDate(e.target.value)}
                    className={inputClass}
                  />
                </div>
              </div>
              <div>
                <label htmlFor="po-notes" className={labelClass}>Notes</label>
                <textarea
                  id="po-notes" rows={3} value={notes} onChange={e => setNotes(e.target.value)}
                  className={inputClass}
                />
              </div>
              {(formError || detailError) && (
                <p className="text-xs text-red-400 bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError || detailError}</p>
              )}
            </div>
          </form>

          {!isNew && effectiveId && po && !po.supplier_id && !suggestedSupplierMatch && (
            <MissingSupplierReview
              purchaseOrderId={effectiveId}
              suggestedName={po.suggested_supplier_name ?? ''}
              suppliers={suppliers}
              onAssigned={supplierId => { setSupplierId(supplierId); refreshDetail() }}
            />
          )}

          {!isNew && effectiveId && (detail?.documents.length ?? 0) > 0 && (
            <DocumentsSection documents={detail!.documents} onDelete={deleteDocument} />
          )}

          {!isNew && effectiveId && (
            <LineItemsSection
              purchaseOrderId={effectiveId}
              lineItems={detail?.line_items ?? []}
              currencyCode={currencyCode}
              initialParsingJobId={initialParsingJobId}
              onAdd={addLineItem}
              onUpdate={updateLineItem}
              onRemove={removeLineItem}
              onRefresh={refreshDetail}
            />
          )}

          {!isNew && effectiveId && (
            <ReceivingSection
              purchaseOrderId={effectiveId}
              warehouseId={po?.warehouse_id ?? null}
              warehouses={warehouses}
              onAssignWarehouse={handleAssignWarehouse}
              lineItems={detail?.line_items ?? []}
            />
          )}
        </div>
      </div>
    </div>
  )
}

// ── Missing supplier review ──────────────────────────────────────────────────

interface MissingSupplierReviewProps {
  purchaseOrderId: string
  suggestedName: string
  suppliers: Supplier[]
  onAssigned: (supplierId: string) => void
}

// Mirrors the backend's FindByName lookup exactly, so what this shows is
// what submitting will actually do: lower-case, strip periods/commas
// (AI-extracted text and manually-typed names disagree on trailing
// punctuation for legal suffixes like "Lda."/"Lda"), collapse whitespace
// runs -- but leave letters (incl. accented ones) untouched.
function normalizeSupplierName(name: string): string {
  return name.toLowerCase().replace(/[.,]/g, '').replace(/\s+/g, ' ').trim()
}

function MissingSupplierReview({ purchaseOrderId, suggestedName, suppliers, onAssigned }: MissingSupplierReviewProps) {
  const { token } = useAuth()
  const [name, setName] = useState(suggestedName)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const matchedSupplier = suppliers.find(s => normalizeSupplierName(s.name) === normalizeSupplierName(name))

  async function handleCreate(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      const po = await procurementApi.createSupplierAndAssignToPurchaseOrder(token ?? '', purchaseOrderId, { name })
      if (po.supplier_id) onAssigned(po.supplier_id)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create supplier')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="rounded-xl border dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40 p-4 space-y-2">
      <p className="flex items-center gap-1.5 text-xs font-semibold text-amber-400">
        <Sparkles size={12} /> Needs review: this purchase order has no supplier
      </p>
      <p className="text-[11px] dark:text-gray-400 text-gray-500">
        Pick an existing supplier above, or create a new one and assign it in one step.
      </p>
      {suggestedName && (
        <p className="text-[11px] text-blue-400">Suggested from the uploaded document — edit if it's wrong.</p>
      )}
      {matchedSupplier && (
        <p className="text-[11px] text-emerald-400">
          Matches existing supplier "{matchedSupplier.name}" — this will assign it, not create a new one.
        </p>
      )}
      <form onSubmit={handleCreate} className="flex items-center gap-2">
        <input
          aria-label="New supplier name" required value={name} onChange={e => setName(e.target.value)}
          placeholder="New supplier name"
          className="flex-1 px-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500"
        />
        <button type="submit" disabled={saving || !name.trim()}
          className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-semibold hover:bg-blue-700 disabled:opacity-50 whitespace-nowrap">
          {saving ? (matchedSupplier ? 'Assigning…' : 'Creating…') : (matchedSupplier ? 'Assign existing supplier' : 'Create & Assign')}
        </button>
      </form>
      {error && <p className="text-[10px] text-red-400">{error}</p>}
    </div>
  )
}

// ── Documents ────────────────────────────────────────────────────────────────

interface DocumentsSectionProps {
  documents: PurchaseOrderDocument[]
  onDelete: (id: string) => Promise<void>
}

function DocumentsSection({ documents, onDelete }: DocumentsSectionProps) {
  const [deletingId, setDeletingId] = useState<string | null>(null)

  async function handleDelete(doc: PurchaseOrderDocument) {
    if (!window.confirm(`Delete "${doc.file_name}"? This does not affect any line items already created from it.`)) return
    setDeletingId(doc.id)
    try {
      await onDelete(doc.id)
    } finally {
      setDeletingId(null)
    }
  }

  return (
    <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
      <div className="px-4 py-3 dark:bg-[#111827]/50 bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
        <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">Documents</h3>
      </div>
      <div className="divide-y dark:divide-[#2D3748] divide-gray-200">
        {documents.map(doc => (
          <div key={doc.id} className="flex items-center gap-2 px-4 py-2.5 text-xs dark:text-gray-300 text-gray-700">
            <FileText size={13} className="dark:text-gray-500 text-gray-400 shrink-0" />
            <a href={doc.file_url} target="_blank" rel="noreferrer" className="flex-1 min-w-0 truncate hover:underline">
              {doc.file_name}
            </a>
            {doc.is_potential_duplicate && (
              <span className="flex items-center gap-1 px-1.5 py-0.5 rounded-full text-[10px] font-medium text-amber-400 bg-amber-500/10 shrink-0">
                <AlertTriangle size={10} /> Possible duplicate
              </span>
            )}
            <button
              type="button"
              title="Delete"
              onClick={() => handleDelete(doc)}
              disabled={deletingId === doc.id}
              className="p-1 rounded dark:text-gray-500 text-gray-400 hover:text-red-400 disabled:opacity-50 transition-colors shrink-0"
            >
              <Trash2 size={13} />
            </button>
          </div>
        ))}
      </div>
    </div>
  )
}

// ── Line Items ───────────────────────────────────────────────────────────────

interface LineItemsSectionProps {
  purchaseOrderId: string
  lineItems: PurchaseOrderLineItem[]
  currencyCode: string
  // Set only right after a document-upload create -- picked up once (on
  // mount) to poll that already-in-flight parse instead of the create flow
  // blocking on it.
  initialParsingJobId?: string | null
  onAdd: (input: AddLineItemInput) => Promise<void>
  onUpdate: (id: string, input: UpdateLineItemInput) => Promise<void>
  onRemove: (id: string) => Promise<void>
  onRefresh: () => void
}

function LineItemsSection({ purchaseOrderId, lineItems, currencyCode, initialParsingJobId, onAdd, onUpdate, onRemove, onRefresh }: LineItemsSectionProps) {
  const { token } = useAuth()
  const [adding, setAdding] = useState(false)
  const [editingLineId, setEditingLineId] = useState<string | null>(null)
  const [reviewingLineId, setReviewingLineId] = useState<string | null>(null)
  const [uploadStatus, setUploadStatus] = useState<'idle' | 'uploading' | 'parsing'>(initialParsingJobId ? 'parsing' : 'idle')
  const [parsingSlow, setParsingSlow] = useState(false)
  const [uploadError, setUploadError] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  // Picks up a parse that was already kicked off by the create-from-document
  // flow -- runs once (initialParsingJobId only has a value on the very
  // first mount right after that create, and this component isn't reused
  // across different POs without unmounting first).
  useEffect(() => {
    if (!initialParsingJobId) return
    let cancelled = false
    pollParsingJob(
      () => procurementApi.getDocumentParsingJob(token ?? '', initialParsingJobId),
      { onSlow: () => { if (!cancelled) setParsingSlow(true) } },
    )
      .then(finalJob => {
        if (cancelled) return
        if (finalJob.status === 'failed') {
          setUploadError(friendlyParsingError(finalJob.error_message))
        }
        onRefresh()
      })
      .catch(err => {
        if (!cancelled) setUploadError(err instanceof Error ? err.message : 'Failed to parse document')
      })
      .finally(() => {
        if (!cancelled) { setUploadStatus('idle'); setParsingSlow(false) }
      })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [initialParsingJobId])

  async function handleRemove(id: string) {
    if (!window.confirm('Remove this line item?')) return
    await onRemove(id)
  }

  async function handleFileSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return

    // Only PDFs need AI to parse (CSV/XLSX go through the deterministic
    // path) -- warn up front rather than let the upload silently fail to
    // extract anything, so the user knows manual entry is the fallback.
    const isPDF = file.type === 'application/pdf' || file.name.toLowerCase().endsWith('.pdf')
    if (isPDF) {
      const aiConnection = await integrationsApi.getAIConnection(token ?? '').catch(() => null)
      if (!aiConnection && !window.confirm(
        'No AI provider is configured for this workspace, so this PDF won’t auto-parse into line items. ' +
        'You can still upload it and add line items manually below. Continue?',
      )) {
        return
      }
    }

    setUploadError('')
    setUploadStatus('uploading')
    try {
      const job = await procurementApi.attachDocumentToPurchaseOrder(token ?? '', purchaseOrderId, file)
      setUploadStatus('parsing')
      const finalJob = await pollParsingJob(
        () => procurementApi.getDocumentParsingJob(token ?? '', job.id),
        { onSlow: () => setParsingSlow(true) },
      )
      if (finalJob.status === 'failed') {
        setUploadError(friendlyParsingError(finalJob.error_message))
      }
      // Always refresh, even on a failed parse -- the document itself was
      // still saved and should show up in the Documents section so it can
      // be opened for manual entry.
      onRefresh()
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : 'Failed to upload document')
    } finally {
      setUploadStatus('idle')
      setParsingSlow(false)
    }
  }

  return (
    <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 dark:bg-[#111827]/50 bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
        <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">Line Items</h3>
        <div className="flex items-center gap-2">
          {uploadStatus === 'uploading' && (
            <span className="flex items-center gap-1 text-[10px] dark:text-gray-500 text-gray-400">
              <Loader2 size={11} className="animate-spin" /> Uploading…
            </span>
          )}
          {uploadStatus === 'parsing' && (
            <span className="flex items-center gap-1 text-[10px] text-amber-400">
              <Loader2 size={11} className="animate-spin" />
              {parsingSlow ? 'Still parsing… this can take a bit longer' : 'Parsing document…'}
            </span>
          )}
          <input ref={fileInputRef} type="file" aria-label="Upload document" accept=".pdf,.csv,.xlsx" className="hidden" onChange={handleFileSelected} />
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadStatus !== 'idle'}
            className="flex items-center gap-1 px-2.5 py-1 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-[10px] dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 disabled:opacity-50 transition-colors"
          >
            <Upload size={11} /> Upload Document
          </button>
          {!adding && (
            <button
              type="button"
              onClick={() => setAdding(true)}
              className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-blue-600 text-white text-[10px] font-medium hover:bg-blue-700"
            >
              <Plus size={11} /> Add Line Item
            </button>
          )}
        </div>
      </div>
      {uploadError && (
        <p className="text-[10px] text-red-400 bg-red-50 dark:bg-red-500/10 px-4 py-1.5">{uploadError}</p>
      )}

      <table className="w-full text-xs">
        <thead>
          <tr className="dark:bg-[#111827]/30 bg-gray-50">
            <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Description</th>
            <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Type</th>
            <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Linked SKU</th>
            <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Batch</th>
            <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Qty</th>
            <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Unit Cost</th>
            <th className="px-4 py-2 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider text-[10px]">Total</th>
            <th className="px-4 py-2 w-20" />
          </tr>
        </thead>
        <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
          {lineItems.length === 0 && !adding && (
            <tr><td colSpan={8} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No line items yet</td></tr>
          )}
          {lineItems.map(li => {
            const isCharge = isChargeLineType(li.line_type)
            return editingLineId === li.id ? (
              <LineItemFormRow
                key={li.id}
                initial={li}
                currencyCode={currencyCode}
                onSubmit={async input => { await onUpdate(li.id, input); setEditingLineId(null) }}
                onCancel={() => setEditingLineId(null)}
              />
            ) : (
              <Fragment key={li.id}>
                <tr className={clsx('transition-colors', li.sku_id || isCharge ? 'hover:dark:bg-[#2D3748]/30 hover:bg-gray-50' : 'dark:bg-amber-500/5 bg-amber-50/40')}>
                  <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800">{li.raw_description}</td>
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">
                    {isCharge && (
                      <div className="flex items-center gap-1.5">
                        <span className="text-[10px]">{LINE_TYPE_LABELS[li.line_type as Exclude<LineItemType, ''>]}</span>
                        {!li.line_type_confirmed && (
                          <button
                            type="button"
                            onClick={() => onUpdate(li.id, {
                              sku_id: li.sku_id ?? null, raw_sku: li.raw_sku, raw_description: li.raw_description,
                              quantity: li.quantity, unit_cost_minor_units: li.unit_cost_minor_units, line_type: li.line_type,
                            })}
                            title="AI-classified -- confirm this is correct"
                            className="text-[10px] font-medium text-blue-400 hover:underline"
                          >
                            Confirm
                          </button>
                        )}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">
                    {isCharge ? (
                      <span className="text-[10px]">—</span>
                    ) : li.sku_id ? (
                      li.product_name ? (
                        <span className="text-[10px]">
                          {li.product_name} <span className="font-mono dark:text-gray-500 text-gray-400">— {li.sku_code}</span>
                          {li.units_per_purchased_item > 1 && (
                            <span className="ml-1 text-amber-400" title="Units per purchased item">
                              × {li.units_per_purchased_item}
                            </span>
                          )}
                        </span>
                      ) : (
                        <span className="font-mono text-[10px]">{li.sku_id}</span>
                      )
                    ) : (
                      <button
                        type="button"
                        onClick={() => setReviewingLineId(reviewingLineId === li.id ? null : li.id)}
                        className="flex items-center gap-1 text-[10px] font-medium text-amber-400 hover:underline"
                      >
                        <Sparkles size={10} /> Needs review
                      </button>
                    )}
                  </td>
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">
                    {li.suggested_lot_number ? (
                      <span className="text-[10px]" title="Parsed from the uploaded document -- confirmed when the batch is assigned during booking">
                        {li.suggested_lot_number}
                        {li.suggested_expiry_date && (
                          <span className="dark:text-gray-500 text-gray-400"> · exp {li.suggested_expiry_date}</span>
                        )}
                      </span>
                    ) : (
                      <span className="text-[10px]">—</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800">{li.quantity}</td>
                  <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800">{formatMoney(li.unit_cost_minor_units, currencyCode)}</td>
                  <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800 font-medium">
                    {formatMoney(signedLineItemAmount(li), currencyCode)}
                  </td>
                  <td className="px-4 py-2.5">
                    <div className="flex items-center gap-1">
                      <button
                        type="button"
                        onClick={() => setEditingLineId(li.id)}
                        className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                        title="Edit"
                      >
                        <Edit3 size={12} className="dark:text-gray-400 text-gray-500" />
                      </button>
                      <button
                        type="button"
                        onClick={() => handleRemove(li.id)}
                        className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                        title="Remove"
                      >
                        <Trash2 size={12} className="dark:text-gray-400 text-gray-500 hover:text-red-400" />
                      </button>
                    </div>
                  </td>
                </tr>
                {reviewingLineId === li.id && (
                  <tr>
                    <td colSpan={8} className="px-4 pb-3 dark:bg-amber-500/5 bg-amber-50/40">
                      <ReviewUnmatchedLineItem
                        lineItem={li}
                        onResolved={() => { setReviewingLineId(null); onRefresh() }}
                        onCancel={() => setReviewingLineId(null)}
                      />
                    </td>
                  </tr>
                )}
              </Fragment>
            )
          })}
          {adding && (
            <LineItemFormRow
              currencyCode={currencyCode}
              onSubmit={async input => { await onAdd(input); setAdding(false) }}
              onCancel={() => setAdding(false)}
            />
          )}
          {!adding && (
            <tr>
              <td colSpan={8} className="px-4 py-2">
                <button
                  type="button"
                  onClick={() => setAdding(true)}
                  className="flex items-center gap-1 text-[10px] font-medium text-blue-400 hover:underline"
                >
                  <Plus size={11} /> Add Line Item
                </button>
              </td>
            </tr>
          )}
        </tbody>
        {lineItems.length > 0 && (
          <tfoot>
            <tr className="border-t dark:border-[#2D3748] border-gray-200 font-medium">
              <td colSpan={6} className="px-4 py-2.5 text-right dark:text-gray-400 text-gray-500 uppercase tracking-wider text-[10px]">
                Total
              </td>
              <td className="px-4 py-2.5 dark:text-gray-100 text-gray-900">
                {formatMoney(lineItems.reduce((sum, li) => sum + signedLineItemAmount(li), 0), currencyCode)}
              </td>
              <td className="px-4 py-2.5" />
            </tr>
          </tfoot>
        )}
      </table>
    </div>
  )
}

interface LineItemFormRowProps {
  initial?: PurchaseOrderLineItem
  currencyCode: string
  onSubmit: (input: AddLineItemInput) => Promise<void>
  onCancel: () => void
}

function LineItemFormRow({ initial, onSubmit, onCancel }: LineItemFormRowProps) {
  const [description, setDescription] = useState(initial?.raw_description ?? '')
  const [quantity, setQuantity] = useState(initial ? String(initial.quantity) : '1')
  const [unitCost, setUnitCost] = useState(initial ? (initial.unit_cost_minor_units / 100).toFixed(2) : '0.00')
  const [skuId, setSkuId] = useState<string | null>(initial?.sku_id ?? null)
  // Friendly "Product Name — SKU code" for display -- falls back to null
  // (renders the raw id) only when the initial item has no resolved name,
  // e.g. the linked SKU was deleted after linking.
  const [skuLabel, setSkuLabel] = useState<string | null>(
    initial?.sku_id && initial.product_name ? `${initial.product_name} — ${initial.sku_code}` : null,
  )
  const [unitsPerPurchasedItem, setUnitsPerPurchasedItem] = useState(initial ? String(initial.units_per_purchased_item) : '1')
  const [lineType, setLineType] = useState<LineItemType>(initial?.line_type || 'product')
  const [saving, setSaving] = useState(false)
  const [rowError, setRowError] = useState('')
  const [pickerOpen, setPickerOpen] = useState(false)
  const isCharge = isChargeLineType(lineType)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setRowError('')
    setSaving(true)
    try {
      await onSubmit({
        // A charge row (shipping/discount/fee/credit note) never has a
        // SKU, even if one was picked before switching the type.
        sku_id: isCharge ? null : skuId,
        raw_description: description,
        quantity: parseInt(quantity, 10),
        units_per_purchased_item: isCharge ? 1 : parseInt(unitsPerPurchasedItem || '1', 10),
        unit_cost_minor_units: Math.round(parseFloat(unitCost || '0') * 100),
        line_type: lineType,
      })
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to save line item')
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#0B1220] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500'
  const idPrefix = initial?.id ?? 'new'

  return (
    <tr className="dark:bg-[#111827]/40 bg-blue-50/30">
      <td colSpan={8} className="px-4 py-3">
          <div className="grid grid-cols-4 gap-3">
            <div className="col-span-2">
              <label htmlFor={`li-description-${idPrefix}`} className={labelClass}>Description</label>
              <input
                id={`li-description-${idPrefix}`}
                aria-label="Line item description" required value={description} onChange={e => setDescription(e.target.value)}
                className={inputClass} placeholder="Description"
              />
            </div>
            <div>
              <label htmlFor={`li-type-${idPrefix}`} className={labelClass}>Type</label>
              <select
                id={`li-type-${idPrefix}`}
                aria-label="Line item type" value={lineType} onChange={e => setLineType(e.target.value as LineItemType)}
                className={inputClass}
              >
                {(Object.keys(LINE_TYPE_LABELS) as Exclude<LineItemType, ''>[]).map(t => (
                  <option key={t} value={t}>{LINE_TYPE_LABELS[t]}</option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor={`li-quantity-${idPrefix}`} className={labelClass}>Quantity</label>
              <input
                id={`li-quantity-${idPrefix}`}
                aria-label="Quantity" type="number" min="1" required value={quantity} onChange={e => setQuantity(e.target.value)}
                className={inputClass}
              />
            </div>

            <div className="col-span-2">
              <span className={labelClass}>Linked product</span>
              {isCharge ? (
                <p className="text-xs dark:text-gray-500 text-gray-400 py-2">— (charge rows don't link to a product)</p>
              ) : skuId ? (
                <div className="flex items-center gap-2 py-1">
                  <span
                    className={clsx('text-xs truncate', skuLabel ? 'dark:text-gray-200 text-gray-800' : 'font-mono dark:text-gray-400 text-gray-500')}
                    title={skuLabel ?? skuId}
                  >
                    {skuLabel ?? skuId}
                  </span>
                  <button
                    type="button" onClick={() => { setSkuId(null); setSkuLabel(null) }}
                    className="flex items-center gap-1 text-[10px] dark:text-gray-500 text-gray-400 hover:text-red-400 shrink-0"
                  >
                    <X size={11} /> Unlink
                  </button>
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => setPickerOpen(true)}
                  className="flex items-center gap-1 text-xs dark:text-blue-400 text-blue-600 hover:underline py-2"
                >
                  <Search size={12} /> Link product
                </button>
              )}
            </div>
            {!isCharge && skuId && (
              <div>
                <label htmlFor={`li-units-per-item-${idPrefix}`} className={labelClass}>Units per purchased item</label>
                <input
                  id={`li-units-per-item-${idPrefix}`}
                  aria-label="Units per purchased item" type="number" min="1" step="1"
                  value={unitsPerPurchasedItem} onChange={e => setUnitsPerPurchasedItem(e.target.value)}
                  className={inputClass}
                />
                <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-1">
                  Stock units one purchased item holds — e.g. 36 if this line is a box of 36 units.
                </p>
              </div>
            )}

            <div>
              <label htmlFor={`li-unit-cost-${idPrefix}`} className={labelClass}>Unit cost</label>
              <input
                id={`li-unit-cost-${idPrefix}`}
                aria-label="Unit cost" type="number" min="0" step="0.01" required value={unitCost} onChange={e => setUnitCost(e.target.value)}
                className={inputClass}
              />
            </div>
            <div>
              <span className={labelClass}>Total</span>
              <p className="px-3 py-2 text-xs dark:text-gray-300 text-gray-700">
                {(signedLineItemAmount({
                  line_type: lineType, quantity: parseInt(quantity || '0', 10),
                  unit_cost_minor_units: Math.round(parseFloat(unitCost || '0') * 100),
                }) / 100).toFixed(2)}
              </p>
            </div>
          </div>

          {rowError && <p className="text-[10px] text-red-400 mt-2">{rowError}</p>}

          {pickerOpen && (
            <div className="mt-3">
              <ProductSkuPicker
                onSelect={(sku, label) => {
                  setSkuId(sku.id); setSkuLabel(label)
                  if (!description) setDescription(label)
                  setPickerOpen(false)
                }}
                onClose={() => setPickerOpen(false)}
              />
            </div>
          )}

          <div className="flex items-center gap-2 mt-3">
            <button
              type="button" disabled={saving} onClick={handleSubmit}
              className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50"
            >
              <Check size={12} /> Save Line
            </button>
            <button
              type="button" onClick={onCancel}
              className="px-3 py-1.5 rounded-lg text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100"
            >
              Cancel
            </button>
          </div>
      </td>
    </tr>
  )
}

// ── Receiving ────────────────────────────────────────────────────────────────

interface ReceivingSectionProps {
  purchaseOrderId: string
  // Nil until stock has been received against this PO at least once --
  // ReceiveDiscrepancyForm prompts for one inline the first time, via
  // onAssignWarehouse.
  warehouseId: string | null
  warehouses: Warehouse[]
  onAssignWarehouse: (warehouseId: string) => Promise<void>
  lineItems: PurchaseOrderLineItem[]
}

function ReceivingSection({ purchaseOrderId, warehouseId, warehouses, onAssignWarehouse, lineItems }: ReceivingSectionProps) {
  const { token } = useAuth()
  const { records, loading, error, refresh, upload } = useReceivingRecords(token ?? '', purchaseOrderId)
  const [expandedId, setExpandedId] = useState<string | null>(null)
  const [uploadStatus, setUploadStatus] = useState<'idle' | 'uploading' | 'parsing'>('idle')
  const [uploadError, setUploadError] = useState('')
  const fileInputRef = useRef<HTMLInputElement>(null)

  async function handleFileSelected(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0]
    e.target.value = ''
    if (!file) return
    setUploadError('')
    setUploadStatus('uploading')
    try {
      const job = await upload(file)
      setUploadStatus('parsing')
      const finalJob = await pollParsingJob(() => procurementApi.getReceivingParsingJob(token ?? '', job.id))
      if (finalJob.status === 'failed') {
        setUploadError(finalJob.error_message || 'Failed to parse invoice')
      }
      refresh()
    } catch (err) {
      setUploadError(err instanceof Error ? err.message : 'Failed to upload invoice')
    } finally {
      setUploadStatus('idle')
    }
  }

  return (
    <div className="rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 dark:bg-[#111827]/50 bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
        <h3 className="text-xs font-semibold dark:text-gray-100 text-gray-900">Receiving</h3>
        <div className="flex items-center gap-2">
          {uploadStatus === 'uploading' && <span className="text-[10px] dark:text-gray-500 text-gray-400">Uploading…</span>}
          {uploadStatus === 'parsing' && <span className="text-[10px] text-amber-400">Parsing invoice…</span>}
          <input ref={fileInputRef} type="file" aria-label="Upload invoice" accept=".pdf,.csv,.xlsx" className="hidden" onChange={handleFileSelected} />
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadStatus !== 'idle'}
            className="flex items-center gap-1 px-2.5 py-1 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#1F2937] bg-white text-[10px] dark:text-gray-300 text-gray-700 hover:dark:bg-[#2D3748] hover:bg-gray-50 disabled:opacity-50 transition-colors"
          >
            <Upload size={11} /> Upload Invoice
          </button>
        </div>
      </div>
      {(uploadError || error) && (
        <p className="text-[10px] text-red-400 bg-red-50 dark:bg-red-500/10 px-4 py-1.5">{uploadError || error}</p>
      )}
      <div className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
        {loading && <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
        {!loading && records.length === 0 && (
          <p className="px-4 py-6 text-center text-xs dark:text-gray-500 text-gray-400">No invoices uploaded yet</p>
        )}
        {records.map(record => (
          <div key={record.id}>
            <button
              type="button"
              onClick={() => setExpandedId(expandedId === record.id ? null : record.id)}
              className="w-full flex items-center gap-3 px-4 py-2.5 text-left hover:dark:bg-[#1F2937]/40 hover:bg-gray-50 transition-colors"
            >
              <FileText size={13} className="dark:text-gray-500 text-gray-400 shrink-0" />
              <span className="flex-1 text-xs dark:text-gray-200 text-gray-800 truncate">{record.invoice_file_name}</span>
              {record.is_potential_duplicate && (
                <span
                  title="Another invoice with identical file content was already uploaded for this supplier"
                  className="flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium bg-amber-100 text-amber-700 dark:bg-amber-500/10 dark:text-amber-400"
                >
                  <AlertTriangle size={10} /> Possible duplicate
                </span>
              )}
              <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', RECEIVING_STATUS_STYLE[record.status])}>
                {record.status.replace('_', ' ')}
              </span>
            </button>
            {expandedId === record.id && (
              <div className="px-4 pb-3 dark:bg-[#0B1220]/40 bg-gray-50/60">
                <ReceivingRecordCard
                  recordId={record.id}
                  lineItems={lineItems}
                  warehouseId={warehouseId}
                  warehouses={warehouses}
                  onAssignWarehouse={onAssignWarehouse}
                  invoiceFileName={record.invoice_file_name}
                  onCompleted={() => { refresh() }}
                />
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}

interface ReceivingRecordCardProps {
  recordId: string
  lineItems: PurchaseOrderLineItem[]
  warehouseId: string | null
  warehouses: Warehouse[]
  onAssignWarehouse: (warehouseId: string) => Promise<void>
  invoiceFileName: string
  onCompleted: () => void
}

function ReceivingRecordCard({ recordId, lineItems, warehouseId, warehouses, onAssignWarehouse, invoiceFileName, onCompleted }: ReceivingRecordCardProps) {
  const { token } = useAuth()
  const [detail, setDetail] = useState<ReceivingRecordDetail | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')
  const [completing, setCompleting] = useState(false)

  const load = async () => {
    setLoading(true)
    setError('')
    try {
      setDetail(await procurementApi.getReceivingRecord(token ?? '', recordId))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load receiving record')
    } finally {
      setLoading(false)
    }
  }

  useEffect(() => { load() }, [recordId])

  async function handleComplete() {
    setCompleting(true)
    setError('')
    try {
      await procurementApi.completeReceiving(token ?? '', recordId)
      await load()
      onCompleted()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to complete receiving')
    } finally {
      setCompleting(false)
    }
  }

  if (loading) return <p className="text-[11px] dark:text-gray-500 text-gray-400 py-2">Loading…</p>
  if (!detail) return error ? <p className="text-[10px] text-red-400 py-2">{error}</p> : null

  const record = detail.receiving_record
  const isApplied = record.status === 'applied'

  return (
    <div className="space-y-2 pt-1">
      {detail.discrepancies.length === 0 ? (
        <p className="text-[11px] dark:text-gray-400 text-gray-500">
          The invoice matched the purchase order exactly — no discrepancies.
        </p>
      ) : (
        <div className="space-y-2">
          {detail.discrepancies.map(d => (
            <DiscrepancyRow
              key={d.id}
              discrepancy={d}
              lineItem={lineItems.find(li => li.id === d.line_item_id)}
              warehouseId={warehouseId}
              warehouses={warehouses}
              onAssignWarehouse={onAssignWarehouse}
              invoiceFileName={invoiceFileName}
              disabled={isApplied}
            />
          ))}
        </div>
      )}
      {error && <p className="text-[10px] text-red-400">{error}</p>}
      <div className="flex items-center justify-end pt-1">
        {isApplied ? (
          <span className="flex items-center gap-1 text-[10px] font-medium text-green-400">
            <PackageCheck size={11} /> Receiving completed
          </span>
        ) : (
          <button
            type="button" onClick={handleComplete} disabled={completing}
            className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-[10px] font-semibold hover:bg-blue-700 disabled:opacity-50"
          >
            <PackageCheck size={11} /> {completing ? 'Completing…' : 'Mark Receiving Complete'}
          </button>
        )}
      </div>
    </div>
  )
}

interface DiscrepancyRowProps {
  discrepancy: ReceivingDiscrepancy
  lineItem: PurchaseOrderLineItem | undefined
  warehouseId: string | null
  warehouses: Warehouse[]
  onAssignWarehouse: (warehouseId: string) => Promise<void>
  invoiceFileName: string
  disabled: boolean
}

function DiscrepancyRow({ discrepancy, lineItem, warehouseId, warehouses, onAssignWarehouse, invoiceFileName, disabled }: DiscrepancyRowProps) {
  const [formOpen, setFormOpen] = useState(false)
  const canReceive = !disabled && !formOpen && lineItem?.sku_id && discrepancy.received_qty > 0

  return (
    <div className="rounded-lg border dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40 p-2.5 space-y-1.5">
      <div className="flex items-center justify-between gap-2">
        <span className="text-[10px] font-semibold text-amber-400">{DISCREPANCY_LABEL[discrepancy.type]}</span>
        <span className="text-[10px] dark:text-gray-500 text-gray-400">
          expected {discrepancy.expected_qty} · invoice {discrepancy.received_qty}
        </span>
      </div>
      <p className="text-[11px] dark:text-gray-300 text-gray-600">{discrepancy.details}</p>
      {canReceive && (
        <button
          type="button" onClick={() => setFormOpen(true)}
          className="flex items-center gap-1 text-[10px] font-medium dark:text-blue-400 text-blue-600 hover:underline"
        >
          <PackageCheck size={10} /> Receive stock
        </button>
      )}
      {!disabled && lineItem && !lineItem.sku_id && discrepancy.received_qty > 0 && (
        <p className="text-[10px] dark:text-gray-500 text-gray-400">Link this line item to a product before receiving stock for it.</p>
      )}
      {formOpen && lineItem?.sku_id && (
        <ReceiveDiscrepancyForm
          skuId={lineItem.sku_id}
          warehouseId={warehouseId}
          warehouses={warehouses}
          onAssignWarehouse={onAssignWarehouse}
          defaultQuantity={discrepancy.received_qty}
          reason={`Receiving invoice ${invoiceFileName}`}
          onDone={() => setFormOpen(false)}
        />
      )}
    </div>
  )
}

interface ReceiveDiscrepancyFormProps {
  skuId: string
  warehouseId: string | null
  warehouses: Warehouse[]
  onAssignWarehouse: (warehouseId: string) => Promise<void>
  defaultQuantity: number
  reason: string
  onDone: () => void
}

function ReceiveDiscrepancyForm({ skuId, warehouseId, warehouses, onAssignWarehouse, defaultQuantity, reason, onDone }: ReceiveDiscrepancyFormProps) {
  const { token } = useAuth()
  const [quantity, setQuantity] = useState(String(defaultQuantity))
  const [lotNumber, setLotNumber] = useState('')
  const [expiryDate, setExpiryDate] = useState('')
  // Only used when the PO has no warehouse yet -- picking one here assigns
  // it to the PO for good (see onAssignWarehouse), it's not a per-scan choice.
  const [pickedWarehouseId, setPickedWarehouseId] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [done, setDone] = useState(false)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      let effectiveWarehouseId = warehouseId
      if (!effectiveWarehouseId) {
        if (!pickedWarehouseId) throw new Error('Pick a warehouse first')
        await onAssignWarehouse(pickedWarehouseId)
        effectiveWarehouseId = pickedWarehouseId
      }
      await inventoryApi.receiveStock(token ?? '', skuId, {
        warehouse_id: effectiveWarehouseId, quantity: parseInt(quantity, 10),
        lot_number: lotNumber || undefined, expiry_date: expiryDate || null, reason,
      })
      setDone(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to receive stock')
    } finally {
      setSaving(false)
    }
  }

  if (done) {
    return <p className="text-[10px] font-medium text-green-400">Stock received.</p>
  }

  const inputClass = 'w-full px-2 py-1 text-[11px] rounded dark:bg-[#0B1220] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500'

  return (
    <form onSubmit={handleSubmit} className="flex items-end gap-2 pt-1 flex-wrap">
      {!warehouseId && (
        <div>
          <label htmlFor={`receive-warehouse-${skuId}`} className="block text-[9px] dark:text-gray-500 text-gray-400 mb-0.5">Warehouse</label>
          <select
            id={`receive-warehouse-${skuId}`} required value={pickedWarehouseId}
            onChange={e => setPickedWarehouseId(e.target.value)} className={clsx(inputClass, 'w-28')}
          >
            <option value="">Select…</option>
            {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
          </select>
        </div>
      )}
      <div>
        <label htmlFor={`receive-qty-${skuId}`} className="block text-[9px] dark:text-gray-500 text-gray-400 mb-0.5">Qty</label>
        <input id={`receive-qty-${skuId}`} type="number" min="1" required value={quantity} onChange={e => setQuantity(e.target.value)} className={clsx(inputClass, 'w-14')} />
      </div>
      <div>
        <label htmlFor={`receive-lot-${skuId}`} className="block text-[9px] dark:text-gray-500 text-gray-400 mb-0.5">Lot #</label>
        <input id={`receive-lot-${skuId}`} value={lotNumber} onChange={e => setLotNumber(e.target.value)} className={clsx(inputClass, 'w-20')} />
      </div>
      <div>
        <label htmlFor={`receive-expiry-${skuId}`} className="block text-[9px] dark:text-gray-500 text-gray-400 mb-0.5">Expiry</label>
        <input id={`receive-expiry-${skuId}`} type="date" value={expiryDate} onChange={e => setExpiryDate(e.target.value)} className={inputClass} />
      </div>
      <button type="submit" disabled={saving} className="px-2.5 py-1 rounded-lg bg-blue-600 text-white text-[10px] font-semibold hover:bg-blue-700 disabled:opacity-50">
        {saving ? 'Saving…' : 'Save receipt'}
      </button>
      <button type="button" onClick={onDone} className="px-2 py-1 text-[10px] dark:text-gray-500 text-gray-400 hover:underline">
        Cancel
      </button>
      {error && <p className="text-[10px] text-red-400">{error}</p>}
    </form>
  )
}

// ── Review unmatched line item ──────────────────────────────────────────────

interface ReviewUnmatchedLineItemProps {
  lineItem: PurchaseOrderLineItem
  onResolved: () => void
  onCancel: () => void
}

function ReviewUnmatchedLineItem({ lineItem, onResolved, onCancel }: ReviewUnmatchedLineItemProps) {
  const { token } = useAuth()
  const [mode, setMode] = useState<'choose' | 'link' | 'create'>('choose')
  const [productName, setProductName] = useState(lineItem.raw_description)
  const [skuCode, setSkuCode] = useState(lineItem.raw_sku)
  const [price, setPrice] = useState('0.00')
  const [unitsPerPurchasedItem, setUnitsPerPurchasedItem] = useState('1')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleLink(sku: SKU) {
    setError('')
    setSaving(true)
    try {
      await procurementApi.linkLineItemToSku(token ?? '', lineItem.id, {
        sku_id: sku.id, units_per_purchased_item: parseInt(unitsPerPurchasedItem || '1', 10),
      })
      onResolved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to link line item')
      setSaving(false)
    }
  }

  async function handleCreateProduct(e: React.FormEvent) {
    e.preventDefault()
    setError('')
    setSaving(true)
    try {
      await procurementApi.createProductAndLinkLineItem(token ?? '', lineItem.id, {
        product_name: productName, sku_code: skuCode, currency_code: 'USD',
        price_minor_units: Math.round(parseFloat(price || '0') * 100),
      })
      onResolved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create product')
      setSaving(false)
    }
  }

  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 outline-none focus:ring-1 focus:ring-blue-500'
  const labelClass = 'block text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-1'

  return (
    <div className="rounded-lg border dark:border-amber-500/30 border-amber-300 dark:bg-[#0B1220] bg-white p-3 space-y-2">
      <div className="text-[11px] dark:text-gray-300 text-gray-600">
        Extracted from document{lineItem.raw_sku ? <> — printed SKU/barcode <span className="font-mono">{lineItem.raw_sku}</span></> : null}, but no matching product was found automatically.
      </div>

      {mode === 'choose' && (
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => setMode('link')}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-blue-600 text-white text-[10px] font-medium hover:bg-blue-700">
            <Search size={10} /> Link to existing product
          </button>
          <button type="button" onClick={() => setMode('create')}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg dark:bg-[#2D3748] bg-gray-100 text-[10px] dark:text-gray-300 text-gray-700 font-medium hover:dark:bg-[#374151] hover:bg-gray-200">
            <Plus size={10} /> Create new product
          </button>
          <button type="button" onClick={onCancel} className="text-[10px] dark:text-gray-500 text-gray-400 hover:underline ml-auto">
            Dismiss
          </button>
        </div>
      )}

      {mode === 'link' && (
        <div className="space-y-2">
          <div>
            <label htmlFor={`review-units-per-item-${lineItem.id}`} className={labelClass}>
              Units per purchased item
            </label>
            <input
              id={`review-units-per-item-${lineItem.id}`} type="number" min="1" step="1"
              value={unitsPerPurchasedItem} onChange={e => setUnitsPerPurchasedItem(e.target.value)}
              title="e.g. 36 if this line's purchased item is a box holding 36 of the linked product"
              className={clsx(inputClass, 'w-24')}
            />
          </div>
          <ProductSkuPicker onSelect={sku => handleLink(sku)} onClose={() => setMode('choose')} />
        </div>
      )}

      {mode === 'create' && (
        <form onSubmit={handleCreateProduct} className="space-y-2">
          <div className="grid grid-cols-2 gap-2">
            <div>
              <label htmlFor={`review-name-${lineItem.id}`} className={labelClass}>Product name</label>
              <input id={`review-name-${lineItem.id}`} required value={productName} onChange={e => setProductName(e.target.value)} className={inputClass} />
            </div>
            <div>
              <label htmlFor={`review-sku-${lineItem.id}`} className={labelClass}>SKU code</label>
              <input id={`review-sku-${lineItem.id}`} required value={skuCode} onChange={e => setSkuCode(e.target.value)} className={inputClass} />
            </div>
            <div>
              <label htmlFor={`review-price-${lineItem.id}`} className={labelClass}>Price</label>
              <input id={`review-price-${lineItem.id}`} type="number" min="0" step="0.01" value={price} onChange={e => setPrice(e.target.value)} className={inputClass} />
            </div>
          </div>
          <div className="flex items-center gap-2">
            <button type="submit" disabled={saving}
              className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-[10px] font-semibold hover:bg-blue-700 disabled:opacity-50">
              {saving ? 'Creating…' : 'Create & Link'}
            </button>
            <button type="button" onClick={() => setMode('choose')} className="text-[10px] dark:text-gray-500 text-gray-400 hover:underline">
              Back
            </button>
          </div>
        </form>
      )}

      {error && <p className="text-[10px] text-red-400">{error}</p>}
    </div>
  )
}

// ── Product/SKU Picker ───────────────────────────────────────────────────────

export interface ProductSkuPickerProps {
  onSelect: (sku: SKU, label: string) => void
  onClose: () => void
}

export function ProductSkuPicker({ onSelect, onClose }: ProductSkuPickerProps) {
  const { token } = useAuth()
  const [query, setQuery] = useState('')
  const { products } = useProducts(token ?? '', { search: query, perPage: 5 })
  const [expandedProductId, setExpandedProductId] = useState<string | null>(null)
  const [skus, setSkus] = useState<SKU[]>([])
  const [loadingSkus, setLoadingSkus] = useState(false)

  async function toggleProduct(productId: string) {
    if (expandedProductId === productId) {
      setExpandedProductId(null)
      return
    }
    setExpandedProductId(productId)
    setLoadingSkus(true)
    try {
      const detail = await pmsApi.getProduct(token ?? '', productId)
      setSkus(detail.variants.flatMap(v => v.skus ?? []))
    } finally {
      setLoadingSkus(false)
    }
  }

  return (
    <div className="rounded-lg border dark:border-[#2D3748] border-gray-200 dark:bg-[#0B1220] bg-white p-3 space-y-2">
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            autoFocus value={query} onChange={e => setQuery(e.target.value)}
            placeholder="Search products by name…"
            className="w-full pl-7 pr-2 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <button type="button" onClick={onClose} title="Close">
          <X size={14} className="dark:text-gray-500 text-gray-400" />
        </button>
      </div>
      <div className="max-h-48 overflow-auto space-y-1">
        {products.length === 0 && <p className="text-[10px] dark:text-gray-500 text-gray-400 px-1">No matching products</p>}
        {products.map(p => (
          <div key={p.id}>
            <button
              type="button"
              onClick={() => toggleProduct(p.id)}
              className="w-full text-left px-2 py-1.5 text-xs rounded dark:hover:bg-[#1F2937] hover:bg-gray-50 dark:text-gray-200 text-gray-800"
            >
              {p.name}
            </button>
            {expandedProductId === p.id && (
              <div className="pl-4 space-y-0.5">
                {loadingSkus && <p className="text-[10px] dark:text-gray-500 text-gray-400">Loading…</p>}
                {!loadingSkus && skus.length === 0 && <p className="text-[10px] dark:text-gray-500 text-gray-400">No SKUs on this product</p>}
                {!loadingSkus && skus.map(s => (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => onSelect(s, `${p.name} — ${s.sku_code}`)}
                    className="w-full flex items-center justify-between text-left px-2 py-1 text-[10px] rounded dark:hover:bg-[#1F2937] hover:bg-gray-50"
                  >
                    <span className="font-mono dark:text-gray-300 text-gray-600">{s.sku_code}</span>
                    <span className="dark:text-gray-500 text-gray-400">{(s.price_minor_units / 100).toFixed(2)} {s.currency_code}</span>
                  </button>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  )
}
