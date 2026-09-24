import { useEffect, useState } from 'react'
import { listBatches } from '../../inventory/api'
import type { Batch } from '../../inventory/types'

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

function batchLabel(b: Batch): string {
  const lot = b.lot_number || 'No lot'
  const expiry = b.expiry_date ? `exp ${new Date(b.expiry_date).toLocaleDateString()}` : 'no expiry'
  return `${lot} · ${expiry} · ${b.quantity} left`
}

// ClearanceBatchSelect turns a promotion line into a clearance line for one
// of its SKU's batches. The line then ends by itself once that batch is
// sold out, discarded, removed or expired.
export function ClearanceBatchSelect({ token, skuId, value, onChange }: ClearanceBatchSelectProps) {
  const [batches, setBatches] = useState<Batch[]>([])
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setError('')
    listBatches(token, skuId)
      .then(rows => { if (!cancelled) setBatches(rows) })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load batches') })
    return () => { cancelled = true }
  }, [token, skuId])

  const now = new Date()
  // The line's current batch stays listed even once finished, so editing a
  // running or ended clearance never silently drops it.
  const options = batches.filter(b => !b.is_general_ledger && (b.id === value || isSellable(b, now)))

  return (
    <div className="min-w-0">
      <label htmlFor={`batch-${skuId}`} className="sr-only">Clearance batch</label>
      <select
        id={`batch-${skuId}`}
        value={value ?? ''}
        onChange={e => onChange(e.target.value === '' ? null : e.target.value)}
        className="w-full px-2 py-2 text-xs rounded-lg dark:bg-night-800 bg-white border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
      >
        <option value="">Whole SKU (no batch)</option>
        {options.map(b => <option key={b.id} value={b.id}>{batchLabel(b)}</option>)}
      </select>
      {error && <p className="text-2xs text-danger mt-1">{error}</p>}
    </div>
  )
}
