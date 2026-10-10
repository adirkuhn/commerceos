import { useState, useEffect } from 'react'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { Modal, inputClass, labelClass } from './components/Modal'
import * as inventoryApi from './api'
import type { Batch } from './types'
import type { DateInput } from '../format/date'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'

function formatBatchOption(b: Batch, formatDate: (value: DateInput) => string): string {
  const label = b.lot_number || b.id
  if (!b.expiry_date) return label
  return i18next.t('inventory:portion.batchOption', { lot: label, date: formatDate(b.expiry_date) })
}

// Portion one sellable item off a bulk weight pool (see
// pms.SKU.portioned_from_sku_id). The Batch picker must list the BULK sku's
// own batches (that's whose lot is being drawn down and, via
// TargetBatchID on the backend, whose expiry can be carried onto the new
// item) -- fetched here directly by bulkSkuId rather than threaded through
// the parent's own useSKUStock(token, sku.id), which is scoped to the
// SKU being viewed (the target/portioned item, not the bulk pool) and was
// the source of a real bug: the picker used to silently show the wrong
// SKU's (empty) batch list, so batch-linked portioning never actually
// worked through this UI -- only "General ledger" was ever selectable.
export function PortionFromBulkModal({
  token, bulkSkuId, warehouses, nominalWeightGrams, onSave, onClose,
}: {
  token: string
  bulkSkuId: string
  warehouses: { id: string; name: string }[]
  nominalWeightGrams?: number | null
  onSave: (input: { warehouse_id: string; batch_id?: string | null; weight_grams?: number; reason?: string }) => Promise<unknown>
  onClose: () => void
}) {
  const { formatDate } = useDateFormat()
  const [batches, setBatches] = useState<Batch[]>([])
  const [batchesError, setBatchesError] = useState<string | null>(null)
  const [warehouseId, setWarehouseId] = useState(warehouses[0]?.id ?? '')
  const [batchId, setBatchId] = useState('')
  const [weightGrams, setWeightGrams] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const { t } = useTranslation(['inventory', 'common'])
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    inventoryApi.listBatches(token, bulkSkuId)
      .then(result => { if (!cancelled) setBatches(result) })
      .catch(e => { if (!cancelled) setBatchesError(apiErrorText(e) ?? t('portion.batchesFailed')) })
    return () => { cancelled = true }
  }, [token, bulkSkuId])

  const submit = async () => {
    let weight: number | undefined
    if (weightGrams.trim() !== '') {
      const n = parseInt(weightGrams, 10)
      if (Number.isNaN(n) || n <= 0) {
        setError(t('portion.enterWeight'))
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
      setError(apiErrorText(e) ?? t('portion.failed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title={t('portion.title')} onClose={onClose}>
      <div className="space-y-3">
        <p className="text-2xs text-ink-4">
          {t('portion.hint')}
        </p>
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className={labelClass} htmlFor="portion-warehouse">{t('modal.warehouse')}</label>
            <select id="portion-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          <div>
            <label className={labelClass} htmlFor="portion-batch">{t('portion.batch')}</label>
            <select id="portion-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
              <option value="">{t('transfer.generalLedger')}</option>
              {batches.filter(b => !b.is_general_ledger).map(b => <option key={b.id} value={b.id}>{formatBatchOption(b, formatDate)}</option>)}
            </select>
            {batchesError && <p className="mt-1 text-2xs text-danger">{batchesError}</p>}
          </div>
        </div>
        <div>
          <label className={labelClass} htmlFor="portion-weight">
            {nominalWeightGrams != null ? t('portion.weightDefault', { grams: nominalWeightGrams }) : t('portion.weight')}
          </label>
          <input
            id="portion-weight" type="number" min={1} value={weightGrams} onChange={e => setWeightGrams(e.target.value)}
            placeholder={nominalWeightGrams != null ? String(nominalWeightGrams) : undefined} className={inputClass}
          />
        </div>
        <div>
          <label className={labelClass} htmlFor="portion-reason">{t('modal.reasonOptional')}</label>
          <input id="portion-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            {t('common:actions.cancel')}
          </button>
          <button type="button" onClick={submit} disabled={saving} className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50">
            {saving ? t('portion.adding') : t('portion.submit')}
          </button>
        </div>
      </div>
    </Modal>
  )
}
