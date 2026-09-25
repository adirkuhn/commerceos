import { FormCard } from './FormCard'
import { BarcodeAliasesEditor } from './SkuAliasEditors'
import { CURRENCIES } from './productFormat'
import { inputClass, labelClass } from './formStyles'
import type { FlatSkuForm, SetFlatSkuField } from './flatSkuForm'
import type { TaxPreview } from './skuTax'
import type { TaxClass } from '../tax/types'
import type { SKUStatus } from './types'

interface FlatSkuPricingCardProps {
  form: FlatSkuForm
  setField: SetFlatSkuField
  // Set once the SKU exists -- alternate barcodes hang off a real SKU id.
  currentSkuId: string | null
  taxClasses: TaxClass[]
  defaultTaxClass: TaxClass | null
  taxPreview: TaxPreview | null
}

// "Pricing & SKU" card of the flat single-SKU form.
export function FlatSkuPricingCard({ form, setField, currentSkuId, taxClasses, defaultTaxClass, taxPreview }: FlatSkuPricingCardProps) {
  const { scaleCode, currency } = form
  return (
    <FormCard title="Pricing & SKU">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label htmlFor="product-sku-code" className={labelClass}>SKU code</label>
          <input
            id="product-sku-code"
            type="text" required placeholder="e.g. SHIRT-BLK-M" value={form.skuCode} onChange={e => setField('skuCode', e.target.value)}
            className={inputClass}
          />
        </div>
        <div>
          <label htmlFor="product-barcode" className={labelClass}>Barcode / EAN</label>
          <input
            id="product-barcode"
            type="text" placeholder={scaleCode ? 'Not used — see Weight-embedded barcode below' : 'Optional'}
            disabled={!!scaleCode} value={scaleCode ? '' : form.barcode} onChange={e => setField('barcode', e.target.value)}
            className={inputClass}
          />
          {scaleCode && (
            <p className="text-2xs text-ink-4 mt-1">
              Not used for this SKU — its real printed barcode changes per package (weight baked in).
            </p>
          )}
          {currentSkuId && !scaleCode && <div className="mt-1"><BarcodeAliasesEditor skuId={currentSkuId} /></div>}
        </div>
        <div>
          <label htmlFor="product-price" className={labelClass}>Price</label>
          <input
            id="product-price"
            type="number" step="0.01" min="0" required placeholder="0.00" value={form.price} onChange={e => setField('price', e.target.value)}
            className={inputClass}
          />
          {taxPreview && (
            <p className="mt-1 text-2xs text-ink-4">
              Excl. tax: {currency} {taxPreview.exclusive.toFixed(2)} · Incl. tax: {currency} {taxPreview.inclusive.toFixed(2)} ({taxPreview.ratePercent}% {taxPreview.country})
            </p>
          )}
        </div>
        <div>
          <label htmlFor="product-sale-price" className={labelClass}>Sale price (optional)</label>
          <input
            id="product-sale-price"
            type="number" step="0.01" min="0" placeholder="None" value={form.salePrice} onChange={e => setField('salePrice', e.target.value)}
            className={inputClass}
          />
        </div>
        <div>
          <label htmlFor="product-sale-end-date" className={labelClass}>Sale ends (optional)</label>
          <input
            id="product-sale-end-date"
            type="date" value={form.saleEndDate} onChange={e => setField('saleEndDate', e.target.value)}
            className={inputClass}
          />
        </div>
        <div>
          <label htmlFor="product-currency" className={labelClass}>Currency</label>
          <select id="product-currency" value={currency} onChange={e => setField('currency', e.target.value)} className={inputClass}>
            {CURRENCIES.map(c => <option key={c} value={c}>{c}</option>)}
          </select>
        </div>
        <div>
          <label htmlFor="product-tax-class" className={labelClass}>Tax class</label>
          <select
            id="product-tax-class"
            value={form.taxClassId ?? ''}
            onChange={e => setField('taxClassId', e.target.value || null)}
            className={inputClass}
          >
            <option value="">Tenant default{defaultTaxClass ? ` (${defaultTaxClass.name})` : ''}</option>
            {taxClasses.map(c => <option key={c.id} value={c.id}>{c.name}</option>)}
          </select>
        </div>
        <div>
          <label htmlFor="product-sku-status" className={labelClass}>SKU status</label>
          <select id="product-sku-status" value={form.skuStatus} onChange={e => setField('skuStatus', e.target.value as SKUStatus)} className={inputClass}>
            <option value="active">Active</option>
            <option value="inactive">Inactive</option>
          </select>
        </div>
      </div>
    </FormCard>
  )
}
