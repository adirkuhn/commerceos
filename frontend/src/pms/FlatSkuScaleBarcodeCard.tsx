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
  return (
    <FormCard
      title="Weight-embedded barcode"
      description="For a butcher/deli scale that prints a barcode with the package's weight baked in — every package gets a different full barcode, so scanning it is matched by this scale code instead."
    >
      <div>
        <label htmlFor="product-scale-code" className={labelClass}>Scale code</label>
        <input
          id="product-scale-code"
          type="text" placeholder="Not sold via a scale barcode" value={scaleCode}
          onChange={e => setField('scaleCode', normalizeScaleCodeInput(e.target.value))}
          maxLength={7} inputMode="numeric"
          className={inputClass}
        />
        <p className="text-2xs text-ink-4 mt-1">
          The full code programmed into the scale for this product, including its prefix (e.g. 2900633) — type it exactly as shown on the scale, leave blank if it has a normal fixed barcode.
        </p>
      </div>
      <div className="grid grid-cols-2 gap-3 mt-2">
        <div>
          <label htmlFor="product-pricing-mode" className={labelClass}>Pricing</label>
          <select
            id="product-pricing-mode" value={pricingMode}
            onChange={e => setField('pricingMode', e.target.value as SKUPricingMode)}
            className={inputClass}
          >
            <option value="fixed">Fixed — same price per package (or a normal per-unit product)</option>
            <option value="per_weight">Per kilo — price = rate × scanned weight</option>
          </select>
          {pricingMode === 'per_weight' && !scaleCode && (
            <p className="text-2xs text-warning mt-1">
              Set a scale code above too — per-kilo pricing only applies once a package's weight can actually be scanned.
            </p>
          )}
        </div>
        {pricingMode === 'per_weight' && (
          <div>
            <label htmlFor="product-price-per-kilo" className={labelClass}>Price per kilo ({currency})</label>
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
