import { Dialog } from '../components/common/Dialog'

interface ScanErrorDialogProps {
  message: string
  onDismiss: () => void
}

// Blocking alert for a mis-scan -- staff must acknowledge it before the next.
export function ScanErrorDialog({ message, onDismiss }: ScanErrorDialogProps) {
  return (
    <Dialog onClose={onDismiss} label="Scan error" initialFocus="dialog" className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-xs rounded-xl bg-surface border dark:border-red-500/40 border-red-300 p-5 text-center shadow-xl">
        <p className="text-sm font-semibold text-danger mb-4">{message}</p>
        <button
          type="button" onClick={onDismiss}
          className="w-full py-2 rounded-lg bg-red-600 text-white text-xs font-semibold hover:bg-red-700"
        >
          OK
        </button>
      </div>
    </Dialog>
  )
}

interface ResetSessionDialogProps {
  resetting: boolean
  onCancel: () => void
  onConfirm: () => void
}

export function ResetSessionDialog({ resetting, onCancel, onConfirm }: ResetSessionDialogProps) {
  return (
    <Dialog onClose={onCancel} label="Start over?" dismissible={!resetting} className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-xs rounded-xl bg-surface border border-line p-5 text-center shadow-xl">
        <p className="text-sm font-semibold text-ink mb-1">Start over?</p>
        <p className="text-xs text-ink-3 mb-4">
          This discards every item packed so far on this order and starts the session fresh. This can't be undone.
        </p>
        <div className="flex gap-2">
          <button
            type="button" onClick={onCancel} disabled={resetting}
            className="flex-1 py-2 rounded-lg border border-line text-xs font-medium text-ink-2 disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="button" onClick={onConfirm} disabled={resetting}
            className="flex-1 py-2 rounded-lg bg-red-600 text-white text-xs font-semibold hover:bg-red-700 disabled:opacity-50"
          >
            {resetting ? 'Resetting…' : 'Start over'}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
