import { X } from 'lucide-react'
import type { PickLine } from './usePickLineSheet'

interface PickLineSheetProps {
  pickLine: PickLine
  pickQty: number
  onPickQtyChange: (qty: number) => void
  onConfirm: () => void
  onClose: () => void
  busy: boolean
}

const inputClass = 'w-full px-3 py-2 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'

// Pick confirm panel -- picking only picks; substitutes, extra items and
// refunds are recorded in packing.
export function PickLineSheet({ pickLine, pickQty, onPickQtyChange, onConfirm, onClose, busy }: PickLineSheetProps) {
  return (
    <div className="w-80 border-l border-line bg-sunken flex flex-col shrink-0">
      <div className="flex items-center justify-between px-4 py-3 border-b border-line">
        <p className="text-xs font-semibold text-ink">{pickLine.lineItem.name}</p>
        <button type="button" onClick={onClose} aria-label="Close" className="p-1 -m-1 rounded text-ink-4 hover:text-ink-2 hover:bg-hover">
          <X size={14} />
        </button>
      </div>
      <div className="p-4 space-y-4">
        <div>
          <label htmlFor="pick-qty" className="block text-2xs font-medium text-ink-4 uppercase tracking-wider mb-1.5">Quantity</label>
          <input
            id="pick-qty" type="number" min={1} max={pickLine.lineItem.quantity - pickLine.pickedSoFar || undefined}
            value={pickQty} onChange={e => onPickQtyChange(Math.max(1, parseInt(e.target.value, 10) || 1))}
            className={inputClass}
          />
        </div>
        <button
          type="button" onClick={onConfirm} disabled={busy}
          className="w-full px-3 py-2 rounded-lg bg-accent-solid text-white text-xs font-semibold hover:bg-accent-solid-hover disabled:opacity-50"
        >
          Confirm pick
        </button>
        <p className="text-2xs text-ink-4">Out of stock or needs replacing? Substitutes and refunds happen in packing.</p>
      </div>
    </div>
  )
}
