import { useState, useEffect, useCallback, useRef, type ReactNode } from 'react'
import {
  Search, ChevronRight, ChevronDown, Plus, ArrowRightLeft, AlertTriangle,
  Trash2, Edit3, X, PackageMinus, ClipboardList, Tag, ArrowLeft, History, Boxes, Scale, Image as ImageIcon,
} from 'lucide-react'
import clsx from 'clsx'
import { useLocation } from 'react-router-dom'
import { useAuth } from '../auth/AuthContext'
import { parseInventoryPath, inventoryPaths, INVENTORY_PATH } from '../inventory/inventoryRoutes'
import { useInventoryHistoryTarget } from '../inventory/useInventoryHistoryTarget'
import { useSkuActionRoute } from '../inventory/useSkuActionRoute'
import type { SkuAction } from '../inventory/inventoryRoutes'
import { useStackNavigation } from '../routing/useStackNavigation'
import { useQueryEnum, useQueryPage, useQuerySet, useQueryString } from '../routing/useQueryState'
import { RecordLoading, RecordNotFound } from '../components/common/RecordStatus'
import Pagination from '../components/common/Pagination'
import { useProducts } from '../pms/hooks/useProducts'
import { useProductDetail } from '../pms/hooks/useProductDetail'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useSKUStock } from '../inventory/hooks/useSKUStock'
import { useExpiredBatchReview } from '../inventory/hooks/useExpiredBatchReview'
import { useExpiringBatches } from '../inventory/hooks/useExpiringBatches'
import { useDiscards } from '../inventory/hooks/useDiscards'
import { useDiscardBatch } from '../inventory/hooks/useDiscardBatch'
import * as pmsApi from '../pms/api'
import * as inventoryApi from '../inventory/api'
import type { SKU } from '../pms/types'
import type { Batch, MovementType, StockMovement, ExpiringBatch, ExpiredBatchReview } from '../inventory/types'
import { Modal, inputClass, labelClass } from '../inventory/components/Modal'
import { PortionFromBulkModal } from '../inventory/PortionFromBulkModal'

const PAGE_SIZE = 20

// No per-SKU reorder threshold exists in the backend yet -- this is a
// client-side heuristic ("low" = at or below this many units, but not
// zero) until a real configurable threshold field is added.
const LOW_STOCK_THRESHOLD = 10

const STOCK_FILTERS = ['all', 'low', 'out'] as const
type StockFilter = typeof STOCK_FILTERS[number]

function formatDate(iso?: string | null): string {
  if (!iso) return '—'
  return new Date(iso).toLocaleDateString()
}

function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}

function parsePriceToMinorUnits(price: string): number {
  return Math.round(parseFloat(price || '0') * 100)
}

function parseOptionalPriceToMinorUnits(price: string): number | null {
  if (price.trim() === '') return null
  return Math.round(parseFloat(price) * 100)
}

// formatSKUQuantity shows a kilo SKU's stock (stored in grams) as kg for
// readability -- every other SKU's stock is a plain unit count.
function formatSKUQuantity(sku: SKU, quantity: number): string {
  if (sku.unit_type === 'kilo') return `${(quantity / 1000).toFixed(2)}kg`
  return String(quantity)
}

function formatMinorUnitsAsInput(minorUnits: number): string {
  return (minorUnits / 100).toFixed(2)
}

function isExpired(iso?: string | null): boolean {
  if (!iso) return false
  return new Date(iso).getTime() < Date.now()
}

function isExpiringSoon(iso?: string | null, withinDays = 30): boolean {
  if (!iso) return false
  const ms = new Date(iso).getTime() - Date.now()
  return ms >= 0 && ms <= withinDays * 24 * 60 * 60 * 1000
}

// Modal/inputClass/labelClass live in ../inventory/components/Modal (see
// import above) -- shared by every stock-action modal in this file.

// BatchScope pins a modal opened by clicking a batch row to that exact
// row -- warehouseId/batchId feed straight into the mutation, label is
// just for display. batchId '' means the general-ledger row (same sentinel
// AdjustStockModal/RecountModal/etc already use for "no batch"). Only
// Adjust/Recount/Damaged-Lost, and Transfer's FROM side, ever get locked
// this way -- Transfer's TO side stays a free pick even from a row click.
interface BatchScope {
  warehouseId: string
  batchId: string
  label: string
}

// LockedField replaces a warehouse/batch <select> with its resolved value
// when a modal was opened by clicking a specific batch row -- the whole
// point of that entry point is "this exact row", so it isn't re-editable.
function LockedField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className={labelClass}>{label}</span>
      <div className={clsx(inputClass, 'opacity-70 cursor-not-allowed')}>{value}</div>
    </div>
  )
}

// ── Batch form (create/edit) ─────────────────────────────────────────────────

function BatchFormModal({
  batch, warehouses, onSave, onClose,
}: {
  batch: Batch | null
  warehouses: { id: string; name: string }[]
  onSave: (input: { warehouse_id: string; lot_number?: string; expiry_date?: string | null }) => Promise<void>
  onClose: () => void
}) {
  const [warehouseId, setWarehouseId] = useState(batch?.warehouse_id ?? warehouses[0]?.id ?? '')
  const [lotNumber, setLotNumber] = useState(batch?.lot_number ?? '')
  const [expiryDate, setExpiryDate] = useState(batch?.expiry_date ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    setSaving(true)
    setError(null)
    try {
      await onSave({ warehouse_id: warehouseId, lot_number: lotNumber, expiry_date: expiryDate || null })
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to save batch')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={batch ? 'Edit Batch' : 'New Batch'} onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className={labelClass} htmlFor="batch-warehouse">Warehouse</label>
          <select id="batch-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
            {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
          </select>
        </div>
        <div>
          <label className={labelClass} htmlFor="batch-lot">Lot number</label>
          <input id="batch-lot" value={lotNumber} onChange={e => setLotNumber(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="batch-expiry">Expiry date (leave empty for no expiry)</label>
          <input id="batch-expiry" type="date" value={expiryDate ?? ''} onChange={e => setExpiryDate(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-[10px] text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            Cancel
          </button>
          <button type="button" onClick={submit} disabled={saving || !warehouseId} className="px-3 py-1.5 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Adjust stock ──────────────────────────────────────────────────────────────

function AdjustStockModal({
  warehouses, batches, scope, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  scope?: BatchScope
  onSave: (input: { warehouse_id: string; batch_id?: string | null; type: MovementType; quantity_delta: number; reason?: string }) => Promise<void>
  onClose: () => void
}) {
  const [type, setType] = useState<'receipt' | 'sale' | 'adjustment'>('receipt')
  const [warehouseId, setWarehouseId] = useState(scope?.warehouseId ?? warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState(scope?.batchId ?? '')
  const [quantity, setQuantity] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    const n = parseInt(quantity, 10)
    if (Number.isNaN(n) || n === 0) {
      setError('Enter a non-zero quantity')
      return
    }
    setSaving(true)
    setError(null)
    try {
      const delta = type === 'sale' ? -Math.abs(n) : type === 'receipt' ? Math.abs(n) : n
      await onSave({ warehouse_id: warehouseId, batch_id: batchId || null, type, quantity_delta: delta, reason })
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to record movement')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Adjust Stock" onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className={labelClass} htmlFor="adjust-type">Type</label>
          <select id="adjust-type" value={type} onChange={e => setType(e.target.value as typeof type)} className={inputClass}>
            <option value="receipt">Receipt (stock in)</option>
            <option value="sale">Sale (stock out)</option>
            <option value="adjustment">Adjustment (+/-)</option>
          </select>
        </div>
        {scope ? (
          <LockedField label="Warehouse" value={warehouses.find(w => w.id === scope.warehouseId)?.name ?? scope.warehouseId} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="adjust-warehouse">Warehouse</label>
            <select id="adjust-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
        )}
        {scope ? (
          <LockedField label="Batch" value={scope.label} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="adjust-batch">Batch (leave empty for general ledger)</label>
            <select id="adjust-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
              <option value="">General ledger (no batch)</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
        )}
        <div>
          <label className={labelClass} htmlFor="adjust-qty">
            Quantity {type === 'adjustment' ? '(use a minus sign to decrease)' : ''}
          </label>
          <input id="adjust-qty" type="number" value={quantity} onChange={e => setQuantity(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="adjust-reason">Reason (optional)</label>
          <input id="adjust-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-[10px] text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            Cancel
          </button>
          <button type="button" onClick={submit} disabled={saving || !warehouseId} className="px-3 py-1.5 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
            {saving ? 'Saving…' : 'Record'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Recount ───────────────────────────────────────────────────────────────────

function RecountModal({
  warehouses, batches, scope, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  scope?: BatchScope
  onSave: (input: { warehouse_id: string; batch_id?: string | null; counted_quantity: number; reason?: string }) => Promise<void>
  onClose: () => void
}) {
  const [warehouseId, setWarehouseId] = useState(scope?.warehouseId ?? warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState(scope?.batchId ?? '')
  const [counted, setCounted] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    const n = parseInt(counted, 10)
    if (Number.isNaN(n) || n < 0) {
      setError('Enter the counted quantity')
      return
    }
    setSaving(true)
    setError(null)
    try {
      await onSave({ warehouse_id: warehouseId, batch_id: batchId || null, counted_quantity: n, reason })
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to record recount')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Recount Stock" onClose={onClose}>
      <div className="space-y-3">
        {scope ? (
          <LockedField label="Warehouse" value={warehouses.find(w => w.id === scope.warehouseId)?.name ?? scope.warehouseId} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="recount-warehouse">Warehouse</label>
            <select id="recount-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
        )}
        {scope ? (
          <LockedField label="Batch" value={scope.label} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="recount-batch">Batch (leave empty for general ledger)</label>
            <select id="recount-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
              <option value="">General ledger (no batch)</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
        )}
        <div>
          <label className={labelClass} htmlFor="recount-qty">Counted quantity</label>
          <input id="recount-qty" type="number" min={0} value={counted} onChange={e => setCounted(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="recount-reason">Reason (optional)</label>
          <input id="recount-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-[10px] text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            Cancel
          </button>
          <button type="button" onClick={submit} disabled={saving || !warehouseId} className="px-3 py-1.5 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
            {saving ? 'Saving…' : 'Record'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Transfer (also covers general-ledger <-> batch moves) ───────────────────

function TransferModal({
  warehouses, batches, scope, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  scope?: BatchScope
  onSave: (input: {
    quantity: number; from_warehouse_id: string; from_batch_id?: string | null
    to_warehouse_id: string; to_batch_id?: string | null; reason?: string
  }) => Promise<void>
  onClose: () => void
}) {
  const [quantity, setQuantity] = useState('')
  const [fromWarehouse, setFromWarehouse] = useState(scope?.warehouseId ?? warehouses[0]?.id ?? '')
  const [fromBatch, setFromBatch] = useState(scope?.batchId ?? '')
  const [toWarehouse, setToWarehouse] = useState(warehouses[0]?.id ?? '')
  const [toBatch, setToBatch] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    const n = parseInt(quantity, 10)
    if (Number.isNaN(n) || n <= 0) {
      setError('Enter a positive quantity')
      return
    }
    setSaving(true)
    setError(null)
    try {
      await onSave({
        quantity: n, from_warehouse_id: fromWarehouse, from_batch_id: fromBatch || null,
        to_warehouse_id: toWarehouse, to_batch_id: toBatch || null, reason,
      })
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to transfer stock')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Transfer Stock" onClose={onClose}>
      <div className="space-y-3">
        <p className="text-[10px] text-ink-4">
          Leaving a batch empty means the general ledger (un-lotted stock) — use this to move stock into or out of a batch too.
        </p>
        <div className="grid grid-cols-2 gap-2">
          {scope ? (
            <LockedField label="From warehouse" value={warehouses.find(w => w.id === scope.warehouseId)?.name ?? scope.warehouseId} />
          ) : (
            <div>
              <label className={labelClass} htmlFor="transfer-from-wh">From warehouse</label>
              <select id="transfer-from-wh" value={fromWarehouse} onChange={e => setFromWarehouse(e.target.value)} className={inputClass}>
                {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
              </select>
            </div>
          )}
          <div>
            <label className={labelClass} htmlFor="transfer-to-wh">To warehouse</label>
            <select id="transfer-to-wh" value={toWarehouse} onChange={e => setToWarehouse(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          {scope ? (
            <LockedField label="From batch" value={scope.label} />
          ) : (
            <div>
              <label className={labelClass} htmlFor="transfer-from-batch">From batch</label>
              <select id="transfer-from-batch" value={fromBatch} onChange={e => setFromBatch(e.target.value)} className={inputClass}>
                <option value="">General ledger</option>
                {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
              </select>
            </div>
          )}
          <div>
            <label className={labelClass} htmlFor="transfer-to-batch">To batch</label>
            <select id="transfer-to-batch" value={toBatch} onChange={e => setToBatch(e.target.value)} className={inputClass}>
              <option value="">General ledger</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
        </div>
        <div>
          <label className={labelClass} htmlFor="transfer-qty">Quantity</label>
          <input id="transfer-qty" type="number" min={1} value={quantity} onChange={e => setQuantity(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="transfer-reason">Reason (optional)</label>
          <input id="transfer-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-[10px] text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            Cancel
          </button>
          <button type="button" onClick={submit} disabled={saving} className="px-3 py-1.5 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
            {saving ? 'Saving…' : 'Transfer'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Split case into units (see pms.SKU.case_of_sku_id) ──────────────────────

function SplitCaseModal({
  warehouses, batches, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  onSave: (input: { warehouse_id: string; batch_id?: string | null; cases_to_split: number; reason?: string }) => Promise<unknown>
  onClose: () => void
}) {
  const [warehouseId, setWarehouseId] = useState(warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState('')
  const [casesToSplit, setCasesToSplit] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    const n = parseInt(casesToSplit, 10)
    if (Number.isNaN(n) || n <= 0) {
      setError('Enter a positive number of cases')
      return
    }
    setSaving(true)
    setError(null)
    try {
      await onSave({ warehouse_id: warehouseId, batch_id: batchId || null, cases_to_split: n, reason })
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to split case stock')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Split Case Into Units" onClose={onClose}>
      <div className="space-y-3">
        <p className="text-[10px] text-ink-4">
          One-way: the cases you split leave stock, and that many times the case size arrives as units of the linked product. This can't be undone by re-assembling units back into cases.
        </p>
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className={labelClass} htmlFor="split-warehouse">Warehouse</label>
            <select id="split-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          <div>
            <label className={labelClass} htmlFor="split-batch">Batch</label>
            <select id="split-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
              <option value="">General ledger</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
        </div>
        <div>
          <label className={labelClass} htmlFor="split-cases">Cases to split</label>
          <input id="split-cases" type="number" min={1} value={casesToSplit} onChange={e => setCasesToSplit(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="split-reason">Reason (optional)</label>
          <input id="split-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-[10px] text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            Cancel
          </button>
          <button type="button" onClick={submit} disabled={saving} className="px-3 py-1.5 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
            {saving ? 'Splitting…' : 'Split'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// PortionFromBulkModal lives in ../inventory/PortionFromBulkModal (see
// import above) -- also fixes a real bug: the batch picker used to source
// the currently-viewed (target) SKU's own batches instead of the BULK
// pool's, so batch-linked portioning never worked through this UI.

// ── Mark damaged / lost ───────────────────────────────────────────────────────

function DamagedLostModal({
  warehouses, batches, scope, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  scope?: BatchScope
  onSave: (input: { warehouse_id: string; batch_id?: string | null; quantity: number; type: 'damaged' | 'lost'; reason?: string }) => Promise<void>
  onClose: () => void
}) {
  const [type, setType] = useState<'damaged' | 'lost'>('damaged')
  const [warehouseId, setWarehouseId] = useState(scope?.warehouseId ?? warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState(scope?.batchId ?? '')
  const [quantity, setQuantity] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    const n = parseInt(quantity, 10)
    if (Number.isNaN(n) || n <= 0) {
      setError('Enter a positive quantity')
      return
    }
    setSaving(true)
    setError(null)
    try {
      await onSave({ warehouse_id: warehouseId, batch_id: batchId || null, quantity: n, type, reason })
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to write off stock')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Mark Damaged / Lost" onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className={labelClass} htmlFor="dl-type">Type</label>
          <select id="dl-type" value={type} onChange={e => setType(e.target.value as typeof type)} className={inputClass}>
            <option value="damaged">Damaged</option>
            <option value="lost">Lost</option>
          </select>
        </div>
        {scope ? (
          <LockedField label="Warehouse" value={warehouses.find(w => w.id === scope.warehouseId)?.name ?? scope.warehouseId} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="dl-warehouse">Warehouse</label>
            <select id="dl-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
        )}
        {scope ? (
          <LockedField label="Batch" value={scope.label} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="dl-batch">Batch (leave empty for general ledger)</label>
            <select id="dl-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
              <option value="">General ledger (no batch)</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
        )}
        <div>
          <label className={labelClass} htmlFor="dl-qty">Quantity</label>
          <input id="dl-qty" type="number" min={1} value={quantity} onChange={e => setQuantity(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="dl-reason">Reason (optional)</label>
          <input id="dl-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-[10px] text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            Cancel
          </button>
          <button type="button" onClick={submit} disabled={saving || !warehouseId} className="px-3 py-1.5 text-xs rounded-lg bg-red-600 text-white hover:bg-red-700 disabled:opacity-50">
            {saving ? 'Saving…' : 'Write off'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Price / sale price (owned by pms, edited here as a convenience — same
// pattern as ATUM's inventory screen writing back to WooCommerce's own
// product price, not a separate price store) ────────────────────────────────

function PriceModal({ sku, onSave, onClose }: {
  sku: SKU; onSave: (price: string, salePrice: string, saleEndDate: string) => Promise<void>; onClose: () => void
}) {
  const [price, setPrice] = useState(formatMinorUnitsAsInput(sku.price_minor_units))
  const [salePrice, setSalePrice] = useState(sku.sale_price_minor_units != null ? formatMinorUnitsAsInput(sku.sale_price_minor_units) : '')
  const [saleEndDate, setSaleEndDate] = useState(sku.sale_end_date ?? '')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    setSaving(true)
    setError(null)
    try {
      await onSave(price, salePrice, saleEndDate)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to update price')
    } finally {
      setSaving(false)
    }
  }

  const priceNum = parseFloat(price || '0')
  const salePriceNum = salePrice.trim() === '' ? null : parseFloat(salePrice)
  const discountPct = salePriceNum != null && priceNum > 0 ? Math.round((1 - salePriceNum / priceNum) * 100) : null

  return (
    <Modal title="Edit Price" onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className={labelClass} htmlFor="price-edit-price">Regular price ({sku.currency_code})</label>
          <input id="price-edit-price" type="number" step="0.01" min="0" value={price} onChange={e => setPrice(e.target.value)} className={inputClass} />
        </div>

        <div className="rounded-lg border border-line p-2.5 space-y-2">
          <div className="flex items-center justify-between">
            <label className={clsx(labelClass, 'mb-0')} htmlFor="price-edit-sale-price">Sale price (optional)</label>
            {discountPct != null && discountPct > 0 && (
              <span className="text-[9px] px-1.5 py-px rounded bg-green-500/15 text-success font-medium">-{discountPct}%</span>
            )}
            {discountPct != null && discountPct <= 0 && (
              <span className="text-[9px] text-danger">must be below regular price</span>
            )}
          </div>
          <input id="price-edit-sale-price" type="number" step="0.01" min="0" value={salePrice} onChange={e => {
            setSalePrice(e.target.value)
            if (e.target.value.trim() === '') setSaleEndDate('')
          }} className={inputClass} />

          <div>
            <label className={labelClass} htmlFor="price-edit-sale-end-date">Sale ends (optional)</label>
            <input id="price-edit-sale-end-date" type="date" value={saleEndDate} onChange={e => setSaleEndDate(e.target.value)} className={inputClass} disabled={salePriceNum == null} />
            <p className="text-[9px] text-ink-4 mt-1">Reverts to the regular price automatically the day after this date.</p>
          </div>
        </div>

        {error && <p className="text-[10px] text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            Cancel
          </button>
          <button type="button" onClick={submit} disabled={saving} className="px-3 py-1.5 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
            {saving ? 'Saving…' : 'Save'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Per-SKU stock panel ───────────────────────────────────────────────────────

// BatchFormModal for a new batch: a successful create closes back to the
// list with zero-stock batches shown (a new batch starts at zero), while
// Cancel closes normally.
function NewBatchModal({ warehouses, onSave, onClose, onCreated }: {
  warehouses: { id: string; name: string }[]
  onSave: (input: { warehouse_id: string; lot_number?: string; expiry_date?: string | null }) => Promise<void>
  onClose: () => void
  onCreated: () => void
}) {
  const created = useRef(false)
  return (
    <BatchFormModal batch={null} warehouses={warehouses}
      onSave={input => onSave(input).then(() => { created.current = true })}
      onClose={() => (created.current ? onCreated() : onClose())} />
  )
}

// BatchRowActionModal is what clicking a batch row opens -- a chooser
// scoped to that one row, since a click alone can't say which of
// Adjust/Recount/Transfer/Damaged-Lost staff actually want. Picking one
// hands off to that same modal, pre-locked to this row (see BatchScope);
// Cancel here just closes without picking anything.
function BatchRowActionModal({ scope, onChoose, onClose }: {
  scope: BatchScope
  onChoose: (action: 'adjust' | 'recount' | 'transfer' | 'damaged-lost') => void
  onClose: () => void
}) {
  const actions: { key: 'adjust' | 'recount' | 'transfer' | 'damaged-lost'; label: string; icon: ReactNode; danger?: boolean }[] = [
    { key: 'adjust', label: 'Adjust (add/remove)', icon: <Plus size={13} /> },
    { key: 'recount', label: 'Recount', icon: <ClipboardList size={13} /> },
    { key: 'transfer', label: 'Transfer', icon: <ArrowRightLeft size={13} /> },
    { key: 'damaged-lost', label: 'Damaged / Lost', icon: <PackageMinus size={13} />, danger: true },
  ]
  return (
    <Modal title={scope.label} onClose={onClose}>
      <div className="space-y-1.5">
        {actions.map(a => (
          <button
            key={a.key} type="button" onClick={() => onChoose(a.key)}
            className={clsx(
              'w-full flex items-center gap-2 px-3 py-2 text-xs rounded-lg border border-line hover:bg-hover',
              a.danger ? 'text-danger' : 'text-ink',
            )}
          >
            {a.icon} {a.label}
          </button>
        ))}
        <button type="button" onClick={onClose} className="w-full px-3 py-2 text-xs rounded-lg text-ink-3 hover:bg-hover">
          Cancel
        </button>
      </div>
    </Modal>
  )
}

function SKUStockPanel({ token, productId, sku, variantName, warehouses, onSKUUpdated, onStockChanged, onViewHistory, onViewBatchHistory }: {
  token: string; productId: string; sku: SKU; variantName?: string; warehouses: { id: string; name: string }[]
  // onSKUUpdated refetches product/variant/SKU metadata (name, price, ...)
  // -- reserved for price edits. Stock-quantity mutations must use
  // onStockChanged instead: onSKUUpdated flips the parent's `loading` flag,
  // which unmounts/remounts this whole panel (see ProductStockRow's
  // `{!loading && skus.map(...)}`), discarding showAllBatches/modal state
  // on every adjust/recount/transfer/etc. onStockChanged only refetches the
  // collapsed row's aggregate total, no unmount.
  onSKUUpdated: () => void
  onStockChanged: () => void
  onViewHistory: (sku: SKU) => void; onViewBatchHistory: (sku: SKU, batch: Batch) => void
}) {
  const { level, batches, loading, error, addBatch, editBatch, removeBatch, adjust, recount, transfer, splitCase, portionFromBulk, markDamagedOrLost } =
    useSKUStock(token, sku.id)
  // The open modal, and the batch row it's locked to, come from the URL
  // (see useSkuActionRoute), as does "show zero-stock batches"
  // (?allBatches=) -- a refresh or Back keeps or closes them like pages.
  const route = useSkuActionRoute(productId, sku.id)
  const allBatches = useQuerySet('allBatches')
  const showAllBatches = allBatches.has(sku.id)
  const [deleteError, setDeleteError] = useState<string | null>(null)

  // Resolves ?batch= against the loaded batches (the general-ledger row
  // has no id, so it's addressed by warehouse via ?ledger=). A modal that
  // needs its row waits until the batches are in rather than opening
  // unlocked.
  const rowBatch = route.row && 'batch' in route.row ? batches.find(b => b.id === (route.row as { batch: string }).batch) ?? null : null
  const rowScope: BatchScope | null = !route.row
    ? null
    : 'ledger' in route.row
      ? { warehouseId: route.row.ledger, batchId: '', label: 'General ledger' }
      : rowBatch && { warehouseId: rowBatch.warehouse_id, batchId: rowBatch.id, label: rowBatch.lot_number || 'Batch' }
  const modal = !route.row || rowScope ? route.action : null
  const setModal = (action: SkuAction) => route.openAction(action)
  const closeModal = route.close
  const openRowAction = (b: Batch) => {
    route.openAction('choose', b.is_general_ledger ? { ledger: b.warehouse_id } : { batch: b.id })
  }

  // Old zero-stock batches (even with movement history) clutter this table
  // by default -- hidden unless explicitly shown. The general-ledger row
  // and any batch still carrying stock are always shown. Modals below still
  // get the full, unfiltered `batches` -- e.g. Transfer needs to offer a
  // zero-stock batch as a valid destination.
  const zeroStockBatchCount = batches.filter(b => !b.is_general_ledger && b.quantity === 0).length
  const visibleBatches = showAllBatches ? batches : batches.filter(b => b.is_general_ledger || b.quantity !== 0)

  const handleSavePrice = async (price: string, salePrice: string, saleEndDate: string) => {
    // Full round-trip of every SKU field, not just price -- UpdateSKU is a
    // full replace, not a patch, so omitting a field (tracks_expiry,
    // case_of_sku_id, unit_type, ...) would silently wipe it.
    await pmsApi.updateSKU(token, sku.id, {
      sku_code: sku.sku_code,
      barcode: sku.barcode ?? undefined,
      price_minor_units: parsePriceToMinorUnits(price),
      sale_price_minor_units: parseOptionalPriceToMinorUnits(salePrice),
      sale_end_date: saleEndDate || null,
      currency_code: sku.currency_code,
      weight_grams: sku.weight_grams,
      length_mm: sku.length_mm,
      width_mm: sku.width_mm,
      height_mm: sku.height_mm,
      is_virtual: sku.is_virtual,
      tracks_expiry: sku.tracks_expiry,
      case_of_sku_id: sku.case_of_sku_id,
      units_per_case: sku.units_per_case,
      unit_type: sku.unit_type,
      portioned_from_sku_id: sku.portioned_from_sku_id,
      status: sku.status,
    })
    onSKUUpdated()
  }

  const handleDeleteBatch = async (id: string) => {
    setDeleteError(null)
    try {
      await removeBatch(id)
      onStockChanged()
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Failed to delete batch')
    }
  }

  return (
    <div className="pl-8 pr-4 py-3 border-t border-line/60 dark:bg-night-950/40 bg-gray-50/60">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-3">
          <span className="font-mono text-[11px] text-ink-2">
            {variantName && <span className="font-sans text-ink-3">{variantName} · </span>}
            {sku.sku_code}
          </span>
          <button
            type="button"
            onClick={() => setModal('price')}
            title="Edit price"
            className="flex items-center gap-1 text-[10px] text-ink-4 hover:text-accent"
          >
            <Tag size={11} /> Price
          </button>
          <button
            type="button"
            onClick={() => onViewHistory(sku)}
            title="View movement history"
            className="flex items-center gap-1 text-[10px] text-ink-4 hover:text-accent"
          >
            <History size={11} /> History
          </button>
          {level && (
            <span data-testid="stock-level" className="text-[10px] text-ink-3">
              Total: <strong className="text-ink">{formatSKUQuantity(sku, level.total)}</strong>
              {' · '}Syncable: <strong className="text-ink">{formatSKUQuantity(sku, level.syncable)}</strong>
            </span>
          )}
          {loading && <span className="text-[10px] text-ink-4">Loading…</span>}
        </div>
        <div className="flex items-center gap-1.5">
          <button type="button" onClick={() => setModal('adjust')} className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] bg-raised border border-line text-ink-2 hover:bg-hover">
            <Plus size={11} /> Adjust
          </button>
          <button type="button" onClick={() => setModal('recount')} className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] bg-raised border border-line text-ink-2 hover:bg-hover">
            <ClipboardList size={11} /> Recount
          </button>
          <button type="button" onClick={() => setModal('transfer')} className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] bg-raised border border-line text-ink-2 hover:bg-hover">
            <ArrowRightLeft size={11} /> Transfer
          </button>
          {sku.case_of_sku_id && (
            <button type="button" onClick={() => setModal('split-case')} title="Split sealed cases into individual units" className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] bg-raised border border-line text-ink-2 hover:bg-hover">
              <Boxes size={11} /> Split case
            </button>
          )}
          {sku.portioned_from_sku_id && (
            <button type="button" onClick={() => setModal('portion')} title="Add a sellable item peeled off the linked weight pool" className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] bg-raised border border-line text-ink-2 hover:bg-hover">
              <Scale size={11} /> Add item from weight pool
            </button>
          )}
          <button type="button" onClick={() => setModal('damaged-lost')} className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] bg-raised border border-line text-danger hover:bg-hover">
            <PackageMinus size={11} /> Damaged/Lost
          </button>
        </div>
      </div>

      {error && <p className="text-[10px] text-danger mb-2">{error}</p>}
      {deleteError && <p className="text-[10px] text-danger mb-2">{deleteError}</p>}

      <div className="rounded-lg border border-line overflow-hidden">
        <table className="w-full text-[10px]">
          <thead className="bg-fill">
            <tr>
              {['Lot', 'Warehouse', 'Qty', 'Expiry', ''].map(h => (
                <th key={h} className="px-2.5 py-1.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {visibleBatches.map(b => (
              <tr
                key={b.is_general_ledger ? `general-${b.warehouse_id}` : b.id}
                onClick={() => openRowAction(b)}
                title="Click to adjust, recount, transfer, or mark damaged/lost"
                className={clsx(
                  'cursor-pointer dark:hover:bg-night-800 hover:bg-gray-100/80',
                  b.is_general_ledger && 'dark:bg-night-900/40 bg-gray-100/60',
                )}
              >
                <td className="px-2.5 py-1.5 font-mono text-ink-2">
                  {b.is_general_ledger ? 'General ledger' : (b.lot_number || '—')}
                </td>
                <td className="px-2.5 py-1.5 text-ink-3">{warehouses.find(w => w.id === b.warehouse_id)?.name ?? b.warehouse_id}</td>
                <td className="px-2.5 py-1.5 font-mono text-ink-2">{b.quantity}</td>
                <td className="px-2.5 py-1.5">
                  {!b.is_general_ledger && (
                    <span className={clsx(
                      'font-mono',
                      isExpired(b.expiry_date) ? 'text-danger' : isExpiringSoon(b.expiry_date) ? 'text-warning' : 'text-ink-3',
                    )}>
                      {formatDate(b.expiry_date)}
                    </span>
                  )}
                </td>
                <td className="px-2.5 py-1.5" onClick={e => e.stopPropagation()}>
                  {!b.is_general_ledger && (
                    <div className="flex items-center justify-end gap-2">
                      <button type="button" onClick={() => onViewBatchHistory(sku, b)} aria-label={`View movement history for batch ${b.lot_number}`} title="View movement history" className="text-ink-4 hover:text-accent">
                        <History size={11} />
                      </button>
                      <button type="button" onClick={() => route.openAction('edit-batch', { batch: b.id })} aria-label={`Edit batch ${b.lot_number}`} className="text-ink-4 hover:text-ink-2">
                        <Edit3 size={11} />
                      </button>
                      <button type="button" onClick={() => handleDeleteBatch(b.id)} aria-label={`Delete batch ${b.lot_number}`} className="text-ink-4 hover:text-danger">
                        <Trash2 size={11} />
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
            {batches.length === 0 && (
              <tr>
                <td colSpan={5} className="px-2.5 py-3 text-center text-ink-4">No batches — all stock is in the general ledger.</td>
              </tr>
            )}
            {batches.length > 0 && visibleBatches.length === 0 && (
              <tr>
                <td colSpan={5} className="px-2.5 py-3 text-center text-ink-4">
                  All {zeroStockBatchCount} batch{zeroStockBatchCount === 1 ? '' : 'es'} for this SKU {zeroStockBatchCount === 1 ? 'has' : 'have'} zero stock.
                </td>
              </tr>
            )}
          </tbody>
        </table>
        <div className="px-2.5 py-1.5 border-t border-line flex items-center gap-3">
          <button type="button" onClick={() => setModal('new-batch')} className="text-[10px] text-accent hover:text-accent/80">
            + New batch
          </button>
          {zeroStockBatchCount > 0 && (
            <button type="button" onClick={() => allBatches.toggle(sku.id)} className="text-[10px] text-ink-4 hover:text-accent">
              {showAllBatches ? 'Hide zero-stock batches' : `Show ${zeroStockBatchCount} zero-stock batch${zeroStockBatchCount === 1 ? '' : 'es'}`}
            </button>
          )}
        </div>
      </div>

      {modal === 'choose' && rowScope && (
        <BatchRowActionModal scope={rowScope} onChoose={route.switchAction} onClose={closeModal} />
      )}
      {modal === 'adjust' && (
        <AdjustStockModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} onClose={closeModal}
          onSave={input => adjust(input).then(() => onStockChanged())} />
      )}
      {modal === 'recount' && (
        <RecountModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} onClose={closeModal}
          onSave={input => recount(input).then(() => onStockChanged())} />
      )}
      {modal === 'transfer' && (
        <TransferModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} onClose={closeModal}
          onSave={input => transfer(input).then(() => onStockChanged())} />
      )}
      {modal === 'split-case' && (
        <SplitCaseModal warehouses={warehouses} batches={batches} onClose={closeModal}
          onSave={input => splitCase(input).then(() => onStockChanged())} />
      )}
      {modal === 'portion' && sku.portioned_from_sku_id && (
        <PortionFromBulkModal
          token={token} bulkSkuId={sku.portioned_from_sku_id} warehouses={warehouses} nominalWeightGrams={sku.weight_grams}
          onClose={closeModal} onSave={input => portionFromBulk(input).then(() => onStockChanged())}
        />
      )}
      {modal === 'damaged-lost' && (
        <DamagedLostModal warehouses={warehouses} batches={batches} scope={rowScope ?? undefined} onClose={closeModal}
          onSave={input => markDamagedOrLost(input).then(() => onStockChanged())} />
      )}
      {modal === 'new-batch' && (
        <NewBatchModal warehouses={warehouses} onClose={closeModal} onCreated={route.closeShowingAllBatches}
          onSave={input => addBatch(input).then(() => onStockChanged())} />
      )}
      {modal === 'price' && (
        <PriceModal sku={sku} onClose={closeModal} onSave={handleSavePrice} />
      )}
      {modal === 'edit-batch' && rowBatch && (
        <BatchFormModal batch={rowBatch} warehouses={warehouses} onClose={closeModal}
          onSave={input => editBatch(rowBatch.id, input).then(() => onStockChanged())} />
      )}
    </div>
  )
}

// ── Product row ───────────────────────────────────────────────────────────────

// A product with more than one SKU (named variants) has no single
// "the" price/sale price to show at the product-row level — the price
// columns show a min–max range instead, and the sale-price column shows
// "-" rather than guessing which variant's sale is "the" one.
export interface ProductStockSummary {
  totalStock: number
  skuCount: number
}

function ProductStockRow({ token, productId, productName, productImageUrl, warehouses, onSummary, onViewHistory, onViewBatchHistory, onViewProductHistory }: {
  token: string; productId: string; productName: string; productImageUrl?: string | null; warehouses: { id: string; name: string }[]
  onSummary: (productId: string, summary: ProductStockSummary) => void
  onViewHistory: (sku: SKU) => void
  onViewBatchHistory: (sku: SKU, batch: Batch) => void
  onViewProductHistory: (productName: string, skus: SKU[]) => void
}) {
  // Expanded rows live in ?open= so a refresh, Back or a shared link keeps
  // them; a SKU action URL for this product (its modal) also expands it.
  const openRows = useQuerySet('open')
  const location = useLocation()
  const routeView = parseInventoryPath(location.pathname)
  const routeSkuId = routeView.kind === 'skuAction' && routeView.productId === productId ? routeView.skuId : null
  const expanded = openRows.has(productId) || routeSkuId !== null
  const toggleExpanded = () => openRows.toggle(productId)

  const { detail, loading, refresh } = useProductDetail(token, productId)
  const skus = (detail?.variants ?? []).flatMap(v => v.skus ?? [])
  const { exitToOpener } = useStackNavigation()
  const navigateToList = () => exitToOpener(INVENTORY_PATH)
  const skusKey = skus.map(s => s.id).join(',')
  // Kept as a separate lookup rather than threading a variantName field
  // through SKU itself -- every other consumer of `skus` here (movement
  // history, price range) has no use for it.
  const variantNameBySKU = Object.fromEntries(
    (detail?.variants ?? []).flatMap(v => (v.skus ?? []).map(s => [s.id, v.name] as const)),
  )

  const [stockBySKU, setStockBySKU] = useState<Record<string, number>>({})
  const [stockLoading, setStockLoading] = useState(true)

  // Also called directly (not just from the effect below) after any stock
  // mutation in a child SKUStockPanel, via onSKUUpdated -- so the collapsed
  // row's total never goes stale without a full page reload.
  const loadStock = useCallback(() => {
    if (!token || skus.length === 0) {
      setStockBySKU({})
      setStockLoading(false)
      return
    }
    setStockLoading(true)
    Promise.all(skus.map(sku => inventoryApi.getSKUStockLevel(token, sku.id).then(l => [sku.id, l.total] as const).catch(() => [sku.id, 0] as const)))
      .then(pairs => setStockBySKU(Object.fromEntries(pairs)))
      .finally(() => setStockLoading(false))
    // eslint-disable-next-line react-hooks/exhaustive-deps -- skusKey is the stable dependency; skus itself is a new array every render
  }, [token, skusKey])

  useEffect(() => { loadStock() }, [loadStock])

  const totalStock = Object.values(stockBySKU).reduce((sum, n) => sum + n, 0)

  useEffect(() => {
    if (!stockLoading) onSummary(productId, { totalStock, skuCount: skus.length })
    // eslint-disable-next-line react-hooks/exhaustive-deps -- onSummary identity is stable per Inventory render; including it would re-fire on every parent render
  }, [productId, totalStock, skus.length, stockLoading])

  const singleSKU = skus.length === 1 ? skus[0] : null
  const prices = skus.map(s => s.price_minor_units)
  const priceRange = prices.length > 0 ? [Math.min(...prices), Math.max(...prices)] as const : null

  return (
    <>
      <tr className="hover:bg-hover/60 transition-colors cursor-pointer" onClick={toggleExpanded}>
        <td className="px-4 py-2.5 w-8">
          {expanded ? <ChevronDown size={13} className="text-ink-4" /> : <ChevronRight size={13} className="text-ink-4" />}
        </td>
        <td className="px-2 py-2.5">
          {productImageUrl ? (
            <img src={productImageUrl} alt="" className="w-6 h-6 rounded object-cover dark:bg-night-700 bg-gray-200" />
          ) : (
            <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
              <ImageIcon size={12} className="text-ink-4" />
            </div>
          )}
        </td>
        <td className="px-3 py-2.5 font-medium text-ink">{productName || detail?.product.name}</td>
        <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
          {singleSKU ? (
            <span className="text-xs text-ink-2">{formatPrice(singleSKU.price_minor_units, singleSKU.currency_code)}</span>
          ) : priceRange ? (
            <span className="text-xs text-ink-2">
              {priceRange[0] === priceRange[1]
                ? formatPrice(priceRange[0], skus[0].currency_code)
                : `${formatPrice(priceRange[0], skus[0].currency_code)} – ${formatPrice(priceRange[1], skus[0].currency_code)}`}
            </span>
          ) : (
            <span className="text-xs text-ink-4">—</span>
          )}
        </td>
        <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
          {singleSKU && singleSKU.sale_price_minor_units != null ? (
            <span className="flex items-center gap-1">
              <span className="text-xs font-medium text-success">{formatPrice(singleSKU.sale_price_minor_units, singleSKU.currency_code)}</span>
              {singleSKU.sale_end_date && (
                <span className="text-[9px] text-ink-4">until {formatDate(singleSKU.sale_end_date)}</span>
              )}
            </span>
          ) : (
            <span className="text-xs text-ink-4">—</span>
          )}
        </td>
        <td className="px-3 py-2.5">
          {stockLoading ? (
            <span className="text-xs text-ink-4">…</span>
          ) : (
            <span className={clsx(
              'text-xs font-medium tabular-nums',
              totalStock <= 0 ? 'text-danger' : totalStock <= LOW_STOCK_THRESHOLD ? 'text-warning' : 'text-ink-2',
            )}>
              {totalStock}
            </span>
          )}
        </td>
        <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
          <button
            type="button"
            onClick={() => onViewProductHistory(productName, skus)}
            aria-label="View full movement history"
            title="View full movement history"
            className="flex items-center gap-1 text-[10px] text-ink-4 hover:text-accent"
          >
            <History size={11} /> History
          </button>
        </td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={7} className="p-0">
            {loading && <p className="pl-8 py-3 text-xs text-ink-4">Loading SKUs…</p>}
            {!loading && skus.length === 0 && <p className="pl-8 py-3 text-xs text-ink-4">No SKUs on this product yet.</p>}
            {!loading && detail && routeSkuId && !skus.some(s => s.id === routeSkuId) && (
              <div className="py-4"><RecordNotFound label="SKU" onBack={() => navigateToList()} /></div>
            )}
            {!loading && skus.map(sku => (
              <SKUStockPanel key={sku.id} token={token} productId={productId} sku={sku} variantName={variantNameBySKU[sku.id]} warehouses={warehouses} onSKUUpdated={refresh} onStockChanged={loadStock} onViewHistory={onViewHistory} onViewBatchHistory={onViewBatchHistory} />
            ))}
          </td>
        </tr>
      )}
    </>
  )
}

// ── SKU movement history (full-page, entered via the "History" button —
// deliberately not an inline table in the expanded row, to keep that
// compact) ────────────────────────────────────────────────────────────────

const MOVEMENT_TYPE_LABEL: Record<MovementType, string> = {
  receipt: 'Receipt', sale: 'Sale', adjustment: 'Adjustment',
  transfer_out: 'Transfer out', transfer_in: 'Transfer in', damaged: 'Damaged', lost: 'Lost',
  split_out: 'Case split (out)', split_in: 'Case split (in)',
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

type ProductMovementRow = StockMovement & { skuCode: string; balance: number }

function ProductMovementHistoryPage({ token, productName, skus, warehouses, onClose }: {
  token: string; productName: string; skus: SKU[]; warehouses: { id: string; name: string }[]; onClose: () => void
}) {
  const [rows, setRows] = useState<ProductMovementRow[] | null>(null)
  const [batches, setBatches] = useState<Batch[]>([])
  const [error, setError] = useState<string | null>(null)
  const skusKey = skus.map(s => s.id).join(',')

  useEffect(() => {
    let cancelled = false
    Promise.all(skus.map(sku => Promise.all([
      inventoryApi.listStockMovements(token, sku.id),
      inventoryApi.listBatches(token, sku.id),
    ]).then(([movements, batches]) => ({ sku, movements, batches }))))
      .then(perSKU => {
        if (cancelled) return
        // One running balance across every SKU of this product -- an
        // account book for the product as a whole, not per-variant --
        // computed oldest-first, then reversed for newest-first display.
        const combined = perSKU.flatMap(({ sku, movements }) => movements.map(m => ({ ...m, skuCode: sku.sku_code })))
        combined.sort((a, b) => a.occurred_at.localeCompare(b.occurred_at))
        let balance = 0
        const withBalance: ProductMovementRow[] = combined.map(m => {
          balance += m.quantity_delta
          return { ...m, balance }
        })
        setRows(withBalance.reverse())
        setBatches(perSKU.flatMap(p => p.batches))
      })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load movement history') })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- skusKey is the stable dependency; skus itself is a new array every render
  }, [token, skusKey])

  const warehouseName = (id: string) => warehouses.find(w => w.id === id)?.name ?? id
  const batchLabel = (batchId: string | null | undefined) => {
    if (!batchId) return 'General ledger'
    const batch = batches.find(b => b.id === batchId)
    return batch?.lot_number || batchId
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Inventory
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">Movement History</h1>
          <p className="text-xs text-ink-4 mt-0.5">{productName} — every SKU, running balance</p>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-danger">{error}</p>}
        {!error && rows === null && <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>}
        {!error && rows !== null && rows.length === 0 && (
          <p className="px-6 py-4 text-xs text-ink-4">No stock movements recorded for this product yet.</p>
        )}
        {!error && rows !== null && rows.length > 0 && (
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
              <tr>
                {['Date', 'SKU', 'Type', 'Warehouse', 'Batch', 'Quantity', 'Balance', 'Reason'].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line/60">
              {rows.map(m => (
                <tr key={m.id} className="hover:bg-hover/60 transition-colors">
                  <td className="px-4 py-2.5 text-ink-3 whitespace-nowrap">{formatDateTime(m.occurred_at)}</td>
                  <td className="px-4 py-2.5 font-mono text-ink-2">{m.skuCode}</td>
                  <td className="px-4 py-2.5 text-ink">{MOVEMENT_TYPE_LABEL[m.type]}</td>
                  <td className="px-4 py-2.5 text-ink-3">{warehouseName(m.warehouse_id)}</td>
                  <td className="px-4 py-2.5 text-ink-3">{batchLabel(m.batch_id)}</td>
                  <td className={clsx('px-4 py-2.5 font-medium tabular-nums', m.quantity_delta >= 0 ? 'text-success' : 'text-danger')}>
                    {m.quantity_delta >= 0 ? '+' : ''}{m.quantity_delta}
                  </td>
                  <td className="px-4 py-2.5 font-medium tabular-nums text-ink">{m.balance}</td>
                  <td className="px-4 py-2.5 text-ink-4">{m.reason || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

function SKUMovementHistoryPage({ token, sku, warehouses, onClose }: {
  token: string; sku: SKU; warehouses: { id: string; name: string }[]; onClose: () => void
}) {
  const [movements, setMovements] = useState<StockMovement[] | null>(null)
  const [batches, setBatches] = useState<Batch[]>([])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    Promise.all([inventoryApi.listStockMovements(token, sku.id), inventoryApi.listBatches(token, sku.id)])
      .then(([m, b]) => { if (!cancelled) { setMovements(m); setBatches(b) } })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load movement history') })
    return () => { cancelled = true }
  }, [token, sku.id])

  const warehouseName = (id: string) => warehouses.find(w => w.id === id)?.name ?? id
  const batchLabel = (batchId: string | null | undefined) => {
    if (!batchId) return 'General ledger'
    const batch = batches.find(b => b.id === batchId)
    return batch?.lot_number || batchId
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Inventory
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">Movement History</h1>
          <p className="text-xs font-mono text-ink-4 mt-0.5">{sku.sku_code}</p>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-danger">{error}</p>}
        {!error && movements === null && <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>}
        {!error && movements !== null && movements.length === 0 && (
          <p className="px-6 py-4 text-xs text-ink-4">No stock movements recorded for this SKU yet.</p>
        )}
        {!error && movements !== null && movements.length > 0 && (
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
              <tr>
                {['Date', 'Type', 'Warehouse', 'Batch', 'Quantity', 'Reason'].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line/60">
              {[...movements].reverse().map(m => (
                <tr key={m.id} className="hover:bg-hover/60 transition-colors">
                  <td className="px-4 py-2.5 text-ink-3 whitespace-nowrap">{formatDateTime(m.occurred_at)}</td>
                  <td className="px-4 py-2.5 text-ink">{MOVEMENT_TYPE_LABEL[m.type]}</td>
                  <td className="px-4 py-2.5 text-ink-3">{warehouseName(m.warehouse_id)}</td>
                  <td className="px-4 py-2.5 text-ink-3">{batchLabel(m.batch_id)}</td>
                  <td className={clsx('px-4 py-2.5 font-medium tabular-nums', m.quantity_delta >= 0 ? 'text-success' : 'text-danger')}>
                    {m.quantity_delta >= 0 ? '+' : ''}{m.quantity_delta}
                  </td>
                  <td className="px-4 py-2.5 text-ink-4">{m.reason || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

function BatchMovementHistoryPage({ token, sku, batch, warehouses, onClose }: {
  token: string; sku: SKU; batch: Batch; warehouses: { id: string; name: string }[]; onClose: () => void
}) {
  const [movements, setMovements] = useState<StockMovement[] | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    inventoryApi.listBatchMovements(token, batch.id)
      .then(m => { if (!cancelled) setMovements(m) })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : 'Failed to load movement history') })
    return () => { cancelled = true }
  }, [token, batch.id])

  const warehouseName = (id: string) => warehouses.find(w => w.id === id)?.name ?? id

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Inventory
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">Batch Movement History</h1>
          <p className="text-xs font-mono text-ink-4 mt-0.5">{sku.sku_code} · Lot {batch.lot_number || '—'}</p>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-danger">{error}</p>}
        {!error && movements === null && <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>}
        {!error && movements !== null && movements.length === 0 && (
          <p className="px-6 py-4 text-xs text-ink-4">No stock movements recorded for this batch yet.</p>
        )}
        {!error && movements !== null && movements.length > 0 && (
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
              <tr>
                {['Date', 'Type', 'Warehouse', 'Quantity', 'Reason'].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line/60">
              {[...movements].reverse().map(m => (
                <tr key={m.id} className="hover:bg-hover/60 transition-colors">
                  <td className="px-4 py-2.5 text-ink-3 whitespace-nowrap">{formatDateTime(m.occurred_at)}</td>
                  <td className="px-4 py-2.5 text-ink">{MOVEMENT_TYPE_LABEL[m.type]}</td>
                  <td className="px-4 py-2.5 text-ink-3">{warehouseName(m.warehouse_id)}</td>
                  <td className={clsx('px-4 py-2.5 font-medium tabular-nums', m.quantity_delta >= 0 ? 'text-success' : 'text-danger')}>
                    {m.quantity_delta >= 0 ? '+' : ''}{m.quantity_delta}
                  </td>
                  <td className="px-4 py-2.5 text-ink-4">{m.reason || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

// A single discard target, whichever tab it was opened from -- carries
// everything DiscardBatchInput needs plus enough to refresh the right
// list(s) afterwards.
interface DiscardTarget {
  skuId: string
  warehouseId: string
  batchId?: string
  quantity: number
  label: string
  source: 'expired' | 'expiring'
}

function DiscardModal({ target, submitting, error, onCancel, onConfirm }: {
  target: DiscardTarget
  submitting: boolean
  error: string
  onCancel: () => void
  onConfirm: (reason: string, photo?: File) => void
}) {
  const [reason, setReason] = useState('')
  const [photo, setPhoto] = useState<File | undefined>(undefined)

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/40 px-4">
      <div className="w-full max-w-sm rounded-2xl bg-surface border border-line p-5 space-y-4">
        <div className="flex items-start justify-between gap-3">
          <div>
            <h2 className="text-sm font-semibold text-ink">Discard {target.label}</h2>
            <p className="text-xs text-ink-4 mt-0.5">
              Records a Lost movement for {target.quantity} unit{target.quantity === 1 ? '' : 's'} — cannot be undone.
            </p>
          </div>
          <button onClick={onCancel} title="Close" className="shrink-0 text-ink-4 dark:hover:text-ink-2">
            <X size={16} />
          </button>
        </div>
        {error && <p className="text-xs text-danger">{error}</p>}
        <div>
          <label htmlFor="discard-reason" className="block text-xs text-ink-3 mb-1.5">Reason</label>
          <input
            id="discard-reason" value={reason} onChange={e => setReason(e.target.value)}
            className="w-full px-3 py-2 text-sm rounded-lg dark:bg-night-800 bg-gray-50 border border-line text-ink outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div>
          <label className="block text-xs text-ink-3 mb-1.5">Photo (optional)</label>
          <label className="flex items-center gap-2 px-3 py-2 text-xs rounded-lg dark:bg-night-800 bg-gray-50 border border-line text-ink-2 cursor-pointer">
            <ImageIcon size={14} />
            <span className="truncate">{photo ? photo.name : 'Choose a photo'}</span>
            <input
              type="file" accept="image/*" className="hidden"
              onChange={e => setPhoto(e.target.files?.[0] ?? undefined)}
            />
          </label>
        </div>
        <div className="flex gap-2">
          <button
            onClick={onCancel}
            className="flex-1 py-2 rounded-lg dark:bg-night-800 bg-gray-100 text-ink-2 text-xs font-medium hover:bg-hover"
          >
            Cancel
          </button>
          <button
            onClick={() => onConfirm(reason, photo)}
            disabled={submitting}
            className="flex-1 py-2 rounded-lg bg-red-600 text-white text-xs font-medium hover:bg-red-700 disabled:opacity-50"
          >
            {submitting ? 'Discarding…' : 'Confirm'}
          </button>
        </div>
      </div>
    </div>
  )
}

const EXPIRY_TABS = ['expired', 'expiring', 'discarded'] as const
type ExpiryTab = typeof EXPIRY_TABS[number]

// The tab lives in ?tab= (see inventoryRoutes). Defaults to 'expired' --
// the permanent "Damaged / Lost / Discarded" submenu entry opens straight
// to 'discarded' instead, since that's the whole reason it exists: unlike
// the red "Expired batches" alert (only shown when something needs review
// right now), this history is only reachable through an explicit menu
// entry, always -- not gated behind any current-state condition.
function ExpiryDiscardPage({
  onClose, expiredReview, expiringBatches, discardsReport, discardHook,
}: {
  onClose: () => void
  expiredReview: ReturnType<typeof useExpiredBatchReview>
  expiringBatches: ReturnType<typeof useExpiringBatches>
  discardsReport: ReturnType<typeof useDiscards>
  discardHook: ReturnType<typeof useDiscardBatch>
}) {
  const { batches, loading, error, extend, dismiss } = expiredReview
  const [tab, setTab] = useQueryEnum<ExpiryTab>('tab', EXPIRY_TABS, 'expired')
  const [rowError, setRowError] = useState('')
  const [extendingBatchId, setExtendingBatchId] = useState<string | null>(null)
  const [newExpiryDate, setNewExpiryDate] = useState('')
  const [actingBatchId, setActingBatchId] = useState<string | null>(null)
  const [discardTarget, setDiscardTarget] = useState<DiscardTarget | null>(null)

  const openExtend = (batchId: string) => {
    setRowError('')
    setNewExpiryDate('')
    setExtendingBatchId(batchId)
  }

  function openDiscard(source: 'expired' | 'expiring', row: ExpiredBatchReview | ExpiringBatch) {
    setDiscardTarget({
      skuId: row.sku_id, warehouseId: row.warehouse_id, batchId: row.batch_id, quantity: row.quantity,
      label: `${row.product_name} (${row.sku_code})`, source,
    })
  }

  async function handleConfirmDiscard(reason: string, photo?: File) {
    if (!discardTarget) return
    try {
      await discardHook.discard(discardTarget.skuId, {
        warehouse_id: discardTarget.warehouseId, batch_id: discardTarget.batchId,
        quantity: discardTarget.quantity, reason, photo,
      })
      setDiscardTarget(null)
      if (discardTarget.source === 'expired') expiredReview.refresh()
      else expiringBatches.refresh()
      discardsReport.refresh()
    } catch {
      // discardHook.error already holds the message; modal stays open.
    }
  }

  const handleConfirmExtend = async (batchId: string) => {
    if (!newExpiryDate) return
    setRowError('')
    setActingBatchId(batchId)
    try {
      await extend(batchId, newExpiryDate)
      setExtendingBatchId(null)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to extend expiry')
    } finally {
      setActingBatchId(null)
    }
  }

  const handleDismiss = async (batchId: string) => {
    setRowError('')
    setActingBatchId(batchId)
    try {
      await dismiss(batchId)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to dismiss batch')
    } finally {
      setActingBatchId(null)
    }
  }

  const TABS: { id: ExpiryTab; label: string }[] = [
    { id: 'expired', label: `Expired${batches.length ? ` (${batches.length})` : ''}` },
    { id: 'expiring', label: 'Expiring soon' },
    { id: 'discarded', label: 'Discarded' },
  ]

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Inventory
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">Expiry & Discards</h1>
          <p className="text-xs text-ink-4 mt-0.5">Review expired/expiring stock, discard it, and see what's already been thrown away</p>
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
                : 'bg-raised text-ink-3 hover:bg-hover',
            )}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="flex-1 overflow-auto">
        {/* discardHook.error is deliberately not shown here -- the modal (still open on failure) already surfaces it inline. */}
        {(error || rowError) && (
          <p className="px-6 py-4 text-xs text-danger">{error || rowError}</p>
        )}

        {tab === 'expired' && (
          <>
            {!error && loading && batches.length === 0 && <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>}
            {!error && !loading && batches.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">No expired batches pending review.</p>
            )}
            {batches.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                  <tr>
                    {['SKU', 'Product', 'Warehouse', 'Lot', 'Expired', 'Qty', ''].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {batches.map(b => (
                    <tr key={b.batch_id} className="hover:bg-hover/60 transition-colors">
                      <td className="px-4 py-2.5 font-mono text-ink-2">{b.sku_code}</td>
                      <td className="px-4 py-2.5 text-ink">{b.product_name}</td>
                      <td className="px-4 py-2.5 text-ink-3">{b.warehouse_name}</td>
                      <td className="px-4 py-2.5 text-ink-3">{b.lot_number || '—'}</td>
                      <td className="px-4 py-2.5 text-danger">{formatDate(b.expiry_date)}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{b.quantity}</td>
                      <td className="px-4 py-2.5">
                        {extendingBatchId === b.batch_id ? (
                          <div className="flex items-center gap-1.5">
                            <input
                              type="date"
                              aria-label="New expiry date"
                              value={newExpiryDate}
                              onChange={e => setNewExpiryDate(e.target.value)}
                              className="px-2 py-1 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-blue-500"
                            />
                            <button
                              onClick={() => handleConfirmExtend(b.batch_id)}
                              disabled={!newExpiryDate || actingBatchId === b.batch_id}
                              className="px-2 py-1 rounded-lg bg-blue-600 text-white text-[11px] font-medium hover:bg-blue-700 disabled:opacity-50"
                            >
                              Confirm
                            </button>
                            <button
                              onClick={() => setExtendingBatchId(null)}
                              className="px-2 py-1 rounded-lg bg-fill text-ink-2 text-[11px] font-medium hover:bg-hover"
                            >
                              Cancel
                            </button>
                          </div>
                        ) : (
                          <div className="flex items-center gap-1.5">
                            <button
                              onClick={() => openDiscard('expired', b)}
                              disabled={actingBatchId === b.batch_id}
                              className="px-2 py-1 rounded-lg dark:bg-red-500/10 bg-red-50 text-danger text-[11px] font-medium hover:dark:bg-red-500/20 hover:bg-red-100 disabled:opacity-50"
                            >
                              Discard
                            </button>
                            <button
                              onClick={() => openExtend(b.batch_id)}
                              disabled={actingBatchId === b.batch_id}
                              className="px-2 py-1 rounded-lg bg-fill text-ink-2 text-[11px] font-medium hover:bg-hover disabled:opacity-50"
                            >
                              Extend
                            </button>
                            <button
                              onClick={() => handleDismiss(b.batch_id)}
                              disabled={actingBatchId === b.batch_id}
                              className="px-2 py-1 rounded-lg bg-fill text-ink-2 text-[11px] font-medium hover:bg-hover disabled:opacity-50"
                            >
                              Dismiss
                            </button>
                          </div>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}

        {tab === 'expiring' && (
          <>
            {!expiringBatches.error && expiringBatches.loading && expiringBatches.batches.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>
            )}
            {!expiringBatches.error && !expiringBatches.loading && expiringBatches.batches.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">Nothing expiring soon.</p>
            )}
            {expiringBatches.batches.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                  <tr>
                    {['SKU', 'Product', 'Warehouse', 'Lot', 'Expires', 'Qty', ''].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {expiringBatches.batches.map(b => (
                    <tr key={b.batch_id} className="hover:bg-hover/60 transition-colors">
                      <td className="px-4 py-2.5 font-mono text-ink-2">{b.sku_code}</td>
                      <td className="px-4 py-2.5 text-ink">{b.product_name}</td>
                      <td className="px-4 py-2.5 text-ink-3">{b.warehouse_name}</td>
                      <td className="px-4 py-2.5 text-ink-3">{b.lot_number || '—'}</td>
                      <td className="px-4 py-2.5 text-ink-2">{formatDate(b.expiry_date)}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{b.quantity}</td>
                      <td className="px-4 py-2.5">
                        <button
                          onClick={() => openDiscard('expiring', b)}
                          className="px-2 py-1 rounded-lg dark:bg-red-500/10 bg-red-50 text-danger text-[11px] font-medium hover:dark:bg-red-500/20 hover:bg-red-100"
                        >
                          Discard
                        </button>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}

        {tab === 'discarded' && (
          <>
            {!discardsReport.error && discardsReport.loading && discardsReport.discards.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">Loading…</p>
            )}
            {!discardsReport.error && !discardsReport.loading && discardsReport.discards.length === 0 && (
              <p className="px-6 py-4 text-xs text-ink-4">Nothing discarded yet.</p>
            )}
            {discardsReport.discards.length > 0 && (
              <table className="w-full text-xs">
                <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
                  <tr>
                    {['Photo', 'SKU', 'Product', 'Type', 'Warehouse', 'Lot', 'Qty', 'Reason', 'By', 'Discarded'].map(h => (
                      <th key={h} className="px-4 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{h}</th>
                    ))}
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {discardsReport.discards.map(d => (
                    <tr key={d.id} className="hover:bg-hover/60 transition-colors">
                      <td className="px-4 py-2.5">
                        {d.photo_url ? (
                          <img src={d.photo_url} alt={`${d.product_name} discard photo`} className="w-8 h-8 rounded object-cover" />
                        ) : (
                          <div className="w-8 h-8 rounded dark:bg-night-800 bg-gray-100 flex items-center justify-center">
                            <ImageIcon size={12} className="text-ink-4" />
                          </div>
                        )}
                      </td>
                      <td className="px-4 py-2.5 font-mono text-ink-2">{d.sku_code}</td>
                      <td className="px-4 py-2.5 text-ink">{d.product_name}</td>
                      <td className="px-4 py-2.5">
                        <span className={clsx(
                          'px-1.5 py-0.5 rounded text-[10px] font-medium capitalize',
                          d.type === 'damaged' ? 'bg-amber-500/15 text-warning' : 'bg-red-500/15 text-danger',
                        )}>
                          {d.type}
                        </span>
                      </td>
                      <td className="px-4 py-2.5 text-ink-3">{d.warehouse_name}</td>
                      <td className="px-4 py-2.5 text-ink-3">{d.lot_number || '—'}</td>
                      <td className="px-4 py-2.5 tabular-nums text-ink-2">{d.quantity}</td>
                      <td className="px-4 py-2.5 text-ink-3">{d.reason || '—'}</td>
                      <td className="px-4 py-2.5 text-ink-3">{d.discarded_by || '—'}</td>
                      <td className="px-4 py-2.5 text-ink-3">{formatDate(d.discarded_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </>
        )}
      </div>

      {discardTarget && (
        <DiscardModal
          target={discardTarget}
          submitting={discardHook.discarding}
          error={discardHook.error ?? ''}
          onCancel={() => setDiscardTarget(null)}
          onConfirm={handleConfirmDiscard}
        />
      )}
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function Inventory() {
  const { token } = useAuth()
  // Search/page/stock filter (?q=&page=&stock=) and every full-screen view
  // (history, expiry review -- see inventoryRoutes) live in the URL, so a
  // refresh or Back keeps the user where they were.
  const location = useLocation()
  const view = parseInventoryPath(location.pathname)
  const { open, exitToOpener } = useStackNavigation()
  const backToList = () => exitToOpener(INVENTORY_PATH)
  const [urlSearch, setUrlSearch] = useQueryString('q', { resets: ['page'] })
  const search = urlSearch ?? ''
  const [searchInput, setSearchInput] = useState(search)
  const [page, setPage] = useQueryPage()
  const [stockFilter, setStockFilter] = useQueryEnum<StockFilter>('stock', STOCK_FILTERS, 'all')
  const [summaryByProduct, setSummaryByProduct] = useState<Record<string, ProductStockSummary>>({})
  const expiredReview = useExpiredBatchReview(token ?? '')
  const expiringBatches = useExpiringBatches(token ?? '')
  const discardsReport = useDiscards(token ?? '')
  const discardHook = useDiscardBatch(token ?? '')

  // Debounced into ?q=; only writes a real change, so landing on a URL
  // with ?page= doesn't get reset by the search write (which clears page).
  useEffect(() => {
    if (searchInput === search) return
    const timeout = setTimeout(() => setUrlSearch(searchInput), 300)
    return () => clearTimeout(timeout)
  }, [searchInput, search, setUrlSearch])
  useEffect(() => { setSearchInput(search) }, [search])

  const { products, total, loading, error } = useProducts(token ?? '', { page, perPage: PAGE_SIZE, search })
  const { warehouses } = useWarehouses(token ?? '')

  // Rows report their own stock total once loaded — low-stock/out-of-stock
  // filtering is client-side over the current page (the product search API
  // has no stock-aware filtering yet), so a row not yet reporting a
  // summary is still shown rather than hidden while it resolves.
  const handleSummary = (productId: string, summary: ProductStockSummary) => {
    setSummaryByProduct(prev => {
      const existing = prev[productId]
      if (existing && existing.totalStock === summary.totalStock && existing.skuCount === summary.skuCount) return prev
      return { ...prev, [productId]: summary }
    })
  }

  const visibleProducts = products.filter(p => {
    if (stockFilter === 'all') return true
    const summary = summaryByProduct[p.id]
    if (!summary) return true
    if (stockFilter === 'out') return summary.totalStock <= 0
    return summary.totalStock > 0 && summary.totalStock <= LOW_STOCK_THRESHOLD
  })

  if (view.kind === 'productHistory' || view.kind === 'skuHistory' || view.kind === 'batchHistory') {
    return <InventoryHistoryView token={token ?? ''} view={view} warehouses={warehouses} onClose={backToList} />
  }

  if (view.kind === 'expiry') {
    return (
      <ExpiryDiscardPage
        onClose={backToList}
        expiredReview={expiredReview}
        expiringBatches={expiringBatches}
        discardsReport={discardsReport}
        discardHook={discardHook}
      />
    )
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={searchInput}
            onChange={e => setSearchInput(e.target.value)}
            placeholder="Search product..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div className="flex items-center rounded-lg border border-line overflow-hidden shrink-0">
          {([
            ['all', 'All'],
            ['low', 'Low Stock'],
            ['out', 'Out of Stock'],
          ] as const).map(([value, label]) => (
            <button
              key={value}
              type="button"
              onClick={() => setStockFilter(value)}
              className={clsx(
                'px-2.5 py-1.5 text-[11px] font-medium transition-colors',
                stockFilter === value
                  ? 'bg-blue-600 text-white'
                  : 'bg-raised text-ink-3 hover:bg-hover',
              )}
            >
              {label}
            </button>
          ))}
        </div>
        {expiredReview.batches.length > 0 && (
          <button
            onClick={() => open(inventoryPaths.expiry)}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg dark:bg-red-500/10 bg-red-50 text-danger text-[11px] font-medium hover:dark:bg-red-500/20 hover:bg-red-100 transition-colors shrink-0"
          >
            <AlertTriangle size={12} /> Expired batches ({expiredReview.batches.length})
          </button>
        )}
        <button
          onClick={() => open(`${inventoryPaths.expiry}?tab=discarded`)}
          className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg bg-raised border border-line text-ink-3 text-[11px] font-medium hover:bg-hover transition-colors shrink-0"
        >
          <Trash2 size={12} /> Damaged / Lost / Discarded
        </button>
        <span className="text-xs text-ink-4 ml-auto">{total} product{total === 1 ? '' : 's'}</span>
      </div>

      {error && (
        <div className="flex items-center gap-1.5 px-5 py-2 text-xs text-danger">
          <AlertTriangle size={12} /> {error}
        </div>
      )}

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Product</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Price</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Sale Price</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Stock</th>
              <th className="px-3 py-2.5 w-8" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && visibleProducts.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">No products found.</td></tr>
            )}
            {/* A SKU action URL for a product not on this list page (shared
                link, or the list moved on) still opens -- pinned on top. */}
            {!loading && view.kind === 'skuAction' && !visibleProducts.some(p => p.id === view.productId) && (
              <ProductStockRow
                key={`pinned-${view.productId}`} token={token ?? ''} productId={view.productId} productName="" warehouses={warehouses}
                onSummary={handleSummary} onViewHistory={sku => open(inventoryPaths.skuHistory(view.productId, sku.id))}
                onViewBatchHistory={(sku, batch) => open(inventoryPaths.batchHistory(view.productId, sku.id, batch.id))}
                onViewProductHistory={() => open(inventoryPaths.productHistory(view.productId))}
              />
            )}
            {!loading && visibleProducts.map(p => (
              <ProductStockRow
                key={p.id} token={token ?? ''} productId={p.id} productName={p.name} productImageUrl={p.image_url} warehouses={warehouses}
                onSummary={handleSummary} onViewHistory={sku => open(inventoryPaths.skuHistory(p.id, sku.id))}
                onViewBatchHistory={(sku, batch) => open(inventoryPaths.batchHistory(p.id, sku.id, batch.id))}
                onViewProductHistory={() => open(inventoryPaths.productHistory(p.id))}
              />
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
    </div>
  )
}

// Movement history page for the product / SKU / batch named in the URL
// (see inventoryRoutes) -- reloads its subject from the ids, so it works
// straight from a refresh or a shared link.
function InventoryHistoryView({ token, view, warehouses, onClose }: {
  token: string
  view: Parameters<typeof useInventoryHistoryTarget>[1]
  warehouses: { id: string; name: string }[]
  onClose: () => void
}) {
  const target = useInventoryHistoryTarget(token, view)
  const data = target.data
  if (!data) {
    if (target.loading) return <RecordLoading />
    // A missing SKU/batch reports itself; anything else means the product
    // lookup (the first request) failed.
    const subject = target.error === 'SKU not found' ? 'SKU' : target.error === 'Batch not found' ? 'Batch' : 'Product'
    return <RecordNotFound label={subject} onBack={onClose} />
  }
  if (view.kind === 'batchHistory' && data.sku && data.batch) {
    return <BatchMovementHistoryPage token={token} sku={data.sku} batch={data.batch} warehouses={warehouses} onClose={onClose} />
  }
  if (view.kind === 'skuHistory' && data.sku) {
    return <SKUMovementHistoryPage token={token} sku={data.sku} warehouses={warehouses} onClose={onClose} />
  }
  return <ProductMovementHistoryPage token={token} productName={data.productName} skus={data.skus} warehouses={warehouses} onClose={onClose} />
}
