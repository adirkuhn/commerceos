import { useTranslation } from 'react-i18next'
import { fieldInputClass, fieldLabelClass } from './formStyles'
import type { DimensionInputs } from './skuPayloads'


// Compact weight/length/width/height inputs, shared by the variant row's
// edit and add-SKU panels.
export function DimensionInputsGrid({ values, onChange }: { values: DimensionInputs; onChange: (key: keyof DimensionInputs, value: string) => void }) {
  const { t } = useTranslation('pms')
  const DIMENSION_FIELDS: { key: keyof DimensionInputs; label: string }[] = [
    { key: 'weightGrams', label: t('shipping.weight') },
    { key: 'lengthMM', label: t('shipping.length') },
    { key: 'widthMM', label: t('shipping.width') },
    { key: 'heightMM', label: t('shipping.height') },
  ]
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
