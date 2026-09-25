import { useState } from 'react'
import { Search, X, Check } from 'lucide-react'
import clsx from 'clsx'
import type { PurchaseOrderLineItem, AddLineItemInput, LineItemType } from './types'
import { ProductSkuPicker } from '../components/common/ProductSkuPicker'
import { LINE_TYPE_LABELS, isChargeLineType, signedLineItemAmount } from './procurementHelpers'

interface LineItemFormRowProps {
  initial?: PurchaseOrderLineItem
  currencyCode: string
  onSubmit: (input: AddLineItemInput) => Promise<void>
  onCancel: () => void
}

export function LineItemFormRow({ initial, onSubmit, onCancel }: LineItemFormRowProps) {
  const [description, setDescription] = useState(initial?.raw_description ?? '')
  const [quantity, setQuantity] = useState(initial ? String(initial.quantity) : '1')
  const [unitCost, setUnitCost] = useState(initial ? (initial.unit_cost_minor_units / 100).toFixed(2) : '0.00')
  const [skuId, setSkuId] = useState<string | null>(initial?.sku_id ?? null)
  // Friendly "Product Name — SKU code" for display -- falls back to null
  // (renders the raw id) only when the initial item has no resolved name,
  // e.g. the linked SKU was deleted after linking.
  const [skuLabel, setSkuLabel] = useState<string | null>(
    initial?.sku_id && initial.product_name ? `${initial.product_name} — ${initial.sku_code}` : null,
  )
  const [unitsPerPurchasedItem, setUnitsPerPurchasedItem] = useState(initial ? String(initial.units_per_purchased_item) : '1')
  const [lineType, setLineType] = useState<LineItemType>(initial?.line_type || 'product')
  const [saving, setSaving] = useState(false)
  const [rowError, setRowError] = useState('')
  const [pickerOpen, setPickerOpen] = useState(false)
  const isCharge = isChargeLineType(lineType)

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    setRowError('')
    setSaving(true)
    try {
      await onSubmit({
        // A charge row (shipping/discount/fee/credit note) never has a
        // SKU, even if one was picked before switching the type.
        sku_id: isCharge ? null : skuId,
        raw_description: description,
        quantity: parseInt(quantity, 10),
        units_per_purchased_item: isCharge ? 1 : parseInt(unitsPerPurchasedItem || '1', 10),
        unit_cost_minor_units: Math.round(parseFloat(unitCost || '0') * 100),
        line_type: lineType,
      })
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Failed to save line item')
    } finally {
      setSaving(false)
    }
  }

  const labelClass = 'block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1'
  const inputClass = 'w-full px-3 py-2 text-xs rounded-lg dark:bg-night-950 bg-white border border-line text-ink outline-none focus:ring-1 focus:ring-focus'
  const idPrefix = initial?.id ?? 'new'

  return (
    <tr className="bg-accent/5">
      <td colSpan={8} className="px-4 py-3">
          <div className="grid grid-cols-4 gap-3">
            <div className="col-span-2">
              <label htmlFor={`li-description-${idPrefix}`} className={labelClass}>Description</label>
              <input
                id={`li-description-${idPrefix}`}
                aria-label="Line item description" required value={description} onChange={e => setDescription(e.target.value)}
                className={inputClass} placeholder="Description"
              />
            </div>
            <div>
              <label htmlFor={`li-type-${idPrefix}`} className={labelClass}>Type</label>
              <select
                id={`li-type-${idPrefix}`}
                aria-label="Line item type" value={lineType} onChange={e => setLineType(e.target.value as LineItemType)}
                className={inputClass}
              >
                {(Object.keys(LINE_TYPE_LABELS) as Exclude<LineItemType, ''>[]).map(t => (
                  <option key={t} value={t}>{LINE_TYPE_LABELS[t]}</option>
                ))}
              </select>
            </div>
            <div>
              <label htmlFor={`li-quantity-${idPrefix}`} className={labelClass}>Quantity</label>
              <input
                id={`li-quantity-${idPrefix}`}
                aria-label="Quantity" type="number" min="1" required value={quantity} onChange={e => setQuantity(e.target.value)}
                className={inputClass}
              />
            </div>

            <div className="col-span-2">
              <span className={labelClass}>Linked product</span>
              {isCharge ? (
                <p className="text-xs text-ink-4 py-2">— (charge rows don't link to a product)</p>
              ) : skuId ? (
                <div className="flex items-center gap-2 py-1">
                  <span
                    className={clsx('text-xs truncate', skuLabel ? 'text-ink' : 'font-mono text-ink-3')}
                    title={skuLabel ?? skuId}
                  >
                    {skuLabel ?? skuId}
                  </span>
                  <button
                    type="button" onClick={() => { setSkuId(null); setSkuLabel(null) }}
                    className="flex items-center gap-1 text-2xs text-ink-4 hover:text-danger shrink-0"
                  >
                    <X size={11} /> Unlink
                  </button>
                </div>
              ) : (
                <button
                  type="button"
                  onClick={() => setPickerOpen(true)}
                  className="flex items-center gap-1 text-xs text-accent hover:underline py-2"
                >
                  <Search size={12} /> Link product
                </button>
              )}
            </div>
            {!isCharge && skuId && (
              <div>
                <label htmlFor={`li-units-per-item-${idPrefix}`} className={labelClass}>Units per purchased item</label>
                <input
                  id={`li-units-per-item-${idPrefix}`}
                  aria-label="Units per purchased item" type="number" min="1" step="1"
                  value={unitsPerPurchasedItem} onChange={e => setUnitsPerPurchasedItem(e.target.value)}
                  className={inputClass}
                />
                <p className="text-2xs text-ink-4 mt-1">
                  Stock units one purchased item holds — e.g. 36 if this line is a box of 36 units.
                </p>
              </div>
            )}

            <div>
              <label htmlFor={`li-unit-cost-${idPrefix}`} className={labelClass}>Unit cost</label>
              <input
                id={`li-unit-cost-${idPrefix}`}
                aria-label="Unit cost" type="number" min="0" step="0.01" required value={unitCost} onChange={e => setUnitCost(e.target.value)}
                className={inputClass}
              />
            </div>
            <div>
              <span className={labelClass}>Total</span>
              <p className="px-3 py-2 text-xs text-ink-2">
                {(signedLineItemAmount({
                  line_type: lineType, quantity: parseInt(quantity || '0', 10),
                  unit_cost_minor_units: Math.round(parseFloat(unitCost || '0') * 100),
                }) / 100).toFixed(2)}
              </p>
            </div>
          </div>

          {rowError && <p className="text-2xs text-danger mt-2">{rowError}</p>}

          {pickerOpen && (
            <div className="mt-3">
              <ProductSkuPicker
                onSelect={(sku, label) => {
                  setSkuId(sku.id); setSkuLabel(label)
                  if (!description) setDescription(label)
                  setPickerOpen(false)
                }}
                onClose={() => setPickerOpen(false)}
              />
            </div>
          )}

          <div className="flex items-center gap-2 mt-3">
            <button
              type="button" disabled={saving} onClick={handleSubmit}
              className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50"
            >
              <Check size={12} /> Save line
            </button>
            <button
              type="button" onClick={onCancel}
              className="px-3 py-1.5 rounded-lg text-xs text-ink-2 hover:bg-hover"
            >
              Cancel
            </button>
          </div>
      </td>
    </tr>
  )
}
