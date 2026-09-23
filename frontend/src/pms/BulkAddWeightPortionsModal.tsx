import { useState, useEffect } from 'react'
import { Modal, inputClass, labelClass } from '../inventory/components/Modal'
import { useWarehouses } from '../inventory/hooks/useWarehouses'
import * as inventoryApi from '../inventory/api'
import * as pmsApi from '../pms/api'
import type { Batch } from '../inventory/types'
import type { CreatedPortionedSKU } from './types'

function parseWeights(raw: string): { weights: number[]; error: string } {
  const parts = raw.split(/[,\n]/).map(s => s.trim()).filter(s => s !== '')
  if (parts.length === 0) return { weights: [], error: '' }
  const weights: number[] = []
  for (const part of parts) {
    const n = Number(part)
    if (!Number.isFinite(n) || !Number.isInteger(n) || n <= 0) {
      return { weights: [], error: `"${part}" isn't a positive whole number of grams` }
    }
    weights.push(n)
  }
  return { weights, error: '' }
}

// Computes the same "round-half-up to the nearest minor unit" preview the
// backend authoritatively applies (see backend's lineTotalForScannedWeight)
// -- display only, the created SKU's real price always comes from the
// server response.
function previewPriceMinorUnits(pricePerKiloMinorUnits: number, weightGrams: number): number {
  return Math.floor((pricePerKiloMinorUnits * weightGrams + 500) / 1000)
}

function formatPrice(minorUnits: number, currency: string): string {
  return `${currency} ${(minorUnits / 100).toFixed(2)}`
}

// "Type a list of weights, get N sellable items" -- bulk-creates portioned
// SKUs off this weight-pool bulk SKU, one per entered weight, replacing the
// previous one-by-one create-SKU-then-portion-from-bulk round trip. See
// application/pms.BulkAddWeightPortionsUseCase's own doc comment for the
// full mechanics (barcode generation, price calc, batch/expiry linking).
export function BulkAddWeightPortionsModal({
  token, bulkSkuId, pricePerKiloMinorUnits, currencyCode, onCreated, onClose,
}: {
  token: string
  bulkSkuId: string
  pricePerKiloMinorUnits: number
  currencyCode: string
  onCreated: (created: CreatedPortionedSKU[]) => void
  onClose: () => void
}) {
  const { warehouses } = useWarehouses(token)
  const [batches, setBatches] = useState<Batch[]>([])
  const [warehouseId, setWarehouseId] = useState('')
  const [batchId, setBatchId] = useState('')
  const [weightsRaw, setWeightsRaw] = useState('')
  const [skuCodePrefix, setSkuCodePrefix] = useState('')
  const [reason, setReason] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (warehouses[0] && !warehouseId) setWarehouseId(warehouses[0].id)
  }, [warehouses, warehouseId])

  useEffect(() => {
    inventoryApi.listBatches(token, bulkSkuId).then(setBatches).catch(() => {})
  }, [token, bulkSkuId])

  const { weights, error: parseError } = parseWeights(weightsRaw)

  async function submit() {
    if (weights.length === 0 || !warehouseId) return
    setSaving(true)
    setError('')
    try {
      const result = await pmsApi.bulkAddWeightPortions(token, bulkSkuId, {
        warehouse_id: warehouseId, bulk_batch_id: batchId || null, weights_grams: weights,
        sku_code_prefix: skuCodePrefix || undefined, reason: reason || undefined,
      })
      onCreated(result.created)
      onClose()
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create packages')
    } finally {
      setSaving(false)
    }
  }

  return (
    <Modal title="Bulk add by weight" onClose={onClose}>
      <div className="space-y-3">
        <p className="text-2xs text-ink-4">
          Type or paste each physical package's weight (grams), one per line or comma-separated. Each becomes its
          own sellable SKU, auto-priced from this pool's price per kilo and auto-barcoded to match your scale.
        </p>
        <div className="grid grid-cols-2 gap-2">
          <div>
            <label className={labelClass} htmlFor="bulk-portion-warehouse">Warehouse</label>
            <select id="bulk-portion-warehouse" value={warehouseId} onChange={e => setWarehouseId(e.target.value)} className={inputClass}>
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          <div>
            <label className={labelClass} htmlFor="bulk-portion-batch">Bulk lot (optional)</label>
            <select id="bulk-portion-batch" value={batchId} onChange={e => setBatchId(e.target.value)} className={inputClass}>
              <option value="">General ledger</option>
              {batches.filter(b => !b.is_general_ledger).map(b => (
                <option key={b.id} value={b.id}>
                  {b.lot_number || b.id}{b.expiry_date ? ` — exp ${new Date(b.expiry_date).toLocaleDateString()}` : ''}
                </option>
              ))}
            </select>
            <p className="mt-1 text-2xs text-ink-4">
              Picking a lot copies its expiry onto every new package -- synced to WooCommerce automatically.
            </p>
          </div>
        </div>
        <div>
          <label className={labelClass} htmlFor="bulk-portion-weights">Weights (grams)</label>
          <textarea
            id="bulk-portion-weights" rows={4} value={weightsRaw} onChange={e => setWeightsRaw(e.target.value)}
            placeholder={'404\n442\n458'} className={inputClass}
          />
          {parseError && <p className="mt-1 text-2xs text-danger">{parseError}</p>}
        </div>
        <div>
          <label className={labelClass} htmlFor="bulk-portion-prefix">SKU code prefix (optional)</label>
          <input
            id="bulk-portion-prefix" value={skuCodePrefix} onChange={e => setSkuCodePrefix(e.target.value)}
            placeholder="Defaults to this SKU's own code" className={inputClass}
          />
        </div>
        <div>
          <label className={labelClass} htmlFor="bulk-portion-reason">Reason (optional)</label>
          <input id="bulk-portion-reason" value={reason} onChange={e => setReason(e.target.value)} className={inputClass} />
        </div>
        {weights.length > 0 && !parseError && (
          <div className="rounded-lg border border-line divide-y divide-line max-h-40 overflow-y-auto">
            {weights.map((w, i) => (
              <div key={i} className="flex items-center justify-between px-2 py-1 text-2xs">
                <span className="text-ink-2">{w}g</span>
                <span className="text-ink-3">{formatPrice(previewPriceMinorUnits(pricePerKiloMinorUnits, w), currencyCode)}</span>
              </div>
            ))}
          </div>
        )}
        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            Cancel
          </button>
          <button
            type="button" onClick={submit} disabled={saving || weights.length === 0 || !!parseError || !warehouseId}
            className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50"
          >
            {saving ? 'Creating…' : `Create ${weights.length || ''} item${weights.length === 1 ? '' : 's'}`}
          </button>
        </div>
      </div>
    </Modal>
  )
}
