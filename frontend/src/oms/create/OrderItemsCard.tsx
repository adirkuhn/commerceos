import clsx from 'clsx'
import { Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Money } from '../../format/amountVisibility'
import { ProductPriceSearch } from '../../pricelist/ProductPriceSearch'
import { MoneyField } from './MoneyField'
import { addLine, type DraftLine } from './orderDraft'

const TH = 'py-2 px-2 text-2xs font-medium text-ink-4 caps-label'

export function OrderItemsCard({ token, lines, lineTotals, currency, onChange }: {
  token: string
  lines: DraftLine[]
  lineTotals: number[]
  currency: string
  onChange: (lines: DraftLine[]) => void
}) {
  const { t } = useTranslation(['orders', 'common'])
  const edit = (key: string, patch: Partial<DraftLine>) => onChange(lines.map(l => (l.key === key ? { ...l, ...patch } : l)))

  return (
    <section className="rounded-xl border border-line bg-surface p-4 space-y-3">
      <h3 className="text-xs font-semibold text-ink">{t('form.items')}</h3>
      <ProductPriceSearch token={token} onAdd={row => onChange(addLine(lines, row))} />
      {lines.length === 0 ? (
        <p className="py-6 text-center text-xs text-ink-4">{t('form.noItems')}</p>
      ) : (
        <table className="w-full text-xs">
          <thead className="border-b border-line">
            <tr>
              <th className={`${TH} text-left`}>{t('common:fields.product')}</th>
              <th className={`${TH} text-right`}>{t('form.stock')}</th>
              <th className={`${TH} text-right`}>{t('form.qty')}</th>
              <th className={`${TH} text-right`}>{t('form.unitPrice')}</th>
              <th className={`${TH} text-right`}>{t('form.lineTotal')}</th>
              <th className="w-8" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {lines.map((l, i) => {
              const over = l.quantity > l.available
              return (
                <tr key={l.key} data-testid={`order-line-${l.skuId}`}>
                  <td className="py-2 px-2">
                    <p className="text-ink font-medium">{l.label}</p>
                    <p className="text-2xs font-mono text-ink-4">{l.skuCode}</p>
                  </td>
                  <td className={clsx('py-2 px-2 text-right tabular-nums', over ? 'text-warning font-medium' : 'text-ink-3')}
                    title={over ? t('form.overStock', { available: l.available }) : undefined}>{l.available}</td>
                  <td className="py-2 px-2 text-right">
                    <input type="number" min={1} step={1} value={l.quantity} aria-label={`${t('form.qty')} ${l.skuCode}`}
                      onChange={e => edit(l.key, { quantity: Math.max(1, Math.floor(Number(e.target.value) || 1)) })}
                      className={clsx('w-16 px-2 py-1.5 text-xs text-right tabular-nums rounded-lg bg-raised border text-ink', over ? 'border-warning' : 'border-line')} />
                  </td>
                  <td className="py-2 px-2 text-right">
                    <MoneyField value={l.unitPrice} onChange={p => edit(l.key, { unitPrice: p })} label={`${t('form.unitPrice')} ${l.skuCode}`} />
                  </td>
                  <td className="py-2 px-2 text-right tabular-nums font-medium text-ink"><Money minor={lineTotals[i] ?? 0} currency={currency} /></td>
                  <td className="py-2 px-1 text-right">
                    <button type="button" onClick={() => onChange(lines.filter(x => x.key !== l.key))} aria-label={t('form.remove', { name: l.label })}
                      className="p-1 rounded hover:bg-hover text-ink-4 hover:text-danger"><Trash2 size={12} /></button>
                  </td>
                </tr>
              )
            })}
          </tbody>
        </table>
      )}
    </section>
  )
}
