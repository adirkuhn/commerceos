import { useTranslation } from 'react-i18next'
import { Plus, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import type { BolPricingLevel } from '../../integrations/types'
import { inputClass, labelClass } from '../BolContentFields'
import { addTierRow, setBundleEnabled, type BolPricingForm, type BolTierRow } from './bolPricingForm'
import type { BolPricingPreviewState } from './useBolPricingPreview'

function money(minor: number): string {
  return (minor / 100).toFixed(2)
}

function percent(basisPoints: number): string {
  return String(basisPoints / 100)
}

// The Bol tab's price settings: an optional custom price and fee, bulk
// discount levels, and the exact prices a push would send with what each
// unit leaves you (from the backend preview, never computed here).
export function BolPricingSection({ form, onChange, preview }: {
  form: BolPricingForm
  onChange: (form: BolPricingForm) => void
  preview: BolPricingPreviewState
}) {
  const { t } = useTranslation(['pms'])
  const p = preview.preview
  const levelFor = (quantity: number) => p?.levels.find(l => l.quantity === quantity)
  const first = p?.levels[0]

  const setTier = (index: number, patch: Partial<BolTierRow>) =>
    onChange({ ...form, tiers: form.tiers.map((row, i) => (i === index ? { ...row, ...patch } : row)) })
  const removeTier = (index: number) => onChange({ ...form, tiers: form.tiers.filter((_, i) => i !== index) })

  return (
    <div className="pt-3 border-t border-line space-y-3">
      <h4 className="text-xs font-semibold text-ink">{t('bol.pricing.title')}</h4>

      <div className="grid grid-cols-1 sm:grid-cols-2 gap-3">
        <div>
          <label htmlFor="bol-custom-price" className={labelClass}>{t('bol.pricing.price')}</label>
          <input
            id="bol-custom-price" inputMode="decimal" value={form.customPrice}
            onChange={e => onChange({ ...form, customPrice: e.target.value })}
            placeholder={first ? money(first.unit_price_minor_units) : ''} className={inputClass}
          />
          <p className="text-2xs text-ink-4 mt-1">{t('bol.pricing.priceHint')}</p>
        </div>
        <div>
          <label htmlFor="bol-fee-percent" className={labelClass}>{t('bol.pricing.feePercent')}</label>
          <input
            id="bol-fee-percent" inputMode="decimal" value={form.commissionPercent}
            onChange={e => onChange({ ...form, commissionPercent: e.target.value })}
            placeholder={first ? percent(first.fee_percent_basis_points) : ''} className={inputClass}
          />
          <p className="text-2xs text-ink-4 mt-1">{t('bol.pricing.feePercentHint')}</p>
        </div>
      </div>

      {p && (
        <p className="text-2xs text-ink-4">
          {t('bol.pricing.basis', {
            price: money(p.regular_price_minor_units), vat: percent(p.product_vat_basis_points), shipping: money(p.shipping_minor_units),
          })}
        </p>
      )}

      <div className="flex items-start gap-2">
        <input
          id="bol-bulk" type="checkbox" className="mt-0.5" checked={form.bundleEnabled} aria-describedby="bol-bulk-hint"
          onChange={e => onChange(setBundleEnabled(form, e.target.checked))}
        />
        <div>
          <label htmlFor="bol-bulk" className="text-xs text-ink-2">{t('bol.pricing.bulk')}</label>
          <p id="bol-bulk-hint" className="text-2xs text-ink-4">{t('bol.pricing.bulkHint')}</p>
        </div>
      </div>

      {form.bundleEnabled && (
        <div className="space-y-2">
          {form.tiers.map((row, i) => {
            const quantity = parseInt(row.quantity, 10)
            const auto = levelFor(quantity)
            return (
              <div key={i} className="flex items-end gap-2">
                <div className="w-24">
                  <label htmlFor={`bol-tier-qty-${i}`} className={labelClass}>{t('bol.pricing.fromUnits')}</label>
                  <input
                    id={`bol-tier-qty-${i}`} aria-label={`${t('bol.pricing.fromUnits')} ${i + 1}`} inputMode="numeric"
                    value={row.quantity} onChange={e => setTier(i, { quantity: e.target.value })} className={inputClass}
                  />
                </div>
                <div className="flex-1">
                  <label htmlFor={`bol-tier-price-${i}`} className={labelClass}>{t('bol.pricing.priceEach')}</label>
                  <input
                    id={`bol-tier-price-${i}`} aria-label={`${t('bol.pricing.priceEach')} ${i + 1}`} inputMode="decimal"
                    value={row.price} onChange={e => setTier(i, { price: e.target.value })}
                    placeholder={auto ? money(auto.unit_price_minor_units) : ''} className={inputClass}
                  />
                </div>
                <button
                  type="button" onClick={() => removeTier(i)} aria-label={t('bol.pricing.removeLevel')}
                  className="p-2 rounded-lg text-ink-4 hover:text-danger hover:bg-sunken transition-colors"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            )
          })}
          <button
            type="button" onClick={() => onChange(addTierRow(form))}
            className="inline-flex items-center gap-1 text-2xs text-accent hover:underline"
          >
            <Plus size={12} /> {t('bol.pricing.addLevel')}
          </button>
          <p className="text-2xs text-ink-4">{t('bol.pricing.levelsHint')}</p>
        </div>
      )}

      {preview.error && <p className="text-2xs text-danger">{preview.error}</p>}
      {!preview.error && preview.loading && !p && <p className="text-2xs text-ink-4">{t('bol.pricing.calculating')}</p>}
      {!preview.error && p && p.levels.length > 0 && <PriceLevelsTable levels={p.levels} />}
    </div>
  )
}

function PriceLevelsTable({ levels }: { levels: BolPricingLevel[] }) {
  const { t } = useTranslation(['pms'])
  const headers = [
    t('bol.pricing.colQuantity'), t('bol.pricing.colPrice'), t('bol.pricing.colVat'), t('bol.pricing.colFee'),
    t('bol.pricing.colShipping'), t('bol.pricing.colYouReceive'), t('bol.pricing.colShop'), t('bol.pricing.colProfit'),
  ]
  return (
    <div className="overflow-x-auto rounded-lg bg-sunken">
      <table className="w-full text-2xs tabular-nums">
        <thead>
          <tr className="text-ink-4 text-left">
            {headers.map(h => <th key={h} className="px-2 py-1.5 font-medium whitespace-nowrap">{h}</th>)}
          </tr>
        </thead>
        <tbody>
          {levels.map(l => (
            <tr key={l.quantity} className="border-t border-line text-ink-2">
              <td className="px-2 py-1.5 whitespace-nowrap">{l.quantity === 1 ? '1' : `${l.quantity}+`}</td>
              <td className="px-2 py-1.5 whitespace-nowrap">
                <span className="font-semibold text-ink">{money(l.unit_price_minor_units)}</span>
                {l.auto && <span className="ml-1 text-ink-4">{t('bol.pricing.auto')}</span>}
              </td>
              <td className="px-2 py-1.5">{money(l.product_vat_minor_units)}</td>
              <td className="px-2 py-1.5">{money(l.bol_fee_incl_vat_minor_units)}</td>
              <td className="px-2 py-1.5">{money(l.shipping_share_minor_units)}</td>
              <td className={clsx('px-2 py-1.5 whitespace-nowrap', l.below_regular ? 'text-warning' : 'text-success')}>
                {money(l.net_minor_units)}
                {l.below_regular && <span className="block">{t('bol.pricing.belowShop')}</span>}
              </td>
              <td className="px-2 py-1.5">{money(l.regular_net_minor_units)}</td>
              <td className="px-2 py-1.5">{l.profit_minor_units != null ? money(l.profit_minor_units) : '–'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
