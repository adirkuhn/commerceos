import { CaseOfSKUPicker } from './VariantsSection'
import { CURRENCIES, normalizeScaleCodeInput } from './productFormat'
import { fieldInputClass, fieldLabelClass } from './formStyles'
import { DimensionInputsGrid } from './DimensionInputsGrid'
import type { useVariantAddSku } from './hooks/useVariantAddSku'
import type { SKUPricingMode, SKUStatus, SKUUnitType } from './types'

interface VariantAddSkuPanelProps {
  addSku: ReturnType<typeof useVariantAddSku>
}

// A variant row's "+ Add SKU" panel. All state lives in useVariantAddSku.
export function VariantAddSkuPanel({ addSku }: VariantAddSkuPanelProps) {
  const { draft, setField, patch, portionPickerOpen, setPortionPickerOpen, error, submitting, submit, setOpen } = addSku
  const { unitType, portionedFromSKUID, portionedFromLabel, pricingMode, scaleCode } = draft

  return (
    // Plain div, not <form> — same nested-<form>-inside-the-page's-outer-
    // form bug as the Add Variant form (see VariantsSection).
    <div className="mt-2 space-y-1.5 p-2 rounded-lg dark:bg-night-800/60 bg-gray-50 border border-line">
      <div className="grid grid-cols-2 gap-1.5">
        <div>
          <label className={fieldLabelClass}>SKU code</label>
          <input type="text" required placeholder="SKU code" value={draft.skuCode} onChange={e => setField('skuCode', e.target.value)} className={fieldInputClass} />
        </div>
        <div>
          <label className={fieldLabelClass}>Barcode</label>
          <input type="text" placeholder="Optional" value={draft.barcode} onChange={e => setField('barcode', e.target.value)} className={fieldInputClass} />
        </div>
        <div>
          <label className={fieldLabelClass}>Price</label>
          <input type="number" step="0.01" min="0" required placeholder="0.00" value={draft.price} onChange={e => setField('price', e.target.value)} className={fieldInputClass} />
        </div>
        <div>
          <label className={fieldLabelClass}>Sale price</label>
          <input type="number" step="0.01" min="0" placeholder="Optional" value={draft.salePrice} onChange={e => setField('salePrice', e.target.value)} className={fieldInputClass} />
        </div>
        <div>
          <label className={fieldLabelClass}>Sale ends</label>
          <input type="date" title="Sale ends (optional)" value={draft.saleEndDate} onChange={e => setField('saleEndDate', e.target.value)} className={fieldInputClass} />
        </div>
        <div>
          <label className={fieldLabelClass}>Currency</label>
          <select value={draft.currency} onChange={e => setField('currency', e.target.value)} className={fieldInputClass}>
            {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>
      </div>
      <div>
        <label className={fieldLabelClass}>Status</label>
        <select value={draft.status} onChange={e => setField('status', e.target.value as SKUStatus)} className={fieldInputClass}>
          <option value="active">Active</option>
          <option value="inactive">Inactive</option>
        </select>
      </div>
      <label className="flex items-center gap-1.5 text-2xs text-ink-3">
        <input type="checkbox" checked={draft.isVirtual} onChange={e => setField('isVirtual', e.target.checked)} />
        Virtual (no shipping)
      </label>
      {!draft.isVirtual && <DimensionInputsGrid values={draft} onChange={setField} />}
      <label className="flex items-center gap-1.5 text-2xs text-ink-3">
        Item type
        <select
          value={unitType}
          onChange={e => {
            const next = e.target.value as SKUUnitType
            patch(next === 'kilo' ? { unitType: next, portionedFromSKUID: null, portionedFromLabel: '' } : { unitType: next })
          }}
          className="px-2 py-1 text-2xs rounded bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
        >
          <option value="unit">Unit</option>
          <option value="box">Box</option>
          <option value="kilo">Kilo (weight pool)</option>
        </select>
      </label>
      {unitType !== 'kilo' && (
        portionedFromSKUID ? (
          <div className="flex items-center gap-1.5 text-2xs text-ink-3">
            <span className="flex-1">Portioned from <span className="font-medium">{portionedFromLabel || portionedFromSKUID}</span></span>
            <button type="button" onClick={() => patch({ portionedFromSKUID: null, portionedFromLabel: '' })} className="hover:underline">Unlink</button>
          </div>
        ) : portionPickerOpen ? (
          <CaseOfSKUPicker
            excludeSKUID=""
            filterSku={s => s.unit_type === 'kilo'}
            onSelect={(s, label) => { patch({ portionedFromSKUID: s.id, portionedFromLabel: label }); setPortionPickerOpen(false) }}
            onClose={() => setPortionPickerOpen(false)}
          />
        ) : (
          <button type="button" onClick={() => setPortionPickerOpen(true)} className="text-2xs text-accent hover:text-accent/80 transition-colors">
            + Link to a weight pool
          </button>
        )
      )}
      <div>
        <label className={fieldLabelClass}>Scale code (weight-embedded barcode)</label>
        <input type="text" placeholder="Not sold via a scale barcode" value={scaleCode} onChange={e => setField('scaleCode', normalizeScaleCodeInput(e.target.value))} maxLength={7} inputMode="numeric" className={fieldInputClass} />
      </div>
      <div className="grid grid-cols-2 gap-1.5">
        <label className="flex flex-col gap-0.5 text-2xs text-ink-3">
          Pricing
          <select value={pricingMode} onChange={e => setField('pricingMode', e.target.value as SKUPricingMode)} className={fieldInputClass}>
            <option value="fixed">Fixed / package</option>
            <option value="per_weight">Per kilo</option>
          </select>
        </label>
        {pricingMode === 'per_weight' && (
          <div>
            <label className={fieldLabelClass}>Price/kilo ({draft.currency})</label>
            <input type="number" step="0.01" min="0" required value={draft.pricePerKilo} onChange={e => setField('pricePerKilo', e.target.value)} className={fieldInputClass} />
          </div>
        )}
      </div>
      {pricingMode === 'per_weight' && !scaleCode && (
        <p className="text-2xs text-warning">Set a scale code above too — per-kilo pricing only applies once a package's weight can actually be scanned.</p>
      )}
      {error && <p className="text-2xs text-danger">{error}</p>}
      <div className="flex gap-1.5">
        <button type="button" onClick={submit} disabled={submitting} className="flex-1 py-1 rounded bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors">
          {submitting ? 'Adding…' : 'Add'}
        </button>
        <button type="button" onClick={() => setOpen(false)} className="px-2 py-1 rounded bg-fill text-2xs text-ink-3 transition-colors">
          Cancel
        </button>
      </div>
    </div>
  )
}
