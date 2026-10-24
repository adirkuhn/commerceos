import { CornerDownRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../../format/DateFormatProvider'
import { Money } from '../../format/amountVisibility'
import type { SalesOrderLineChange } from '../types'
import { changeProductLabel, isExtraItem } from './lineChanges'

// What picking/packing did to one order line, shown under it: replaced by
// another product, refunded in part, or added as an extra item.
interface LineChangeNotesProps {
  changes: SalesOrderLineChange[]
  currency: string
}

export function LineChangeNotes({ changes, currency }: LineChangeNotesProps) {
  if (changes.length === 0) return null
  return (
    <ul className="mt-1 space-y-1">
      {changes.map(c => (
        <li key={c.id}>
          <LineChangeNote change={c} currency={currency} />
        </li>
      ))}
    </ul>
  )
}

function LineChangeNote({ change: c, currency }: { change: SalesOrderLineChange; currency: string }) {
  const { t } = useTranslation('orders')
  const { formatDateTime } = useDateFormat()
  const meta = [
    !isExtraItem(c) && c.reason ? `“${c.reason}”` : '',
    formatDateTime(c.at),
    c.step === 'picking' ? t('lineChanges.atPicking') : '',
    c.action === 'substitute' && c.return_to_stock ? t('lineChanges.backToStock') : '',
  ].filter(Boolean).join(' · ')

  if (isExtraItem(c)) {
    return (
      <p className="text-2xs text-ink-4">
        <span className="inline-block rounded bg-accent/10 text-accent px-1.5 py-px font-medium mr-1.5">
          {t(c.action === 'freebie' ? 'lineChanges.extraFree' : 'lineChanges.extraCharged')}
        </span>
        {meta}
      </p>
    )
  }

  return (
    <div className="flex gap-1 text-2xs">
      <CornerDownRight className="w-3 h-3 mt-px shrink-0 text-ink-4" aria-hidden />
      <div>
        {c.action === 'substitute' ? (
          <p className="text-ink-2">
            {t('lineChanges.replacedBy', { product: changeProductLabel(c), quantity: c.quantity })}
            {c.sku_code && <span className="text-ink-4"> ({c.sku_code})</span>}
            {c.original_qty_covered != null && <span className="text-ink-4"> · {t('lineChanges.covers', { count: c.original_qty_covered })}</span>}
          </p>
        ) : (
          <p className="text-warning">
            {t('lineChanges.refunded', { quantity: c.quantity })}
            {c.amount_minor_units != null && <> (<Money minor={c.amount_minor_units} currency={currency} />)</>}
          </p>
        )}
        <p className="text-ink-4">{meta}</p>
      </div>
    </div>
  )
}

// Extras staff chose to ship without (no stock) -- never became a line.
export function NotAddedExtras({ changes }: { changes: SalesOrderLineChange[] }) {
  const { t } = useTranslation('orders')
  if (changes.length === 0) return null
  return (
    <ul className="space-y-0.5 text-2xs text-ink-4">
      {changes.map(c => (
        <li key={c.id}>{t('lineChanges.notAdded', { product: changeProductLabel(c), quantity: c.quantity })}</li>
      ))}
    </ul>
  )
}
