import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import type { Shipment } from './types'
import * as shippingApi from './api'
import { Dialog } from '../components/common/Dialog'

interface ExistingShipmentConfirmModalProps {
  token: string
  shipment: Shipment
  onReprint: () => void
  // Called once staff choose to proceed with a fresh shipment after
  // attempting to cancel the existing one (wrong-carrier fix) -- regardless
  // of whether MyParcel itself actually deleted it (see cancelShipment's
  // doc comment: CreateShipmentUseCase's own duplicate-guard already skips
  // a cancelled shipment, so a new one is guaranteed to actually be created).
  onCreateNew: () => void
  onCancel: () => void
}

// Shown instead of silently creating a new MyParcel shipment when one
// already exists for this order -- CreateShipmentUseCase itself now returns
// the existing shipment rather than duplicating it (defense in depth), but
// staff should still get to choose "just reprint" over quietly reusing an
// old label without being told, which is what this confirms. Also offers a
// "wrong carrier" escape hatch: attempt to delete the shipment on MyParcel,
// then let staff create a corrected one regardless of that outcome.
export function ExistingShipmentConfirmModal({ token, shipment, onReprint, onCreateNew, onCancel }: ExistingShipmentConfirmModalProps) {
  const [cancelling, setCancelling] = useState(false)
  const [cancelResult, setCancelResult] = useState<{ deleted_on_myparcel: boolean; message: string } | null>(null)
  const [cancelError, setCancelError] = useState('')
  const { t } = useTranslation(['shipping', 'common'])

  async function handleWrongCarrier() {
    setCancelling(true)
    setCancelError('')
    try {
      const result = await shippingApi.cancelShipment(token, shipment.id)
      setCancelResult(result)
    } catch (err) {
      setCancelError(err instanceof Error ? err.message : t('existing.cancelFailed'))
    } finally {
      setCancelling(false)
    }
  }

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onCancel} />
      <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
        {!cancelResult ? (
          <>
            <h3 className="text-sm font-semibold text-ink mb-2">{t('existing.title')}</h3>
            <p className="text-xs text-ink-3 mb-4">
              {t('existing.message', { barcode: shipment.barcode || '—' })}
            </p>
            <div className="flex items-center gap-2">
              <button
                type="button" onClick={onCancel}
                className="flex-1 px-4 py-2 rounded-lg bg-fill text-ink-2 text-xs font-medium hover:bg-hover transition-colors"
              >
                {t('common:actions.cancel')}
              </button>
              <button
                type="button" onClick={onReprint}
                className="flex-1 px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
              >
                {t('existing.reprint')}
              </button>
            </div>
            <button
              type="button" onClick={handleWrongCarrier} disabled={cancelling}
              className="mt-3 w-full text-2xs text-danger hover:underline disabled:opacity-50 disabled:no-underline"
            >
              {cancelling ? t('existing.cancelling') : t('existing.wrongCarrier')}
            </button>
            {cancelError && <p className="text-2xs text-danger mt-2">{cancelError}</p>}
          </>
        ) : (
          <>
            <h3 className="text-sm font-semibold text-ink mb-2">{t('existing.cancelled')}</h3>
            <p className="text-xs text-ink-3 mb-4">{cancelResult.message}</p>
            <div className="flex items-center gap-2">
              <button
                type="button" onClick={onCancel}
                className="flex-1 px-4 py-2 rounded-lg bg-fill text-ink-2 text-xs font-medium hover:bg-hover transition-colors"
              >
                {t('common:actions.close')}
              </button>
              <button
                type="button" onClick={onCreateNew}
                className="flex-1 px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
              >
                {t('existing.createNew')}
              </button>
            </div>
          </>
        )}
      </div>
    </Dialog>
  )
}
