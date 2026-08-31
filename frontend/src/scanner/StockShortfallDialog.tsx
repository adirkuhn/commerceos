import { useState } from 'react'
import type { StockShortfall, StockShortfallResolution } from './types'

interface StockShortfallDialogProps {
  shortfalls: StockShortfall[]
  submitting: boolean
  onConfirm: (resolutions: Record<string, StockShortfallResolution>) => void
  onCancel: () => void
}

// Shown when finishing a picking session hits a substitute/freebie/
// new-item pick whose SKU doesn't have enough tracked stock (see backend's
// CompletePickingUseCase doc comment) -- staff decide, per item, whether to
// ship it anyway (fixes the inventory record, reduces stock for real) or
// ship without it (the line just doesn't go out, same as any other unpicked
// line).
export function StockShortfallDialog({ shortfalls, submitting, onConfirm, onCancel }: StockShortfallDialogProps) {
  const [choices, setChoices] = useState<Record<string, StockShortfallResolution>>(
    () => Object.fromEntries(shortfalls.map(s => [s.scan_id, 'with_item' as StockShortfallResolution])),
  )

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onCancel} />
      <div className="relative w-full max-w-md dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
        <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Not enough stock</h3>
        <p className="text-xs dark:text-gray-400 text-gray-500 mb-4">
          Tracked stock is short for these items. Decide how to finish for each.
        </p>
        <div className="space-y-3 mb-4">
          {shortfalls.map(s => (
            <div key={s.scan_id} className="rounded-lg border dark:border-[#2D3748] border-gray-200 p-3">
              <p className="text-xs font-medium dark:text-gray-200 text-gray-800">{s.name}</p>
              <p className="text-[11px] dark:text-gray-500 text-gray-400 mb-2">
                Needed {s.needed}, {s.short} short
              </p>
              <div className="flex items-center gap-2">
                <button
                  type="button"
                  onClick={() => setChoices(prev => ({ ...prev, [s.scan_id]: 'with_item' }))}
                  aria-pressed={choices[s.scan_id] === 'with_item'}
                  className={`flex-1 px-2 py-1.5 rounded-lg text-[11px] font-medium transition-colors ${choices[s.scan_id] === 'with_item' ? 'bg-blue-600 text-white' : 'dark:bg-[#111827] bg-gray-100 dark:text-gray-300 text-gray-600'}`}
                >
                  Send it (fix inventory)
                </button>
                <button
                  type="button"
                  onClick={() => setChoices(prev => ({ ...prev, [s.scan_id]: 'without_item' }))}
                  aria-pressed={choices[s.scan_id] === 'without_item'}
                  className={`flex-1 px-2 py-1.5 rounded-lg text-[11px] font-medium transition-colors ${choices[s.scan_id] === 'without_item' ? 'bg-blue-600 text-white' : 'dark:bg-[#111827] bg-gray-100 dark:text-gray-300 text-gray-600'}`}
                >
                  Skip it
                </button>
              </div>
            </div>
          ))}
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onCancel} disabled={submitting}
            className="px-4 py-2 rounded-lg dark:bg-[#111827] bg-gray-100 dark:text-gray-300 text-gray-700 text-xs font-medium disabled:opacity-50 transition-colors"
          >
            Cancel
          </button>
          <button
            type="button" onClick={() => onConfirm(choices)} disabled={submitting}
            className="flex-1 px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
          >
            {submitting ? 'Completing…' : 'Confirm & Complete'}
          </button>
        </div>
      </div>
    </div>
  )
}
