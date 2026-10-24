import { useTranslation } from 'react-i18next'
import { Money } from '../../format/amountVisibility'
import { LineChangeNotes } from '../lineChanges/LineChangeNotes'
import { SoldExpiryNote } from '../SoldExpiryNote'
import type { SalesOrderLineChange, SalesOrderLineItem } from '../types'
import { lineMoney } from './lineMoney'

interface SalesOrderLinesTableProps {
  lines: SalesOrderLineItem[]
  changesByLine: Map<string, SalesOrderLineChange[]>
  currency: string
}

// The order's lines with their money without VAT, the VAT and with VAT --
// whichever way each line was stored -- and what picking/packing changed.
export function SalesOrderLinesTable({ lines, changesByLine, currency }: SalesOrderLinesTableProps) {
  const { t } = useTranslation(['orders', 'common'])
  const th = 'text-right py-1.5 font-medium text-ink-4 pl-2'
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-xs">
        <thead>
          <tr className="border-b border-line">
            <th className="text-left py-1.5 font-medium text-ink-4">{t('common:fields.product')}</th>
            <th className={th}>{t('detail.qty')}</th>
            <th className={th}>{t('detail.exclVat')}</th>
            <th className={th}>{t('detail.vat')}</th>
            <th className={th}>{t('detail.inclVat')}</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-line/60">
          {lines.map(li => {
            const money = lineMoney(li)
            return (
              <tr key={li.id}>
                <td className="py-1.5">
                  <p className="text-ink">{li.name}</p>
                  {li.sku_code && <p className="text-2xs text-ink-4">{li.sku_code}</p>}
                  {li.barcode && <p className="text-2xs text-ink-4">{li.barcode}</p>}
                  <SoldExpiryNote date={li.sold_expiry_date} />
                  {!li.sku_id && <p className="text-2xs text-warning">{t('detail.notLinked')}</p>}
                  <LineChangeNotes changes={changesByLine.get(li.id) ?? []} currency={currency} />
                </td>
                <td className="py-1.5 pl-2 text-right text-ink-2 align-top">
                  {li.quantity}
                  <p className="text-2xs text-ink-4 whitespace-nowrap">
                    <Money minor={money.unitInclTax} currency={currency} /> {t('detail.each')}
                  </p>
                </td>
                <td className="py-1.5 pl-2 text-right text-ink-2 align-top"><Money minor={money.exclTax} currency={currency} /></td>
                <td className="py-1.5 pl-2 text-right text-ink-3 align-top"><Money minor={money.tax} currency={currency} /></td>
                <td className="py-1.5 pl-2 text-right font-medium text-ink align-top"><Money minor={money.inclTax} currency={currency} /></td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}
