import { useEffect, useState } from 'react'
import { useNavigate, useLocation } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { Search, Trash2, ChevronLeft, ChevronRight } from 'lucide-react'
import { useAuth } from '../../auth/AuthContext'
import { apiErrorText } from '../../api/apiErrorText'
import { useDateFormat } from '../../format/DateFormatProvider'
import { useConfirm } from '../../components/common/useConfirm'
import { useQueryEnum, useQueryPage, useQueryString } from '../../routing/useQueryState'
import { subPathSegments } from '../../routes'
import * as api from '../api'
import type { SentEmailList, SentEmailSource, SentEmailStatus } from '../types'
import { SENT_PATH } from '../messageCenterPaths'
import { SentEmailDetail } from './SentEmailDetail'
import {
  SENT_EMAIL_SOURCES, SENT_EMAIL_STATUSES, STATUS_CLASSES, dayEnd, dayStart, sentReason, sentSourceLabels, sentStatusLabels,
} from './sentLabels'

const PER_PAGE = 50
const ALL = 'all'
const FILTER_RESETS = { resets: ['page'] } as const

// SentEmailsPanel is the Message Center's email history: every email sent,
// failed or skipped, kept until deleted. Filters and page live in the URL;
// /message-center/sent/<id> opens one email.
export default function SentEmailsPanel() {
  const { token } = useAuth()
  const { t } = useTranslation(['messages', 'common'])
  const { formatDateTime } = useDateFormat()
  const { confirm, confirmDialog } = useConfirm()
  const navigate = useNavigate()
  const location = useLocation()
  const [openId] = subPathSegments(location.pathname, SENT_PATH)

  const [status, setStatus] = useQueryEnum<SentEmailStatus | typeof ALL>('status', [ALL, ...SENT_EMAIL_STATUSES], ALL, FILTER_RESETS)
  const [source, setSource] = useQueryEnum<SentEmailSource | typeof ALL>('source', [ALL, ...SENT_EMAIL_SOURCES], ALL, FILTER_RESETS)
  const [q, setQ] = useQueryString('q', FILTER_RESETS)
  const [from, setFrom] = useQueryString('from', FILTER_RESETS)
  const [to, setTo] = useQueryString('to', FILTER_RESETS)
  const [page, setPage] = useQueryPage()

  const [list, setList] = useState<SentEmailList | null>(null)
  const [error, setError] = useState('')
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [reload, setReload] = useState(0)

  useEffect(() => {
    let current = true
    setError('')
    api.listSentEmails(token ?? '', {
      status: status === ALL ? undefined : status, source: source === ALL ? undefined : source, q: q ?? undefined,
      from: from ? dayStart(from) : undefined, to: to ? dayEnd(to) : undefined, page, per_page: PER_PAGE,
    }).then(
      data => { if (current) { setList(data); setSelected(new Set()) } },
      err => { if (current) setError(apiErrorText(err) ?? t('common:status.error')) },
    )
    return () => { current = false }
  }, [token, status, source, q, from, to, page, reload, t])

  const statusLabels = sentStatusLabels(t)
  const sourceLabels = sentSourceLabels(t)
  const pages = list ? Math.max(1, Math.ceil(list.total / list.per_page)) : 1

  function toggle(id: string) {
    setSelected(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function deleteSelected() {
    const ids = [...selected]
    if (!(await confirm({ title: t('sent.deleteTitle', { count: ids.length }), message: t('sent.deleteMessage'), confirmLabel: t('sent.deleteConfirm'), destructive: true }))) return
    try {
      await api.deleteSentEmails(token ?? '', ids)
      setReload(n => n + 1)
    } catch (err) {
      setError(apiErrorText(err) ?? t('common:status.error'))
    }
  }

  if (openId) {
    return <SentEmailDetail id={openId} onBack={() => navigate(`${SENT_PATH}${location.search}`)} onDeleted={() => { setReload(n => n + 1); navigate(SENT_PATH) }} />
  }

  const input = 'px-2 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus'
  return (
    <div className="flex flex-col h-full">
      <div className="flex flex-wrap items-center gap-2 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 min-w-[180px] max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input value={q ?? ''} onChange={e => setQ(e.target.value)} placeholder={t('sent.search')} aria-label={t('sent.search')}
            className={`w-full pl-8 ${input}`} />
        </div>
        <select aria-label={t('sent.colStatus')} value={status} onChange={e => setStatus(e.target.value as SentEmailStatus | typeof ALL)} className={input}>
          <option value={ALL}>{t('sent.allStatuses')}</option>
          {SENT_EMAIL_STATUSES.map(s => <option key={s} value={s}>{statusLabels[s]}</option>)}
        </select>
        <select aria-label={t('sent.colSource')} value={source} onChange={e => setSource(e.target.value as SentEmailSource | typeof ALL)} className={input}>
          <option value={ALL}>{t('sent.allSources')}</option>
          {SENT_EMAIL_SOURCES.map(s => <option key={s} value={s}>{sourceLabels[s]}</option>)}
        </select>
        <label className="flex items-center gap-1 text-xs text-ink-4">
          {t('sent.from')}
          <input type="date" value={from ?? ''} onChange={e => setFrom(e.target.value)} className={input} />
        </label>
        <label className="flex items-center gap-1 text-xs text-ink-4">
          {t('sent.to')}
          <input type="date" value={to ?? ''} onChange={e => setTo(e.target.value)} className={input} />
        </label>
        {selected.size > 0 && (
          <button onClick={deleteSelected} className="ml-auto flex items-center gap-1.5 px-3 py-1.5 text-xs font-medium rounded-lg text-danger hover:bg-hover">
            <Trash2 size={13} /> {t('sent.deleteSelected', { count: selected.size })}
          </button>
        )}
      </div>

      <div className="flex items-center gap-4 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{t('sent.count', { count: list?.total ?? 0 })}</span>
        {error && <span role="alert" className="text-xs text-danger">{error}</span>}
        <div className="ml-auto flex items-center gap-2 text-xs text-ink-4">
          <button aria-label={t('sent.previousPage')} disabled={page <= 1} onClick={() => setPage(page - 1)} className="p-1 rounded hover:bg-hover disabled:opacity-40">
            <ChevronLeft size={14} />
          </button>
          {t('sent.pageOf', { page, pages })}
          <button aria-label={t('sent.nextPage')} disabled={page >= pages} onClick={() => setPage(page + 1)} className="p-1 rounded hover:bg-hover disabled:opacity-40">
            <ChevronRight size={14} />
          </button>
        </div>
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-3 py-2.5 w-8" />
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('sent.colDate')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('sent.colRecipient')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('sent.colSubject')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('sent.colSource')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('sent.colStatus')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {!list && !error && <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>}
            {list && list.items.length === 0 && <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">{t('sent.empty')}</td></tr>}
            {list?.items.map(e => (
              <tr key={e.id} className="hover:bg-hover/60 cursor-pointer" onClick={() => navigate(`${SENT_PATH}/${encodeURIComponent(e.id)}${location.search}`)}>
                <td className="px-3 py-2.5" onClick={ev => ev.stopPropagation()}>
                  <input type="checkbox" aria-label={t('sent.select', { subject: e.subject })} checked={selected.has(e.id)} onChange={() => toggle(e.id)} />
                </td>
                <td className="px-3 py-2.5 whitespace-nowrap text-ink-3">{formatDateTime(e.created_at)}</td>
                <td className="px-3 py-2.5 text-ink-2">{e.recipient || '—'}</td>
                <td className="px-3 py-2.5 max-w-[320px] truncate text-ink">{e.subject}</td>
                <td className="px-3 py-2.5 text-ink-3">{sourceLabels[e.source]}</td>
                <td className="px-3 py-2.5">
                  <span className={STATUS_CLASSES[e.status]}>{statusLabels[e.status]}</span>
                  {e.status !== 'sent' && <span className="block text-2xs text-ink-4 max-w-[220px] truncate" title={sentReason(t, e)}>{sentReason(t, e)}</span>}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {confirmDialog}
    </div>
  )
}
