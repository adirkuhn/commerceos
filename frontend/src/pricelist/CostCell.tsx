import { useState } from 'react'
import { Check, Pencil, X } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../api/apiErrorText'
import { fromMinorUnits, toMinorUnits } from '../collections/form'
import { Money } from '../format/amountVisibility'
import { setAutomaticCostPrice } from './api'
import type { PriceListRow } from './types'

type CostRow = Pick<PriceListRow, 'sku_id' | 'sku_code' | 'currency_code' | 'cost_price_minor_units'>

// The SKU's cost with a pencil to set it. Saved as the real SKU cost, but
// automatic: the next goods receipt still blends into it.
export function CostCell({ token, row, onSaved }: { token: string; row: CostRow; onSaved: () => void }) {
  const { t } = useTranslation('priceList')
  const [text, setText] = useState<string | null>(null)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')
  const cost = row.cost_price_minor_units

  function cancel() {
    setText(null)
    setError('')
  }

  async function save() {
    const minor = toMinorUnits(text ?? '')
    if (minor === null) {
      setError(t('cost.invalid'))
      return
    }
    setSaving(true)
    setError('')
    try {
      await setAutomaticCostPrice(token, row.sku_id, minor)
      setText(null)
      onSaved()
    } catch (e) {
      setError(apiErrorText(e) ?? t('cost.saveFailed'))
    } finally {
      setSaving(false)
    }
  }

  if (text === null) {
    return (
      <td className="px-3 py-2 text-right tabular-nums text-ink-3 group/cost">
        <span className="inline-flex items-center gap-1">
          {cost != null ? <Money minor={cost} currency={row.currency_code} /> : '—'}
          <button onClick={() => setText(fromMinorUnits(cost))} aria-label={t('cost.edit', { sku: row.sku_code })} title={t('cost.hint')}
            className="w-5 h-5 flex items-center justify-center rounded text-ink-4 hover:text-ink-2 hover:bg-hover opacity-60 group-hover/cost:opacity-100 focus:opacity-100">
            <Pencil size={11} />
          </button>
        </span>
      </td>
    )
  }

  return (
    <td className="px-3 py-2 text-right">
      <span className="inline-flex items-center gap-1">
        <input
          autoFocus
          value={text}
          inputMode="decimal"
          onChange={e => setText(e.target.value)}
          onKeyDown={e => {
            if (e.key === 'Enter') save()
            if (e.key === 'Escape') cancel()
          }}
          aria-label={t('cost.label', { sku: row.sku_code })}
          disabled={saving}
          className="w-20 py-1 px-2 text-xs rounded-md bg-raised border border-line text-ink outline-none focus:ring-1 focus:ring-focus tabular-nums text-right"
        />
        <button onClick={save} disabled={saving} aria-label={t('cost.save')}
          className="w-5 h-5 flex items-center justify-center rounded text-accent hover:bg-hover disabled:opacity-40"><Check size={12} /></button>
        <button onClick={cancel} disabled={saving} aria-label={t('cost.cancel')}
          className="w-5 h-5 flex items-center justify-center rounded text-ink-4 hover:bg-hover"><X size={12} /></button>
      </span>
      {error && <p role="alert" className="mt-1 text-2xs text-danger">{error}</p>}
    </td>
  )
}
