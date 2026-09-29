import type { ReactNode } from 'react'
import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SalesOrder } from '../oms/types'
import type { PackingSessionDetail } from '../scanner/types'
import { computeMissingPackItems } from '../scanner/packingProgress'
import { customerName, shippingAddressLine } from '../oms/salesOrderDisplay'

interface PackingReviewViewProps {
  order: SalesOrder
  session: PackingSessionDetail
  error: string
  completing: boolean
  // The create/print label button, if any.
  labelAction: ReactNode
  onConfirm: () => void
  onBack: () => void
}

// "Review before completing": which lines aren't fully packed yet, then
// confirm (or go back to packing).
export function PackingReviewView({ order, session, error, completing, labelAction, onConfirm, onBack }: PackingReviewViewProps) {
  const { t } = useTranslation('packing')
  const missing = computeMissingPackItems(session.line_items, session.scans)
  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="flex items-center gap-3 px-6 py-4 border-b border-line shrink-0">
        <button aria-label={t('review.back')} onClick={onBack} className="p-1 -ml-1" title={t('review.back')}>
          <ArrowLeft size={18} className="text-ink-3" />
        </button>
        <h2 className="text-sm font-semibold text-ink flex-1">{t('review.title')}</h2>
      </div>
      <div className="px-6 py-3 border-b border-line shrink-0">
        <p className="text-sm font-medium text-ink">
          #{order.number || order.external_order_id} · {customerName(order)}
        </p>
        {shippingAddressLine(order.shipping) && (
          <p className="text-xs text-ink-4 mt-0.5">{shippingAddressLine(order.shipping)}</p>
        )}
      </div>
      <div className="flex-1 overflow-y-auto px-6 py-4 space-y-2">
        {missing.length === 0 ? (
          <p className="text-sm text-ink-3">{t('review.allPacked')}</p>
        ) : (
          <>
            <p className="text-sm text-warning">{t('review.notFullyPacked', { count: missing.length })}</p>
            {missing.map(m => (
              <div key={m.line_item_id} className="rounded-lg bg-raised border border-line px-3 py-2">
                <p className="text-sm text-ink">{m.name}</p>
                <p className="text-xs text-ink-4">{t('review.expectedPacked', { expected: m.expected_qty, packed: m.packed_qty })}</p>
              </div>
            ))}
          </>
        )}
      </div>
      <div className="p-6 border-t border-line shrink-0 space-y-2">
        {error && <p className="text-sm text-danger">{error}</p>}
        {labelAction}
        <button
          onClick={onConfirm} disabled={completing}
          className="w-full py-3 rounded-xl bg-green-600 text-white font-semibold hover:bg-green-700 disabled:opacity-50"
        >
          {completing ? t('review.completing') : t('review.confirm')}
        </button>
        <button
          onClick={onBack}
          className="w-full py-2.5 rounded-xl dark:bg-night-800 bg-gray-100 text-ink-2 text-sm font-medium"
        >
          {t('review.back')}
        </button>
      </div>
    </div>
  )
}
