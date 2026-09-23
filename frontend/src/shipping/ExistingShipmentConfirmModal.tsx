import { useState } from 'react'
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

  async function handleWrongCarrier() {
    setCancelling(true)
    setCancelError('')
    try {
      const result = await shippingApi.cancelShipment(token, shipment.id)
      setCancelResult(result)
    } catch (err) {
      setCancelError(err instanceof Error ? err.message : 'Failed to cancel shipment')
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
            <h3 className="text-sm font-semibold text-ink mb-2">Label already exists</h3>
            <p className="text-xs text-ink-3 mb-4">
              A shipping label was already created for this order (barcode {shipment.barcode || '—'}). Download and print it again?
            </p>
            <div className="flex items-center gap-2">
              <button
                type="button" onClick={onCancel}
                className="flex-1 px-4 py-2 rounded-lg bg-fill text-ink-2 text-xs font-medium hover:bg-hover transition-colors"
              >
                Cancel
              </button>
              <button
                type="button" onClick={onReprint}
                className="flex-1 px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
              >
                Download & print again
              </button>
            </div>
            <button
              type="button" onClick={handleWrongCarrier} disabled={cancelling}
              className="mt-3 w-full text-2xs text-danger hover:underline disabled:opacity-50 disabled:no-underline"
            >
              {cancelling ? 'Cancelling…' : 'Wrong carrier? Delete & create new'}
            </button>
            {cancelError && <p className="text-2xs text-danger mt-2">{cancelError}</p>}
          </>
        ) : (
          <>
            <h3 className="text-sm font-semibold text-ink mb-2">Shipment cancelled</h3>
            <p className="text-xs text-ink-3 mb-4">{cancelResult.message}</p>
            <div className="flex items-center gap-2">
              <button
                type="button" onClick={onCancel}
                className="flex-1 px-4 py-2 rounded-lg bg-fill text-ink-2 text-xs font-medium hover:bg-hover transition-colors"
              >
                Close
              </button>
              <button
                type="button" onClick={onCreateNew}
                className="flex-1 px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors"
              >
                Create new shipment
              </button>
            </div>
          </>
        )}
      </div>
    </Dialog>
  )
}
