import { useTranslation } from 'react-i18next'
import { FormCard } from './FormCard'
import { ScaleCodeAliasesEditor } from './SkuAliasEditors'
import { normalizeScaleCodeInput } from './productFormat'
import { inputClass, labelClass } from './formStyles'
import type { FlatSkuForm, SetFlatSkuField } from './flatSkuForm'
import type { SKUPricingMode } from './types'

interface FlatSkuScaleBarcodeCardProps {
  form: Pick<FlatSkuForm, 'scaleCode' | 'pricingMode' | 'pricePerKilo' | 'currency'>
  setField: SetFlatSkuField
  // Set once the SKU exists -- extra scale codes hang off a real SKU id.
  currentSkuId: string | null
}

// "Weight-embedded barcode" card: scale code + per-kilo pricing for
// butcher/deli-scale packages.
export function FlatSkuScaleBarcodeCard({ form, setField, currentSkuId }: FlatSkuScaleBarcodeCardProps) {
  const { scaleCode, pricingMode, pricePerKilo, currency } = form
  const { t } = useTranslation('pms')
  return (
    <FormCard title={t('scale.title')} description={t('scale.description')}>
      <div>
        <label htmlFor="product-scale-code" className={labelClass}>{t('scale.code')}</label>
        <input
          id="product-scale-code"
          type="text" placeholder={t('scale.codePlaceholder')} value={scaleCode}
          onChange={e => setField('scaleCode', normalizeScaleCodeInput(e.target.value))}
          maxLength={7} inputMode="numeric"
          className={inputClass}
        />
        <p className="text-2xs text-ink-4 mt-1">
          {t('scale.codeHint')}
        </p>
      </div>
      <div className="grid grid-cols-2 gap-3 mt-2">
        <div>
          <label htmlFor="product-pricing-mode" className={labelClass}>{t('scale.pricing')}</label>
          <select
            id="product-pricing-mode" value={pricingMode}
            onChange={e => setField('pricingMode', e.target.value as SKUPricingMode)}
            className={inputClass}
          >
            <option value="fixed">{t('scale.fixed')}</option>
            <option value="per_weight">{t('scale.perWeight')}</option>
          </select>
          {pricingMode === 'per_weight' && !scaleCode && (
            <p className="text-2xs text-warning mt-1">
              {t('scale.needCode')}
            </p>
          )}
        </div>
        {pricingMode === 'per_weight' && (
          <div>
            <label htmlFor="product-price-per-kilo" className={labelClass}>{t('scale.pricePerKilo', { currency })}</label>
            <input
              id="product-price-per-kilo"
              type="number" step="0.01" min="0" required value={pricePerKilo}
              onChange={e => setField('pricePerKilo', e.target.value)}
              className={inputClass}
            />
          </div>
        )}
      </div>
      {scaleCode && currentSkuId && <div className="mt-2"><ScaleCodeAliasesEditor skuId={currentSkuId} /></div>}
    </FormCard>
  )
}
