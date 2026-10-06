import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Supplier } from '../suppliers/types'
import { SELECT } from './PlanControls'
import type { PlanOrderLine } from './types'

interface Props {
  // Null for products never bought on a PO -- a supplier is picked here.
  supplierId: string | null
  suppliers: Pick<Supplier, 'id' | 'name'>[]
  lines: PlanOrderLine[]
  onCreateOrder: (supplierId: string, lines: PlanOrderLine[]) => Promise<void>
}

export function OrderActions({ supplierId, suppliers, lines, onCreateOrder }: Props) {
  const { t } = useTranslation('purchasePlanning')
  const [picked, setPicked] = useState('')
  const [creating, setCreating] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const orderFrom = supplierId ?? picked
  const pickerId = `order-from-${supplierId ?? 'unsourced'}`

  async function create() {
    setCreating(true)
    setError(null)
    try {
      await onCreateOrder(orderFrom, lines)
    } catch (e) {
      setError(e instanceof Error ? e.message : t('errors.unknown'))
    } finally {
      setCreating(false)
    }
  }

  return (
    <div className="basis-full flex flex-wrap items-center justify-end gap-2 pt-1">
      {error && <span role="alert" className="mr-auto text-xs text-danger">{error}</span>}
      {!supplierId && (
        <label htmlFor={pickerId} className="flex items-center gap-1.5 text-xs text-ink-3">
          {t('order.orderFrom')}
          <select id={pickerId} className={SELECT} value={picked} onChange={e => setPicked(e.target.value)}>
            <option value="">{t('order.pickSupplier')}</option>
            {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
          </select>
        </label>
      )}
      <button
        onClick={create}
        disabled={creating || !orderFrom || lines.length === 0}
        className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors disabled:opacity-50"
      >
        {creating ? t('order.creating') : t('order.create')}
      </button>
    </div>
  )
}
