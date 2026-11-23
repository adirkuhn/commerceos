import { useTranslation } from 'react-i18next'
import { Money } from '../format/amountVisibility'
import { ProposalLinesTable, ShortfallList } from './ProposalLines'
import type { ConversionProposalPreview, ProposalCustomer, QuoteProposalPreview } from './types'

function Customer({ customer }: { customer?: ProposalCustomer | null }) {
  const { t } = useTranslation('assistant')
  return (
    <p data-testid="proposal-customer" className="text-xs text-ink-2">
      <span className="text-ink-4">{t('proposal.customer')}: </span>
      {customer ? (
        <>
          <span className="text-ink">{customer.name}</span>
          {customer.email && <span className="text-ink-4"> · {customer.email}</span>}
        </>
      ) : t('proposal.noCustomer')}
    </p>
  )
}

function Total({ minor, currency }: { minor: number; currency: string }) {
  const { t } = useTranslation('assistant')
  return (
    <p className="flex justify-end gap-2 text-xs">
      <span className="text-ink-4">{t('proposal.total')}</span>
      <span data-testid="proposal-total" className="font-semibold tabular-nums text-ink"><Money minor={minor} currency={currency} /></span>
    </p>
  )
}

export function QuotePreview({ preview }: { preview: QuoteProposalPreview }) {
  const { t } = useTranslation('assistant')
  return (
    <div className="flex flex-col gap-2">
      <p className="text-sm font-medium text-ink">{preview.name}</p>
      <Customer customer={preview.customer} />
      <p className="flex flex-wrap gap-x-3 text-2xs text-ink-4">
        {preview.discount_basis_points > 0 && <span>{t('proposal.discount', { percent: preview.discount_basis_points / 100 })}</span>}
        <span>{preview.prices_include_tax ? t('proposal.pricesInclTax') : t('proposal.pricesExclTax')}</span>
        <span>{preview.currency_code}</span>
      </p>
      <ProposalLinesTable lines={preview.lines} currency={preview.currency_code} showListPrice />
      <Total minor={preview.total_minor_units} currency={preview.currency_code} />
    </div>
  )
}

// The conversion's warehouse is picked here; the card sends the choice
// with the confirm.
export function ConversionPreview({ preview, warehouseId, onWarehouse, editable }: {
  preview: ConversionProposalPreview
  warehouseId: string
  onWarehouse: (id: string) => void
  editable: boolean
}) {
  const { t } = useTranslation('assistant')
  const id = `proposal-warehouse-${preview.quote_id}`
  return (
    <div className="flex flex-col gap-2">
      <p className="text-sm font-medium text-ink">{preview.name}</p>
      <Customer customer={preview.customer} />
      <ProposalLinesTable lines={preview.lines} currency={preview.currency_code} showListPrice={false} />
      <Total minor={preview.total_minor_units} currency={preview.currency_code} />
      <div className="flex items-center gap-2">
        <label htmlFor={id} className="text-2xs text-ink-4 caps-label">{t('proposal.warehouse')}</label>
        <select
          id={id} value={warehouseId} onChange={e => onWarehouse(e.target.value)} disabled={!editable}
          className="px-2 py-1 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-focus disabled:opacity-60"
        >
          {preview.warehouses.map(w => <option key={w.id} value={w.id}>{w.name}</option>)}
        </select>
      </div>
      {warehouseId === preview.warehouse_id
        ? <ShortfallList shortfall={preview.shortfall} />
        : <p className="text-2xs text-ink-4">{t('proposal.stockCheckedFor', { warehouse: preview.warehouses.find(w => w.id === preview.warehouse_id)?.name ?? '' })}</p>}
    </div>
  )
}
