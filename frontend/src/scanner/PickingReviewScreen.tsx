import { ArrowLeft } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { SalesOrder } from '../oms/types'
import type { MissingPickItem } from './types'
import { ScanSessionOrderBand } from './ScanSessionOrderBand'
import { MissingLineCard } from './ScanSessionLines'
import { PickingBillingLine, PickingShippingMethodLine } from './PickingOrderDetails'

interface PickingReviewScreenProps {
  order: SalesOrder
  missing: MissingPickItem[]
  error: string
  finishing: boolean
  onBack: () => void
  onConfirm: () => void
}

// "Review before finishing": what isn't fully picked yet, and the confirm
// button.
export function PickingReviewScreen({ order, missing, error, finishing, onBack, onConfirm }: PickingReviewScreenProps) {
  const { t } = useTranslation('scanner')
  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label={t('review.backToPicking')} onClick={onBack} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title={t('review.backToPicking')}>
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">{t('review.title')}</h1>
      </div>
      {/* Customer/address stays visible here too, not just during active
          picking -- staff confirming completion should never have to go
          back just to double-check who this ships to (or create the label
          below without knowing the destination). */}
      <ScanSessionOrderBand order={order}>
        <PickingBillingLine order={order} />
        <PickingShippingMethodLine order={order} />
      </ScanSessionOrderBand>
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {missing.length === 0 ? (
          <p className="text-center text-sm text-gray-400 py-8">{t('review.allPicked')}</p>
        ) : (
          <>
            <p className="text-sm text-amber-400">{t('review.notFullyPicked', { count: missing.length })}</p>
            {missing.map(m => (
              <MissingLineCard key={m.line_item_id} activity="picking" name={m.name} expectedQty={m.expected_qty} actualQty={m.picked_qty} />
            ))}
          </>
        )}
      </div>
      <div className="p-4 border-t border-night-700 shrink-0 space-y-2">
        {error && <p className="text-sm text-red-400">{error}</p>}
        <button
          onClick={onConfirm}
          disabled={finishing}
          className="w-full py-3 rounded-xl bg-green-600 text-white font-semibold hover:bg-green-700 disabled:opacity-50"
        >
          {finishing ? t('review.completing') : t('review.confirm')}
        </button>
        <button onClick={onBack} className="w-full py-2.5 rounded-xl bg-night-800 text-gray-300 text-sm font-medium">
          {t('review.backScanMore')}
        </button>
      </div>
    </div>
  )
}
