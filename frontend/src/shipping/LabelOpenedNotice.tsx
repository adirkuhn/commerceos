import { Printer, X } from 'lucide-react'

interface LabelOpenedNoticeProps {
  onPrint: () => void
  onDismiss: () => void
}

// Shown after a shipping label opens in a new browser tab (see
// openLabel.ts) -- an explicit in-app Print button, since the browser's
// own PDF-viewer print control lives in a small toolbar easy to miss.
// onPrint calls .print() on the tab directly (a cross-origin-safelisted
// call, works even though that tab now shows the browser's native PDF
// viewer rather than a page of ours).
export function LabelOpenedNotice({ onPrint, onDismiss }: LabelOpenedNoticeProps) {
  return (
    <div className="fixed bottom-4 right-4 z-50 flex items-center gap-3 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 shadow-lg px-4 py-3">
      <p className="text-xs dark:text-gray-300 text-gray-700">Label opened in a new tab</p>
      <button
        type="button" onClick={onPrint}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
      >
        <Printer size={13} /> Print
      </button>
      <button
        type="button" onClick={onDismiss} aria-label="Dismiss"
        className="dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600"
      >
        <X size={14} />
      </button>
    </div>
  )
}
