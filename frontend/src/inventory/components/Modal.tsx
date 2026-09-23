import { X } from 'lucide-react'
import type { ReactNode } from 'react'

// Generic modal shell + shared field styles -- used by every inventory
// stock-action modal (adjust/recount/transfer/portion-from-bulk/damaged-
// lost/etc). Extracted out of pages/Inventory.tsx so each modal can live in
// its own file instead of growing that already-large page further.
export function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-sm rounded-xl bg-raised border border-line shadow-xl">
        <div className="flex items-center justify-between px-4 py-3 border-b border-line">
          <h3 className="text-sm font-semibold text-ink">{title}</h3>
          <button type="button" onClick={onClose} aria-label="Close" className="text-ink-4 hover:text-ink-2">
            <X size={16} />
          </button>
        </div>
        <div className="p-4">{children}</div>
      </div>
    </div>
  )
}

export const inputClass = 'w-full px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-blue-500'
export const labelClass = 'block text-[10px] font-medium text-ink-3 mb-1'
