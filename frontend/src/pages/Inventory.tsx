import { useState, useEffect, type ReactNode } from 'react'
import {
  Search, ChevronRight, ChevronDown, Plus, ArrowRightLeft, AlertTriangle,
  Trash2, Edit3, X, PackageMinus, ClipboardList, Tag, ArrowLeft, History, Boxes, Scale,
} from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useProducts } from '../pms/hooks/useProducts'
import { useProductDetail } from '../pms/hooks/useProductDetail'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import { useSKUStock } from '../inventory/hooks/useSKUStock'
import { useExpiredBatchReview } from '../inventory/hooks/useExpiredBatchReview'
import * as pmsApi from '../pms/api'
import * as inventoryApi from '../inventory/api'
import type { SKU } from '../pms/types'
import type { Batch, MovementType, StockMovement } from '../inventory/types'

const PAGE_SIZE = 20

// No per-SKU reorder threshold exists in the backend yet -- this is a
// client-side heuristic ("low" = at or below this many units, but not
// zero) until a real configurable threshold field is added.
const LOW_STOCK_THRESHOLD = 10

type StockFilter = 'all' | 'low' | 'out'

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

// ── Generic modal shell ──────────────────────────────────────────────────────

function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-sm rounded-xl dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 shadow-xl">
        <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">{title}</h3>
          <button type="button" onClick={onClose} aria-label="Close" className="dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600">
            <X size={16} />
          </button>
        </div>
        <div className="p-4">{children}</div>
      </div>
    </div>
  )
}

const inputClass = 'w-full px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500'
const labelClass = 'block text-[10px] font-medium dark:text-gray-400 text-gray-500 mb-1'

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
        {error && <p className="text-[10px] text-red-500">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
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
  warehouses, batches, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  onSave: (input: { warehouse_id: string; batch_id?: string | null; type: MovementType; quantity_delta: number; reason?: string }) => Promise<void>
  onClose: () => void
}) {
  const [type, setType] = useState<'receipt' | 'sale' | 'adjustment'>('receipt')
  const [warehouseId, setWarehouseId] = useState(warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState('')
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
        <div>
          <label className={labelClass} htmlFor="adjust-warehouse">Warehouse</label>
          <select id="adjust-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
            {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
          </select>
        </div>
        <div>
          <label className={labelClass} htmlFor="adjust-batch">Batch (leave empty for general ledger)</label>
          <select id="adjust-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
            <option value="">General ledger (no batch)</option>
            {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
          </select>
        </div>
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
        {error && <p className="text-[10px] text-red-500">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
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
  warehouses, batches, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  onSave: (input: { warehouse_id: string; batch_id?: string | null; counted_quantity: number; reason?: string }) => Promise<void>
  onClose: () => void
}) {
  const [warehouseId, setWarehouseId] = useState(warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState('')
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
        <div>
          <label className={labelClass} htmlFor="recount-warehouse">Warehouse</label>
          <select id="recount-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
            {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
          </select>
        </div>
        <div>
          <label className={labelClass} htmlFor="recount-batch">Batch (leave empty for general ledger)</label>
          <select id="recount-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
            <option value="">General ledger (no batch)</option>
            {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
          </select>
        </div>
        <div>
          <label className={labelClass} htmlFor="recount-qty">Counted quantity</label>
          <input id="recount-qty" type="number" min={0} value={counted} onChange={e => setCounted(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="recount-reason">Reason (optional)</label>
          <input id="recount-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-[10px] text-red-500">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
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
  warehouses, batches, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  onSave: (input: {
    quantity: number; from_warehouse_id: string; from_batch_id?: string | null
    to_warehouse_id: string; to_batch_id?: string | null; reason?: string
  }) => Promise<void>
  onClose: () => void
}) {
  const [quantity, setQuantity] = useState('')
  const [fromWarehouse, setFromWarehouse] = useState(warehouses[0]?.id ?? '')
  const [fromBatch, setFromBatch] = useState('')
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
        <p className="text-[10px] dark:text-gray-500 text-gray-400">
          Leaving a batch empty means the general ledger (un-lotted stock) — use this to move stock into or out of a batch too.
        </p>
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className={labelClass} htmlFor="transfer-from-wh">From warehouse</label>
            <select id="transfer-from-wh" value={fromWarehouse} onChange={e => setFromWarehouse(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          <div>
            <label className={labelClass} htmlFor="transfer-to-wh">To warehouse</label>
            <select id="transfer-to-wh" value={toWarehouse} onChange={e => setToWarehouse(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          <div>
            <label className={labelClass} htmlFor="transfer-from-batch">From batch</label>
            <select id="transfer-from-batch" value={fromBatch} onChange={e => setFromBatch(e.target.value)} className={inputClass}>
              <option value="">General ledger</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
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
        {error && <p className="text-[10px] text-red-500">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
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
        <p className="text-[10px] dark:text-gray-500 text-gray-400">
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
        {error && <p className="text-[10px] text-red-500">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
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

// ── Portion one sellable item from a bulk pool (see pms.SKU.portioned_from_sku_id) ──

function PortionFromBulkModal({
  warehouses, batches, nominalWeightGrams, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  nominalWeightGrams?: number | null
  onSave: (input: { warehouse_id: string; batch_id?: string | null; weight_grams?: number; reason?: string }) => Promise<unknown>
  onClose: () => void
}) {
  const [warehouseId, setWarehouseId] = useState(warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState('')
  const [weightGrams, setWeightGrams] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    let weight: number | undefined
    if (weightGrams.trim() !== '') {
      const n = parseInt(weightGrams, 10)
      if (Number.isNaN(n) || n <= 0) {
        setError('Enter a positive weight in grams, or leave blank to use the nominal weight')
        return
      }
      weight = n
    }
    setSaving(true)
    setError(null)
    try {
      await onSave({ warehouse_id: warehouseId, batch_id: batchId || null, weight_grams: weight, reason })
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to portion from bulk stock')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Add Item From Weight Pool" onClose={onClose}>
      <div className="space-y-3">
        <p className="text-[10px] dark:text-gray-500 text-gray-400">
          Peels one sellable unit off the linked weight pool. Repeatable as many times as you like.
        </p>
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className={labelClass} htmlFor="portion-warehouse">Warehouse</label>
            <select id="portion-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          <div>
            <label className={labelClass} htmlFor="portion-batch">Batch</label>
            <select id="portion-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
              <option value="">General ledger</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
        </div>
        <div>
          <label className={labelClass} htmlFor="portion-weight">
            Actual weight (grams){nominalWeightGrams != null && ` — defaults to ${nominalWeightGrams}g`}
          </label>
          <input
            id="portion-weight" type="number" min={1} value={weightGrams} onChange={e => setWeightGrams(e.target.value)}
            placeholder={nominalWeightGrams != null ? String(nominalWeightGrams) : undefined} className={inputClass}
          />
        </div>
        <div>
          <label className={labelClass} htmlFor="portion-reason">Reason (optional)</label>
          <input id="portion-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-[10px] text-red-500">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
            Cancel
          </button>
          <button type="button" onClick={submit} disabled={saving} className="px-3 py-1.5 text-xs rounded-lg bg-blue-600 text-white hover:bg-blue-700 disabled:opacity-50">
            {saving ? 'Adding…' : 'Add item'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Mark damaged / lost ───────────────────────────────────────────────────────

function DamagedLostModal({
  warehouses, batches, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  onSave: (input: { warehouse_id: string; batch_id?: string | null; quantity: number; type: 'damaged' | 'lost'; reason?: string }) => Promise<void>
  onClose: () => void
}) {
  const [type, setType] = useState<'damaged' | 'lost'>('damaged')
  const [warehouseId, setWarehouseId] = useState(warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState('')
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
        <div>
          <label className={labelClass} htmlFor="dl-warehouse">Warehouse</label>
          <select id="dl-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
            {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
          </select>
        </div>
        <div>
          <label className={labelClass} htmlFor="dl-batch">Batch (leave empty for general ledger)</label>
          <select id="dl-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
            <option value="">General ledger (no batch)</option>
            {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
          </select>
        </div>
        <div>
          <label className={labelClass} htmlFor="dl-qty">Quantity</label>
          <input id="dl-qty" type="number" min={1} value={quantity} onChange={e => setQuantity(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="dl-reason">Reason (optional)</label>
          <input id="dl-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-[10px] text-red-500">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
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

        <div className="rounded-lg border dark:border-[#2D3748] border-gray-200 p-2.5 space-y-2">
          <div className="flex items-center justify-between">
            <label className={clsx(labelClass, 'mb-0')} htmlFor="price-edit-sale-price">Sale price (optional)</label>
            {discountPct != null && discountPct > 0 && (
              <span className="text-[9px] px-1.5 py-px rounded bg-green-500/15 text-green-400 font-medium">-{discountPct}%</span>
            )}
            {discountPct != null && discountPct <= 0 && (
              <span className="text-[9px] text-red-400">must be below regular price</span>
            )}
          </div>
          <input id="price-edit-sale-price" type="number" step="0.01" min="0" value={salePrice} onChange={e => {
            setSalePrice(e.target.value)
            if (e.target.value.trim() === '') setSaleEndDate('')
          }} className={inputClass} />

          <div>
            <label className={labelClass} htmlFor="price-edit-sale-end-date">Sale ends (optional)</label>
            <input id="price-edit-sale-end-date" type="date" value={saleEndDate} onChange={e => setSaleEndDate(e.target.value)} className={inputClass} disabled={salePriceNum == null} />
            <p className="text-[9px] dark:text-gray-500 text-gray-400 mt-1">Reverts to the regular price automatically the day after this date.</p>
          </div>
        </div>

        {error && <p className="text-[10px] text-red-500">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
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

type ActiveModal = 'adjust' | 'recount' | 'transfer' | 'split-case' | 'portion-from-bulk' | 'damaged-lost' | 'batch-new' | 'price' | { edit: Batch } | null

function SKUStockPanel({ token, sku, warehouses, onSKUUpdated, onViewHistory, onViewBatchHistory }: {
  token: string; sku: SKU; warehouses: { id: string; name: string }[]; onSKUUpdated: () => void
  onViewHistory: (sku: SKU) => void; onViewBatchHistory: (sku: SKU, batch: Batch) => void
}) {
  const { level, batches, loading, error, addBatch, editBatch, removeBatch, adjust, recount, transfer, splitCase, portionFromBulk, markDamagedOrLost } =
    useSKUStock(token, sku.id)
  const [modal, setModal] = useState<ActiveModal>(null)
  const [deleteError, setDeleteError] = useState<string | null>(null)
  const [showAllBatches, setShowAllBatches] = useState(false)

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
    } catch (e) {
      setDeleteError(e instanceof Error ? e.message : 'Failed to delete batch')
    }
  }

  return (
    <div className="pl-8 pr-4 py-3 border-t dark:border-[#2D3748]/60 border-gray-100 dark:bg-[#0B1220]/40 bg-gray-50/60">
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-3">
          <span className="font-mono text-[11px] dark:text-gray-300 text-gray-700">{sku.sku_code}</span>
          <button
            type="button"
            onClick={() => setModal('price')}
            title="Edit price"
            className="flex items-center gap-1 text-[10px] dark:text-gray-500 text-gray-400 hover:text-blue-400"
          >
            <Tag size={11} /> Price
          </button>
          <button
            type="button"
            onClick={() => onViewHistory(sku)}
            title="View movement history"
            className="flex items-center gap-1 text-[10px] dark:text-gray-500 text-gray-400 hover:text-blue-400"
          >
            <History size={11} /> History
          </button>
          {level && (
            <span data-testid="stock-level" className="text-[10px] dark:text-gray-400 text-gray-500">
              Total: <strong className="dark:text-gray-200 text-gray-800">{formatSKUQuantity(sku, level.total)}</strong>
              {' · '}Syncable: <strong className="dark:text-gray-200 text-gray-800">{formatSKUQuantity(sku, level.syncable)}</strong>
            </span>
          )}
          {loading && <span className="text-[10px] dark:text-gray-600 text-gray-400">Loading…</span>}
        </div>
        <div className="flex items-center gap-1.5">
          <button type="button" onClick={() => setModal('adjust')} className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
            <Plus size={11} /> Adjust
          </button>
          <button type="button" onClick={() => setModal('recount')} className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
            <ClipboardList size={11} /> Recount
          </button>
          <button type="button" onClick={() => setModal('transfer')} className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
            <ArrowRightLeft size={11} /> Transfer
          </button>
          {sku.case_of_sku_id && (
            <button type="button" onClick={() => setModal('split-case')} title="Split sealed cases into individual units" className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
              <Boxes size={11} /> Split case
            </button>
          )}
          {sku.portioned_from_sku_id && (
            <button type="button" onClick={() => setModal('portion-from-bulk')} title="Add a sellable item peeled off the linked weight pool" className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
              <Scale size={11} /> Add item from weight pool
            </button>
          )}
          <button type="button" onClick={() => setModal('damaged-lost')} className="flex items-center gap-1 px-2 py-1 rounded-md text-[10px] dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-red-400 hover:dark:bg-[#2D3748] hover:bg-gray-100">
            <PackageMinus size={11} /> Damaged/Lost
          </button>
        </div>
      </div>

      {error && <p className="text-[10px] text-red-500 mb-2">{error}</p>}
      {deleteError && <p className="text-[10px] text-red-500 mb-2">{deleteError}</p>}

      <div className="rounded-lg border dark:border-[#2D3748] border-gray-200 overflow-hidden">
        <table className="w-full text-[10px]">
          <thead className="dark:bg-[#111827] bg-gray-100">
            <tr>
              {['Lot', 'Warehouse', 'Qty', 'Expiry', ''].map(h => (
                <th key={h} className="px-2.5 py-1.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">{h}</th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {visibleBatches.map(b => (
              <tr key={b.is_general_ledger ? `general-${b.warehouse_id}` : b.id} className={clsx(b.is_general_ledger && 'dark:bg-[#111827]/40 bg-gray-100/60')}>
                <td className="px-2.5 py-1.5 font-mono dark:text-gray-300 text-gray-700">
                  {b.is_general_ledger ? 'General ledger' : (b.lot_number || '—')}
                </td>
                <td className="px-2.5 py-1.5 dark:text-gray-400 text-gray-500">{warehouses.find(w => w.id === b.warehouse_id)?.name ?? b.warehouse_id}</td>
                <td className="px-2.5 py-1.5 font-mono dark:text-gray-300 text-gray-700">{b.quantity}</td>
                <td className="px-2.5 py-1.5">
                  {!b.is_general_ledger && (
                    <span className={clsx(
                      'font-mono',
                      isExpired(b.expiry_date) ? 'text-red-400' : isExpiringSoon(b.expiry_date) ? 'text-amber-400' : 'dark:text-gray-400 text-gray-500',
                    )}>
                      {formatDate(b.expiry_date)}
                    </span>
                  )}
                </td>
                <td className="px-2.5 py-1.5">
                  {!b.is_general_ledger && (
                    <div className="flex items-center justify-end gap-2">
                      <button type="button" onClick={() => onViewBatchHistory(sku, b)} aria-label={`View movement history for batch ${b.lot_number}`} title="View movement history" className="dark:text-gray-500 text-gray-400 hover:text-blue-400">
                        <History size={11} />
                      </button>
                      <button type="button" onClick={() => setModal({ edit: b })} aria-label={`Edit batch ${b.lot_number}`} className="dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600">
                        <Edit3 size={11} />
                      </button>
                      <button type="button" onClick={() => handleDeleteBatch(b.id)} aria-label={`Delete batch ${b.lot_number}`} className="dark:text-gray-500 text-gray-400 hover:text-red-400">
                        <Trash2 size={11} />
                      </button>
                    </div>
                  )}
                </td>
              </tr>
            ))}
            {batches.length === 0 && (
              <tr>
                <td colSpan={5} className="px-2.5 py-3 text-center dark:text-gray-600 text-gray-400">No batches — all stock is in the general ledger.</td>
              </tr>
            )}
            {batches.length > 0 && visibleBatches.length === 0 && (
              <tr>
                <td colSpan={5} className="px-2.5 py-3 text-center dark:text-gray-600 text-gray-400">
                  All {zeroStockBatchCount} batch{zeroStockBatchCount === 1 ? '' : 'es'} for this SKU {zeroStockBatchCount === 1 ? 'has' : 'have'} zero stock.
                </td>
              </tr>
            )}
          </tbody>
        </table>
        <div className="px-2.5 py-1.5 border-t dark:border-[#2D3748] border-gray-200 flex items-center gap-3">
          <button type="button" onClick={() => setModal('batch-new')} className="text-[10px] text-blue-400 hover:text-blue-300">
            + New batch
          </button>
          {zeroStockBatchCount > 0 && (
            <button type="button" onClick={() => setShowAllBatches(v => !v)} className="text-[10px] dark:text-gray-500 text-gray-400 hover:text-blue-400">
              {showAllBatches ? 'Hide zero-stock batches' : `Show ${zeroStockBatchCount} zero-stock batch${zeroStockBatchCount === 1 ? '' : 'es'}`}
            </button>
          )}
        </div>
      </div>

      {modal === 'adjust' && (
        <AdjustStockModal warehouses={warehouses} batches={batches} onClose={() => setModal(null)}
          onSave={input => adjust(input).then(() => undefined)} />
      )}
      {modal === 'recount' && (
        <RecountModal warehouses={warehouses} batches={batches} onClose={() => setModal(null)}
          onSave={input => recount(input).then(() => undefined)} />
      )}
      {modal === 'transfer' && (
        <TransferModal warehouses={warehouses} batches={batches} onClose={() => setModal(null)} onSave={transfer} />
      )}
      {modal === 'split-case' && (
        <SplitCaseModal warehouses={warehouses} batches={batches} onClose={() => setModal(null)} onSave={splitCase} />
      )}
      {modal === 'portion-from-bulk' && (
        <PortionFromBulkModal
          warehouses={warehouses} batches={batches} nominalWeightGrams={sku.weight_grams}
          onClose={() => setModal(null)} onSave={portionFromBulk}
        />
      )}
      {modal === 'damaged-lost' && (
        <DamagedLostModal warehouses={warehouses} batches={batches} onClose={() => setModal(null)}
          onSave={input => markDamagedOrLost(input).then(() => undefined)} />
      )}
      {modal === 'batch-new' && (
        <BatchFormModal batch={null} warehouses={warehouses} onClose={() => setModal(null)}
          onSave={input => addBatch(input).then(() => undefined)} />
      )}
      {modal === 'price' && (
        <PriceModal sku={sku} onClose={() => setModal(null)} onSave={handleSavePrice} />
      )}
      {modal !== null && typeof modal === 'object' && (
        <BatchFormModal batch={modal.edit} warehouses={warehouses} onClose={() => setModal(null)}
          onSave={input => editBatch(modal.edit.id, input).then(() => undefined)} />
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

function ProductStockRow({ token, productId, productName, warehouses, onSummary, onViewHistory, onViewBatchHistory, onViewProductHistory }: {
  token: string; productId: string; productName: string; warehouses: { id: string; name: string }[]
  onSummary: (productId: string, summary: ProductStockSummary) => void
  onViewHistory: (sku: SKU) => void
  onViewBatchHistory: (sku: SKU, batch: Batch) => void
  onViewProductHistory: (productName: string, skus: SKU[]) => void
}) {
  const [expanded, setExpanded] = useState(false)
  const { detail, loading, refresh } = useProductDetail(token, productId)
  const skus = (detail?.variants ?? []).flatMap(v => v.skus ?? [])
  const skusKey = skus.map(s => s.id).join(',')

  const [stockBySKU, setStockBySKU] = useState<Record<string, number>>({})
  const [stockLoading, setStockLoading] = useState(true)

  useEffect(() => {
    if (!token || skus.length === 0) {
      setStockBySKU({})
      setStockLoading(false)
      return
    }
    let cancelled = false
    setStockLoading(true)
    Promise.all(skus.map(sku => inventoryApi.getSKUStockLevel(token, sku.id).then(l => [sku.id, l.total] as const).catch(() => [sku.id, 0] as const)))
      .then(pairs => { if (!cancelled) setStockBySKU(Object.fromEntries(pairs)) })
      .finally(() => { if (!cancelled) setStockLoading(false) })
    return () => { cancelled = true }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- skusKey is the stable dependency; skus itself is a new array every render
  }, [token, skusKey])

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
      <tr className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors cursor-pointer" onClick={() => setExpanded(v => !v)}>
        <td className="px-4 py-2.5 w-8">
          {expanded ? <ChevronDown size={13} className="dark:text-gray-500 text-gray-400" /> : <ChevronRight size={13} className="dark:text-gray-500 text-gray-400" />}
        </td>
        <td className="px-3 py-2.5 font-medium dark:text-gray-100 text-gray-900">{productName}</td>
        <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
          {singleSKU ? (
            <span className="text-xs dark:text-gray-300 text-gray-700">{formatPrice(singleSKU.price_minor_units, singleSKU.currency_code)}</span>
          ) : priceRange ? (
            <span className="text-xs dark:text-gray-300 text-gray-700">
              {priceRange[0] === priceRange[1]
                ? formatPrice(priceRange[0], skus[0].currency_code)
                : `${formatPrice(priceRange[0], skus[0].currency_code)} – ${formatPrice(priceRange[1], skus[0].currency_code)}`}
            </span>
          ) : (
            <span className="text-xs dark:text-gray-600 text-gray-300">—</span>
          )}
        </td>
        <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
          {singleSKU && singleSKU.sale_price_minor_units != null ? (
            <span className="flex items-center gap-1">
              <span className="text-xs font-medium text-green-400">{formatPrice(singleSKU.sale_price_minor_units, singleSKU.currency_code)}</span>
              {singleSKU.sale_end_date && (
                <span className="text-[9px] dark:text-gray-500 text-gray-400">until {formatDate(singleSKU.sale_end_date)}</span>
              )}
            </span>
          ) : (
            <span className="text-xs dark:text-gray-600 text-gray-300">—</span>
          )}
        </td>
        <td className="px-3 py-2.5">
          {stockLoading ? (
            <span className="text-xs dark:text-gray-600 text-gray-400">…</span>
          ) : (
            <span className={clsx(
              'text-xs font-medium tabular-nums',
              totalStock <= 0 ? 'text-red-400' : totalStock <= LOW_STOCK_THRESHOLD ? 'text-amber-400' : 'dark:text-gray-300 text-gray-700',
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
            className="flex items-center gap-1 text-[10px] dark:text-gray-500 text-gray-400 hover:text-blue-400"
          >
            <History size={11} /> History
          </button>
        </td>
      </tr>
      {expanded && (
        <tr>
          <td colSpan={6} className="p-0">
            {loading && <p className="pl-8 py-3 text-xs dark:text-gray-500 text-gray-400">Loading SKUs…</p>}
            {!loading && skus.length === 0 && <p className="pl-8 py-3 text-xs dark:text-gray-500 text-gray-400">No SKUs on this product yet.</p>}
            {!loading && skus.map(sku => (
              <SKUStockPanel key={sku.id} token={token} sku={sku} warehouses={warehouses} onSKUUpdated={refresh} onViewHistory={onViewHistory} onViewBatchHistory={onViewBatchHistory} />
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
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Inventory
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">Movement History</h1>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">{productName} — every SKU, running balance</p>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-red-400">{error}</p>}
        {!error && rows === null && <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
        {!error && rows !== null && rows.length === 0 && (
          <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">No stock movements recorded for this product yet.</p>
        )}
        {!error && rows !== null && rows.length > 0 && (
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
              <tr>
                {['Date', 'SKU', 'Type', 'Warehouse', 'Batch', 'Quantity', 'Balance', 'Reason'].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
              {rows.map(m => (
                <tr key={m.id} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500 whitespace-nowrap">{formatDateTime(m.occurred_at)}</td>
                  <td className="px-4 py-2.5 font-mono dark:text-gray-300 text-gray-700">{m.skuCode}</td>
                  <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800">{MOVEMENT_TYPE_LABEL[m.type]}</td>
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{warehouseName(m.warehouse_id)}</td>
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{batchLabel(m.batch_id)}</td>
                  <td className={clsx('px-4 py-2.5 font-medium tabular-nums', m.quantity_delta >= 0 ? 'text-green-400' : 'text-red-400')}>
                    {m.quantity_delta >= 0 ? '+' : ''}{m.quantity_delta}
                  </td>
                  <td className="px-4 py-2.5 font-medium tabular-nums dark:text-gray-200 text-gray-800">{m.balance}</td>
                  <td className="px-4 py-2.5 dark:text-gray-500 text-gray-400">{m.reason || '—'}</td>
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
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Inventory
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">Movement History</h1>
          <p className="text-xs font-mono dark:text-gray-500 text-gray-400 mt-0.5">{sku.sku_code}</p>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-red-400">{error}</p>}
        {!error && movements === null && <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
        {!error && movements !== null && movements.length === 0 && (
          <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">No stock movements recorded for this SKU yet.</p>
        )}
        {!error && movements !== null && movements.length > 0 && (
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
              <tr>
                {['Date', 'Type', 'Warehouse', 'Batch', 'Quantity', 'Reason'].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
              {[...movements].reverse().map(m => (
                <tr key={m.id} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500 whitespace-nowrap">{formatDateTime(m.occurred_at)}</td>
                  <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800">{MOVEMENT_TYPE_LABEL[m.type]}</td>
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{warehouseName(m.warehouse_id)}</td>
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{batchLabel(m.batch_id)}</td>
                  <td className={clsx('px-4 py-2.5 font-medium tabular-nums', m.quantity_delta >= 0 ? 'text-green-400' : 'text-red-400')}>
                    {m.quantity_delta >= 0 ? '+' : ''}{m.quantity_delta}
                  </td>
                  <td className="px-4 py-2.5 dark:text-gray-500 text-gray-400">{m.reason || '—'}</td>
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
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Inventory
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">Batch Movement History</h1>
          <p className="text-xs font-mono dark:text-gray-500 text-gray-400 mt-0.5">{sku.sku_code} · Lot {batch.lot_number || '—'}</p>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {error && <p className="px-6 py-4 text-xs text-red-400">{error}</p>}
        {!error && movements === null && <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
        {!error && movements !== null && movements.length === 0 && (
          <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">No stock movements recorded for this batch yet.</p>
        )}
        {!error && movements !== null && movements.length > 0 && (
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
              <tr>
                {['Date', 'Type', 'Warehouse', 'Quantity', 'Reason'].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
              {[...movements].reverse().map(m => (
                <tr key={m.id} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500 whitespace-nowrap">{formatDateTime(m.occurred_at)}</td>
                  <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800">{MOVEMENT_TYPE_LABEL[m.type]}</td>
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{warehouseName(m.warehouse_id)}</td>
                  <td className={clsx('px-4 py-2.5 font-medium tabular-nums', m.quantity_delta >= 0 ? 'text-green-400' : 'text-red-400')}>
                    {m.quantity_delta >= 0 ? '+' : ''}{m.quantity_delta}
                  </td>
                  <td className="px-4 py-2.5 dark:text-gray-500 text-gray-400">{m.reason || '—'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

function ExpiredBatchReviewPage({ onClose, batches, loading, error, writeOff, extend, dismiss }: {
  onClose: () => void
} & ReturnType<typeof useExpiredBatchReview>) {
  const [rowError, setRowError] = useState('')
  const [extendingBatchId, setExtendingBatchId] = useState<string | null>(null)
  const [newExpiryDate, setNewExpiryDate] = useState('')
  const [actingBatchId, setActingBatchId] = useState<string | null>(null)

  const openExtend = (batchId: string) => {
    setRowError('')
    setNewExpiryDate('')
    setExtendingBatchId(batchId)
  }

  const handleWriteOff = async (batchId: string, skuCode: string) => {
    if (!window.confirm(`Write off remaining stock for ${skuCode}? This records a Lost movement and cannot be undone.`)) return
    setRowError('')
    setActingBatchId(batchId)
    try {
      await writeOff(batchId)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to write off batch')
    } finally {
      setActingBatchId(null)
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

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center justify-between px-6 py-4 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-[11px] font-medium dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Inventory
          </button>
          <h1 className="text-xl font-semibold dark:text-gray-100 text-gray-900 leading-tight">Expired Batches</h1>
          <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Batches pending review — write off, extend, or dismiss</p>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        {(error || rowError) && <p className="px-6 py-4 text-xs text-red-400">{error || rowError}</p>}
        {!error && loading && batches.length === 0 && <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
        {!error && !loading && batches.length === 0 && (
          <p className="px-6 py-4 text-xs dark:text-gray-500 text-gray-400">No expired batches pending review.</p>
        )}
        {batches.length > 0 && (
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
              <tr>
                {['SKU', 'Product', 'Warehouse', 'Lot', 'Expired', 'Qty', ''].map(h => (
                  <th key={h} className="px-4 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
              {batches.map(b => (
                <tr key={b.batch_id} className="hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-2.5 font-mono dark:text-gray-300 text-gray-700">{b.sku_code}</td>
                  <td className="px-4 py-2.5 dark:text-gray-200 text-gray-800">{b.product_name}</td>
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{b.warehouse_name}</td>
                  <td className="px-4 py-2.5 dark:text-gray-400 text-gray-500">{b.lot_number || '—'}</td>
                  <td className="px-4 py-2.5 text-red-400">{formatDate(b.expiry_date)}</td>
                  <td className="px-4 py-2.5 tabular-nums dark:text-gray-300 text-gray-700">{b.quantity}</td>
                  <td className="px-4 py-2.5">
                    {extendingBatchId === b.batch_id ? (
                      <div className="flex items-center gap-1.5">
                        <input
                          type="date"
                          aria-label="New expiry date"
                          value={newExpiryDate}
                          onChange={e => setNewExpiryDate(e.target.value)}
                          className="px-2 py-1 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500"
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
                          className="px-2 py-1 rounded-lg dark:bg-[#111827] bg-gray-100 dark:text-gray-300 text-gray-600 text-[11px] font-medium hover:dark:bg-[#2D3748] hover:bg-gray-200"
                        >
                          Cancel
                        </button>
                      </div>
                    ) : (
                      <div className="flex items-center gap-1.5">
                        <button
                          onClick={() => handleWriteOff(b.batch_id, b.sku_code)}
                          disabled={actingBatchId === b.batch_id}
                          className="px-2 py-1 rounded-lg dark:bg-red-500/10 bg-red-50 text-red-400 text-[11px] font-medium hover:dark:bg-red-500/20 hover:bg-red-100 disabled:opacity-50"
                        >
                          Write off
                        </button>
                        <button
                          onClick={() => openExtend(b.batch_id)}
                          disabled={actingBatchId === b.batch_id}
                          className="px-2 py-1 rounded-lg dark:bg-[#111827] bg-gray-100 dark:text-gray-300 text-gray-600 text-[11px] font-medium hover:dark:bg-[#2D3748] hover:bg-gray-200 disabled:opacity-50"
                        >
                          Extend
                        </button>
                        <button
                          onClick={() => handleDismiss(b.batch_id)}
                          disabled={actingBatchId === b.batch_id}
                          className="px-2 py-1 rounded-lg dark:bg-[#111827] bg-gray-100 dark:text-gray-300 text-gray-600 text-[11px] font-medium hover:dark:bg-[#2D3748] hover:bg-gray-200 disabled:opacity-50"
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
      </div>
    </div>
  )
}

// ── Page ──────────────────────────────────────────────────────────────────────

export default function Inventory() {
  const { token } = useAuth()
  const [searchInput, setSearchInput] = useState('')
  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [stockFilter, setStockFilter] = useState<StockFilter>('all')
  const [summaryByProduct, setSummaryByProduct] = useState<Record<string, ProductStockSummary>>({})
  const [historySKU, setHistorySKU] = useState<SKU | null>(null)
  const [historyBatch, setHistoryBatch] = useState<{ sku: SKU; batch: Batch } | null>(null)
  const [historyProduct, setHistoryProduct] = useState<{ name: string; skus: SKU[] } | null>(null)
  const [showExpiredReview, setShowExpiredReview] = useState(false)
  const expiredReview = useExpiredBatchReview(token ?? '')

  useEffect(() => {
    const timeout = setTimeout(() => {
      setSearch(searchInput)
      setPage(1)
    }, 300)
    return () => clearTimeout(timeout)
  }, [searchInput])

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

  if (historySKU) {
    return <SKUMovementHistoryPage token={token ?? ''} sku={historySKU} warehouses={warehouses} onClose={() => setHistorySKU(null)} />
  }

  if (historyBatch) {
    return (
      <BatchMovementHistoryPage
        token={token ?? ''} sku={historyBatch.sku} batch={historyBatch.batch} warehouses={warehouses}
        onClose={() => setHistoryBatch(null)}
      />
    )
  }

  if (historyProduct) {
    return (
      <ProductMovementHistoryPage
        token={token ?? ''} productName={historyProduct.name} skus={historyProduct.skus} warehouses={warehouses}
        onClose={() => setHistoryProduct(null)}
      />
    )
  }

  if (showExpiredReview) {
    return <ExpiredBatchReviewPage onClose={() => setShowExpiredReview(false)} {...expiredReview} />
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            value={searchInput}
            onChange={e => setSearchInput(e.target.value)}
            placeholder="Search product..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
        <div className="flex items-center rounded-lg border dark:border-[#2D3748] border-gray-200 overflow-hidden shrink-0">
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
                  : 'dark:bg-[#1F2937] bg-white dark:text-gray-400 text-gray-500 hover:dark:bg-[#2D3748] hover:bg-gray-100',
              )}
            >
              {label}
            </button>
          ))}
        </div>
        {expiredReview.batches.length > 0 && (
          <button
            onClick={() => setShowExpiredReview(true)}
            className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg dark:bg-red-500/10 bg-red-50 text-red-400 text-[11px] font-medium hover:dark:bg-red-500/20 hover:bg-red-100 transition-colors shrink-0"
          >
            <AlertTriangle size={12} /> Expired batches ({expiredReview.batches.length})
          </button>
        )}
        <span className="text-xs dark:text-gray-500 text-gray-400 ml-auto">{total} product{total === 1 ? '' : 's'}</span>
      </div>

      {error && (
        <div className="flex items-center gap-1.5 px-5 py-2 text-xs text-red-400">
          <AlertTriangle size={12} /> {error}
        </div>
      )}

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-2 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Product</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Price</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Sale Price</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Stock</th>
              <th className="px-3 py-2.5 w-8" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={6} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && visibleProducts.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No products found.</td></tr>
            )}
            {!loading && visibleProducts.map(p => (
              <ProductStockRow
                key={p.id} token={token ?? ''} productId={p.id} productName={p.name} warehouses={warehouses}
                onSummary={handleSummary} onViewHistory={setHistorySKU}
                onViewBatchHistory={(sku, batch) => setHistoryBatch({ sku, batch })}
                onViewProductHistory={(name, skus) => setHistoryProduct({ name, skus })}
              />
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
    </div>
  )
}
