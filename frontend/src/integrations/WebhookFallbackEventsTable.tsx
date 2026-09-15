import { useState } from 'react'
import Pagination from '../components/common/Pagination'
import { useWebhookFallbackEvents } from './hooks/useWebhookFallbackEvents'

export function WebhookFallbackEventsTable({
  events, total, page, perPage, loading, error, setPage, retry, abandon, abandonAll,
}: ReturnType<typeof useWebhookFallbackEvents>) {
  const [rowError, setRowError] = useState('')
  const [busyId, setBusyId] = useState<string | null>(null)
  const [abandoningAll, setAbandoningAll] = useState(false)

  async function handleRetry(id: string) {
    setRowError('')
    setBusyId(id)
    try {
      await retry(id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Retry failed')
    } finally {
      setBusyId(null)
    }
  }

  async function handleAbandon(id: string) {
    setRowError('')
    setBusyId(id)
    try {
      await abandon(id)
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Abandon failed')
    } finally {
      setBusyId(null)
    }
  }

  async function handleAbandonAll() {
    if (!window.confirm(`Abandon all ${total} pending webhook${total === 1 ? '' : 's'}? This keeps an audit trail but stops any retry.`)) {
      return
    }
    setRowError('')
    setAbandoningAll(true)
    try {
      await abandonAll()
    } catch (err) {
      setRowError(err instanceof Error ? err.message : 'Abandon all failed')
    } finally {
      setAbandoningAll(false)
    }
  }

  if (!loading && events.length === 0 && total === 0 && !error) return null

  return (
    <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 p-4">
      <div className="flex items-start justify-between gap-3 mb-1">
        <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Undelivered webhooks</h2>
        {total > 0 && (
          <button
            onClick={handleAbandonAll}
            disabled={abandoningAll}
            className="shrink-0 px-2 py-1 rounded text-[10px] font-medium dark:bg-[#111827] bg-gray-100 dark:text-gray-400 text-gray-500 hover:text-red-400 disabled:opacity-50 transition-colors"
          >
            {abandoningAll ? 'Abandoning…' : 'Abandon all'}
          </button>
        )}
      </div>
      <p className="text-xs dark:text-gray-500 text-gray-400 mb-3">
        Webhooks that couldn't reach the processing queue after 3 attempts -- saved here instead of lost. Retry re-attempts
        delivery now; Abandon discards it (kept for audit, not deleted).
      </p>

      {error && <p className="text-xs text-red-400 mb-2">{error}</p>}
      {rowError && <p className="text-xs text-red-400 mb-2">{rowError}</p>}

      {loading && events.length === 0 ? (
        <p className="text-xs dark:text-gray-500 text-gray-400">Loading…</p>
      ) : events.length === 0 ? (
        <p className="text-xs dark:text-gray-500 text-gray-400">Nothing pending -- every webhook has been delivered.</p>
      ) : (
        <div className="max-h-96 overflow-y-auto overflow-x-auto border dark:border-[#2D3748] border-gray-100 rounded-lg">
          <table className="w-full text-xs">
            <thead className="sticky top-0 dark:bg-[#1F2937] bg-white">
              <tr className="text-left dark:text-gray-500 text-gray-400 border-b dark:border-[#2D3748] border-gray-200">
                <th className="py-2 pr-3 pl-3 font-medium">Provider</th>
                <th className="py-2 pr-3 font-medium">Event ID</th>
                <th className="py-2 pr-3 font-medium">Received</th>
                <th className="py-2 pr-3 font-medium">Status</th>
                <th className="py-2 pr-3 font-medium"></th>
              </tr>
            </thead>
            <tbody>
              {events.map(ev => (
                <tr key={ev.id} className="border-b dark:border-[#2D3748] border-gray-100 last:border-0">
                  <td className="py-2 pr-3 pl-3 dark:text-gray-300 text-gray-700 capitalize">{ev.provider}</td>
                  <td className="py-2 pr-3 dark:text-gray-400 text-gray-500 font-mono truncate max-w-[160px]" title={ev.event_id}>{ev.event_id}</td>
                  <td className="py-2 pr-3 dark:text-gray-400 text-gray-500">{new Date(ev.received_at).toLocaleString()}</td>
                  <td className="py-2 pr-3 dark:text-gray-400 text-gray-500">{ev.status}</td>
                  <td className="py-2 pr-3">
                    <div className="flex gap-2 justify-end">
                      <button
                        onClick={() => handleRetry(ev.id)}
                        disabled={busyId === ev.id}
                        className="px-2 py-1 rounded dark:bg-[#111827] bg-gray-100 dark:text-gray-400 text-gray-500 hover:text-blue-400 disabled:opacity-50 transition-colors"
                      >
                        Retry
                      </button>
                      <button
                        onClick={() => handleAbandon(ev.id)}
                        disabled={busyId === ev.id}
                        className="px-2 py-1 rounded dark:bg-[#111827] bg-gray-100 dark:text-gray-400 text-gray-500 hover:text-red-400 disabled:opacity-50 transition-colors"
                      >
                        Abandon
                      </button>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
      <Pagination page={page} pageSize={perPage} totalItems={total} onPageChange={setPage} />
    </div>
  )
}
