import { useState } from 'react'
import { useAuth } from '../auth/AuthContext'
import { useWebhookEvents } from '../integrations/hooks/useWebhookEvents'
import * as api from '../integrations/api'
import type { AvailableWebhookTopics } from '../integrations/api'
import type { WebhookEvent } from '../integrations/types'
import { useQueryEnum } from '../routing/useQueryState'
import { providerLabel } from './providerLabel'
import { useDateFormat } from '../format/DateFormatProvider'

const MESSAGE_VIEWS = ['all', 'needsAttention'] as const

// A message "needs attention" if it's Failed, or Received on a
// provider/topic that actually has a dispatch handler (myparcel
// shipment_status_change today) -- meaning it should have moved past
// Received but hasn't. A Received message on any other provider/topic is
// its normal, successful terminal state, not something stuck.
function needsAttention(e: WebhookEvent): boolean {
  if (e.status === 'failed') return true
  return e.status === 'received' && e.provider === 'myparcel' && e.topic === 'shipment_status_change'
}

// Screen 3: "see all messages" -- every webhook this tenant has actually
// received (backend/domain/webhooks.WebhookEvent), filterable by sender.
// Payload is truncated by default and expands inline on click -- these are
// order/shipment bodies, not huge, but a raw dump for every row would still
// bury the table. A view toggle narrows to just Failed/still-pending
// messages, each with its own Retry action (and a bulk one for the whole
// filtered set) -- re-running dispatch right now instead of waiting for
// the next webhook or a manual backfill run.
export function MessagesTab({
  events, loading, error, updateEvent, topics, provider, onProviderChange,
}: ReturnType<typeof useWebhookEvents> & { topics: AvailableWebhookTopics; provider: string; onProviderChange: (provider: string) => void }) {
  const { formatDateTime } = useDateFormat()
  const senders = Array.from(new Set([...Object.keys(topics.active_webhooks), ...Object.keys(topics.all_events)])).sort()
  const [view, setView] = useQueryEnum('view', MESSAGE_VIEWS, 'all')
  const [retryingId, setRetryingId] = useState<string | null>(null)
  const [bulkRetrying, setBulkRetrying] = useState(false)
  const [retryError, setRetryError] = useState('')
  const { token } = useAuth()

  const displayedEvents = view === 'needsAttention' ? events.filter(needsAttention) : events

  async function handleRetry(eventId: string) {
    if (!token) return
    setRetryError('')
    setRetryingId(eventId)
    try {
      const updated = await api.retryWebhookEventDispatch(token, eventId)
      updateEvent(updated)
    } catch (err) {
      setRetryError(err instanceof Error ? err.message : 'Failed to retry')
    } finally {
      setRetryingId(null)
    }
  }

  async function handleRetryAll() {
    if (!token) return
    setRetryError('')
    setBulkRetrying(true)
    try {
      // Sequential, not parallel -- this hits MyParcel's real API once per
      // message, and the dataset this view targets is small by design (see
      // needsAttention's own doc comment).
      for (const e of events.filter(needsAttention)) {
        try {
          const updated = await api.retryWebhookEventDispatch(token, e.id)
          updateEvent(updated)
        } catch (err) {
          setRetryError(err instanceof Error ? err.message : 'Failed to retry one or more messages')
        }
      }
    } finally {
      setBulkRetrying(false)
    }
  }

  return (
    <div className="max-w-4xl">
      <div className="bg-raised rounded-xl border border-line p-4">
        <div className="flex items-start justify-between gap-3 mb-3">
          <div>
            <h2 className="text-sm font-semibold text-ink mb-1">Received messages</h2>
            <p className="text-xs text-ink-4">
              Every webhook this tenant has actually received, newest first.
            </p>
          </div>
          <div className="flex items-end gap-2">
            <div>
              <label htmlFor="message-sender" className="block text-2xs text-ink-3 mb-1">Sender</label>
              <select
                id="message-sender" value={provider} onChange={e => onProviderChange(e.target.value)}
                className="px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink"
              >
                <option value="">Any sender</option>
                {senders.map(s => <option key={s} value={s}>{providerLabel(s)}</option>)}
              </select>
            </div>
            {view === 'needsAttention' && displayedEvents.length > 0 && (
              <button
                type="button" onClick={handleRetryAll} disabled={bulkRetrying}
                className="px-2.5 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
              >
                {bulkRetrying ? 'Retrying…' : `Retry all (${displayedEvents.length})`}
              </button>
            )}
          </div>
        </div>

        <div className="flex gap-1 mb-3 border-b border-line">
          <button aria-pressed={view === 'all'}
            type="button" onClick={() => setView('all')}
            className={`px-2.5 py-1.5 text-xs font-medium border-b-2 transition-colors ${
              view === 'all' ? 'border-accent text-accent' : 'border-transparent text-ink-4 hover:text-ink-2'
            }`}
          >
            All messages
          </button>
          <button aria-pressed={view === 'needsAttention'}
            type="button" onClick={() => setView('needsAttention')}
            className={`px-2.5 py-1.5 text-xs font-medium border-b-2 transition-colors ${
              view === 'needsAttention' ? 'border-accent text-accent' : 'border-transparent text-ink-4 hover:text-ink-2'
            }`}
          >
            Needs attention{events.filter(needsAttention).length > 0 && ` (${events.filter(needsAttention).length})`}
          </button>
        </div>

        {error && <p className="text-xs text-danger mb-2">{error}</p>}
        {retryError && <p className="text-xs text-danger mb-2">{retryError}</p>}

        {loading ? (
          <p className="text-xs text-ink-4">Loading…</p>
        ) : displayedEvents.length === 0 ? (
          <p className="text-xs text-ink-4">
            {view === 'needsAttention' ? 'Nothing needs attention.' : 'No messages received yet.'}
          </p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="text-left text-ink-4 border-b border-line">
                  <th className="py-2 pr-3 font-medium">Sender</th>
                  <th className="py-2 pr-3 font-medium">Topic</th>
                  <th className="py-2 pr-3 font-medium">Received at</th>
                  <th className="py-2 pr-3 font-medium">Status</th>
                  <th className="py-2 pr-3 font-medium">Payload</th>
                  <th className="py-2 pr-3 font-medium"></th>
                </tr>
              </thead>
              <tbody>
                {displayedEvents.map(e => (
                  <tr key={e.id} className="border-b border-line last:border-0 align-top">
                    <td className="py-2 pr-3 text-ink-2">{providerLabel(e.provider)}</td>
                    <td className="py-2 pr-3 text-ink-3">{e.topic}</td>
                    <td className="py-2 pr-3 text-ink-3 whitespace-nowrap">{formatDateTime(e.received_at)}</td>
                    <td className="py-2 pr-3"><MessageStatusBadge status={e.status} dispatchError={e.dispatch_error} /></td>
                    <td className="py-2 pr-3 text-ink-3 font-mono max-w-[360px]">
                      <PayloadPreview payload={e.payload} />
                    </td>
                    <td className="py-2 pr-3">
                      {needsAttention(e) && (
                        <button
                          type="button" onClick={() => handleRetry(e.id)} disabled={retryingId === e.id || bulkRetrying}
                          className="px-2 py-1 rounded bg-fill text-ink-3 hover:text-ink-2 disabled:opacity-50 transition-colors"
                        >
                          {retryingId === e.id ? 'Retrying…' : 'Retry'}
                        </button>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </div>
    </div>
  )
}

// "received" is the normal, successful terminal state for most events --
// no per-topic dispatch handler exists for their provider/topic, so
// nothing more was ever going to happen. Only "processed"/"failed" mean a
// dispatch handler (myparcel's shipment_status_change today) actually ran.
function MessageStatusBadge({ status, dispatchError }: { status: WebhookEvent['status']; dispatchError: string }) {
  if (status === 'failed') {
    return (
      <span
        title={dispatchError || undefined}
        className="px-1.5 py-0.5 rounded text-2xs font-medium bg-red-500/15 text-danger cursor-help"
      >
        Failed
      </span>
    )
  }
  if (status === 'processed') {
    return <span className="px-1.5 py-0.5 rounded text-2xs font-medium bg-emerald-500/15 text-success">Processed</span>
  }
  return <span className="px-1.5 py-0.5 rounded text-2xs font-medium bg-fill text-ink-3">Received</span>
}

// Truncated one-line preview that expands into the full formatted JSON on
// click -- avoids a wall of raw JSON per row while still letting an admin
// actually see what was sent.
function PayloadPreview({ payload }: { payload: unknown }) {
  const [expanded, setExpanded] = useState(false)
  const json = JSON.stringify(payload)
  const pretty = JSON.stringify(payload, null, 2)

  if (!json || json === 'null') {
    return <span className="text-ink-4">—</span>
  }

  if (expanded) {
    return (
      <div>
        <pre className="whitespace-pre-wrap break-all text-2xs bg-sunken rounded-lg p-2 max-h-64 overflow-y-auto">{pretty}</pre>
        <button type="button" onClick={() => setExpanded(false)} className="text-accent hover:underline mt-1">Collapse</button>
      </div>
    )
  }

  const preview = json.length > 60 ? `${json.slice(0, 60)}…` : json
  return (
    <button type="button" onClick={() => setExpanded(true)} className="truncate max-w-[360px] block text-left hover:underline" title="View full payload">
      {preview}
    </button>
  )
}
