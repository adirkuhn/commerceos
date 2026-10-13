import { useTranslation } from 'react-i18next'
import { formatMoney } from '../../format/money'
import type { Profit } from '../../pms/profit'
import type { ClearanceProfit, PriceProfit } from './clearance'

const pct = (basisPoints: number) => `${(basisPoints / 100).toFixed(1)}%`

// What a clearance still makes: the lot's cost, then profit · margin ·
// markup at the regular and the sale price -- excl. and incl. VAT when the
// rate is known (same pair as the product page's margin panel), else at the
// price as stored. Amounts unmasked, like the rest of the dialog.
export function ClearanceProfitPanel({ profit, regularPrice, salePrice, quantity, currency }: {
  profit: ClearanceProfit
  regularPrice: number
  salePrice: number | null
  quantity: number
  currency: string
}) {
  const { t } = useTranslation('inventory')
  const money = (minor: number) => formatMoney(minor, currency)
  const cell = (p: Profit) => `${money(p.profit)} · ${pct(p.marginBasisPoints)} · ${pct(p.markupBasisPoints)}`
  const withVat = profit.regular.exclusive !== null

  const row = (label: string, price: number, p: PriceProfit) => (
    <tr>
      <th scope="row" className="text-left font-normal text-ink-3 pr-2 py-0.5">{label} <span className="tabular-nums text-ink">{money(price)}</span></th>
      {withVat && p.exclusive && p.inclusive ? (
        <>
          <td className={profitClass(p.exclusive)}>{cell(p.exclusive)}</td>
          <td className={profitClass(p.inclusive)}>{cell(p.inclusive)}</td>
        </>
      ) : (
        <td className={profitClass(p.stored)}>{cell(p.stored)}</td>
      )}
    </tr>
  )

  const lossPerUnit = profit.sale ? -(profit.sale.exclusive ?? profit.sale.stored).profit : 0
  return (
    <section aria-label={t('closeToExpiry.profit.title')} className="rounded-lg bg-fill px-3 py-2.5 space-y-2 text-xs">
      <p className="text-ink-3">
        {t('closeToExpiry.profit.cost')} <span className="font-medium text-ink tabular-nums">{money(profit.cost)}</span>{' '}
        <span className="text-ink-4">{t(`closeToExpiry.profit.source.${profit.costSource}`)}</span>
      </p>
      <table className="w-full">
        <thead>
          <tr className="text-2xs text-ink-4">
            <th />
            {withVat ? (
              <>
                <th scope="col" className="text-left font-normal">{t('closeToExpiry.profit.exclVat')}</th>
                <th scope="col" className="text-left font-normal">{t('closeToExpiry.profit.inclVat')}</th>
              </>
            ) : (
              <th scope="col" className="text-left font-normal">{t('closeToExpiry.profit.columns')}</th>
            )}
          </tr>
        </thead>
        <tbody>
          {row(t('closeToExpiry.profit.regular'), regularPrice, profit.regular)}
          {profit.sale && salePrice !== null && row(t('closeToExpiry.profit.sale'), salePrice, profit.sale)}
        </tbody>
      </table>
      {profit.lotProfit !== null && !profit.belowCost && (
        <p className="text-ink-3 tabular-nums">
          {t(withVat ? 'closeToExpiry.profit.lotExclVat' : 'closeToExpiry.profit.lot', { count: quantity, profit: money(profit.lotProfit) })}
        </p>
      )}
      {profit.belowCost && profit.lotProfit !== null && (
        <p role="alert" className="text-danger font-medium tabular-nums">
          {t('closeToExpiry.profit.belowCost', { perUnit: money(lossPerUnit), lot: money(-profit.lotProfit) })}
        </p>
      )}
    </section>
  )
}

function profitClass(p: Profit): string {
  return p.profit < 0 ? 'tabular-nums text-danger py-0.5' : 'tabular-nums text-ink py-0.5'
}
