import { useState, useEffect, useCallback } from 'react'
import { Trash2 } from 'lucide-react'
import * as shippingApi from '../shipping/api'
import type { Shipment, ShipmentTrackingEvent } from '../shipping/types'
import { CARRIERS } from '../shipping/types'
import { useConfirm } from '../components/common/useConfirm'
import { useDateFormat } from '../format/DateFormatProvider'

// One shipment's carrier/barcode header plus its full tracking timeline
// (newest first) and a "Refresh from provider" button -- fetches the
// shipment's current status right now instead of waiting for the next
// webhook/reconciliation tick, and saves it as a new timeline entry (see
// RefreshShipmentTrackingUseCase).
export function ShipmentTrackingSection({ token, shipment, onCancelled }: { token: string; shipment: Shipment; onCancelled: () => void }) {
  const { formatDateTime } = useDateFormat()
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
      .catch(err => setError(err instanceof Error ? err.message : 'Failed to load tracking'))
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
      setError(err instanceof Error ? err.message : 'Failed to fetch from provider')
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
    if (!(await confirm({ title: 'Cancel this shipment?', message: 'This cannot be undone.', confirmLabel: 'Cancel shipment', destructive: true }))) return
    setCancelling(true)
    setError('')
    try {
      const result = await shippingApi.cancelShipment(token, shipment.id)
      setCancelMessage(result.message)
      onCancelled()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to cancel shipment')
    } finally {
      setCancelling(false)
    }
  }

  const carrierLabel = CARRIERS.find(c => c.id === shipment.carrier)?.label ?? `Carrier ${shipment.carrier}`

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
            <span className="ml-2 text-2xs px-1.5 py-0.5 rounded-full bg-red-50 dark:bg-red-500/10 text-danger">Cancelled</span>
          )}
        </div>
        <div className="flex items-center gap-2">
          <button
            type="button" onClick={handleRefresh} disabled={refreshing}
            className="px-2 py-1 rounded bg-fill text-ink-3 hover:text-ink-2 disabled:opacity-50 transition-colors"
          >
            {refreshing ? 'Fetching…' : 'Refresh from provider'}
          </button>
          {!shipment.cancelled_at && (
            <button
              type="button" onClick={handleCancel} disabled={cancelling}
              title="Wrong carrier? Cancel this shipment so a corrected one can be created"
              aria-label="Delete"
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

      {(shipment.cancel_note || cancelMessage) && <p className="text-warning">{shipment.cancel_note || cancelMessage}</p>}
      {error && <p className="text-danger">{error}</p>}

      {loading ? (
        <p className="text-ink-4">Loading tracking…</p>
      ) : events.length === 0 ? (
        <p className="text-ink-4">No tracking updates yet.</p>
      ) : (
        <div className="space-y-1">
          {events.map(e => (
            <div key={e.id} className="flex justify-between gap-3 text-ink-3">
              <span className="text-ink">{e.status_label}</span>
              <span className="shrink-0">{formatDateTime(e.occurred_at)} · {e.source}</span>
            </div>
          ))}
        </div>
      )}
      {confirmDialog}
    </div>
  )
}
