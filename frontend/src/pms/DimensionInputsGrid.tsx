import { fieldInputClass, fieldLabelClass } from './formStyles'
import type { DimensionInputs } from './skuPayloads'

const DIMENSION_FIELDS: { key: keyof DimensionInputs; label: string }[] = [
  { key: 'weightGrams', label: 'Weight (g)' },
  { key: 'lengthMM', label: 'Length (mm)' },
  { key: 'widthMM', label: 'Width (mm)' },
  { key: 'heightMM', label: 'Height (mm)' },
]

// Compact weight/length/width/height inputs, shared by the variant row's
// edit and add-SKU panels.
export function DimensionInputsGrid({ values, onChange }: { values: DimensionInputs; onChange: (key: keyof DimensionInputs, value: string) => void }) {
  return (
    <div className="grid grid-cols-2 gap-1.5">
      {DIMENSION_FIELDS.map(f => (
        <div key={f.key}>
          <label className={fieldLabelClass}>{f.label}</label>
          <input type="number" step="1" min="0" placeholder="0" value={values[f.key]} onChange={e => onChange(f.key, e.target.value)} className={fieldInputClass} />
        </div>
      ))}
    </div>
  )
}
