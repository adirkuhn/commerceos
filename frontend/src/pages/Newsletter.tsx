import { useEffect, useState } from 'react'
import { Download, Search, Trash2 } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useConfirm } from '../components/common/useConfirm'
import { useDateFormat } from '../format/DateFormatProvider'
import { exportSubscribersCSV } from '../newsletter/api'
import { downloadBlob } from '../newsletter/download'
import { SUBSCRIBER_STATUSES, type Subscriber, type SubscriberStatus } from '../newsletter/types'
import { useSubscribers } from '../newsletter/useSubscribers'
import { useQueryEnum, useQueryPage, useQueryString } from '../routing/useQueryState'

const PAGE_SIZE = 50
const SEARCH_DELAY_MS = 300
const ALL = 'all'
type StatusFilter = SubscriberStatus | typeof ALL

const STATUS_LABEL: Record<SubscriberStatus, string> = { pending: 'Awaiting confirmation', confirmed: 'Subscribed', unsubscribed: 'Unsubscribed' }
const STATUS_STYLE: Record<SubscriberStatus, string> = {
  pending: 'bg-amber-500/15 text-warning',
  confirmed: 'bg-green-500/15 text-success',
  unsubscribed: 'bg-gray-500/15 text-ink-4',
}

export default function Newsletter() {
  const { token } = useAuth()
  const { formatDate } = useDateFormat()
  const [status, setStatus] = useQueryEnum<StatusFilter>('status', [ALL, ...SUBSCRIBER_STATUSES], ALL, { resets: ['page'] })
  const [searchParam, setSearch] = useQueryString('q', { resets: ['page'] })
  const [page, setPage] = useQueryPage()
  const [searchDraft, setSearchDraft] = useState(searchParam ?? '')
  const { subscribers, total, loading, error, erase } = useSubscribers(token ?? '', {
    status: status === ALL ? undefined : status, search: searchParam ?? undefined, page, perPage: PAGE_SIZE,
  })
  const { confirm, confirmDialog } = useConfirm()
  const [actionError, setActionError] = useState('')
  const [exporting, setExporting] = useState(false)

  // The URL follows the search box once typing pauses.
  useEffect(() => {
    if (searchDraft === (searchParam ?? '')) return
    const t = setTimeout(() => setSearch(searchDraft), SEARCH_DELAY_MS)
    return () => clearTimeout(t)
  }, [searchDraft, searchParam, setSearch])

  async function handleExport() {
    setActionError('')
    setExporting(true)
    try {
      downloadBlob(await exportSubscribersCSV(token ?? ''), 'newsletter-subscribers.csv')
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to export subscribers')
    } finally {
      setExporting(false)
    }
  }

  async function handleErase(s: Subscriber) {
    if (!(await confirm({
      title: `Erase ${s.email}?`,
      message: 'Their subscription and consent record are deleted for good. Use this for data-removal requests; to stop emails, they can unsubscribe.',
      confirmLabel: 'Erase subscriber', destructive: true,
    }))) return
    setActionError('')
    try {
      await erase(s.id)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Failed to erase subscriber')
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex flex-wrap items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 min-w-48 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input value={searchDraft} onChange={e => setSearchDraft(e.target.value)} placeholder="Search by email..." aria-label="Search subscribers"
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus" />
        </div>
        <select value={status} onChange={e => setStatus(e.target.value as StatusFilter)} aria-label="Status"
          className="px-2 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink outline-none focus:ring-1 focus:ring-focus">
          <option value={ALL}>All statuses</option>
          {SUBSCRIBER_STATUSES.map(s => <option key={s} value={s}>{STATUS_LABEL[s]}</option>)}
        </select>
        <button onClick={() => void handleExport()} disabled={exporting}
          className="ml-auto flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50">
          <Download size={13} /> {exporting ? 'Exporting…' : 'Export subscribed (CSV)'}
        </button>
      </div>
      <p className="px-5 py-2 text-2xs text-ink-4">
        Shoppers join from your storefront's signup form and confirm by email (double opt-in). Only confirmed subscribers are exported.
      </p>
      {(error || actionError) && <p role="alert" className="px-5 py-2 text-xs text-danger">{error || actionError}</p>}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-5 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Email</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Status</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Signed up</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">Confirmed</th>
              <th className="px-3 py-2.5 w-10" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && <tr><td colSpan={5} className="px-4 py-6 text-center text-ink-4">Loading…</td></tr>}
            {!loading && !subscribers.length && <tr><td colSpan={5} className="px-4 py-6 text-center text-ink-4">No subscribers found</td></tr>}
            {!loading && subscribers.map(s => (
              <tr key={s.id} className="group hover:bg-hover/60">
                <td className="px-5 py-2.5 font-medium text-ink">{s.email}</td>
                <td className="px-3 py-2.5">
                  <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium', STATUS_STYLE[s.status])}>{STATUS_LABEL[s.status]}</span>
                </td>
                <td className="px-3 py-2.5 text-ink-3">{formatDate(s.created_at)}</td>
                <td className="px-3 py-2.5 text-ink-3">{s.confirmed_at ? formatDate(s.confirmed_at) : '—'}</td>
                <td className="px-3 py-2.5">
                  <button aria-label={`Erase ${s.email}`} onClick={() => void handleErase(s)}
                    className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover opacity-0 group-hover:opacity-100 focus:opacity-100">
                    <Trash2 size={12} className="text-ink-3 hover:text-danger" />
                  </button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={total} onPageChange={setPage} />
      {confirmDialog}
    </div>
  )
}
