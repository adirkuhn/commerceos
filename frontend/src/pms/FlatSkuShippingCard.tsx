import { FormCard, ToggleRow } from './FormCard'
import { inputClass, labelClass } from './formStyles'
import type { FlatSkuForm, PatchFlatSku, SetFlatSkuField } from './flatSkuForm'
import type { DimensionInputs } from './skuPayloads'

interface FlatSkuShippingCardProps {
  form: Pick<FlatSkuForm, 'isVirtual' | keyof DimensionInputs>
  setField: SetFlatSkuField
  patch: PatchFlatSku
}

const DIMENSION_FIELDS: { key: keyof DimensionInputs; id: string; label: string }[] = [
  { key: 'weightGrams', id: 'product-weight', label: 'Weight (g)' },
  { key: 'lengthMM', id: 'product-length', label: 'Length (mm)' },
  { key: 'widthMM', id: 'product-width', label: 'Width (mm)' },
  { key: 'heightMM', id: 'product-height', label: 'Height (mm)' },
]

// "Shipping" card: virtual toggle, and weight/dimensions for a physical SKU.
export function FlatSkuShippingCard({ form, setField, patch }: FlatSkuShippingCardProps) {
  return (
    <FormCard title="Shipping" description="Leave off for digital/virtual products.">
      <ToggleRow label="Virtual product (no shipping)" checked={form.isVirtual} onChange={() => patch(prev => ({ isVirtual: !prev.isVirtual }))} />
      {!form.isVirtual && (
        <div className="grid grid-cols-2 gap-3">
          {DIMENSION_FIELDS.map(f => (
            <div key={f.key}>
              <label htmlFor={f.id} className={labelClass}>{f.label}</label>
              <input
                id={f.id}
                type="number" step="1" min="0" placeholder="Optional" value={form[f.key]} onChange={e => setField(f.key, e.target.value)}
                className={inputClass}
              />
            </div>
          ))}
        </div>
      )}
    </FormCard>
  )
}
