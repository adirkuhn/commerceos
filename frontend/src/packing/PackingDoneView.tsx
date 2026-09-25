import type { ReactNode } from 'react'
import { CheckCircle } from 'lucide-react'
import type { SalesOrder } from '../oms/types'
import type { MissingPackItem } from '../scanner/types'

interface PackingDoneViewProps {
  order: SalesOrder
  missingItems: MissingPackItem[] | undefined
  // The create/print label button, if any.
  labelAction: ReactNode
  onBack: () => void
}

// Shown once packing is complete: what (if anything) went short, the label,
// and the way back to the queue.
export function PackingDoneView({ order, missingItems, labelAction, onBack }: PackingDoneViewProps) {
  return (
    <div className="flex-1 flex flex-col items-center justify-center p-8">
      <CheckCircle size={40} className="text-success mb-3" />
      <p className="text-sm font-semibold text-ink">
        #{order.number || order.external_order_id} packed
      </p>
      {!!missingItems?.length && (
        <div className="mt-4 w-full max-w-sm rounded-lg border dark:border-amber-500/30 border-amber-300 dark:bg-amber-500/5 bg-amber-50/40 p-3">
          <p className="text-2xs font-medium text-warning mb-2">Not fully packed</p>
          {missingItems.map(m => (
            <p key={m.line_item_id} className="text-2xs text-ink-2">
              {m.name}: {m.packed_qty}/{m.expected_qty}
            </p>
          ))}
        </div>
      )}

      {labelAction}

      <button
        type="button" onClick={onBack}
        className="mt-3 px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover"
      >
        Back to orders
      </button>
    </div>
  )
}
