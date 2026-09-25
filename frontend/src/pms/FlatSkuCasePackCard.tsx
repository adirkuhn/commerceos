import { useState } from 'react'
import { Search } from 'lucide-react'
import { FormCard } from './FormCard'
import { CaseOfSKUPicker } from './VariantsSection'
import { inputClass, labelClass } from './formStyles'
import type { FlatSkuForm, PatchFlatSku, SetFlatSkuField } from './flatSkuForm'

interface FlatSkuCasePackCardProps {
  form: Pick<FlatSkuForm, 'caseOfSKUID' | 'caseOfSKULabel' | 'unitsPerCase'>
  setField: SetFlatSkuField
  patch: PatchFlatSku
  // The SKU being edited -- it can't be a case of itself.
  currentSkuId: string
}

// "Case pack" card: links this SKU as a sealed case of another SKU.
export function FlatSkuCasePackCard({ form, setField, patch, currentSkuId }: FlatSkuCasePackCardProps) {
  const [casePickerOpen, setCasePickerOpen] = useState(false)
  return (
    <FormCard
      title="Case pack"
      description="If a supplier sells this as a sealed case/box of another product in your catalog, link it here — lets you split case stock into unit stock later."
    >
      {form.caseOfSKUID ? (
        <div className="flex items-center gap-2">
          <span className="flex-1 text-xs text-ink">
            Case of <span className="font-medium">{form.caseOfSKULabel || form.caseOfSKUID}</span>
          </span>
          <div className="w-24">
            <label htmlFor="product-units-per-case" className={labelClass}>Units/case</label>
            <input
              id="product-units-per-case"
              type="number" step="1" min="1" required value={form.unitsPerCase} onChange={e => setField('unitsPerCase', e.target.value)}
              className={inputClass}
            />
          </div>
          <button
            type="button"
            onClick={() => patch({ caseOfSKUID: null, caseOfSKULabel: '', unitsPerCase: '' })}
            className="text-2xs text-ink-4 hover:underline"
          >
            Remove
          </button>
        </div>
      ) : casePickerOpen ? (
        <CaseOfSKUPicker
          excludeSKUID={currentSkuId}
          onSelect={(sku, label) => { patch({ caseOfSKUID: sku.id, caseOfSKULabel: label }); setCasePickerOpen(false) }}
          onClose={() => setCasePickerOpen(false)}
        />
      ) : (
        <button
          type="button"
          onClick={() => setCasePickerOpen(true)}
          className="flex items-center gap-1 text-xs font-medium text-accent hover:text-accent/80 transition-colors"
        >
          <Search size={12} /> This is a case of another product
        </button>
      )}
    </FormCard>
  )
}
