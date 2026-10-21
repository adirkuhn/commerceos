import { useState } from 'react'
import { X } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import type { RecipientAddress } from './types'
import { CARRIERS, PACKAGE_TYPES } from './types'
import { Dialog } from '../components/common/Dialog'
import { apiErrorText } from '../api/apiErrorText'

interface LabelPreviewModalProps {
  recipient: RecipientAddress
  carrier: number
  // Optional only so existing call sites/tests that don't care about box
  // type (yet) don't need updating -- both real call sites always pass it.
  packageType?: number
  labelSize: 'A4' | 'A6'
  onLabelSizeChange: (size: 'A4' | 'A6') => void
  // Returns to AddressConfirmModal with recipient/carrier preserved (see
  // that component's initialRecipient/initialCarrier props) -- no API call.
  onBack: () => void
  // The actual shipment-creation call (and, by the caller, opening the
  // print view right after) moved HERE from AddressConfirmModal's old
  // onConfirm -- staff now see exactly what will be sent, label-shaped,
  // before anything is created. May reject (MyParcel rejecting the
  // address/EAN/etc); surfaced inline, the modal stays open so the address
  // can still be fixed via Back without losing the rest of the form.
  onPrint: () => void | Promise<void>
  onClose: () => void
  // The shipping provider's name (e.g. "MyParcel"), shown as a chip so
  // staff see who the label goes through.
  providerName?: string
  // False for a provider that is its own carrier -- the carrier chip would
  // just repeat the provider. Defaults to true.
  showCarrier?: boolean
}

// Second step of the "Create label" flow, after AddressConfirmModal --
// shows exactly what will be sent to MyParcel, laid out like the physical
// label itself (not just a form echo), plus the label-size choice, before
// staff commit to actually creating the shipment.
export function LabelPreviewModal({ recipient, carrier, packageType, labelSize, onLabelSizeChange, onBack, onPrint, onClose, providerName, showCarrier = true }: LabelPreviewModalProps) {
  const [printing, setPrinting] = useState(false)
  const { t } = useTranslation(['shipping', 'common'])
  const [error, setError] = useState('')
  const carrierLabel = CARRIERS.find(c => c.id === carrier)?.label ?? t('preview.carrierFallback', { id: carrier })
  const packageTypeLabel = PACKAGE_TYPES.find(p => p.id === packageType)?.label

  async function handlePrint() {
    setError('')
    setPrinting(true)
    try {
      await onPrint()
    } catch (err) {
      setError(apiErrorText(err) ?? t('preview.createFailed'))
      setPrinting(false)
    }
    // No `finally` -- on success the caller unmounts this modal (moves on
    // to the print view), so leaving `printing` true until then keeps the
    // button disabled rather than flashing back to enabled for an instant.
  }

  return (
    <Dialog onClose={onClose} className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 dark:bg-black/60 bg-black/30" onClick={onClose} />
      <div className="relative w-full max-w-sm bg-raised rounded-xl shadow-2xl border border-line p-5">
        <div className="flex items-center justify-between mb-4">
          <h3 className="text-sm font-semibold text-ink">{t('preview.title')}</h3>
          <button onClick={onClose} aria-label={t('common:actions.close')}><X size={14} className="text-ink-4" /></button>
        </div>

        {/* Label-shaped preview -- deliberately always white/black text
            (like the real printed label) regardless of the app's own
            dark/light theme, so what staff see here matches the paper. */}
        <div className="rounded-lg border-2 border-dashed border-gray-300 p-4 mb-4 bg-white text-gray-900">
          <p className="text-2xs caps-label text-gray-500 mb-1">{t('preview.to')}</p>
          <p className="text-base font-bold leading-tight">{recipient.person || '—'}</p>
          {recipient.company && <p className="text-sm">{recipient.company}</p>}
          <p className="text-sm">
            {recipient.street} {recipient.number}{recipient.number_suffix}
          </p>
          <p className="text-sm">
            {[recipient.postal_code, recipient.city].filter(Boolean).join(' ') || '—'}
          </p>
          <p className="text-sm font-medium">{recipient.cc}</p>
          <div className="flex items-center gap-2 mt-3 pt-3 border-t border-dashed border-gray-200">
            {providerName && (
              <span className="text-2xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-600 font-medium">{providerName}</span>
            )}
            {showCarrier && (
              <span className="text-2xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-600 font-medium">{carrierLabel}</span>
            )}
            {packageTypeLabel && (
              <span className="text-2xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-600 font-medium">{packageTypeLabel}</span>
            )}
            <span className="text-2xs px-2 py-0.5 rounded-full bg-gray-100 text-gray-600 font-medium">{labelSize}</span>
          </div>
        </div>

        <div className="mb-4">
          <label className="block text-2xs font-medium text-ink-3 mb-1">{t('preview.labelSize')}</label>
          <div className="flex items-center rounded-lg border border-line overflow-hidden">
            {(['A4', 'A6'] as const).map(s => (
              <button
                key={s}
                type="button"
                onClick={() => onLabelSizeChange(s)}
                aria-pressed={labelSize === s}
                className={clsx(
                  'flex-1 py-2 text-xs font-medium',
                  labelSize === s ? 'bg-accent-solid text-white' : 'bg-surface text-ink-3',
                )}
              >
                {s}
              </button>
            ))}
          </div>
          <p className="text-2xs text-ink-4 mt-1">{t('preview.sizeHint')}</p>
        </div>

        {error && <p className="text-xs text-danger mb-3">{error}</p>}

        <div className="flex items-center gap-2">
          <button
            type="button" onClick={onBack} disabled={printing}
            className="px-4 py-2 rounded-lg bg-fill text-ink-2 text-xs font-medium hover:bg-hover disabled:opacity-50 transition-colors"
          >
            {t('preview.back')}
          </button>
          <button
            type="button" onClick={handlePrint} disabled={printing}
            className="flex-1 px-4 py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {printing ? t('preview.creating') : t('preview.print')}
          </button>
        </div>
      </div>
    </Dialog>
  )
}
