import { useState, useEffect, useRef, Fragment } from 'react'
import { Plus, ShoppingBag, Trash2, Edit3, ArrowLeft, Search, X, Check, Upload, Sparkles, PackageCheck, FileText, AlertTriangle, Loader2, Link2, ClipboardList, Download, RefreshCw } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useStackNavigation } from '../routing/useStackNavigation'
import { useQueryEnum, useQueryString } from '../routing/useQueryState'
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
import { Dialog } from '../components/common/Dialog'
import { useConfirm } from '../components/common/useConfirm'

const PROCUREMENT_PATH = '/procurement'
const DISCREPANCIES_SEGMENT = 'discrepancies'

const STATUS_TABS: readonly (PurchaseOrderStatus | 'all')[] = ['all', 'draft', 'sent', 'partially_received', 'received', 'needs_review', 'cancelled']

// Same list as Products.tsx's SKU currency select, for consistency.
const CURRENCIES = ['USD', 'EUR', 'GBP', 'CAD', 'AUD']

function todayDateString(): string {
  return new Date().toISOString().slice(0, 10)
}

const STATUS_STYLE: Record<PurchaseOrderStatus, string> = {
  draft: 'bg-gray-500/15 text-ink-4',
  sent: 'bg-accent/15 text-accent',
  partially_received: 'bg-amber-500/15 text-warning',
  received: 'bg-green-500/15 text-success',
  cancelled: 'bg-red-500/15 text-danger',
  needs_review: 'bg-amber-500/20 text-warning',
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
  pending_review: 'bg-amber-500/20 text-warning',
  applied: 'bg-green-500/15 text-success',
}

const DISCREPANCY_LABEL: Record<DiscrepancyType, string> = {
  quantity_mismatch: 'Quantity mismatch',
  missing_item: 'Missing from invoice',
  unexpected_item: 'Not on purchase order',
  expiry_mismatch: 'Expiry mismatch',
  damaged_item: 'Damaged',
}

export default function Procurement() {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { purchaseOrders, loading, error, create, remove, refresh } = usePurchaseOrders(token ?? '')
  const { suppliers } = useSuppliers(token ?? '')
  const { warehouses } = useWarehouses(token ?? '')

  // Everything that decides what's on screen lives in the URL, so refresh
  // and Back keep the user in place:
  //   /procurement?status=sent          list, filtered
  //   /procurement/new | /procurement/:id   PO editor
  //   /procurement/:id?parsing=<jobId>  editor still polling an upload's parse
  //   /procurement/discrepancies?tab=   booking discrepancies report
  const [statusFilter, setStatusFilter] = useQueryEnum('status', STATUS_TABS, 'all')
  const { editingId, openNew, openEdit, close, showSaved } = useRoutedEditor(PROCUREMENT_PATH)
  const { open } = useStackNavigation()
  // Set right after a document-upload create -- the PO page picks it up (see
  // LineItemsSection's initialParsingJobId) to poll the still-in-flight
  // parse and show a spinner instead of the create flow blocking on it.
  const [pendingParsingJobId] = useQueryString('parsing')
  const [deleteError, setDeleteError] = useState('')
  const [uploadModalOpen, setUploadModalOpen] = useState(false)

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
    if (!(await confirm({ title: 'Delete this purchase order?', message: 'This cannot be undone.', confirmLabel: 'Delete order', destructive: true }))) return
    setDeleteError('')
    try {
      await remove(po.id)
    } catch (err) {
      setDeleteError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  if (editingId === DISCREPANCIES_SEGMENT) {
    return <BookingDiscrepanciesPage token={token ?? ''} onClose={close} />
  }

  if (editingId !== null) {
    return (
      <PurchaseOrderEditPage
        key={editingId}
        purchaseOrderId={editingId === NEW_RECORD ? null : editingId}
        initialParsingJobId={editingId === NEW_RECORD ? null : pendingParsingJobId}
        onCreate={create}
        onCreated={showSaved}
        onClose={() => { close(); refresh() }}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="flex items-center gap-1 overflow-x-auto">
          {STATUS_TABS.map(s => (
            <button
              key={s}
              onClick={() => setStatusFilter(s)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-xs font-medium capitalize transition-colors whitespace-nowrap',
                statusFilter === s
                  ? 'dark:bg-night-800 bg-gray-100 text-ink'
                  : 'text-ink-4 hover:text-ink-2',
              )}
            >
              {s.replace('_', ' ')}
            </button>
          ))}
        </div>
        <div className="flex items-center gap-2 ml-auto shrink-0">
          <button
            onClick={() => open(`${PROCUREMENT_PATH}/${DISCREPANCIES_SEGMENT}`)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs text-ink-2 hover:bg-hover transition-colors font-medium"
          >
            <AlertTriangle size={13} /> Discrepancies
          </button>
          <button
            onClick={() => setUploadModalOpen(true)}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs text-ink-2 hover:bg-hover transition-colors font-medium"
          >
            <Upload size={13} /> Upload document
          </button>
          <button
            onClick={openNew}
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
          >
            <Plus size={13} /> Add purchase order
          </button>
        </div>
      </div>

      {uploadModalOpen && (
        <CreateFromDocumentModal
          onClose={() => setUploadModalOpen(false)}
          onCreated={async (poId, jobId) => {
            setUploadModalOpen(false)
            await refresh()
            open(`${PROCUREMENT_PATH}/${encodeURIComponent(poId)}?parsing=${encodeURIComponent(jobId)}`)
          }}
        />
      )}

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{filtered.length} purchase order{filtered.length === 1 ? '' : 's'}</span>
        {(error || deleteError) && <span className="text-xs text-danger ml-auto">{error || deleteError}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Supplier</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Warehouse</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Expected</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Currency</th>
              <th className="px-3 py-2.5 text-right font-medium text-ink-4 uppercase tracking-wider">Total</th>
              <th className="px-3 py-2.5 w-16" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={8} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={8} className="px-4 py-6 text-center text-ink-4">No purchase orders yet</td></tr>
            )}
            {filtered.map(po => (
              <tr
                key={po.id}
                onClick={() => openEdit(po.id)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
                    <ShoppingBag size={12} className="text-ink-4" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[280px]">
                  {supplierName(po.supplier_id) ? (
                    <p className="font-medium text-ink truncate">{supplierName(po.supplier_id)}</p>
                  ) : (
                    <p className="flex items-center gap-1 font-medium text-warning truncate">
                      <Sparkles size={11} /> Missing supplier
                    </p>
                  )}
                  {po.is_potential_duplicate && (
                    <button
                      type="button"
                      onClick={e => { e.stopPropagation(); if (po.duplicate_of_purchase_order_id) openEdit(po.duplicate_of_purchase_order_id) }}
                      className="flex items-center gap-1 mt-0.5 text-2xs font-medium text-danger hover:underline"
                    >
                      <AlertTriangle size={10} /> Possible duplicate order
                    </button>
                  )}
                </td>
                <td className="px-3 py-2.5 text-ink-3">{warehouseName(po.warehouse_id) ?? '—'}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize', STATUS_STYLE[po.status])}>
                    {po.status.replace('_', ' ')}
                  </span>
                </td>
                <td className="px-3 py-2.5 text-ink-3">{po.expected_date ?? '—'}</td>
                <td className="px-3 py-2.5 text-ink-3">{po.currency_code}</td>
                <td className="px-3 py-2.5 text-right text-ink font-medium">
                  {formatMoney(po.total_minor_units, po.currency_code)}
                </td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label="Edit"
                      onClick={e => { e.stopPropagation(); openEdit(po.id) }}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title="Edit"
                    >
                      <Edit3 size={12} className="text-ink-3" />
                    </button>
                    <button aria-label="Delete"
                      onClick={e => { e.stopPropagation(); handleDelete(po) }}
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
      {confirmDialog}
    </div>
  )
}

// ── Booking discrepancies report ────────────────────────────────────────────

function formatDiscrepancyDate(iso?: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString()
}

const DISCREPANCY_TABS = ['missing', 'damaged-lost'] as const
type DiscrepancyTab = typeof DISCREPANCY_TABS[number]

// Tenant-wide, derived on demand (not persisted) from every completed
// booking's line items vs scans -- same "not a real table" reasoning as
// the Inventory module's Expiry & Discards report.
function BookingDiscrepanciesPage({ token, onClose }: { token: string; onClose: () => void }) {
  const { report, loading, error } = useBookingDiscrepancies(token)
  const [tab, setTab] = useQueryEnum('tab', DISCREPANCY_TABS, 'missing')

  const TABS: { id: DiscrepancyTab; label: string }[] = [
    { id: 'missing', label: `Missing${report.missing_items.length ? ` (${report.missing_items.length})` : ''}` },
    { id: 'damaged-lost', label: `Damaged / Lost${report.damaged_lost.length ? ` (${report.damaged_lost.length})` : ''}` },
  ]

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Procurement
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">Booking discrepancies</h1>
          <p className="text-xs text-ink-4 mt-0.5">Every completed booking's under-scanned line items and damaged/lost stock</p>
        </div>
      </div>

      <div className="flex gap-2 px-6 pt-4 shrink-0">
        {TABS.map(t => (
          <button aria-pressed={tab === t.id}
            key={t.id}
            onClick={() => setTab(t.id)}
            className={clsx(
              'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              tab === t.id
                ? 'bg-accent-solid text-white'
                : 'bg-raised text-ink-3 hover:bg-hover',
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-danger">{error}</p>}

        {tab === 'missing' && (
          <>
            {!error && loading && report.missing_items.length === 0 && <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>}
            {!error && !loading && report.missing_items.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">No missing items -- every completed booking was fully scanned.</p>
            )}
            {report.missing_items.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                  <tr>
                    {['SKU', 'Product', 'Expected', 'Scanned', 'Missing', 'Completed'].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {report.missing_items.map(m => (
                    <tr key={m.line_item_id} className="hover:bg-hover/60 transition-colors">
                      <td className="px-4 py-2.5 font-mono text-ink-2">{m.sku_code ?? '—'}</td>
                      <td className="px-4 py-2.5 text-ink">{m.product_name ?? m.raw_description}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{m.expected_qty}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{m.scanned_qty}</td>
                      <td className="px-4 py-2.5 tabular-nums text-danger">{m.missing_qty}</td>
                      <td className="px-4 py-2.5 text-ink-3">{formatDiscrepancyDate(m.completed_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}

        {tab === 'damaged-lost' && (
          <>
            {!error && loading && report.damaged_lost.length === 0 && <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>}
            {!error && !loading && report.damaged_lost.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">Nothing damaged or lost across any completed booking.</p>
            )}
            {report.damaged_lost.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                  <tr>
                    {['SKU', 'Product', 'Type', 'Qty', 'Reason', 'Scanned by', 'Date'].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {report.damaged_lost.map(d => (
                    <tr key={d.scan_id} className="hover:bg-hover/60 transition-colors">
                      <td className="px-4 py-2.5 font-mono text-ink-2">{d.sku_code ?? '—'}</td>
                      <td className="px-4 py-2.5 text-ink">{d.product_name ?? '—'}</td>
                      <td className="px-4 py-2.5 capitalize text-danger">{d.action_type}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{d.quantity}</td>
                      <td className="px-4 py-2.5 text-ink-3">{d.reason || '—'}</td>
                      <td className="px-4 py-2.5 text-ink-3">{d.scanned_by || '—'}</td>
                      <td className="px-4 py-2.5 text-ink-3">{formatDiscrepancyDate(d.scanned_at)}</td>
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

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5'

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-md rounded-xl bg-surface border border-line shadow-xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-line">
          <h2 className="text-sm font-semibold text-ink">Upload document</h2>
          <button aria-label="Close" type="button" onClick={onClose} title="Close">
            <X size={16} className="text-ink-4" />
          </button>
        </div>
        <form onSubmit={handleSubmit} className="p-5 space-y-4">
          <p className="text-2xs text-ink-4">
            Upload a supplier PDF, CSV, or XLSX. Supplier and line items are extracted and matched to products
            automatically — anything that can't be matched will need review.
          </p>
          <div>
            <label htmlFor="upload-file" className={labelClass}>Document</label>
            <input id="upload-file" type="file" accept=".pdf,.csv,.xlsx"
              disabled={status !== 'idle'}
              onChange={e => setFile(e.target.files?.[0] ?? null)}
              className="w-full text-xs text-ink-2 file:mr-3 file:px-3 file:py-1.5 file:rounded-lg file:border-0 file:bg-accent-solid file:text-white file:text-xs file:font-medium file:cursor-pointer disabled:opacity-50" />
          </div>
          {formError && <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError}</p>}
          <div className="flex items-center justify-end gap-2 pt-1">
            <button type="button" onClick={onClose}
              className="px-4 py-2 rounded-lg border border-line bg-raised text-xs font-semibold text-ink-2 hover:bg-hover transition-colors">
              Cancel
            </button>
            <button type="submit" disabled={status !== 'idle' || !file}
              className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors">
              {status === 'uploading' ? 'Uploading…' : 'Upload & create'}
            </button>
          </div>
        </form>
      </div>
    </Dialog>
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
  // Called with the new id right after a create, so the URL can switch from
  // /procurement/new to the PO's own URL.
  onCreated: (id: string) => void
  onClose: () => void
}

function PurchaseOrderEditPage({ purchaseOrderId, initialParsingJobId, onCreate, onCreated, onClose }: PurchaseOrderEditPageProps) {
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
        onCreated(created.id)
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
            <ArrowLeft size={11} /> Purchase Orders
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">
            {isNew ? 'New purchase order' : 'Edit purchase order'}
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
            type="submit" form="po-form" disabled={saving || !supplierId}
            className="px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {saving ? 'Saving…' : isNew ? 'Create purchase order' : 'Save'}
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="max-w-3xl mx-auto p-6 space-y-5">
          <form id="po-form" onSubmit={handleSave} className="space-y-5">
            <div className="rounded-xl border border-line p-4 space-y-3">
              <div className="flex items-center justify-between">
                <h3 className="text-xs font-semibold text-ink">General</h3>
                {!isNew && (
                  <select
                    aria-label="Status"
                    value={status}
                    onChange={e => setStatus(e.target.value as PurchaseOrderStatus)}
                    className="px-2 py-1 text-2xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60 capitalize"
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
                        className="text-2xs font-semibold text-accent hover:underline disabled:opacity-50"
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
                    <p className="text-2xs text-accent mt-1">Suggested from the uploaded document — confirm to assign.</p>
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
                <p className="text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{formError || detailError}</p>
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
            <DocumentsSection documents={detail!.documents} onDelete={deleteDocument} onRefresh={refreshDetail} />
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
      <p className="flex items-center gap-1.5 text-xs font-semibold text-warning">
        <Sparkles size={12} /> Needs review: this purchase order has no supplier
      </p>
      <p className="text-2xs text-ink-3">
        Pick an existing supplier above, or create a new one and assign it in one step.
      </p>
      {suggestedName && (
        <p className="text-2xs text-accent">Suggested from the uploaded document — edit if it's wrong.</p>
      )}
      {matchedSupplier && (
        <p className="text-2xs text-success">
          Matches existing supplier "{matchedSupplier.name}" — this will assign it, not create a new one.
        </p>
      )}
      <form onSubmit={handleCreate} className="flex items-center gap-2">
        <input
          aria-label="New supplier name" required value={name} onChange={e => setName(e.target.value)}
          placeholder="New supplier name"
          className="flex-1 px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
        <button type="submit" disabled={saving || !name.trim()}
          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50 whitespace-nowrap">
          {saving ? (matchedSupplier ? 'Assigning…' : 'Creating…') : (matchedSupplier ? 'Assign existing supplier' : 'Create & assign')}
        </button>
      </form>
      {error && <p className="text-2xs text-danger">{error}</p>}
    </div>
  )
}

// ── Documents ────────────────────────────────────────────────────────────────

interface DocumentsSectionProps {
  documents: PurchaseOrderDocument[]
  onDelete: (id: string) => Promise<void>
  onRefresh: () => void
}

function DocumentsSection({ documents, onDelete, onRefresh }: DocumentsSectionProps) {
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const [deletingId, setDeletingId] = useState<string | null>(null)
  const [reparsingId, setReparsingId] = useState<string | null>(null)
  const [reparseError, setReparseError] = useState<{ id: string; message: string } | null>(null)

  async function handleDelete(doc: PurchaseOrderDocument) {
    if (!(await confirm({ title: `Delete "${doc.file_name}"?`, message: 'Line items already created from it are kept.', confirmLabel: 'Delete document', destructive: true }))) return
    setDeletingId(doc.id)
    try {
      await onDelete(doc.id)
    } finally {
      setDeletingId(null)
    }
  }

  async function handleReparse(doc: PurchaseOrderDocument) {
    setReparseError(null)
    setReparsingId(doc.id)
    try {
      const job = await procurementApi.retryDocumentParsingJob(token ?? '', doc.id)
      const finalJob = await pollParsingJob(() => procurementApi.getDocumentParsingJob(token ?? '', job.id))
      if (finalJob.status === 'failed') {
        setReparseError({ id: doc.id, message: friendlyParsingError(finalJob.error_message) })
      }
      onRefresh()
    } catch (err) {
      setReparseError({ id: doc.id, message: err instanceof Error ? err.message : 'Failed to reparse document' })
    } finally {
      setReparsingId(null)
    }
  }

  return (
    <div className="rounded-xl border border-line overflow-hidden">
      <div className="px-4 py-3 dark:bg-night-900/50 bg-gray-50 border-b border-line">
        <h3 className="text-xs font-semibold text-ink">Documents</h3>
      </div>
      <div className="divide-y divide-line">
        {documents.map(doc => (
          <div key={doc.id} className="px-4 py-2.5">
            <div className="flex items-center gap-2 text-xs text-ink-2">
              <FileText size={13} className="text-ink-4 shrink-0" />
              <a href={doc.file_url} target="_blank" rel="noreferrer" className="flex-1 min-w-0 truncate hover:underline">
                {doc.file_name}
              </a>
              {doc.is_potential_duplicate && (
                <span className="flex items-center gap-1 px-1.5 py-0.5 rounded-full text-2xs font-medium text-warning bg-amber-500/10 shrink-0">
                  <AlertTriangle size={10} /> Possible duplicate
                </span>
              )}
              <button aria-label="Reparse"
                type="button"
                title="Reparse"
                onClick={() => handleReparse(doc)}
                disabled={reparsingId === doc.id}
                className="p-1 rounded text-ink-4 hover:text-accent disabled:opacity-50 transition-colors shrink-0"
              >
                <RefreshCw size={13} className={reparsingId === doc.id ? 'animate-spin' : ''} />
              </button>
              <button aria-label="Delete"
                type="button"
                title="Delete"
                onClick={() => handleDelete(doc)}
                disabled={deletingId === doc.id}
                className="p-1 rounded text-ink-4 hover:text-danger disabled:opacity-50 transition-colors shrink-0"
              >
                <Trash2 size={13} />
              </button>
            </div>
            {reparseError?.id === doc.id && (
              <p className="mt-1 text-2xs text-danger pl-[21px]">{reparseError.message}</p>
            )}
          </div>
        ))}
      </div>
      {confirmDialog}
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
  const { confirm, confirmDialog } = useConfirm()
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
    if (!(await confirm({ title: 'Remove this line item?', confirmLabel: 'Remove line item', destructive: true }))) return
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
      if (!aiConnection && !(await confirm({
        title: 'Upload without auto-parsing?',
        message: 'No AI provider is configured, so this PDF won’t be parsed into line items. You can still upload it and add line items manually.',
        confirmLabel: 'Upload anyway',
      }))) {
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
    <div className="rounded-xl border border-line overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 dark:bg-night-900/50 bg-gray-50 border-b border-line">
        <h3 className="text-xs font-semibold text-ink">Line items</h3>
        <div className="flex items-center gap-2">
          {uploadStatus === 'uploading' && (
            <span className="flex items-center gap-1 text-2xs text-ink-4">
              <Loader2 size={11} className="animate-spin" /> Uploading…
            </span>
          )}
          {uploadStatus === 'parsing' && (
            <span className="flex items-center gap-1 text-2xs text-warning">
              <Loader2 size={11} className="animate-spin" />
              {parsingSlow ? 'Still parsing… this can take a bit longer' : 'Parsing document…'}
            </span>
          )}
          <input ref={fileInputRef} type="file" aria-label="Upload document" accept=".pdf,.csv,.xlsx" className="hidden" onChange={handleFileSelected} />
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadStatus !== 'idle'}
            className="flex items-center gap-1 px-2.5 py-1 rounded-lg border border-line bg-raised text-2xs text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
          >
            <Upload size={11} /> Upload document
          </button>
          {!adding && (
            <button
              type="button"
              onClick={() => setAdding(true)}
              className="flex items-center gap-1 px-2.5 py-1 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover"
            >
              <Plus size={11} /> Add line item
            </button>
          )}
        </div>
      </div>
      {uploadError && (
        <p className="text-2xs text-danger bg-red-50 dark:bg-red-500/10 px-4 py-1.5">{uploadError}</p>
      )}

      <table className="w-full text-xs">
        <thead>
          <tr className="dark:bg-night-900/30 bg-gray-50">
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">Description</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">Type</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">Linked SKU</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">Batch</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">Qty</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">Unit cost</th>
            <th className="px-4 py-2 text-left font-medium text-ink-4 uppercase tracking-wider text-2xs">Total</th>
            <th className="px-4 py-2 w-20" />
          </tr>
        </thead>
        <tbody className="divide-y divide-line/50">
          {lineItems.length === 0 && !adding && (
            <tr><td colSpan={8} className="px-4 py-6 text-center text-ink-4">No line items yet</td></tr>
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
                <tr className={clsx('transition-colors', li.sku_id || isCharge ? 'hover:bg-hover/30' : 'dark:bg-amber-500/5 bg-amber-50/40')}>
                  <td className="px-4 py-2.5 text-ink">{li.raw_description}</td>
                  <td className="px-4 py-2.5 text-ink-3">
                    {isCharge && (
                      <div className="flex items-center gap-1.5">
                        <span className="text-2xs">{LINE_TYPE_LABELS[li.line_type as Exclude<LineItemType, ''>]}</span>
                        {!li.line_type_confirmed && (
                          <button
                            type="button"
                            onClick={() => onUpdate(li.id, {
                              sku_id: li.sku_id ?? null, raw_sku: li.raw_sku, raw_description: li.raw_description,
                              quantity: li.quantity, unit_cost_minor_units: li.unit_cost_minor_units, line_type: li.line_type,
                            })}
                            title="AI-classified -- confirm this is correct"
                            className="text-2xs font-medium text-accent hover:underline"
                          >
                            Confirm
                          </button>
                        )}
                      </div>
                    )}
                  </td>
                  <td className="px-4 py-2.5 text-ink-3">
                    {isCharge ? (
                      <span className="text-2xs">—</span>
                    ) : li.sku_id ? (
                      li.product_name ? (
                        <span className="text-2xs">
                          {li.product_name} <span className="font-mono text-ink-4">— {li.sku_code}</span>
                          {li.units_per_purchased_item > 1 && (
                            <span className="ml-1 text-warning" title="Units per purchased item">
                              × {li.units_per_purchased_item}
                            </span>
                          )}
                        </span>
                      ) : (
                        <span className="font-mono text-2xs">{li.sku_id}</span>
                      )
                    ) : (
                      <button
                        type="button"
                        onClick={() => setReviewingLineId(reviewingLineId === li.id ? null : li.id)}
                        className="flex items-center gap-1 text-2xs font-medium text-warning hover:underline"
                      >
                        <Sparkles size={10} /> Needs review
                      </button>
                    )}
                  </td>
                  <td className="px-4 py-2.5 text-ink-3">
                    {li.suggested_lot_number ? (
                      <span className="text-2xs" title="Parsed from the uploaded document -- confirmed when the batch is assigned during booking">
                        {li.suggested_lot_number}
                        {li.suggested_expiry_date && (
                          <span className="text-ink-4"> · exp {li.suggested_expiry_date}</span>
                        )}
                      </span>
                    ) : (
                      <span className="text-2xs">—</span>
                    )}
                  </td>
                  <td className="px-4 py-2.5 text-ink">{li.quantity}</td>
                  <td className="px-4 py-2.5 text-ink">{formatMoney(li.unit_cost_minor_units, currencyCode)}</td>
                  <td className="px-4 py-2.5 text-ink font-medium">
                    {formatMoney(signedLineItemAmount(li), currencyCode)}
                  </td>
                  <td className="px-4 py-2.5">
                    <div className="flex items-center gap-1">
                      <button aria-label="Edit"
                        type="button"
                        onClick={() => setEditingLineId(li.id)}
                        className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                        title="Edit"
                      >
                        <Edit3 size={12} className="text-ink-3" />
                      </button>
                      <button aria-label="Remove"
                        type="button"
                        onClick={() => handleRemove(li.id)}
                        className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                        title="Remove"
                      >
                        <Trash2 size={12} className="text-ink-3 hover:text-danger" />
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
                  className="flex items-center gap-1 text-2xs font-medium text-accent hover:underline"
                >
                  <Plus size={11} /> Add line item
                </button>
              </td>
            </tr>
          )}
        </tbody>
        {lineItems.length > 0 && (
          <tfoot>
            <tr className="border-t border-line font-medium">
              <td colSpan={6} className="px-4 py-2.5 text-right text-ink-3 uppercase tracking-wider text-2xs">
                Total
              </td>
              <td className="px-4 py-2.5 text-ink">
                {formatMoney(lineItems.reduce((sum, li) => sum + signedLineItemAmount(li), 0), currencyCode)}
              </td>
              <td className="px-4 py-2.5" />
            </tr>
          </tfoot>
        )}
      </table>
      {confirmDialog}
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

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-night-950 bg-white border border-line text-ink outline-none focus:ring-1 focus:ring-focus'
  const idPrefix = initial?.id ?? 'new'

  return (
    <tr className="bg-accent/5">
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
                <p className="text-xs text-ink-4 py-2">— (charge rows don't link to a product)</p>
              ) : skuId ? (
                <div className="flex items-center gap-2 py-1">
                  <span
                    className={clsx('text-xs truncate', skuLabel ? 'text-ink' : 'font-mono text-ink-3')}
                    title={skuLabel ?? skuId}
                  >
                    {skuLabel ?? skuId}
                  </span>
                  <button
                    type="button" onClick={() => { setSkuId(null); setSkuLabel(null) }}
                    className="flex items-center gap-1 text-2xs text-ink-4 hover:text-danger shrink-0"
                  >
                    <X size={11} /> Unlink
                  </button>
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => setPickerOpen(true)}
                  className="flex items-center gap-1 text-xs text-accent hover:underline py-2"
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
                <p className="text-2xs text-ink-4 mt-1">
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
              <p className="px-3 py-2 text-xs text-ink-2">
                {(signedLineItemAmount({
                  line_type: lineType, quantity: parseInt(quantity || '0', 10),
                  unit_cost_minor_units: Math.round(parseFloat(unitCost || '0') * 100),
                }) / 100).toFixed(2)}
              </p>
            </div>
          </div>

          {rowError && <p className="text-2xs text-danger mt-2">{rowError}</p>}

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
              className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
            >
              <Check size={12} /> Save line
            </button>
            <button
              type="button" onClick={onCancel}
              className="px-3 py-1.5 rounded-lg text-xs text-ink-2 hover:bg-hover"
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
    <div className="rounded-xl border border-line overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 dark:bg-night-900/50 bg-gray-50 border-b border-line">
        <h3 className="text-xs font-semibold text-ink">Receiving</h3>
        <div className="flex items-center gap-2">
          {uploadStatus === 'uploading' && <span className="text-2xs text-ink-4">Uploading…</span>}
          {uploadStatus === 'parsing' && <span className="text-2xs text-warning">Parsing invoice…</span>}
          <input ref={fileInputRef} type="file" aria-label="Upload invoice" accept=".pdf,.csv,.xlsx" className="hidden" onChange={handleFileSelected} />
          <button
            type="button"
            onClick={() => fileInputRef.current?.click()}
            disabled={uploadStatus !== 'idle'}
            className="flex items-center gap-1 px-2.5 py-1 rounded-lg border border-line bg-raised text-2xs text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
          >
            <Upload size={11} /> Upload invoice
          </button>
        </div>
      </div>
      {(uploadError || error) && (
        <p className="text-2xs text-danger bg-red-50 dark:bg-red-500/10 px-4 py-1.5">{uploadError || error}</p>
      )}
      <div className="divide-y divide-line/50">
        {loading && <p className="px-4 py-6 text-center text-xs text-ink-4">Loading…</p>}
        {!loading && records.length === 0 && (
          <p className="px-4 py-6 text-center text-xs text-ink-4">No invoices uploaded yet</p>
        )}
        {records.map(record => (
          <div key={record.id}>
            <button
              type="button"
              onClick={() => setExpandedId(expandedId === record.id ? null : record.id)}
              className="w-full flex items-center gap-3 px-4 py-2.5 text-left hover:bg-hover/40 transition-colors"
            >
              <FileText size={13} className="text-ink-4 shrink-0" />
              <span className="flex-1 text-xs text-ink truncate">{record.invoice_file_name}</span>
              {record.is_potential_duplicate && (
                <span
                  title="Another invoice with identical file content was already uploaded for this supplier"
                  className="flex items-center gap-1 px-2 py-0.5 rounded-full text-2xs font-medium bg-amber-100 dark:bg-amber-500/10 text-warning"
                >
                  <AlertTriangle size={10} /> Possible duplicate
                </span>
              )}
              <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium capitalize', RECEIVING_STATUS_STYLE[record.status])}>
                {record.status.replace('_', ' ')}
              </span>
            </button>
            {expandedId === record.id && (
              <div className="px-4 pb-3 dark:bg-night-950/40 bg-gray-50/60">
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
  const [damagedFormOpen, setDamagedFormOpen] = useState(false)
  const [reportOpen, setReportOpen] = useState(false)

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

  if (loading) return <p className="text-2xs text-ink-4 py-2">Loading…</p>
  if (!detail) return error ? <p className="text-2xs text-danger py-2">{error}</p> : null

  const record = detail.receiving_record
  const isApplied = record.status === 'applied'

  return (
    <div className="space-y-2 pt-1">
      {detail.discrepancies.length === 0 ? (
        <p className="text-2xs text-ink-3">
          The invoice matched the purchase order exactly — no discrepancies.
        </p>
      ) : (
        <div className="space-y-2">
          {detail.discrepancies.map(d => (
            <DiscrepancyRow
              key={d.id}
              discrepancy={d}
              allDiscrepancies={detail.discrepancies}
              lineItem={lineItems.find(li => li.id === d.line_item_id)}
              warehouseId={warehouseId}
              warehouses={warehouses}
              onAssignWarehouse={onAssignWarehouse}
              invoiceFileName={invoiceFileName}
              disabled={isApplied}
              onReload={load}
            />
          ))}
        </div>
      )}
      {error && <p className="text-2xs text-danger">{error}</p>}
      {!isApplied && (
        damagedFormOpen ? (
          <FlagDamagedItemForm recordId={recordId} lineItems={lineItems} onDone={() => { setDamagedFormOpen(false); load() }} onCancel={() => setDamagedFormOpen(false)} />
        ) : (
          <button
            type="button" onClick={() => setDamagedFormOpen(true)}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:underline"
          >
            <AlertTriangle size={10} /> Flag damaged item
          </button>
        )
      )}
      <div className="flex items-center justify-between pt-1">
        <button
          type="button" onClick={() => setReportOpen(true)}
          className="flex items-center gap-1 px-3 py-1.5 rounded-lg dark:bg-white/5 bg-gray-100 text-ink-2 text-2xs font-semibold hover:dark:bg-white/10 hover:bg-gray-200"
        >
          <ClipboardList size={11} /> Generate report
        </button>
        {isApplied ? (
          <span className="flex items-center gap-1 text-2xs font-medium text-success">
            <PackageCheck size={11} /> Receiving completed
          </span>
        ) : (
          <button
            type="button" onClick={handleComplete} disabled={completing}
            className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
          >
            <PackageCheck size={11} /> {completing ? 'Completing…' : 'Mark receiving complete'}
          </button>
        )}
      </div>
      {reportOpen && (
        <ReceivingReportModal
          discrepancies={detail.discrepancies} invoiceFileName={invoiceFileName}
          onClose={() => setReportOpen(false)}
        />
      )}
    </div>
  )
}

interface DiscrepancyRowProps {
  discrepancy: ReceivingDiscrepancy
  allDiscrepancies: ReceivingDiscrepancy[]
  lineItem: PurchaseOrderLineItem | undefined
  warehouseId: string | null
  warehouses: Warehouse[]
  onAssignWarehouse: (warehouseId: string) => Promise<void>
  invoiceFileName: string
  disabled: boolean
  onReload: () => Promise<void> | void
}

function DiscrepancyRow({
  discrepancy, allDiscrepancies, lineItem, warehouseId, warehouses, onAssignWarehouse, invoiceFileName, disabled, onReload,
}: DiscrepancyRowProps) {
  const { token } = useAuth()
  const [formOpen, setFormOpen] = useState(false)
  const [linkOpen, setLinkOpen] = useState(false)
  const [linkTargetId, setLinkTargetId] = useState('')
  const [acting, setActing] = useState(false)
  const [actionError, setActionError] = useState('')
  const canReceive = !disabled && !formOpen && lineItem?.sku_id && discrepancy.received_qty > 0
  const isAcknowledged = discrepancy.status === 'acknowledged'

  // Candidates for manual linking: sibling discrepancies of the opposite
  // missing/unexpected type in the same record that are still open -- the
  // matcher's own fuzzy pass already handled everything it could connect
  // confidently, this is the human escape hatch for what's left.
  const linkCandidates = !isAcknowledged && (discrepancy.type === 'unexpected_item' || discrepancy.type === 'missing_item')
    ? allDiscrepancies.filter(d => d.status !== 'acknowledged' && d.type === (discrepancy.type === 'unexpected_item' ? 'missing_item' : 'unexpected_item'))
    : []

  async function handleAcknowledge() {
    setActing(true)
    setActionError('')
    try {
      await procurementApi.acknowledgeReceivingDiscrepancy(token ?? '', discrepancy.id)
      await onReload()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to acknowledge discrepancy')
    } finally {
      setActing(false)
    }
  }

  async function handleLink() {
    if (!linkTargetId) return
    setActing(true)
    setActionError('')
    try {
      const [unexpectedId, missingId] = discrepancy.type === 'unexpected_item'
        ? [discrepancy.id, linkTargetId]
        : [linkTargetId, discrepancy.id]
      await procurementApi.linkReceivingDiscrepancies(token ?? '', unexpectedId, missingId)
      setLinkOpen(false)
      await onReload()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to link discrepancies')
    } finally {
      setActing(false)
    }
  }

  return (
    <div className={clsx(
      'rounded-lg border p-2.5 space-y-1.5',
      isAcknowledged
        ? 'dark:border-white/10 border-gray-200 dark:bg-white/[0.02] bg-gray-50'
        : 'dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40',
    )}>
      <div className="flex items-center justify-between gap-2">
        <span className={clsx('text-2xs font-semibold', isAcknowledged ? 'text-ink-4' : 'text-warning')}>
          {DISCREPANCY_LABEL[discrepancy.type]}
        </span>
        <span className="text-2xs text-ink-4">
          expected {discrepancy.expected_qty} · invoice {discrepancy.received_qty}
        </span>
      </div>
      <p className="text-2xs text-ink-2">{discrepancy.details}</p>
      {isAcknowledged ? (
        <span className="flex items-center gap-1 text-2xs font-medium text-success">
          <Check size={10} /> Acknowledged
        </span>
      ) : (
        <div className="flex flex-wrap items-center gap-3">
          {canReceive && (
            <button
              type="button" onClick={() => setFormOpen(true)}
              className="flex items-center gap-1 text-2xs font-medium text-accent hover:underline"
            >
              <PackageCheck size={10} /> Receive stock
            </button>
          )}
          {!disabled && (
            <button
              type="button" onClick={handleAcknowledge} disabled={acting}
              className="flex items-center gap-1 text-2xs font-medium text-ink-3 hover:underline disabled:opacity-50"
            >
              <Check size={10} /> Mark as valid
            </button>
          )}
          {!disabled && linkCandidates.length > 0 && !linkOpen && (
            <button
              type="button" onClick={() => setLinkOpen(true)}
              className="flex items-center gap-1 text-2xs font-medium text-ink-3 hover:underline"
            >
              <Link2 size={10} /> Link to…
            </button>
          )}
        </div>
      )}
      {linkOpen && (
        <div className="flex items-center gap-2 pt-1">
          <select
            aria-label="Link to discrepancy"
            value={linkTargetId} onChange={e => setLinkTargetId(e.target.value)}
            className="flex-1 text-2xs rounded-md border border-line dark:bg-night-950 bg-white text-ink-2 px-2 py-1"
          >
            <option value="">Select the matching item…</option>
            {linkCandidates.map(c => <option key={c.id} value={c.id}>{c.details}</option>)}
          </select>
          <button
            type="button" onClick={handleLink} disabled={acting || !linkTargetId}
            className="text-2xs font-medium text-accent hover:underline disabled:opacity-50"
          >
            Confirm
          </button>
          <button type="button" onClick={() => setLinkOpen(false)} className="text-2xs text-ink-4 hover:underline">
            Cancel
          </button>
        </div>
      )}
      {actionError && <p className="text-2xs text-danger">{actionError}</p>}
      {!disabled && lineItem && !lineItem.sku_id && discrepancy.received_qty > 0 && (
        <p className="text-2xs text-ink-4">Link this line item to a product before receiving stock for it.</p>
      )}
      {formOpen && lineItem?.sku_id && (
        <ReceiveDiscrepancyForm
          skuId={lineItem.sku_id}
          warehouseId={warehouseId}
          warehouses={warehouses}
          onAssignWarehouse={onAssignWarehouse}
          defaultQuantity={discrepancy.received_qty}
          unitsPerPurchasedItem={lineItem.units_per_purchased_item}
          reason={`Receiving invoice ${invoiceFileName}`}
          onDone={() => setFormOpen(false)}
        />
      )}
    </div>
  )
}

interface FlagDamagedItemFormProps {
  recordId: string
  lineItems: PurchaseOrderLineItem[]
  onDone: () => void
  onCancel: () => void
}

function FlagDamagedItemForm({ recordId, lineItems, onDone, onCancel }: FlagDamagedItemFormProps) {
  const { token } = useAuth()
  const [lineItemId, setLineItemId] = useState(lineItems[0]?.id ?? '')
  const [qty, setQty] = useState('1')
  const [note, setNote] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    const damagedQty = Number(qty)
    if (!lineItemId || !Number.isFinite(damagedQty) || damagedQty <= 0) {
      setError('Pick a line item and a positive quantity')
      return
    }
    setSaving(true)
    setError('')
    try {
      await procurementApi.flagDamagedLineItem(token ?? '', recordId, lineItemId, damagedQty, note)
      onDone()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to flag damaged item')
      setSaving(false)
    }
  }

  return (
    <form onSubmit={handleSubmit} className="rounded-lg border border-line p-2.5 space-y-2">
      <div className="flex items-center gap-2">
        <select
          value={lineItemId} onChange={e => setLineItemId(e.target.value)} required
          className="flex-1 text-2xs rounded-md border border-line dark:bg-night-950 bg-white text-ink-2 px-2 py-1"
        >
          {lineItems.map(li => <option key={li.id} value={li.id}>{li.raw_description}</option>)}
        </select>
        <input
          type="number" min="1" value={qty} onChange={e => setQty(e.target.value)} required
          className="w-16 text-2xs rounded-md border border-line dark:bg-night-950 bg-white text-ink-2 px-2 py-1"
        />
      </div>
      <input
        type="text" placeholder="Note (optional)" value={note} onChange={e => setNote(e.target.value)}
        className="w-full text-2xs rounded-md border border-line dark:bg-night-950 bg-white text-ink-2 px-2 py-1"
      />
      {error && <p className="text-2xs text-danger">{error}</p>}
      <div className="flex items-center gap-3">
        <button
          type="submit" disabled={saving}
          className="text-2xs font-medium text-accent hover:underline disabled:opacity-50"
        >
          {saving ? 'Saving…' : 'Flag as damaged'}
        </button>
        <button type="button" onClick={onCancel} className="text-2xs text-ink-4 hover:underline">
          Cancel
        </button>
      </div>
    </form>
  )
}

interface ReceivingReportModalProps {
  discrepancies: ReceivingDiscrepancy[]
  invoiceFileName: string
  onClose: () => void
}

function ReceivingReportModal({ discrepancies, invoiceFileName, onClose }: ReceivingReportModalProps) {
  const grouped = (Object.keys(DISCREPANCY_LABEL) as DiscrepancyType[]).map(type => ({
    type, items: discrepancies.filter(d => d.type === type),
  })).filter(g => g.items.length > 0)

  function handleExportCsv() {
    const header = ['Type', 'Expected Qty', 'Received Qty', 'Status', 'Details']
    const rows = discrepancies.map(d => [
      DISCREPANCY_LABEL[d.type], String(d.expected_qty), String(d.received_qty), d.status, d.details,
    ])
    const csv = [header, ...rows]
      .map(row => row.map(cell => `"${cell.replace(/"/g, '""')}"`).join(','))
      .join('\n')
    const blob = new Blob([csv], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `receiving-report-${invoiceFileName || 'export'}.csv`
    document.body.appendChild(a)
    a.click()
    a.remove()
    URL.revokeObjectURL(url)
  }

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 p-4">
      <div className="w-full max-w-lg max-h-[85vh] flex flex-col rounded-xl dark:bg-night-900 bg-white border border-line shadow-xl">
        <div className="flex items-center justify-between px-5 py-4 border-b border-line">
          <h2 className="text-sm font-semibold text-ink">Receiving discrepancy report</h2>
          <button aria-label="Close" type="button" onClick={onClose} title="Close">
            <X size={16} className="text-ink-4" />
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-5 space-y-4">
          {grouped.length === 0 ? (
            <p className="text-2xs text-ink-3">No discrepancies to report.</p>
          ) : (
            grouped.map(g => (
              <div key={g.type} className="space-y-1.5">
                <h3 className="text-2xs font-semibold text-ink-2">
                  {DISCREPANCY_LABEL[g.type]} <span className="dark:text-ink-4 font-normal">({g.items.length})</span>
                </h3>
                <ul className="space-y-1">
                  {g.items.map(d => (
                    <li key={d.id} className="text-2xs text-ink-3 flex items-center justify-between gap-2">
                      <span>{d.details}</span>
                      <span className={clsx('shrink-0', d.status === 'acknowledged' ? 'text-green-400' : 'text-amber-400')}>
                        {d.status === 'acknowledged' ? 'Acknowledged' : 'Open'}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            ))
          )}
        </div>
        <div className="flex items-center justify-end gap-2 px-5 py-3 border-t border-line">
          <button
            type="button" onClick={handleExportCsv}
            className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-fill text-ink-2 text-2xs font-semibold hover:bg-hover"
          >
            <Download size={11} /> Export CSV
          </button>
        </div>
      </div>
    </Dialog>
  )
}

interface ReceiveDiscrepancyFormProps {
  skuId: string
  warehouseId: string | null
  warehouses: Warehouse[]
  onAssignWarehouse: (warehouseId: string) => Promise<void>
  defaultQuantity: number
  // Same convention as PurchaseOrderLineItem.UnitsPerPurchasedItem: how many
  // stock units one purchased item (box/case) represents. defaultQuantity
  // and the "Qty" field below stay in purchased-item units throughout --
  // matching received_qty/expected_qty (see diffInvoiceAgainstPurchaseOrder,
  // which diffs the invoice's own quantity against the PO line's Quantity,
  // both already in that unit) -- only the actual receive-stock call is
  // scaled up to stock units, same split as ScanBookingItemUseCase's
  // scaleByLineItem on the scanner side.
  unitsPerPurchasedItem: number
  reason: string
  onDone: () => void
}

function ReceiveDiscrepancyForm({ skuId, warehouseId, warehouses, onAssignWarehouse, defaultQuantity, unitsPerPurchasedItem, reason, onDone }: ReceiveDiscrepancyFormProps) {
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
      const effectiveMultiplier = unitsPerPurchasedItem > 0 ? unitsPerPurchasedItem : 1
      await inventoryApi.receiveStock(token ?? '', skuId, {
        warehouse_id: effectiveWarehouseId, quantity: parseInt(quantity, 10) * effectiveMultiplier,
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
    return <p className="text-2xs font-medium text-green-400">Stock received.</p>
  }

  const inputClass = 'w-full px-2 py-1 text-2xs rounded dark:bg-night-950 bg-white border dark:border-night-700 text-ink outline-none focus:ring-1 focus:ring-focus'

  return (
    <form onSubmit={handleSubmit} className="flex items-end gap-2 pt-1 flex-wrap">
      {!warehouseId && (
        <div>
          <label htmlFor={`receive-warehouse-${skuId}`} className="block text-2xs text-ink-4 mb-0.5">Warehouse</label>
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
        <label htmlFor={`receive-qty-${skuId}`} className="block text-2xs text-ink-4 mb-0.5">Qty</label>
        <input id={`receive-qty-${skuId}`} type="number" min="1" required value={quantity} onChange={e => setQuantity(e.target.value)} className={clsx(inputClass, 'w-14')} />
        {unitsPerPurchasedItem > 1 && Number.isFinite(parseInt(quantity, 10)) && (
          <p className="text-2xs text-ink-4 mt-0.5">= {parseInt(quantity, 10) * unitsPerPurchasedItem} units</p>
        )}
      </div>
      <div>
        <label htmlFor={`receive-lot-${skuId}`} className="block text-2xs text-ink-4 mb-0.5">Lot #</label>
        <input id={`receive-lot-${skuId}`} value={lotNumber} onChange={e => setLotNumber(e.target.value)} className={clsx(inputClass, 'w-20')} />
      </div>
      <div>
        <label htmlFor={`receive-expiry-${skuId}`} className="block text-2xs text-ink-4 mb-0.5">Expiry</label>
        <input id={`receive-expiry-${skuId}`} type="date" value={expiryDate} onChange={e => setExpiryDate(e.target.value)} className={inputClass} />
      </div>
      <button type="submit" disabled={saving} className="px-2.5 py-1 rounded-lg bg-accent-solid text-white text-2xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50">
        {saving ? 'Saving…' : 'Save receipt'}
      </button>
      <button type="button" onClick={onDone} className="px-2 py-1 text-2xs text-ink-4 hover:underline">
        Cancel
      </button>
      {error && <p className="text-2xs text-red-400">{error}</p>}
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
  const { settings: tenantSettings } = useTenantSettings(token ?? '')
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
        product_name: productName, sku_code: skuCode,
        currency_code: tenantSettings?.default_currency || 'USD',
        price_minor_units: Math.round(parseFloat(price || '0') * 100),
      })
      onResolved()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to create product')
      setSaving(false)
    }
  }

  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-night-800 bg-white border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'
  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1'

  return (
    <div className="rounded-lg border dark:border-amber-500/30 border-amber-300 dark:bg-night-950 bg-white p-3 space-y-2">
      <div className="text-2xs text-ink-3">
        Extracted from document{lineItem.raw_sku ? <> — printed SKU/barcode <span className="font-mono">{lineItem.raw_sku}</span></> : null}, but no matching product was found automatically.
      </div>

      {mode === 'choose' && (
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => setMode('link')}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover">
            <Search size={10} /> Link to existing product
          </button>
          <button type="button" onClick={() => setMode('create')}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg bg-fill text-2xs text-ink-2 font-medium hover:bg-hover">
            <Plus size={10} /> Create new product
          </button>
          <button type="button" onClick={onCancel} className="text-2xs text-ink-4 hover:underline ml-auto">
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
              className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50">
              {saving ? 'Creating…' : 'Create & link'}
            </button>
            <button type="button" onClick={() => setMode('choose')} className="text-2xs text-ink-4 hover:underline">
              Back
            </button>
          </div>
        </form>
      )}

      {error && <p className="text-2xs text-red-400">{error}</p>}
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
    <div className="rounded-lg border border-line dark:bg-night-950 bg-white p-3 space-y-2">
      <div className="flex items-center gap-2">
        <div className="relative flex-1">
          <Search size={12} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            autoFocus value={query} onChange={e => setQuery(e.target.value)}
            placeholder="Search products by name…"
            className="w-full pl-7 pr-2 py-1.5 text-xs rounded-lg dark:bg-night-800 bg-white border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
        <button aria-label="Close" type="button" onClick={onClose} title="Close">
          <X size={14} className="text-ink-4" />
        </button>
      </div>
      <div className="max-h-48 overflow-auto space-y-1">
        {products.length === 0 && <p className="text-2xs text-ink-4 px-1">No matching products</p>}
        {products.map(p => (
          <div key={p.id}>
            <button
              type="button"
              onClick={() => toggleProduct(p.id)}
              className="w-full text-left px-2 py-1.5 text-xs rounded hover:bg-hover text-ink"
            >
              {p.name}
            </button>
            {expandedProductId === p.id && (
              <div className="pl-4 space-y-0.5">
                {loadingSkus && <p className="text-2xs text-ink-4">Loading…</p>}
                {!loadingSkus && skus.length === 0 && <p className="text-2xs text-ink-4">No SKUs on this product</p>}
                {!loadingSkus && skus.map(s => (
                  <button
                    key={s.id}
                    type="button"
                    onClick={() => onSelect(s, `${p.name} — ${s.sku_code}`)}
                    className="w-full flex items-center justify-between text-left px-2 py-1 text-2xs rounded hover:bg-hover"
                  >
                    <span className="font-mono text-ink-2">{s.sku_code}</span>
                    <span className="text-ink-4">{(s.price_minor_units / 100).toFixed(2)} {s.currency_code}</span>
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
