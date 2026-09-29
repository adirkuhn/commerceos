import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { X } from 'lucide-react'
import { QuantityStepper } from './QuantityStepper'
import type { PickingSessionLineItem, PickScan } from './types'
import { Dialog } from '../components/common/Dialog'

// Tap a line item row (or scan/confirm its own barcode) to confirm how many
// were picked. Picking only picks -- substitutes, extra items and refunds
// happen in packing.
export function PickLineSheet({ lineItem, pickedSoFar, scans, onDeleteScan, onCancel, onPick }: {
  lineItem: PickingSessionLineItem
  pickedSoFar: number
  scans: PickScan[]
  onDeleteScan: (scanId: string) => void
  onCancel: () => void
  onPick: (quantity: number, reason?: string) => void
}) {
  const remaining = Math.max(lineItem.quantity - pickedSoFar, 0)
  const [qty, setQty] = useState(String(remaining))
  const [reason, setReason] = useState('')
  const n = Number(qty) || 0
  const { t } = useTranslation(['scanner', 'common'])

  return (
    <Dialog onClose={onCancel} className="fixed inset-0 z-50 flex items-end justify-center bg-black/40">
      <div className="w-full max-w-sm rounded-t-2xl bg-night-900 border-t border-night-700 max-h-[85vh] overflow-y-auto">
        <div className="flex items-start justify-between gap-3 px-5 pt-5">
          <div className="min-w-0">
            <h2 className="text-sm font-semibold truncate">{lineItem.name}</h2>
            <p className="text-xs text-gray-400 mt-0.5">{t('line.pickedOf', { done: pickedSoFar, total: lineItem.quantity })}</p>
          </div>
          <button aria-label={t('common:actions.close')} onClick={onCancel} title={t('common:actions.close')} className="shrink-0 w-11 h-11 -mt-3 -mr-3 flex items-center justify-center rounded-lg text-gray-400 hover:text-gray-300">
            <X size={18} />
          </button>
        </div>

        {scans.length > 0 && (
          <div className="px-5 pt-3 space-y-1.5">
            <p className="text-xs text-gray-400">{t('shared.alreadyStaged')}</p>
            {scans.map(s => (
              <div key={s.id} className="flex items-center gap-2 rounded-lg bg-night-800 border border-night-700 px-3 py-2">
                <div className="flex-1 min-w-0">
                  <p className="text-sm text-gray-200">{t('line.picked', { quantity: s.quantity })}</p>
                  {s.reason && <p className="text-xs text-gray-400 truncate">{s.reason}</p>}
                </div>
                <button aria-label={t('shared.deleteStagedScan')}
                  type="button"
                  onClick={() => onDeleteScan(s.id)}
                  title={t('shared.deleteStagedScan')}
                  className="shrink-0 w-11 h-11 -m-2 flex items-center justify-center rounded-lg text-gray-400 hover:text-red-400"
                >
                  <X size={14} />
                </button>
              </div>
            ))}
          </div>
        )}

        <div className="p-5 space-y-4">
          <QuantityStepper id="pick-quantity" label={t('line.qtyPick')} value={qty} onChange={setQty} min={0} max={remaining} />
          <div>
            <label htmlFor="pick-reason" className="block text-xs text-gray-400 mb-1.5">{t('line.noteOptional')}</label>
            <input
              id="pick-reason"
              value={reason}
              onChange={e => setReason(e.target.value)}
              className="w-full px-3 py-2.5 rounded-xl bg-night-800 border border-night-700 text-gray-100 outline-none focus:ring-2 focus:ring-focus"
            />
          </div>
          <button
            onClick={() => onPick(n, reason || undefined)}
            disabled={n <= 0}
            className="w-full py-3 rounded-xl bg-accent-solid text-white font-semibold disabled:opacity-50"
          >
            {t('line.confirmPick')}
          </button>
          <p className="text-xs text-gray-400">{t('line.pickHelp')}</p>
        </div>
      </div>
    </Dialog>
  )
}
