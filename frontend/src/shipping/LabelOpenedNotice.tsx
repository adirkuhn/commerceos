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
    <div className="fixed bottom-4 right-4 z-50 flex items-center gap-3 rounded-lg bg-raised border border-line shadow-lg px-4 py-3">
      <p className="text-xs text-ink-2">Label opened in a new tab</p>
      <button
        type="button" onClick={onPrint}
        className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
      >
        <Printer size={13} /> Print
      </button>
      <button
        type="button" onClick={onDismiss} aria-label="Dismiss"
        className="text-ink-4 hover:text-ink-2"
      >
        <X size={14} />
      </button>
    </div>
  )
}
