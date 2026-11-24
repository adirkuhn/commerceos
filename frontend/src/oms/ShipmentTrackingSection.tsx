import { useState, useEffect, useCallback } from 'react'
import { Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { myParcelStatusText } from '../shipping/myParcelStatusText'
import { cancelOutcomeText, outcomeOfCancelResult, outcomeOfShipment } from '../shipping/cancelOutcomeText'
import * as shippingApi from '../shipping/api'
import type { Shipment, ShipmentTrackingEvent } from '../shipping/types'
import { CARRIERS } from '../shipping/types'
import { useConfirm } from '../components/common/useConfirm'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'

// One shipment's carrier/barcode header plus its full tracking timeline
// (newest first) and a "Refresh from provider" button -- fetches the
// shipment's current status right now instead of waiting for the next
// webhook/reconciliation tick, and saves it as a new timeline entry (see
// RefreshShipmentTrackingUseCase).
export function ShipmentTrackingSection({ token, shipment, onCancelled }: { token: string; shipment: Shipment; onCancelled: () => void }) {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['orders', 'common'])
  const { t: tShipping } = useTranslation('shipping')
  const { confirm, confirmDialog } = useConfirm()
  const [events, setEvents] = useState<ShipmentTrackingEvent[]>([])
  const [loading, setLoading] = useState(true)
  const [refreshing, setRefreshing] = useState(false)
  const [error, setError] = useState('')
  const [cancelling, setCancelling] = useState(false)
  const [cancelMessage, setCancelMessage] = useState('')

  const load = useCallback(() => {
    setLoading(true)
    shippingApi.listShipmentTrackingEvents(token, shipment.id)
      .then(setEvents)
      .catch(err => setError(apiErrorText(err) ?? t('tracking.failedLoad')))
      .finally(() => setLoading(false))
  }, [token, shipment.id])
  useEffect(() => { load() }, [load])

  async function handleRefresh() {
    setRefreshing(true)
    setError('')
    try {
      const event = await shippingApi.refreshShipmentTracking(token, shipment.id)
      setEvents(prev => [event, ...prev])
    } catch (err) {
      setError(apiErrorText(err) ?? t('tracking.failedFetch'))
    } finally {
      setRefreshing(false)
    }
  }

  // Wrong carrier picked -- attempts MyParcel's real delete, then always
  // marks the shipment cancelled locally regardless of that outcome (see
  // cancelShipment's doc comment). onCancelled() re-fetches the parent's
  // shipment list so this card picks up cancelled_at/cancel_note and
  // "Create label" up top stops treating this as the active shipment.
  async function handleCancel() {
    if (!(await confirm({ title: t('tracking.cancelTitle'), message: t('tracking.cancelMessage'), confirmLabel: t('tracking.cancelConfirm'), destructive: true }))) return
    setCancelling(true)
    setError('')
    try {
      const result = await shippingApi.cancelShipment(token, shipment.id)
      setCancelMessage(cancelOutcomeText(tShipping, outcomeOfCancelResult(result), result.message))
      onCancelled()
    } catch (err) {
      setError(apiErrorText(err) ?? t('tracking.failedCancel'))
    } finally {
      setCancelling(false)
    }
  }

  const cancelText = shipment.cancelled_at ? cancelOutcomeText(tShipping, outcomeOfShipment(shipment), shipment.cancel_note ?? '') || cancelMessage : cancelMessage

  const carrierLabel = CARRIERS.find(c => c.id === shipment.carrier)?.label ?? t('tracking.carrier', { id: shipment.carrier })

  return (
    <div className="border border-line rounded-lg p-3 space-y-2">
      <div className="flex items-center justify-between gap-3">
        <div>
          <span className="text-ink font-medium">{carrierLabel}</span>
          <span className="text-ink-4">
            {' · '}
            {shipment.barcode ? (
              shipment.tracking_url ? (
                <a href={shipment.tracking_url} target="_blank" rel="noreferrer" className="text-accent hover:underline">
                  {shipment.barcode}
                </a>
              ) : (
                shipment.barcode
              )
            ) : (
              'no barcode yet'
            )}
          </span>
          {shipment.cancelled_at && (
            <span className="ml-2 text-2xs px-1.5 py-0.5 rounded-full bg-red-50 dark:bg-danger/10 text-danger">{t('tracking.cancelled')}</span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={handleRefresh} disabled={refreshing}
            className="px-2 py-1 rounded bg-fill text-ink-3 hover:text-ink-2 disabled:opacity-50 transition-colors"
          >
            {refreshing ? t('tracking.fetching') : t('tracking.refresh')}
          </button>
          {!shipment.cancelled_at && (
            <button
              type="button" onClick={handleCancel} disabled={cancelling}
              title={t('tracking.cancelHint')}
              aria-label={t('common:actions.delete')}
              className="w-7 h-7 flex items-center justify-center rounded bg-fill hover:bg-hover disabled:opacity-50 transition-colors"
            >
              {cancelling ? (
                <span className="text-2xs text-danger">…</span>
              ) : (
                <Trash2 size={13} className="text-ink-4 hover:text-danger" />
              )}
            </button>
          )}
        </div>
      </div>

      {cancelText && <p className="text-warning">{cancelText}</p>}
      {error && <p className="text-danger">{error}</p>}

      {loading ? (
        <p className="text-ink-4">{t('tracking.loading')}</p>
      ) : events.length === 0 ? (
        <p className="text-ink-4">{t('tracking.empty')}</p>
      ) : (
        <div className="space-y-1">
          {events.map(e => (
            <div key={e.id} className="flex justify-between gap-3 text-ink-3">
              <span className="text-ink">{myParcelStatusText(tShipping, e.status, e.status_label)}</span>
              <span className="shrink-0">{formatDateTime(e.occurred_at)} · {e.source}</span>
            </div>
          ))}
        </div>
      )}
      {confirmDialog}
    </div>
  )
}
