import clsx from 'clsx'
import { useEffect, useState } from 'react'
import { AlertTriangle, RotateCcw, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { fromMinorUnits, toMinorUnits } from '../collections/form'
import { Money } from '../format/amountVisibility'
import { useDateFormat } from '../format/DateFormatProvider'
import { unitEconomics } from './pricing'
import { belowTargetText, formatQuantity } from './format'
import type { PriceListRow, QuoteLine } from './types'

const INPUT = 'py-1 px-2 text-xs rounded-md bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus tabular-nums text-right'

export interface QuoteLineRowProps {
  line: QuoteLine
  row: PriceListRow
  discountedPrice: number
  pricesIncludeTax: boolean
  onQuantity: (quantity: number) => void
  onUnitPrice: (unitPrice: number | undefined) => void
  onRemove: () => void
}

export function QuoteLineRow({ line, row, discountedPrice, pricesIncludeTax, onQuantity, onUnitPrice, onRemove }: QuoteLineRowProps) {
  const { t } = useTranslation('priceList')
  const { formatDate } = useDateFormat()
  const currency = row.currency_code
  const unitPrice = line.unitPrice ?? discountedPrice
  const economics = unitEconomics(row, unitPrice, pricesIncludeTax)
  const overStock = line.quantity > row.available_quantity

  return (
    <tr data-testid={`quote-line-${row.sku_id}`} className={clsx(economics.belowTarget && 'bg-danger/10')}>
      <td className="px-3 py-2 max-w-[240px]">
        <p className="font-medium text-ink truncate">{row.product_name}{row.variant_name && <span className="text-ink-3"> — {row.variant_name}</span>}</p>
        <p className="text-2xs font-mono text-ink-4 truncate">{row.sku_code}</p>
      </td>
      <td className="px-3 py-2 text-ink-3 whitespace-nowrap">{row.earliest_expiry ? formatDate(row.earliest_expiry) : '—'}</td>
      <td className={clsx('px-3 py-2 text-right tabular-nums', overStock ? 'text-warning font-medium' : 'text-ink-2')}
        title={overStock ? t('overStock', { available: formatQuantity(row, row.available_quantity) }) : undefined}>
        {formatQuantity(row, row.available_quantity)}
      </td>
      <td className="px-3 py-2 text-right tabular-nums text-ink-3">
        {row.cost_price_minor_units != null ? <Money minor={row.cost_price_minor_units} currency={currency} /> : '—'}
      </td>
      <td className="px-3 py-2 text-right">
        <PriceInput value={unitPrice} edited={line.unitPrice !== undefined} onCommit={onUnitPrice}
          label={`${t('col.unitPrice')} ${row.sku_code}`} />
      </td>
      <td className="px-3 py-2 text-right">
        <input
          type="number" min={0} step={1} inputMode="numeric"
          value={line.quantity}
          onChange={e => onQuantity(Math.max(0, Math.floor(Number(e.target.value) || 0)))}
          aria-label={`${t('col.quantity')} ${row.sku_code}`}
          className={clsx(INPUT, 'w-20', overStock && 'border-warning')}
        />
      </td>
      <td className="px-3 py-2 text-right tabular-nums font-medium text-ink"><Money minor={unitPrice * line.quantity} currency={currency} /></td>
      <td className={clsx('px-3 py-2 text-right tabular-nums', economics.belowTarget ? 'text-danger font-medium' : 'text-ink-2')}>
        <span className="inline-flex items-center gap-1" title={economics.belowTarget ? belowTargetText(row, economics.profit) : undefined}>
          {economics.belowTarget && <AlertTriangle size={12} aria-label={belowTargetText(row, economics.profit)} />}
          {economics.profit != null ? <Money minor={economics.profit} currency={currency} /> : <span title={t('noCost')}>—</span>}
        </span>
      </td>
      <td className={clsx('px-3 py-2 text-right tabular-nums', economics.belowTarget ? 'text-danger' : 'text-ink-2')}>
        {economics.profit != null ? <Money minor={economics.profit * line.quantity} currency={currency} /> : '—'}
      </td>
      <td className="px-2 py-2">
        <button onClick={onRemove} aria-label={t('quote.remove', { name: row.sku_code })}
          className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover text-ink-4 hover:text-danger">
          <X size={12} />
        </button>
      </td>
    </tr>
  )
}

// Typed in major units, committed on blur/Enter; empty goes back to the
// discounted price.
function PriceInput({ value, edited, onCommit, label }: {
  value: number
  edited: boolean
  onCommit: (minor: number | undefined) => void
  label: string
}) {
  const { t } = useTranslation('priceList')
  const [text, setText] = useState(fromMinorUnits(value))
  useEffect(() => { setText(fromMinorUnits(value)) }, [value])

  function commit() {
    const minor = toMinorUnits(text)
    if (minor === null) {
      if (text.trim() === '') onCommit(undefined)
      else setText(fromMinorUnits(value))
      return
    }
    if (minor !== value) onCommit(minor)
  }

  return (
    <span className="inline-flex items-center gap-1">
      {edited && (
        <button onClick={() => onCommit(undefined)} aria-label={t('quote.resetPrice')} title={t('quote.resetPrice')}
          className="w-5 h-5 flex items-center justify-center rounded hover:bg-hover text-accent">
          <RotateCcw size={11} />
        </button>
      )}
      <input
        value={text}
        inputMode="decimal"
        onChange={e => setText(e.target.value)}
        onBlur={commit}
        onKeyDown={e => { if (e.key === 'Enter') e.currentTarget.blur() }}
        aria-label={label}
        title={edited ? t('quote.priceEdited') : undefined}
        className={clsx(INPUT, 'w-24', edited && 'border-accent')}
      />
    </span>
  )
}
