import type { ReactNode } from 'react'
import { ArrowLeft } from 'lucide-react'
import type { SalesOrder } from '../oms/types'
import type { MissingPackItem } from './types'
import { ScanSessionOrderBand } from './ScanSessionOrderBand'
import { MissingLineCard } from './ScanSessionLines'

interface PackingReviewScreenProps {
  order: SalesOrder
  missing: MissingPackItem[]
  error: string
  finishing: boolean
  labelButton: ReactNode
  onBack: () => void
  onConfirm: () => void
  // Overlays: the stock-shortfall dialog and the label modals.
  children?: ReactNode
}

// "Review before finishing": what isn't fully packed yet, the label, and
// the confirm button.
export function PackingReviewScreen({ order, missing, error, finishing, labelButton, onBack, onConfirm, children }: PackingReviewScreenProps) {
  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex items-center gap-3 px-4 py-4 border-b border-night-700 shrink-0">
        <button aria-label="Back to packing" onClick={onBack} className="w-11 h-11 -ml-3 -my-2 flex items-center justify-center rounded-lg shrink-0" title="Back to packing">
          <ArrowLeft size={18} className="text-gray-400" />
        </button>
        <h1 className="flex-1 text-base font-semibold">Review before finishing</h1>
      </div>
      <ScanSessionOrderBand order={order} />
      <div className="flex-1 overflow-y-auto px-4 py-3 space-y-2">
        {missing.length === 0 ? (
          <p className="text-center text-sm text-gray-400 py-8">Every line is fully packed.</p>
        ) : (
          <>
            <p className="text-sm text-amber-400">{missing.length} line(s) aren't fully packed yet:</p>
            {missing.map(m => (
              <MissingLineCard key={m.line_item_id} activity="packing" name={m.name} expectedQty={m.expected_qty} actualQty={m.packed_qty} />
            ))}
          </>
        )}
      </div>
      <div className="p-4 border-t border-night-700 shrink-0 space-y-2">
        {error && <p className="text-sm text-red-400">{error}</p>}
        {labelButton}
        <button
          onClick={onConfirm}
          disabled={finishing}
          className="w-full py-3 rounded-xl bg-green-600 text-white font-semibold hover:bg-green-700 disabled:opacity-50"
        >
          {finishing ? 'Completing…' : 'Confirm & complete'}
        </button>
        <button onClick={onBack} className="w-full py-2.5 rounded-xl bg-night-800 text-gray-300 text-sm font-medium">
          Back, scan more
        </button>
      </div>
      {children}
    </div>
  )
}
