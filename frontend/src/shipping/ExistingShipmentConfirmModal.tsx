import type { Shipment } from './types'

interface ExistingShipmentConfirmModalProps {
  shipment: Shipment
  onReprint: () => void
  onCancel: () => void
}

// Shown instead of silently creating a new MyParcel shipment when one
// already exists for this order -- CreateShipmentUseCase itself now returns
// the existing shipment rather than duplicating it (defense in depth), but
// staff should still get to choose "just reprint" over quietly reusing an
// old label without being told, which is what this confirms.
export function ExistingShipmentConfirmModal({ shipment, onReprint, onCancel }: ExistingShipmentConfirmModalProps) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onCancel} />
      <div className="relative w-full max-w-sm dark:bg-[#1F2937] bg-white rounded-xl shadow-2xl border dark:border-[#2D3748] border-gray-200 p-5">
        <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-2">Label already exists</h3>
        <p className="text-xs dark:text-gray-400 text-gray-500 mb-4">
          A shipping label was already created for this order (barcode {shipment.barcode || '—'}). Download and print it again?
        </p>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onCancel}
            className="flex-1 px-4 py-2 rounded-lg dark:bg-[#111827] bg-gray-100 dark:text-gray-300 text-gray-700 text-xs font-medium hover:dark:bg-[#2D3748] hover:bg-gray-200 transition-colors"
          >
            Cancel
          </button>
          <button
            type="button" onClick={onReprint}
            className="flex-1 px-4 py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors"
          >
            Download & print again
          </button>
        </div>
      </div>
    </div>
  )
}
