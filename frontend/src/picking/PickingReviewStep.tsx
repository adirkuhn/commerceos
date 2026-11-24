import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SalesOrder } from '../oms/types'
import type { PickingSessionDetail } from '../scanner/types'
import { computeMissingItems } from '../scanner/pickingMissingItems'
import { customerName, shippingAddressLine, shippingMethodLabel } from '../oms/salesOrderDisplay'

interface PickingReviewStepProps {
  order: SalesOrder
  session: PickingSessionDetail
  error: string
  completing: boolean
  onConfirm: () => void
  onBack: () => void
}

// The review step between "Complete" and actually completing: every line
// not fully picked, plus who/where it ships to.
export function PickingReviewStep({ order, session, error, completing, onConfirm, onBack }: PickingReviewStepProps) {
  const { t } = useTranslation('picking')
  const missing = computeMissingItems(session.line_items, session.scans)
  return (
    <div className="flex-1 flex flex-col overflow-hidden">
      <div className="flex items-center gap-3 px-6 py-4 border-b border-line shrink-0">
        <button aria-label={t('review.back')} onClick={onBack} className="p-1 -ml-1" title={t('review.back')}>
          <ArrowLeft size={18} className="text-ink-3" />
        </button>
        <h2 className="text-sm font-semibold text-ink flex-1">{t('review.title')}</h2>
      </div>
      {/* Customer/address stays visible here too, not just during
          active picking -- staff confirming completion should never
          have to go back just to double-check who this ships to. */}
      <div className="px-6 py-3 border-b border-line shrink-0">
        <p className="text-sm font-medium text-ink">
          #{order.number || order.external_order_id} · {customerName(order)}
        </p>
        {shippingAddressLine(order.shipping) && (
          <p className="text-xs text-ink-4 mt-0.5">{shippingAddressLine(order.shipping)}</p>
        )}
        {shippingMethodLabel(order) && (
          <p className="text-xs text-ink-4 mt-0.5">
            {t('review.shipping', { method: shippingMethodLabel(order) })}
          </p>
        )}
      </div>
      <div className="flex-1 overflow-y-auto px-6 py-4 space-y-2">
        {missing.length === 0 ? (
          <p className="text-sm text-ink-3">{t('review.allPicked')}</p>
        ) : (
          <>
            <p className="text-sm text-warning">{t('review.notFullyPicked', { count: missing.length })}</p>
            {missing.map(m => (
              <div key={m.line_item_id} className="rounded-lg bg-raised border border-line px-3 py-2">
                <p className="text-sm text-ink">{m.name}</p>
                <p className="text-xs text-ink-4">{t('review.expectedPicked', { expected: m.expected_qty, picked: m.picked_qty })}</p>
              </div>
            ))}
          </>
        )}
      </div>
      <div className="p-6 border-t border-line shrink-0 space-y-2">
        {error && <p className="text-sm text-danger">{error}</p>}
        <button
          onClick={onConfirm} disabled={completing}
          className="w-full py-3 rounded-xl bg-green-600 text-white font-semibold hover:bg-green-700 disabled:opacity-50"
        >
          {completing ? t('review.completing') : t('review.confirm')}
        </button>
        <button
          onClick={onBack}
          className="w-full py-2.5 rounded-xl bg-fill text-ink-2 text-sm font-medium"
        >
          {t('review.backToScanning')}
        </button>
      </div>
    </div>
  )
}
