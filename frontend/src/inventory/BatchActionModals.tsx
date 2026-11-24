import { useState, useRef, type ReactNode } from 'react'
import { Plus, ArrowRightLeft, PackageMinus, ClipboardList, Boxes, Scale } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { SKU } from '../pms/types'
import { Modal, inputClass, labelClass } from './components/Modal'
import { formatMinorUnitsAsInput } from './inventoryFormat'
import { BatchFormModal, BatchScope } from './BatchStockModals'
import { apiErrorText } from '../api/apiErrorText'

// ── Price / sale price (owned by pms, edited here as a convenience — same
// pattern as ATUM's inventory screen writing back to WooCommerce's own
// product price, not a separate price store) ────────────────────────────────

export function PriceModal({ sku, onSave, onClose }: {
  sku: SKU; onSave: (price: string, salePrice: string, saleEndDate: string) => Promise<void>; onClose: () => void
}) {
  const [price, setPrice] = useState(formatMinorUnitsAsInput(sku.price_minor_units))
  const [salePrice, setSalePrice] = useState(sku.sale_price_minor_units != null ? formatMinorUnitsAsInput(sku.sale_price_minor_units) : '')
  const [saleEndDate, setSaleEndDate] = useState(sku.sale_end_date ?? '')
  const [saving, setSaving] = useState(false)
  const { t } = useTranslation(['inventory', 'common'])
  const [error, setError] = useState<string | null>(null)

  const submit = async () => {
    setSaving(true)
    setError(null)
    try {
      await onSave(price, salePrice, saleEndDate)
      onClose()
    } catch (e) {
      setError(apiErrorText(e) ?? t('price.updateFailed'))
    } finally {
      setSaving(false)
    }
  }

  const priceNum = parseFloat(price || '0')
  const salePriceNum = salePrice.trim() === '' ? null : parseFloat(salePrice)
  const discountPct = salePriceNum != null && priceNum > 0 ? Math.round((1 - salePriceNum / priceNum) * 100) : null

  return (
    <Modal title={t('price.title')} onClose={onClose}>
      <div className="space-y-3">
        <div>
          <label className={labelClass} htmlFor="price-edit-price">{t('price.regular', { currency: sku.currency_code })}</label>
          <input id="price-edit-price" type="number" step="0.01" min="0" value={price} onChange={e => setPrice(e.target.value)} className={inputClass} />
        </div>

        <div className="rounded-lg border border-line p-2.5 space-y-2">
          <div className="flex items-center justify-between">
            <label className={clsx(labelClass, 'mb-0')} htmlFor="price-edit-sale-price">{t('price.sale')}</label>
            {discountPct != null && discountPct > 0 && (
              <span className="text-2xs px-1.5 py-px rounded bg-success/10 text-success font-medium">-{discountPct}%</span>
            )}
            {discountPct != null && discountPct <= 0 && (
              <span className="text-2xs text-danger">{t('price.mustBeBelow')}</span>
            )}
          </div>
          <input id="price-edit-sale-price" type="number" step="0.01" min="0" value={salePrice} onChange={e => {
            setSalePrice(e.target.value)
            if (e.target.value.trim() === '') setSaleEndDate('')
          }} className={inputClass} />

          <div>
            <label className={labelClass} htmlFor="price-edit-sale-end-date">{t('price.saleEnds')}</label>
            <input id="price-edit-sale-end-date" type="date" value={saleEndDate} onChange={e => setSaleEndDate(e.target.value)} className={inputClass} disabled={salePriceNum == null} />
            <p className="text-2xs text-ink-4 mt-1">{t('price.saleEndsHint')}</p>
          </div>
        </div>

        {error && <p className="text-2xs text-danger">{error}</p>}
        <div className="flex justify-end gap-2 pt-1">
          <button type="button" onClick={onClose} className="px-3 py-1.5 text-xs rounded-lg text-ink-2 hover:bg-hover">
            {t('common:actions.cancel')}
          </button>
          <button type="button" onClick={submit} disabled={saving} className="px-3 py-1.5 text-xs rounded-lg bg-accent-solid text-white hover:bg-accent-solid-hover disabled:opacity-50">
            {saving ? t('common:actions.saving') : t('common:actions.save')}
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
export function NewBatchModal({ warehouses, onSave, onClose, onCreated }: {
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
// Adjust/Recount/Transfer/Split/Damaged-Lost staff actually want. Picking
// one hands off to that same modal, pre-locked to this row (see
// BatchScope); Cancel here just closes without picking anything. Split
// case is only offered for a SKU configured as a case of another SKU.
type BatchRowAction = 'adjust' | 'recount' | 'transfer' | 'split-case' | 'split-packages' | 'damaged-lost'

export function BatchRowActionModal({ scope, canSplitCase = false, canSplitPackages = false, onChoose, onClose }: {
  scope: BatchScope
  canSplitCase?: boolean
  // A scale-sold weight pool's lot can be split into packages.
  canSplitPackages?: boolean
  onChoose: (action: BatchRowAction) => void
  onClose: () => void
}) {
  const { t } = useTranslation(['inventory', 'common'])
  const actions: { key: BatchRowAction; label: string; icon: ReactNode; danger?: boolean }[] = [
    { key: 'adjust', label: t('rowAction.adjust'), icon: <Plus size={13} /> },
    { key: 'recount', label: t('rowAction.recount'), icon: <ClipboardList size={13} /> },
    { key: 'transfer', label: t('rowAction.transfer'), icon: <ArrowRightLeft size={13} /> },
    ...(canSplitCase ? [{ key: 'split-case' as const, label: t('rowAction.splitCase'), icon: <Boxes size={13} /> }] : []),
    ...(canSplitPackages ? [{ key: 'split-packages' as const, label: t('rowAction.splitPackages'), icon: <Scale size={13} /> }] : []),
    { key: 'damaged-lost', label: t('rowAction.damagedLost'), icon: <PackageMinus size={13} />, danger: true },
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
          {t('common:actions.cancel')}
        </button>
      </div>
    </Modal>
  )
}
