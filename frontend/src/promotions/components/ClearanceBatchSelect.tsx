import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { TFunction } from 'i18next'
import { listBatches } from '../../inventory/api'
import type { Batch } from '../../inventory/types'
import type { DateInput } from '../../format/date'
import { useDateFormat } from '../../format/DateFormatProvider'

interface ClearanceBatchSelectProps {
  token: string
  skuId: string
  value: string | null
  onChange: (batchId: string | null) => void
}

// isSellable mirrors the backend's rule for which batches can start a
// clearance: stock left and expiry date still in the future.
function isSellable(b: Batch, now: Date): boolean {
  if (b.quantity <= 0) return false
  return !b.expiry_date || new Date(b.expiry_date) > now
}

function batchLabel(t: TFunction<'promotions'>, b: Batch, formatDate: (value: DateInput) => string): string {
  const lot = b.lot_number || t('batch.noLot')
  const expiry = b.expiry_date ? t('batch.expires', { date: formatDate(b.expiry_date) }) : t('batch.noExpiry')
  return t('batch.option', { lot, expiry, quantity: b.quantity })
}

// ClearanceBatchSelect turns a promotion line into a clearance line for one
// of its SKU's batches. The line then ends by itself once that batch is
// sold out, discarded, removed or expired.
export function ClearanceBatchSelect({ token, skuId, value, onChange }: ClearanceBatchSelectProps) {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation('promotions')
  const [batches, setBatches] = useState<Batch[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setError('')
    listBatches(token, skuId)
      .then(rows => { if (!cancelled) setBatches(rows) })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : t('batch.loadFailed')) })
    return () => { cancelled = true }
  }, [token, skuId])

  const now = new Date()
  // The line's current batch stays listed even once finished, so editing a
  // running or ended clearance never silently drops it.
  const options = batches.filter(b => !b.is_general_ledger && (b.id === value || isSellable(b, now)))

  return (
    <div className="min-w-0">
      <label htmlFor={`batch-${skuId}`} className="sr-only">{t('batch.label')}</label>
      <select
        id={`batch-${skuId}`}
        value={value ?? ''}
        onChange={e => onChange(e.target.value === '' ? null : e.target.value)}
        className="w-full px-2 py-2 text-xs rounded-lg dark:bg-night-800 bg-white border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
      >
        <option value="">{t('batch.wholeSku')}</option>
        {options.map(b => <option key={b.id} value={b.id}>{batchLabel(t, b, formatDate)}</option>)}
      </select>
      {error && <p className="text-2xs text-danger mt-1">{error}</p>}
    </div>
  )
}
