import { useTranslation } from 'react-i18next'
import { Money } from '../../format/amountVisibility'
import { MoneyField } from './MoneyField'
import type { DraftTotals } from './orderDraft'

export function OrderSummaryCard({ totals, currency, pricesIncludeTax, shipping, feeLabel, fee, onShipping, onFeeLabel, onFee }: {
  totals: DraftTotals
  currency: string
  pricesIncludeTax: boolean
  shipping: number
  feeLabel: string
  fee: number
  onShipping: (minor: number) => void
  onFeeLabel: (label: string) => void
  onFee: (minor: number) => void
}) {
  const { t } = useTranslation('orders')
  const row = 'flex items-center justify-between gap-3 text-xs'
  return (
    <section className="rounded-xl border border-line bg-surface p-4 space-y-2.5">
      <h3 className="text-xs font-semibold text-ink">{t('form.summary')}</h3>
      <div className={row}><span className="text-ink-3">{t('form.subtotal')}</span><span data-testid="order-subtotal" className="tabular-nums text-ink"><Money minor={totals.subtotal} currency={currency} /></span></div>
      <div className={row}><span className="text-ink-3">{t('form.shipping')}</span><MoneyField value={shipping} onChange={onShipping} label={t('form.shipping')} /></div>
      <div className="space-y-1.5">
        <input value={feeLabel} onChange={e => onFeeLabel(e.target.value)} placeholder={t('form.feeLabelPlaceholder')} aria-label={t('form.feeLabel')} maxLength={100}
          className="w-full px-2.5 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus" />
        <div className={row}><span className="text-ink-3">{feeLabel.trim() || t('form.feeLabel')}</span><MoneyField value={fee} onChange={onFee} label={t('form.fee')} /></div>
      </div>
      <div className={row}>
        <span className="text-ink-3">{pricesIncludeTax ? t('form.vatIncluded') : t('form.vatAdded')}</span>
        <span data-testid="order-tax" className="tabular-nums text-ink-3"><Money minor={totals.tax} currency={currency} /></span>
      </div>
      <div className={`${row} pt-2.5 border-t border-line`}>
        <span className="text-sm font-semibold text-ink">{t('form.total')}</span>
        <span data-testid="order-total" className="text-sm font-semibold tabular-nums text-ink"><Money minor={totals.total} currency={currency} /></span>
      </div>
    </section>
  )
}
