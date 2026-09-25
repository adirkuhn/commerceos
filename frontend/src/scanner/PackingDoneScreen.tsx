import type { ReactNode } from 'react'
import { PackageCheck } from 'lucide-react'
import type { MissingPackItem } from './types'
import { MissingLineCard } from './ScanSessionLines'

interface PackingDoneScreenProps {
  missingItems?: MissingPackItem[]
  labelButton: ReactNode
  onDone: () => void
  // Overlays: the label modals.
  children?: ReactNode
}

// "Order packed": what wasn't fully packed, the label, and Done.
export function PackingDoneScreen({ missingItems, labelButton, onDone, children }: PackingDoneScreenProps) {
  return (
    <div className="fixed inset-0 flex flex-col bg-night-950 text-gray-100">
      <div className="flex flex-col items-center text-center gap-3 px-6 pt-8 pb-4 shrink-0">
        <div className="w-14 h-14 rounded-2xl bg-green-600/20 flex items-center justify-center">
          <PackageCheck size={26} className="text-green-400" />
        </div>
        <h1 className="text-lg font-semibold">Order packed</h1>
        {!missingItems?.length && <p className="text-sm text-gray-400">Every line was fully packed.</p>}
      </div>
      {!!missingItems?.length && (
        <div className="flex-1 overflow-y-auto px-6 pb-4 space-y-2">
          <p className="text-sm text-amber-400">{missingItems.length} line(s) weren't fully packed:</p>
          {missingItems.map(m => (
            <MissingLineCard key={m.line_item_id} activity="packing" name={m.name} expectedQty={m.expected_qty} actualQty={m.packed_qty} />
          ))}
        </div>
      )}
      <div className="p-6 shrink-0 space-y-2">
        {labelButton}
        <button onClick={onDone} className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold hover:bg-accent-solid-hover">
          Done
        </button>
      </div>

      {children}
    </div>
  )
}
