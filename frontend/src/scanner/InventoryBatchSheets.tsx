import { useState } from 'react'
import { stockUnitFor, type StockUnit } from '../inventory/stockUnit'
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { QuantityStepper } from './QuantityStepper'
import type { SKU } from '../pms/types'
import type { Warehouse } from '../inventory/types'
import { Dialog } from '../components/common/Dialog'
import { formatMinorUnitsAsInput } from './InventoryFormat'

// Standalone batch creation, distinct from Move's inline "+ New batch": that
// one always immediately transfers stock in from a source row (there must
// be a from-row to move out of). This one is for new stock with no source
// -- e.g. a fresh delivery, or a brand-new SKU with nothing to move from
// yet -- so quantity is optional and, when given, is recorded as a receipt
// straight into the new batch rather than a transfer.
export function AddBatchSheet({ warehouses, unit = stockUnitFor(), error, onCancel, onCreate }: {
  unit?: StockUnit
  warehouses: Warehouse[]
  error: string
  onCancel: () => void
  onCreate: (warehouseId: string, lotNumber: string, expiryDate: string, quantity: number, reason: string) => void
}) {
  const { t } = useTranslation(['scanner', 'common'])
  const [warehouseId, setWarehouseId] = useState(warehouses[0]?.id ?? '')
  const [lotNumber, setLotNumber] = useState('')
  const [expiryDate, setExpiryDate] = useState('')
  const [quantity, setQuantity] = useState('')
  const [reason, setReason] = useState('')

  const disabled = !warehouseId || !lotNumber.trim()

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <h2 className="text-sm font-semibold">{t('inventory.addBatch')}</h2>
          <button aria-label={t('common:actions.close')} onClick={onCancel} title={t('common:actions.close')} className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>
        <div className="p-5 space-y-4">
          {error && <p className="text-sm text-red-400">{error}</p>}
          <div>
            <label htmlFor="add-batch-warehouse" className="block text-xs text-gray-400 mb-1.5">{t('inventory.warehouse')}</label>
            <select
              id="add-batch-warehouse"
              value={warehouseId}
              onChange={e => setWarehouseId(e.target.value)}
              className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
            >
              {warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
            </select>
          </div>
          <div>
            <label htmlFor="add-batch-lot" className="block text-xs text-gray-400 mb-1.5">{t('batch.lotNumber')}</label>
            <input id="add-batch-lot" value={lotNumber} onChange={e => setLotNumber(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus" />
          </div>
          <div>
            <label htmlFor="add-batch-expiry" className="block text-xs text-gray-400 mb-1.5">{t('inventory.expiryOptional')}</label>
            <input id="add-batch-expiry" type="date" value={expiryDate} onChange={e => setExpiryDate(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus" />
          </div>
          <QuantityStepper id="add-batch-qty" label={unit.kind === 'kg' ? `${t('inventory.initialQty')} ${t('shared.inKg')}` : t('inventory.initialQty')} value={quantity} onChange={setQuantity} decimal={unit.kind === 'kg'} />
          <div>
            <label htmlFor="add-batch-reason" className="block text-xs text-gray-400 mb-1.5">{t('shared.reason')}</label>
            <input id="add-batch-reason" value={reason} onChange={e => setReason(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus" />
          </div>
          <button
            onClick={() => onCreate(warehouseId, lotNumber.trim(), expiryDate, unit.parse(quantity) ?? 0, reason)}
            disabled={disabled}
            className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold disabled:opacity-50"
          >
            {t('inventory.create')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}

export function PriceSheet({ sku, error, onCancel, onSave }: {
  sku: SKU
  error: string
  onCancel: () => void
  onSave: (price: string, salePrice: string, saleEndDate: string) => void
}) {
  const { t } = useTranslation(['scanner', 'common'])
  const [price, setPrice] = useState(formatMinorUnitsAsInput(sku.price_minor_units))
  const [salePrice, setSalePrice] = useState(sku.sale_price_minor_units != null ? formatMinorUnitsAsInput(sku.sale_price_minor_units) : '')
  const [saleEndDate, setSaleEndDate] = useState(sku.sale_end_date ?? '')

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <h2 className="text-sm font-semibold">{t('inventory.editPrice')}</h2>
          <button aria-label={t('common:actions.close')} onClick={onCancel} title={t('common:actions.close')} className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>
        <div className="p-5 space-y-4">
          {error && <p className="text-sm text-red-400">{error}</p>}
          <div>
            <label htmlFor="price-sheet-price" className="block text-xs text-gray-400 mb-1.5">{t('inventory.regularPrice', { currency: sku.currency_code })}</label>
            <input id="price-sheet-price" type="number" step="0.01" min="0" value={price} onChange={e => setPrice(e.target.value)} className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus" />
          </div>
          <div>
            <label htmlFor="price-sheet-sale-price" className="block text-xs text-gray-400 mb-1.5">{t('inventory.salePrice')}</label>
            <input
              id="price-sheet-sale-price" type="number" step="0.01" min="0" value={salePrice}
              onChange={e => { setSalePrice(e.target.value); if (e.target.value.trim() === '') setSaleEndDate('') }}
              className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
            />
          </div>
          <div>
            <label htmlFor="price-sheet-sale-end" className="block text-xs text-gray-400 mb-1.5">{t('inventory.saleEnds')}</label>
            <input
              id="price-sheet-sale-end" type="date" value={saleEndDate} onChange={e => setSaleEndDate(e.target.value)}
              disabled={salePrice.trim() === ''}
              className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus disabled:opacity-50"
            />
          </div>
          <button
            onClick={() => onSave(price, salePrice, saleEndDate)}
            className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold"
          >
            {t('inventory.save')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
