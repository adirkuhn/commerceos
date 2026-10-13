import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useTenantSettings } from '../settings/hooks/useTenantSettings'
import { DEFAULT_LOW_STOCK_THRESHOLD } from '../settings/types'
import { setProductLowStockThreshold } from './api'
import { FormCard } from './FormCard'
import { inputClass, labelClass } from './formStyles'
import type { Product, VariantWithSKUs } from './types'
import { apiErrorText } from '../api/apiErrorText'

const GRAMS_PER_KG = 1000

// Stock of a weighed product is stored in grams; the field shows kg.
function toField(stockUnits: number, weighed: boolean): string {
  return String(weighed ? stockUnits / GRAMS_PER_KG : stockUnits)
}

function toStockUnits(field: string, weighed: boolean): number | null {
  if (field.trim() === '') return null
  const n = Number(field)
  return Math.round(weighed ? n * GRAMS_PER_KG : n)
}

// isWeighedProduct: every SKU is a weight pool (unit_type kilo), so its
// stock -- and its low stock line -- is in grams.
export function isWeighedProduct(variants: VariantWithSKUs[]): boolean {
  const skus = variants.flatMap(v => v.skus ?? [])
  return skus.length > 0 && skus.every(s => s.unit_type === 'kilo')
}

/** The product's own low stock line, overriding the shop's (Settings ->
 *  Inventory) for the inventory list's "Low stock" filter. Empty = the
 *  shop's. Saved on its own, apart from the product form. */
export function ProductLowStockCard({ product, weighed }: { product: Product; weighed: boolean }) {
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])
  const { settings } = useTenantSettings(token ?? '')
  const own = product.low_stock_threshold
  const [value, setValue] = useState(own == null ? '' : toField(own, weighed))
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const [saved, setSaved] = useState(false)

  useEffect(() => { setValue(own == null ? '' : toField(own, weighed)) }, [own, weighed])

  async function save() {
    setSaving(true)
    setError('')
    setSaved(false)
    try {
      const p = await setProductLowStockThreshold(token ?? '', product.id, toStockUnits(value, weighed))
      setValue(p.low_stock_threshold == null ? '' : toField(p.low_stock_threshold, weighed))
      setSaved(true)
    } catch (err) {
      setError(apiErrorText(err) ?? t('lowStock.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  const shopDefault = toField(settings?.low_stock_threshold ?? DEFAULT_LOW_STOCK_THRESHOLD, weighed)
  return (
    <FormCard title={t('lowStock.title')} description={t('lowStock.help')}>
      <div className="max-w-[220px]">
        <label htmlFor="product-low-stock" className={labelClass}>
          {weighed ? t('lowStock.thresholdKg') : t('lowStock.thresholdPieces')}
        </label>
        <input
          id="product-low-stock" type="number" min={0} step={weighed ? 0.001 : 1}
          value={value} placeholder={t('lowStock.shopDefault', { value: shopDefault })}
          onChange={e => { setValue(e.target.value); setSaved(false) }}
          className={inputClass}
        />
      </div>
      {error && <p role="alert" className="text-xs text-danger">{error}</p>}
      {saved && <p className="text-xs text-success">{t('lowStock.saved')}</p>}
      <button type="button" onClick={() => void save()} disabled={saving}
        className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium disabled:opacity-50">
        {saving ? t('common:actions.saving') : t('lowStock.save')}
      </button>
    </FormCard>
  )
}
