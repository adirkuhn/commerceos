import { CheckCircle, PackageCheck } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SalesOrder } from '../oms/types'
import type { CompletePickingResult } from '../scanner/types'

interface PickingCompleteResultProps {
  order: SalesOrder
  result: CompletePickingResult
  onContinueToPacking: () => void
  onBack: () => void
}

// Shown once picking is completed: what (if anything) wasn't fully picked,
// and the hand-off to packing.
export function PickingCompleteResult({ order, result, onContinueToPacking, onBack }: PickingCompleteResultProps) {
  const { t } = useTranslation('picking')
  return (
    <div className="flex-1 flex flex-col items-center justify-center p-8">
      <CheckCircle size={40} className="text-success mb-3" />
      <p className="text-sm font-semibold text-ink">
        {t('done.fulfilled', { number: order.number || order.external_order_id })}
      </p>
      {!!result.missing_items?.length && (
        <div className="mt-4 w-full max-w-sm rounded-lg border dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40 p-3">
          <p className="text-2xs font-medium text-warning mb-2">{t('done.notFullyPicked')}</p>
          {result.missing_items.map(m => (
            <p key={m.line_item_id} className="text-2xs text-ink-2">
              {m.name}: {m.picked_qty}/{m.expected_qty}
            </p>
          ))}
        </div>
      )}

      <button
        type="button" onClick={onContinueToPacking}
        className="mt-5 px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover flex items-center gap-1.5"
      >
        <PackageCheck size={13} /> {t('done.continue')}
      </button>

      <button
        type="button" onClick={onBack}
        className="mt-3 px-4 py-2 rounded-lg dark:bg-night-800 bg-gray-100 text-ink-2 text-xs font-medium hover:bg-hover transition-colors"
      >
        {t('done.backToOrders')}
      </button>
    </div>
  )
}
