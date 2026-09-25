import { useState } from 'react'
import { Plus, ShoppingBag, Trash2, Edit3, Upload, Sparkles, AlertTriangle } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useRoutedEditor, NEW_RECORD } from '../routing/useRoutedEditor'
import { useStackNavigation } from '../routing/useStackNavigation'
import { useQueryEnum, useQueryString } from '../routing/useQueryState'
import { usePurchaseOrders } from '../procurement/hooks/usePurchaseOrders'
import { useSuppliers } from '../suppliers/hooks/useSuppliers'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import type { PurchaseOrder, PurchaseOrderStatus } from '../procurement/types'
import { useConfirm } from '../components/common/useConfirm'
import { DISCREPANCIES_SEGMENT, PROCUREMENT_PATH, STATUS_STYLE, formatMoney } from '../procurement/procurementHelpers'
import { BookingDiscrepanciesPage } from '../procurement/BookingDiscrepanciesPage'
import { CreateFromDocumentModal } from '../procurement/CreateFromDocumentModal'
import { PurchaseOrderEditPage } from '../procurement/PurchaseOrderEditPage'

const STATUS_TABS: readonly (PurchaseOrderStatus | 'all')[] = ['all', 'draft', 'sent', 'partially_received', 'received', 'needs_review', 'cancelled']

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
