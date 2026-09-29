import { useTranslation } from 'react-i18next'
import { FormCard, ToggleRow } from './FormCard'
import { inputClass, labelClass } from './formStyles'
import type { FlatSkuForm, PatchFlatSku, SetFlatSkuField } from './flatSkuForm'
import type { DimensionInputs } from './skuPayloads'

interface FlatSkuShippingCardProps {
  form: Pick<FlatSkuForm, 'isVirtual' | keyof DimensionInputs>
  setField: SetFlatSkuField
  patch: PatchFlatSku
}


// "Shipping" card: virtual toggle, and weight/dimensions for a physical SKU.
export function FlatSkuShippingCard({ form, setField, patch }: FlatSkuShippingCardProps) {
  const { t } = useTranslation('pms')
  const DIMENSION_FIELDS: { key: keyof DimensionInputs; id: string; label: string }[] = [
    { key: 'weightGrams', id: 'product-weight', label: t('shipping.weight') },
    { key: 'lengthMM', id: 'product-length', label: t('shipping.length') },
    { key: 'widthMM', id: 'product-width', label: t('shipping.width') },
    { key: 'heightMM', id: 'product-height', label: t('shipping.height') },
  ]
  return (
    <FormCard title={t('shipping.title')} description={t('shipping.description')}>
      <ToggleRow label={t('shipping.virtual')} checked={form.isVirtual} onChange={() => patch(prev => ({ isVirtual: !prev.isVirtual }))} />
      {!form.isVirtual && (
        <div className="grid grid-cols-2 gap-3">
          {DIMENSION_FIELDS.map(f => (
            <div key={f.key}>
              <label htmlFor={f.id} className={labelClass}>{f.label}</label>
              <input
                id={f.id}
                type="number" step="1" min="0" placeholder={t('shipping.optional')} value={form[f.key]} onChange={e => setField(f.key, e.target.value)}
                className={inputClass}
              />
            </div>
          ))}
        </div>
      )}
    </FormCard>
  )
}
