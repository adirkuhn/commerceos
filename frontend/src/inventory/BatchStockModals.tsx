import { useState } from 'react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { Batch, MovementType } from './types'
import { Modal, inputClass, labelClass } from './components/Modal'
import { stockUnitFor, type StockUnit } from './stockUnit'
import { apiErrorText } from '../api/apiErrorText'

// Modal/inputClass/labelClass live in ../inventory/components/Modal (see
// import above) -- shared by every stock-action modal in this file.

// BatchScope pins a modal opened by clicking a batch row to that exact
// row -- warehouseId/batchId feed straight into the mutation, label is
// just for display. batchId '' means the general-ledger row (same sentinel
// AdjustStockModal/RecountModal/etc already use for "no batch"). Only
// Adjust/Recount/Damaged-Lost, and Transfer's FROM side, ever get locked
// this way -- Transfer's TO side stays a free pick even from a row click.
export interface BatchScope {
  warehouseId: string
  batchId: string
  label: string
}

// LockedField replaces a warehouse/batch <select> with its resolved value
// when a modal was opened by clicking a specific batch row -- the whole
// point of that entry point is "this exact row", so it isn't re-editable.
export function LockedField({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <span className={labelClass}>{label}</span>
      <div className={clsx(inputClass, 'opacity-70 cursor-not-allowed')}>{value}</div>
    </div>
  )
}

// ── Batch form (create/edit) ─────────────────────────────────────────────────

export function BatchFormModal({
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
  const { t } = useTranslation(['inventory', 'common'])
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    setSaving(true)
    setError(null)
    try {
      await onSave({ warehouse_id: warehouseId, lot_number: lotNumber, expiry_date: expiryDate || null })
      onClose()
    } catch (e) {
      setError(apiErrorText(e) ?? t('modal.saveBatchFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={batch ? t('modal.editBatch') : t('modal.newBatch')} onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className={labelClass} htmlFor="batch-warehouse">{t('modal.warehouse')}</label>
          <select id="batch-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
            {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
          </select>
        </div>
        <div>
          <label className={labelClass} htmlFor="batch-lot">{t('modal.lotNumber')}</label>
          <input id="batch-lot" value={lotNumber} onChange={e => setLotNumber(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="batch-expiry">{t('modal.expiryOptional')}</label>
          <input id="batch-expiry" type="date" value={expiryDate ?? ''} onChange={e => setExpiryDate(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            {t('common:actions.cancel')}
          </button>
          <button type="button" onClick={submit} disabled={saving || !warehouseId} className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50">
            {saving ? t('common:actions.saving') : t('common:actions.save')}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Adjust stock ──────────────────────────────────────────────────────────────

export function AdjustStockModal({
  warehouses, batches, scope, unit = stockUnitFor(), onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  scope?: BatchScope
  // How quantities are typed for this SKU (kg for a weight pool).
  unit?: StockUnit
  onSave: (input: { warehouse_id: string; batch_id?: string | null; type: MovementType; quantity_delta: number; reason?: string }) => Promise<void>
  onClose: () => void
}) {
  const [type, setType] = useState<'receipt' | 'sale' | 'adjustment'>('receipt')
  const [warehouseId, setWarehouseId] = useState(scope?.warehouseId ?? warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState(scope?.batchId ?? '')
  const [quantity, setQuantity] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const { t } = useTranslation(['inventory', 'common'])
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    const n = unit.parse(quantity)
    if (n === null || n === 0) {
      setError(t('modal.enterNonZero'))
      return
    }
    setSaving(true)
    setError(null)
    try {
      const delta = type === 'sale' ? -Math.abs(n) : type === 'receipt' ? Math.abs(n) : n
      await onSave({ warehouse_id: warehouseId, batch_id: batchId || null, type, quantity_delta: delta, reason })
      onClose()
    } catch (e) {
      setError(apiErrorText(e) ?? t('modal.movementFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={t('modal.adjustTitle')} onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className={labelClass} htmlFor="adjust-type">{t('modal.type')}</label>
          <select id="adjust-type" value={type} onChange={e => setType(e.target.value as typeof type)} className={inputClass}>
            <option value="receipt">{t('modal.typeReceipt')}</option>
            <option value="sale">{t('modal.typeSale')}</option>
            <option value="adjustment">{t('modal.typeAdjustment')}</option>
          </select>
        </div>
        {scope ? (
          <LockedField label={t('modal.warehouse')} value={warehouses.find(w => w.id === scope.warehouseId)?.name ?? scope.warehouseId} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="adjust-warehouse">{t('modal.warehouse')}</label>
            <select id="adjust-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
        )}
        {scope ? (
          <LockedField label={t('modal.batch')} value={scope.label} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="adjust-batch">{t('modal.batchOptional')}</label>
            <select id="adjust-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
              <option value="">{t('modal.generalLedgerOption')}</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
        )}
        <div>
          <label className={labelClass} htmlFor="adjust-qty">
            {type === 'adjustment' ? t('modal.quantityAdjustment') : t('modal.quantity')}{unit.kind === 'kg' && <> {t('modal.inKg')}</>}
          </label>
          <input id="adjust-qty" type="number" step={unit.step} inputMode="decimal" value={quantity} onChange={e => setQuantity(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="adjust-reason">{t('modal.reasonOptional')}</label>
          <input id="adjust-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            {t('common:actions.cancel')}
          </button>
          <button type="button" onClick={submit} disabled={saving || !warehouseId} className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50">
            {saving ? t('common:actions.saving') : t('modal.record')}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Recount ───────────────────────────────────────────────────────────────────

export function RecountModal({
  warehouses, batches, scope, unit = stockUnitFor(), onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  scope?: BatchScope
  unit?: StockUnit
  onSave: (input: { warehouse_id: string; batch_id?: string | null; counted_quantity: number; reason?: string }) => Promise<void>
  onClose: () => void
}) {
  const [warehouseId, setWarehouseId] = useState(scope?.warehouseId ?? warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState(scope?.batchId ?? '')
  const [counted, setCounted] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const { t } = useTranslation(['inventory', 'common'])
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    const n = unit.parse(counted)
    if (n === null || n < 0) {
      setError(t('modal.enterCounted'))
      return
    }
    setSaving(true)
    setError(null)
    try {
      await onSave({ warehouse_id: warehouseId, batch_id: batchId || null, counted_quantity: n, reason })
      onClose()
    } catch (e) {
      setError(apiErrorText(e) ?? t('modal.recountFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={t('modal.recountTitle')} onClose={onClose}>
      <div className="space-y-3">
        {scope ? (
          <LockedField label={t('modal.warehouse')} value={warehouses.find(w => w.id === scope.warehouseId)?.name ?? scope.warehouseId} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="recount-warehouse">{t('modal.warehouse')}</label>
            <select id="recount-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
        )}
        {scope ? (
          <LockedField label={t('modal.batch')} value={scope.label} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="recount-batch">{t('modal.batchOptional')}</label>
            <select id="recount-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
              <option value="">{t('modal.generalLedgerOption')}</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
        )}
        <div>
          <label className={labelClass} htmlFor="recount-qty">{t('modal.countedQuantity')}{unit.kind === 'kg' && <> {t('modal.inKg')}</>}</label>
          <input id="recount-qty" type="number" min={0} step={unit.step} inputMode="decimal" value={counted} onChange={e => setCounted(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="recount-reason">{t('modal.reasonOptional')}</label>
          <input id="recount-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            {t('common:actions.cancel')}
          </button>
          <button type="button" onClick={submit} disabled={saving || !warehouseId} className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50">
            {saving ? t('common:actions.saving') : t('modal.record')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
