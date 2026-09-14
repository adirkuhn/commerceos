import { X } from 'lucide-react'
import type { ReactNode } from 'react'

// Generic modal shell + shared field styles -- used by every inventory
// stock-action modal (adjust/recount/transfer/portion-from-bulk/damaged-
// lost/etc). Extracted out of pages/Inventory.tsx so each modal can live in
// its own file instead of growing that already-large page further.
export function Modal({ title, onClose, children }: { title: string; onClose: () => void; children: ReactNode }) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="w-full max-w-sm rounded-xl dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 shadow-xl">
        <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">{title}</h3>
          <button type="button" onClick={onClose} aria-label="Close" className="dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600">
            <X size={16} />
          </button>
        </div>
        <div className="p-4">{children}</div>
      </div>
    </div>
  )
}

export const inputClass = 'w-full px-2.5 py-1.5 text-xs rounded-lg dark:bg-[#111827] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 outline-none focus:ring-1 focus:ring-blue-500'
export const labelClass = 'block text-[10px] font-medium dark:text-gray-400 text-gray-500 mb-1'
