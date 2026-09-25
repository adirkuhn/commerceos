import { useState } from 'react'
import { RefreshCw, AlertTriangle } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSyncLogs } from '../integrations/hooks/useSyncLogs'
import { useSyncLogsNeedingAttention } from '../integrations/hooks/useSyncLogsNeedingAttention'
import * as api from '../integrations/api'
import type { SyncLogStatus, SyncLogDirection } from '../integrations/types'
import { useQueryEnum, useQueryPage } from '../routing/useQueryState'
import { formatDateTime } from '../format/date'

const PAGE_SIZE = 20

const STATUS_STYLES: Record<SyncLogStatus, string> = {
  success: 'bg-green-500/15 text-success',
  error: 'bg-red-500/15 text-danger',
}

const DIRECTION_LABEL: Record<SyncLogDirection, string> = {
  push: 'CommerceOS → Channel',
  import: 'Channel → CommerceOS',
}

function formatValue(v: string): string {
  return v === '' ? '—' : v
}

// SyncLog is the tenant-wide sync history page -- "All" shows every push/
// import/stock attempt, newest first; "Needs attention" narrows to
// listings whose most recent attempt failed and hasn't been dismissed,
// with Retry (re-push, full resync) and Dismiss (silence without
// pretending it succeeded) actions. A successful retry, or a dismiss,
// drops the row out of "Needs attention" on its own -- it's a live query,
// not separate tracked state (see backend's ListNeedingAttention doc).
const SYNC_LOG_VIEWS = ['all', 'needing-attention'] as const
type SyncLogView = typeof SYNC_LOG_VIEWS[number]

export default function SyncLog() {
  const { token } = useAuth()
  // View and page live in the URL (?view=, ?page=) -- refresh and Back keep them.
  const [view, setView] = useQueryEnum('view', SYNC_LOG_VIEWS, 'all', { resets: ['page'] })
  const [page, setPage] = useQueryPage()
  const all = useSyncLogs(token ?? '', { page, perPage: PAGE_SIZE })
  const needingAttention = useSyncLogsNeedingAttention(token ?? '', { page, perPage: PAGE_SIZE })
  const { entries, total, loading, error, refresh } = view === 'all' ? all : needingAttention

  const [busyId, setBusyId] = useState<string | null>(null)
  const [busyAction, setBusyAction] = useState<'retry' | 'dismiss' | null>(null)
  const [actionError, setActionError] = useState('')

  function switchView(next: SyncLogView) {
    setView(next)
  }

  async function handleRetry(id: string) {
    setBusyId(id)
    setBusyAction('retry')
    setActionError('')
    try {
      await api.retryFailedSync(token ?? '', id)
      await needingAttention.refresh()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to retry sync')
    } finally {
      setBusyId(null)
      setBusyAction(null)
    }
  }

  async function handleDismiss(id: string) {
    setBusyId(id)
    setBusyAction('dismiss')
    setActionError('')
    try {
      await api.dismissFailedSync(token ?? '', id)
      await needingAttention.refresh()
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to dismiss sync')
    } finally {
      setBusyId(null)
      setBusyAction(null)
    }
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <h1 className="text-sm font-semibold text-ink">Sync History</h1>
        <div className="flex items-center gap-1 ml-4">
          <button aria-pressed={view === 'all'}
            onClick={() => switchView('all')}
            className={clsx('px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              view === 'all' ? 'bg-accent-solid text-white' : 'text-ink-3 hover:bg-hover')}
          >
            All
          </button>
          <button
            onClick={() => switchView('needing-attention')}
            className={clsx('flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              view === 'needing-attention' ? 'bg-amber-500 text-white' : 'text-ink-3 hover:bg-hover')}
          >
            <AlertTriangle size={12} /> Needs attention
          </button>
        </div>
        <button
          onClick={refresh}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line text-xs text-ink-2 hover:bg-hover transition-colors ml-auto"
        >
          <RefreshCw size={12} /> Refresh
        </button>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">
          {total} {view === 'all' ? 'sync' : 'listing'}{total === 1 ? '' : 's'}
        </span>
        {(error || actionError) && <span className="text-xs text-danger ml-auto">{error || actionError}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Synced</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Product</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Channel</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Direction</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Changes</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Status</th>
              {view === 'needing-attention' && <th className="px-3 py-2.5 w-56" />}
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={view === 'needing-attention' ? 7 : 6} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>
            )}
            {!loading && entries.length === 0 && (
              <tr><td colSpan={view === 'needing-attention' ? 7 : 6} className="px-4 py-6 text-center text-ink-4">
                {view === 'all' ? 'No syncs yet' : "No syncs need attention — everything's resolved"}
              </td></tr>
            )}
            {entries.map(entry => {
              const changedFields = Object.entries(entry.changes)
              return (
                <tr key={entry.id} className="align-top hover:bg-hover/60 transition-colors">
                  <td className="px-3 py-2.5 text-ink-3 whitespace-nowrap">{formatDateTime(entry.synced_at)}</td>
                  <td className="px-3 py-2.5 text-ink">{entry.product_name || entry.product_id}</td>
                  <td className="px-3 py-2.5 text-ink-3">
                    {entry.provider}
                    {entry.store_url && <span className="text-ink-4"> · {entry.store_url}</span>}
                  </td>
                  <td className="px-3 py-2.5 text-ink-3 whitespace-nowrap">{DIRECTION_LABEL[entry.direction]}</td>
                  <td className="px-3 py-2.5">
                    {changedFields.length === 0 ? (
                      <span className="text-ink-4">No changes</span>
                    ) : (
                      <div className="space-y-0.5">
                        {changedFields.map(([field, change]) => (
                          <div key={field} className="flex items-baseline gap-1.5">
                            <span className="text-ink-4 shrink-0">{field}:</span>
                            <span className="text-ink-3 truncate max-w-[220px]" title={change.from}>{formatValue(change.from)}</span>
                            <span className="text-ink-4 shrink-0">→</span>
                            <span className="text-ink font-medium truncate max-w-[220px]" title={change.to}>{formatValue(change.to)}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-2.5">
                    <span className={`px-2 py-0.5 rounded-full text-2xs font-medium whitespace-nowrap ${STATUS_STYLES[entry.status]}`}>
                      {entry.status}
                    </span>
                    {entry.status === 'error' && entry.error_message && (
                      <p className="text-2xs text-danger mt-1 max-w-[220px]">{entry.error_message}</p>
                    )}
                  </td>
                  {view === 'needing-attention' && (
                    <td className="px-3 py-2.5 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => handleDismiss(entry.id)}
                          disabled={busyId === entry.id}
                          className="px-3 py-1.5 rounded-lg border border-line text-ink-2 text-2xs font-medium hover:bg-gray-500/10 disabled:opacity-50 transition-colors"
                        >
                          {busyId === entry.id && busyAction === 'dismiss' ? 'Dismissing…' : 'Dismiss'}
                        </button>
                        <button
                          onClick={() => handleRetry(entry.id)}
                          disabled={busyId === entry.id}
                          className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-2xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
                        >
                          {busyId === entry.id && busyAction === 'retry' ? 'Retrying…' : 'Retry'}
                        </button>
                      </div>
                    </td>
                  )}
                </tr>
              )
            })}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
    </div>
  )
}
