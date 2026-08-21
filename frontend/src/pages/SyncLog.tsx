import { useState } from 'react'
import { RefreshCw, AlertTriangle } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSyncLogs } from '../integrations/hooks/useSyncLogs'
import { useSyncLogsNeedingAttention } from '../integrations/hooks/useSyncLogsNeedingAttention'
import * as api from '../integrations/api'
import type { SyncLogStatus, SyncLogDirection } from '../integrations/types'

const PAGE_SIZE = 20

const STATUS_STYLES: Record<SyncLogStatus, string> = {
  success: 'bg-green-500/15 text-green-400',
  error: 'bg-red-500/15 text-red-400',
}

const DIRECTION_LABEL: Record<SyncLogDirection, string> = {
  push: 'CommerceOS → Channel',
  import: 'Channel → CommerceOS',
}

function formatDateTime(iso: string): string {
  return new Date(iso).toLocaleString()
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
export default function SyncLog() {
  const { token } = useAuth()
  const [view, setView] = useState<'all' | 'needing-attention'>('all')
  const [page, setPage] = useState(1)
  const all = useSyncLogs(token ?? '', { page, perPage: PAGE_SIZE })
  const needingAttention = useSyncLogsNeedingAttention(token ?? '', { page, perPage: PAGE_SIZE })
  const { entries, total, loading, error, refresh } = view === 'all' ? all : needingAttention

  const [busyId, setBusyId] = useState<string | null>(null)
  const [busyAction, setBusyAction] = useState<'retry' | 'dismiss' | null>(null)
  const [actionError, setActionError] = useState('')

  function switchView(next: 'all' | 'needing-attention') {
    setView(next)
    setPage(1)
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
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <h1 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Sync History</h1>
        <div className="flex items-center gap-1 ml-4">
          <button
            onClick={() => switchView('all')}
            className={clsx('px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              view === 'all' ? 'bg-blue-600 text-white' : 'dark:text-gray-400 text-gray-500 hover:dark:bg-[#1F2937] hover:bg-gray-100')}
          >
            All
          </button>
          <button
            onClick={() => switchView('needing-attention')}
            className={clsx('flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
              view === 'needing-attention' ? 'bg-amber-500 text-white' : 'dark:text-gray-400 text-gray-500 hover:dark:bg-[#1F2937] hover:bg-gray-100')}
          >
            <AlertTriangle size={12} /> Needs attention
          </button>
        </div>
        <button
          onClick={refresh}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200 text-xs dark:text-gray-300 text-gray-700 hover:dark:bg-[#1F2937] hover:bg-gray-100 transition-colors ml-auto"
        >
          <RefreshCw size={12} /> Refresh
        </button>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">
          {total} {view === 'all' ? 'sync' : 'listing'}{total === 1 ? '' : 's'}
        </span>
        {(error || actionError) && <span className="text-xs text-red-400 ml-auto">{error || actionError}</span>}
      </div>

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Synced</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Product</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Channel</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Direction</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Changes</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Status</th>
              {view === 'needing-attention' && <th className="px-3 py-2.5 w-56" />}
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={view === 'needing-attention' ? 7 : 6} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && entries.length === 0 && (
              <tr><td colSpan={view === 'needing-attention' ? 7 : 6} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">
                {view === 'all' ? 'No syncs yet' : "No syncs need attention — everything's resolved"}
              </td></tr>
            )}
            {entries.map(entry => {
              const changedFields = Object.entries(entry.changes)
              return (
                <tr key={entry.id} className="align-top hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                  <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500 whitespace-nowrap">{formatDateTime(entry.synced_at)}</td>
                  <td className="px-3 py-2.5 dark:text-gray-200 text-gray-800">{entry.product_name || entry.product_id}</td>
                  <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">
                    {entry.provider}
                    {entry.store_url && <span className="dark:text-gray-500 text-gray-400"> · {entry.store_url}</span>}
                  </td>
                  <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500 whitespace-nowrap">{DIRECTION_LABEL[entry.direction]}</td>
                  <td className="px-3 py-2.5">
                    {changedFields.length === 0 ? (
                      <span className="dark:text-gray-500 text-gray-400">No changes</span>
                    ) : (
                      <div className="space-y-0.5">
                        {changedFields.map(([field, change]) => (
                          <div key={field} className="flex items-baseline gap-1.5">
                            <span className="dark:text-gray-500 text-gray-400 shrink-0">{field}:</span>
                            <span className="dark:text-gray-400 text-gray-500 truncate max-w-[220px]" title={change.from}>{formatValue(change.from)}</span>
                            <span className="dark:text-gray-600 text-gray-300 shrink-0">→</span>
                            <span className="dark:text-gray-100 text-gray-900 font-medium truncate max-w-[220px]" title={change.to}>{formatValue(change.to)}</span>
                          </div>
                        ))}
                      </div>
                    )}
                  </td>
                  <td className="px-3 py-2.5">
                    <span className={`px-2 py-0.5 rounded-full text-[10px] font-medium whitespace-nowrap ${STATUS_STYLES[entry.status]}`}>
                      {entry.status}
                    </span>
                    {entry.status === 'error' && entry.error_message && (
                      <p className="text-[10px] text-red-400 mt-1 max-w-[220px]">{entry.error_message}</p>
                    )}
                  </td>
                  {view === 'needing-attention' && (
                    <td className="px-3 py-2.5 text-right">
                      <div className="flex items-center justify-end gap-2">
                        <button
                          onClick={() => handleDismiss(entry.id)}
                          disabled={busyId === entry.id}
                          className="px-3 py-1.5 rounded-lg border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-600 text-[11px] font-medium hover:bg-gray-500/10 disabled:opacity-50 transition-colors"
                        >
                          {busyId === entry.id && busyAction === 'dismiss' ? 'Dismissing…' : 'Dismiss'}
                        </button>
                        <button
                          onClick={() => handleRetry(entry.id)}
                          disabled={busyId === entry.id}
                          className="px-3 py-1.5 rounded-lg bg-blue-600 text-white text-[11px] font-medium hover:bg-blue-700 disabled:opacity-50 transition-colors"
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
