import { useTranslation } from 'react-i18next'
import { Money } from '../format/amountVisibility'
import type { ProposalLine, ProposalShortfall } from './types'

const TH = 'px-2 py-1.5 text-2xs font-medium text-ink-4 caps-label'
const TD = 'px-2 py-1.5 text-xs text-ink-2'

// "1 kg · FAR-1": the variant and SKU code that tell lines apart.
export function variantLabel(l: { variant_name: string; sku_code: string }): string {
  return [l.variant_name, l.sku_code].filter(Boolean).join(' · ')
}

function ProductCell({ line }: { line: { product_name: string; variant_name: string; sku_code: string } }) {
  const { t } = useTranslation('assistant')
  const sub = variantLabel(line)
  return (
    <td className={TD}>
      <span className="block text-ink">{line.product_name || t('proposal.unknownProduct')}</span>
      {sub && <span className="block text-2xs text-ink-4">{sub}</span>}
    </td>
  )
}

// The lines a proposal would create, at the prices it would use.
export function ProposalLinesTable({ lines, currency, showListPrice }: { lines: ProposalLine[]; currency: string; showListPrice: boolean }) {
  const { t } = useTranslation('assistant')
  return (
    <div className="overflow-x-auto">
      <table className="w-full">
        <thead>
          <tr className="border-b border-line">
            <th className={`${TH} text-left`}>{t('proposal.col.product')}</th>
            <th className={`${TH} text-right`}>{t('proposal.col.quantity')}</th>
            {showListPrice && <th className={`${TH} text-right`}>{t('proposal.col.listPrice')}</th>}
            <th className={`${TH} text-right`}>{t('proposal.col.unitPrice')}</th>
            <th className={`${TH} text-right`}>{t('proposal.col.lineTotal')}</th>
          </tr>
        </thead>
        <tbody>
          {lines.map(l => (
            <tr key={l.sku_id} className="border-b border-line last:border-0">
              <ProductCell line={l} />
              <td className={`${TD} text-right tabular-nums`}>{l.quantity}</td>
              {showListPrice && (
                <td className={`${TD} text-right tabular-nums text-ink-4`}>
                  {l.list_price_minor_units != null && <Money minor={l.list_price_minor_units} currency={currency} />}
                </td>
              )}
              <td className={`${TD} text-right tabular-nums`}><Money minor={l.unit_price_minor_units} currency={currency} /></td>
              <td className={`${TD} text-right tabular-nums`}><Money minor={l.line_total_minor_units} currency={currency} /></td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}

// Lines the warehouse can't fully cover.
export function ShortfallList({ shortfall }: { shortfall: ProposalShortfall[] }) {
  const { t } = useTranslation('assistant')
  if (shortfall.length === 0) return null
  return (
    <div data-testid="proposal-shortfall" className="rounded-lg border border-amber-500/40 bg-amber-50 dark:bg-amber-500/10 px-3 py-2">
      <p className="text-xs font-medium text-warning">{t('proposal.shortfallTitle')}</p>
      <ul className="mt-1 flex flex-col gap-0.5">
        {shortfall.map(s => (
          <li key={s.sku_id} className="text-xs text-ink-2">
            <span className="text-ink">{s.product_name || t('proposal.unknownProduct')}</span>
            {variantLabel(s) && <span className="text-ink-4"> ({variantLabel(s)})</span>}
            {': '}
            {t('proposal.shortfallLine', { requested: s.requested, available: s.available, short: s.short })}
          </li>
        ))}
      </ul>
    </div>
  )
}
