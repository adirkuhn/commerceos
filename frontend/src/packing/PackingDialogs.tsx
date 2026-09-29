import { useTranslation } from 'react-i18next'
import { Dialog } from '../components/common/Dialog'

interface ScanErrorDialogProps {
  message: string
  onDismiss: () => void
}

// Blocking alert for a mis-scan -- staff must acknowledge it before the next.
export function ScanErrorDialog({ message, onDismiss }: ScanErrorDialogProps) {
  const { t } = useTranslation('packing')
  return (
    <Dialog onClose={onDismiss} label={t('dialogs.scanError')} initialFocus="dialog" className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-xs rounded-xl bg-surface border dark:border-red-500/40 border-red-300 p-5 text-center shadow-xl">
        <p className="text-sm font-semibold text-danger mb-4">{message}</p>
        <button
          type="button" onClick={onDismiss}
          className="w-full py-2 rounded-lg bg-red-600 text-white text-xs font-semibold hover:bg-red-700"
        >
          {t('dialogs.ok')}
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
  const { t } = useTranslation(['packing', 'common'])
  return (
    <Dialog onClose={onCancel} label={t('dialogs.startOverTitle')} dismissible={!resetting} className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="w-full max-w-xs rounded-xl bg-surface border border-line p-5 text-center shadow-xl">
        <p className="text-sm font-semibold text-ink mb-1">{t('dialogs.startOverTitle')}</p>
        <p className="text-xs text-ink-3 mb-4">
          {t('dialogs.startOverText')}
        </p>
        <div className="flex gap-2">
          <button
            type="button" onClick={onCancel} disabled={resetting}
            className="flex-1 py-2 rounded-lg border border-line text-xs font-medium text-ink-2 disabled:opacity-50"
          >
            {t('common:actions.cancel')}
          </button>
          <button
            type="button" onClick={onConfirm} disabled={resetting}
            className="flex-1 py-2 rounded-lg bg-red-600 text-white text-xs font-semibold hover:bg-red-700 disabled:opacity-50"
          >
            {resetting ? t('dialogs.resetting') : t('header.startOver')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
