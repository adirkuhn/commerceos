import { Dialog } from '../components/common/Dialog'
import { SCAN_ACTIVITY_WORDS, type ScanActivity } from './scanSessionActivity'

// Blocks scanning after a mis-scan until the user explicitly acknowledges
// it -- focus lands on the dialog so a scanner's Return can't dismiss it.
export function ScanErrorDialog({ message, onDismiss }: { message: string; onDismiss: () => void }) {
  return (
    <Dialog onClose={onDismiss} label="Scan error" initialFocus="dialog" className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-6">
      <div className="w-full max-w-xs rounded-2xl bg-night-900 border border-red-500/40 p-5 text-center">
        <p className="text-sm font-semibold text-red-400 mb-4">{message}</p>
        <button
          type="button" onClick={onDismiss}
          className="w-full py-3 rounded-xl bg-red-600 text-white font-semibold hover:bg-red-700"
        >
          OK
        </button>
      </div>
    </Dialog>
  )
}

interface StartOverDialogProps {
  activity: ScanActivity
  resetting: boolean
  onCancel: () => void
  onConfirm: () => void
}

export function StartOverDialog({ activity, resetting, onCancel, onConfirm }: StartOverDialogProps) {
  return (
    <Dialog onClose={onCancel} label="Start over?" dismissible={!resetting} className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 px-6">
      <div className="w-full max-w-xs rounded-2xl bg-night-900 border border-night-700 p-5 text-center">
        <p className="text-sm font-semibold mb-1">Start over?</p>
        <p className="text-xs text-gray-400 mb-4">
          This discards every item {SCAN_ACTIVITY_WORDS[activity].done} so far on this order and starts the session fresh. This can't be undone.
        </p>
        <div className="flex gap-2">
          <button
            type="button" onClick={onCancel} disabled={resetting}
            className="flex-1 py-3 rounded-xl border border-night-700 text-sm font-medium text-gray-300 disabled:opacity-50"
          >
            Cancel
          </button>
          <button
            type="button" onClick={onConfirm} disabled={resetting}
            className="flex-1 py-3 rounded-xl bg-red-600 text-white text-sm font-semibold hover:bg-red-700 disabled:opacity-50"
          >
            {resetting ? 'Resetting…' : 'Start over'}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
