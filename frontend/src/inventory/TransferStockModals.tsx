import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Batch } from './types'
import { Modal, inputClass, labelClass } from './components/Modal'
import { BatchScope, LockedField } from './BatchStockModals'
import { stockUnitFor, type StockUnit } from './stockUnit'

// ── Transfer (also covers general-ledger <-> batch moves) ───────────────────

export function TransferModal({
  warehouses, batches, scope, unit = stockUnitFor(), onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  scope?: BatchScope
  unit?: StockUnit
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
  const { t } = useTranslation(['inventory', 'common'])
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    const n = unit.parse(quantity)
    if (n === null || n <= 0) {
      setError(t('transfer.enterPositive'))
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
      setError(e instanceof Error ? e.message : t('transfer.failed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={t('transfer.title')} onClose={onClose}>
      <div className="space-y-3">
        <p className="text-2xs text-ink-4">
          {t('transfer.hint')}
        </p>
        <div className="grid grid-cols-2 gap-2">
          {scope ? (
            <LockedField label={t('transfer.fromWarehouse')} value={warehouses.find(w => w.id === scope.warehouseId)?.name ?? scope.warehouseId} />
          ) : (
            <div>
              <label className={labelClass} htmlFor="transfer-from-wh">{t('transfer.fromWarehouse')}</label>
              <select id="transfer-from-wh" value={fromWarehouse} onChange={e => setFromWarehouse(e.target.value)} className={inputClass}>
                {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
              </select>
            </div>
          )}
          <div>
            <label className={labelClass} htmlFor="transfer-to-wh">{t('transfer.toWarehouse')}</label>
            <select id="transfer-to-wh" value={toWarehouse} onChange={e => setToWarehouse(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          {scope ? (
            <LockedField label={t('transfer.fromBatch')} value={scope.label} />
          ) : (
            <div>
              <label className={labelClass} htmlFor="transfer-from-batch">{t('transfer.fromBatch')}</label>
              <select id="transfer-from-batch" value={fromBatch} onChange={e => setFromBatch(e.target.value)} className={inputClass}>
                <option value="">{t('transfer.generalLedger')}</option>
                {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
              </select>
            </div>
          )}
          <div>
            <label className={labelClass} htmlFor="transfer-to-batch">{t('transfer.toBatch')}</label>
            <select id="transfer-to-batch" value={toBatch} onChange={e => setToBatch(e.target.value)} className={inputClass}>
              <option value="">{t('transfer.generalLedger')}</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
        </div>
        <div>
          <label className={labelClass} htmlFor="transfer-qty">{t('modal.quantity')}{unit.kind === 'kg' && <> {t('modal.inKg')}</>}</label>
          <input id="transfer-qty" type="number" min={0} step={unit.step} inputMode="decimal" value={quantity} onChange={e => setQuantity(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="transfer-reason">{t('modal.reasonOptional')}</label>
          <input id="transfer-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            {t('common:actions.cancel')}
          </button>
          <button type="button" onClick={submit} disabled={saving} className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50">
            {saving ? t('common:actions.saving') : t('transfer.submit')}
          </button>
        </div>
      </div>
    </Modal>
  )
}

// ── Split case into units (see pms.SKU.case_of_sku_id) ──────────────────────

export function SplitCaseModal({
  warehouses, batches, scope, onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  scope?: BatchScope
  onSave: (input: { warehouse_id: string; batch_id?: string | null; cases_to_split: number; reason?: string }) => Promise<unknown>
  onClose: () => void
}) {
  const [warehouseId, setWarehouseId] = useState(scope?.warehouseId ?? warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState(scope?.batchId ?? '')
  const [casesToSplit, setCasesToSplit] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const { t } = useTranslation(['inventory', 'common'])
  const [error, setError] = useState<string | null>(null)
  // A batch lives in exactly one warehouse -- only offer that warehouse's
  // lots, and drop a picked lot when the warehouse changes under it.
  const warehouseBatches = batches.filter(b => !b.is_general_ledger && b.warehouse_id === warehouseId)
  const changeWarehouse = (id: string) => {
    setWarehouseId(id)
    setBatchId('')
  }

  const submit = async () => {
    const n = parseInt(casesToSplit, 10)
    if (Number.isNaN(n) || n <= 0) {
      setError(t('split.enterPositive'))
      return
    }
    setSaving(true)
    setError(null)
    try {
      await onSave({ warehouse_id: warehouseId, batch_id: batchId || null, cases_to_split: n, reason })
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('split.failed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={t('split.title')} onClose={onClose}>
      <div className="space-y-3">
        <p className="text-2xs text-ink-4">
          {t('split.hint')}
        </p>
        <div className="grid grid-cols-2 gap-2">
          {scope ? (
            <LockedField label={t('modal.warehouse')} value={warehouses.find(w => w.id === scope.warehouseId)?.name ?? scope.warehouseId} />
          ) : (
            <div>
              <label className={labelClass} htmlFor="split-warehouse">{t('modal.warehouse')}</label>
              <select id="split-warehouse" value={warehouseId} onChange={e => changeWarehouse(e.target.value)} className={inputClass}>
                {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
              </select>
            </div>
          )}
          {scope ? (
            <LockedField label={t('modal.batch')} value={scope.label} />
          ) : (
            <div>
              <label className={labelClass} htmlFor="split-batch">{t('modal.batch')}</label>
              <select id="split-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
                <option value="">{t('transfer.generalLedger')}</option>
                {warehouseBatches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
              </select>
            </div>
          )}
        </div>
        <div>
          <label className={labelClass} htmlFor="split-cases">{t('split.casesToSplit')}</label>
          <input id="split-cases" type="number" min={1} value={casesToSplit} onChange={e => setCasesToSplit(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="split-reason">{t('modal.reasonOptional')}</label>
          <input id="split-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            {t('common:actions.cancel')}
          </button>
          <button type="button" onClick={submit} disabled={saving} className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50">
            {saving ? t('split.splitting') : t('split.submit')}
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
  warehouses, batches, scope, unit = stockUnitFor(), onSave, onClose,
}: {
  warehouses: { id: string; name: string }[]
  batches: Batch[]
  scope?: BatchScope
  unit?: StockUnit
  onSave: (input: { warehouse_id: string; batch_id?: string | null; quantity: number; type: 'damaged' | 'lost'; reason?: string }) => Promise<void>
  onClose: () => void
}) {
  const [type, setType] = useState<'damaged' | 'lost'>('damaged')
  const [warehouseId, setWarehouseId] = useState(scope?.warehouseId ?? warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState(scope?.batchId ?? '')
  const [quantity, setQuantity] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const { t } = useTranslation(['inventory', 'common'])
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    const n = unit.parse(quantity)
    if (n === null || n <= 0) {
      setError(t('transfer.enterPositive'))
      return
    }
    setSaving(true)
    setError(null)
    try {
      await onSave({ warehouse_id: warehouseId, batch_id: batchId || null, quantity: n, type, reason })
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : t('writeOff.failed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={t('writeOff.title')} onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className={labelClass} htmlFor="dl-type">{t('modal.type')}</label>
          <select id="dl-type" value={type} onChange={e => setType(e.target.value as typeof type)} className={inputClass}>
            <option value="damaged">{t('writeOff.damaged')}</option>
            <option value="lost">{t('writeOff.lost')}</option>
          </select>
        </div>
        {scope ? (
          <LockedField label={t('modal.warehouse')} value={warehouses.find(w => w.id === scope.warehouseId)?.name ?? scope.warehouseId} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="dl-warehouse">{t('modal.warehouse')}</label>
            <select id="dl-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
        )}
        {scope ? (
          <LockedField label={t('modal.batch')} value={scope.label} />
        ) : (
          <div>
            <label className={labelClass} htmlFor="dl-batch">{t('modal.batchOptional')}</label>
            <select id="dl-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
              <option value="">{t('modal.generalLedgerOption')}</option>
              {batches.map(b => <option key={b.id} value={b.id}>{b.lot_number || b.id}</option>)}
            </select>
          </div>
        )}
        <div>
          <label className={labelClass} htmlFor="dl-qty">{t('modal.quantity')}{unit.kind === 'kg' && <> {t('modal.inKg')}</>}</label>
          <input id="dl-qty" type="number" min={0} step={unit.step} inputMode="decimal" value={quantity} onChange={e => setQuantity(e.target.value)} className={inputClass} />
        </div>
        <div>
          <label className={labelClass} htmlFor="dl-reason">{t('modal.reasonOptional')}</label>
          <input id="dl-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            {t('common:actions.cancel')}
          </button>
          <button type="button" onClick={submit} disabled={saving || !warehouseId} className="px-3 py-1.5 text-xs rounded-lg bg-red-600 text-white hover:bg-red-700 disabled:opacity-50">
            {saving ? t('common:actions.saving') : t('writeOff.submit')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
