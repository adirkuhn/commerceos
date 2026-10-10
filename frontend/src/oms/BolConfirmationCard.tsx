import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../format/DateFormatProvider'
import { bolFailureText, latestBolConfirmation } from './bolConfirmations'
import type { BolConfirmationAction, BolConfirmationStatus, BolShipmentConfirmation } from './bolConfirmations'

const STATUS_TONE: Record<BolConfirmationStatus, string> = {
  pending: 'text-warning',
  success: 'text-success',
  failure: 'text-danger',
  timeout: 'text-danger',
}

// Status texts and the manual fix differ for a track & trace correction.
const TEXTS: Record<BolConfirmationAction, { status: 'status' | 'correction'; manual: 'manualAction' | 'correctionManualAction' }> = {
  create_shipment: { status: 'status', manual: 'manualAction' },
  update_transport: { status: 'correction', manual: 'correctionManualAction' },
}

// Whether this Bol order's shipment was confirmed on Bol. The newest
// attempt is the headline; a rejection is shown as an alert with Bol's own
// reason and what to do, so an order left open on Bol is never silent.
export function BolConfirmationCard({ confirmations, error }: { confirmations: BolShipmentConfirmation[]; error: string | null }) {
  const { t } = useTranslation('orders')
  const { formatDateTime } = useDateFormat()
  const latest = latestBolConfirmation(confirmations)
  const earlier = confirmations.slice(1)

  return (
    <div className="rounded-xl border border-line p-4 space-y-2 text-xs">
      <p className="font-medium text-ink-2">{t('bolConfirmation.title')}</p>
      {error && <p className="text-danger">{error}</p>}

      {!latest ? (
        <p className="text-ink-4">{t('bolConfirmation.notSent')}</p>
      ) : (
        <div role={latest.needs_manual_action ? 'alert' : undefined} className="space-y-1">
          <p className={`font-medium ${STATUS_TONE[latest.status]}`}>{t(`bolConfirmation.${TEXTS[latest.action].status}.${latest.status}`)}</p>
          <p className="text-ink-3">
            {[latest.transporter_code, latest.track_and_trace].filter(Boolean).join(' · ')}
            {' — '}
            {formatDateTime(latest.resolved_at ?? latest.created_at)}
          </p>
          {bolFailureText(t, latest) && <p className="text-danger">{bolFailureText(t, latest)}</p>}
          {latest.needs_manual_action && (
            <p className="text-ink">{t(`bolConfirmation.${TEXTS[latest.action].manual}`, { order: latest.external_order_id })}</p>
          )}
        </div>
      )}

      {earlier.length > 0 && (
        <div className="pt-2 border-t border-line space-y-1">
          <p className="text-ink-4">{t('bolConfirmation.earlier')}</p>
          {earlier.map(c => (
            <p key={c.id} className="text-ink-3">
              {t(`bolConfirmation.${TEXTS[c.action].status}.${c.status}`)} · {formatDateTime(c.created_at)}
              {bolFailureText(t, c) && ` · ${bolFailureText(t, c)}`}
            </p>
          ))}
        </div>
      )}
    </div>
  )
}
