import { useState } from 'react'
import { Search } from 'lucide-react'
import { FormCard } from './FormCard'
import { CaseOfSKUPicker } from './VariantsSection'
import { inputClass, labelClass } from './formStyles'
import type { FlatSkuForm, PatchFlatSku } from './flatSkuForm'
import type { SKUUnitType } from './types'

interface FlatSkuWeightPoolCardProps {
  form: Pick<FlatSkuForm, 'unitType' | 'portionedFromSKUID' | 'portionedFromLabel'>
  patch: PatchFlatSku
  // The SKU being edited (if saved yet) -- it can't be portioned from itself.
  currentSkuId: string | null
}

// "Weight pool" card: a SKU is either a weight pool itself (unit_type
// kilo), or portioned from one, never both.
export function FlatSkuWeightPoolCard({ form, patch, currentSkuId }: FlatSkuWeightPoolCardProps) {
  const [portionPickerOpen, setPortionPickerOpen] = useState(false)
  const { unitType, portionedFromSKUID, portionedFromLabel } = form
  return (
    <FormCard
      title="Weight pool"
      description="Book a bulk lot once (e.g. 20kg), then peel off individually-weighed sellable items from it over time as they're packaged."
    >
      <div>
        <label htmlFor="product-unit-type" className={labelClass}>Item type</label>
        <select
          id="product-unit-type" value={unitType}
          onChange={e => {
            const next = e.target.value as SKUUnitType
            patch(next === 'kilo' ? { unitType: next, portionedFromSKUID: null, portionedFromLabel: '' } : { unitType: next })
          }}
          className={inputClass}
        >
          <option value="unit">Unit</option>
          <option value="box">Box</option>
          <option value="kilo">Kilo (weight pool)</option>
        </select>
      </div>
      {unitType !== 'kilo' && (
        portionedFromSKUID ? (
          <div className="flex items-center gap-2 mt-2">
            <span className="flex-1 text-xs text-ink">
              Portioned from <span className="font-medium">{portionedFromLabel || portionedFromSKUID}</span>
            </span>
            <button
              type="button"
              onClick={() => patch({ portionedFromSKUID: null, portionedFromLabel: '' })}
              className="text-2xs text-ink-4 hover:underline"
            >
              Remove
            </button>
          </div>
        ) : portionPickerOpen ? (
          <div className="mt-2">
            <CaseOfSKUPicker
              excludeSKUID={currentSkuId ?? ''}
              filterSku={s => s.unit_type === 'kilo'}
              onSelect={(sku, label) => { patch({ portionedFromSKUID: sku.id, portionedFromLabel: label }); setPortionPickerOpen(false) }}
              onClose={() => setPortionPickerOpen(false)}
            />
          </div>
        ) : (
          <button
            type="button"
            onClick={() => setPortionPickerOpen(true)}
            className="flex items-center gap-1 text-xs font-medium text-accent hover:text-accent/80 transition-colors mt-2"
          >
            <Search size={12} /> This is portioned from a weight pool
          </button>
        )
      )}
    </FormCard>
  )
}
