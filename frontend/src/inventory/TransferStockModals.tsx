import { useState } from 'react'
import type { Batch } from './types'
import { Modal, inputClass, labelClass } from './components/Modal'
import { BatchScope, LockedField } from './BatchStockModals'

// ── Transfer (also covers general-ledger <-> batch moves) ───────────────────

export function TransferModal({
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
    <Modal title="Transfer stock" onClose={onClose}>
      <div className="space-y-3">
        <p className="text-2xs text-ink-4">
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
        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            Cancel
          </button>
          <button type="button" onClick={submit} disabled={saving} className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50">
            {saving ? 'Saving…' : 'Transfer'}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Split case into units (see pms.SKU.case_of_sku_id) ──────────────────────

export function SplitCaseModal({
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
    <Modal title="Split case into units" onClose={onClose}>
      <div className="space-y-3">
        <p className="text-2xs text-ink-4">
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
        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            Cancel
          </button>
          <button type="button" onClick={submit} disabled={saving} className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50">
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

export function DamagedLostModal({
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
    <Modal title="Mark damaged / lost" onClose={onClose}>
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
        {error && <p className="text-2xs text-danger">{error}</p>}
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
