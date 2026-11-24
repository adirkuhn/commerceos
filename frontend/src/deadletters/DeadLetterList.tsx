import { useEffect, useState, type FormEvent } from 'react'
import { useSearchParams } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useDateFormat } from '../format/DateFormatProvider'
import { useQueryEnum } from '../routing/useQueryState'
import { useResource } from '../routing/useResource'
import { useStackNavigation } from '../routing/useStackNavigation'
import * as api from './api'
import { deadLetterPath, waitingSince } from './format'
import { DEAD_LETTER_STREAMS, type DeadLetterStream, type DeadLetterSummary } from './types'

// Filter params live in the URL so a refresh or Back keeps the view; the
// cursor is cleared whenever a filter or the stream changes.
const FILTER_PARAMS = ['subject', 'consumer', 'tenant_id'] as const
type FilterParam = typeof FILTER_PARAMS[number]

// DeadLetterList is one stream's dead letters, oldest first, with a count
// on each stream tab, filters, and cursor paging. A row opens its own URL.
export function DeadLetterList() {
  const { t } = useTranslation(['settings', 'common'])
  const { token } = useAuth()
  const { formatDateTime } = useDateFormat()
  const { open } = useStackNavigation()
  const [searchParams, setSearchParams] = useSearchParams()
  const [stream, setStream] = useQueryEnum<DeadLetterStream>('stream', DEAD_LETTER_STREAMS, 'events', { resets: ['cursor'] })
  const filters = Object.fromEntries(FILTER_PARAMS.map(p => [p, searchParams.get(p) ?? ''])) as Record<FilterParam, string>
  const cursor = Number(searchParams.get('cursor')) || undefined
  const [draft, setDraft] = useState(filters)
  useEffect(() => { setDraft(filters) }, [filters.subject, filters.consumer, filters.tenant_id]) // eslint-disable-line react-hooks/exhaustive-deps

  const page = useResource(
    () => api.listDeadLetters(token ?? '', { stream, subject: filters.subject, consumer: filters.consumer, tenantId: filters.tenant_id, cursor }),
    [token, stream, filters.subject, filters.consumer, filters.tenant_id, cursor],
  )
  // Each tab's badge is that stream's total, filters ignored.
  const totals = useResource(async () => {
    const pages = await Promise.all(DEAD_LETTER_STREAMS.map(s => api.listDeadLetters(token ?? '', { stream: s, limit: 1 })))
    return Object.fromEntries(DEAD_LETTER_STREAMS.map((s, i) => [s, pages[i].total])) as Record<DeadLetterStream, number>
  }, [token])

  function writeParams(update: (next: URLSearchParams) => void) {
    setSearchParams(prev => {
      const next = new URLSearchParams(prev)
      update(next)
      return next
    }, { replace: true })
  }

  function applyFilters(e: FormEvent) {
    e.preventDefault()
    writeParams(next => {
      for (const p of FILTER_PARAMS) {
        const v = draft[p].trim()
        if (v) next.set(p, v)
        else next.delete(p)
      }
      next.delete('cursor')
    })
  }

  function clearFilters() {
    writeParams(next => { for (const p of [...FILTER_PARAMS, 'cursor']) next.delete(p) })
  }

  const filtered = FILTER_PARAMS.some(p => filters[p])
  const items = page.data?.items ?? []

  return (
    <div className="p-5 flex flex-col gap-4">
      <p className="text-xs text-ink-4 max-w-2xl">{t('deadLetters.intro')}</p>

      <div className="flex gap-1 border-b border-line" role="tablist">
        {DEAD_LETTER_STREAMS.map(s => (
          <button
            key={s} type="button" role="tab" aria-selected={stream === s} onClick={() => setStream(s)}
            className={`px-2.5 py-1.5 text-xs font-medium border-b-2 transition-colors flex items-center gap-1.5 ${
              stream === s ? 'border-accent text-accent' : 'border-transparent text-ink-4 hover:text-ink-2'
            }`}
          >
            {t(`deadLetters.streams.${s}`)}
            {totals.data && (
              <span
                data-testid={`count-${s}`}
                className={`px-1.5 rounded-full text-2xs tabular-nums ${totals.data[s] > 0 ? 'bg-danger/10 text-danger' : 'bg-fill text-ink-4'}`}
              >
                {totals.data[s]}
              </span>
            )}
          </button>
        ))}
      </div>

      <form onSubmit={applyFilters} className="flex flex-wrap items-end gap-2">
        {FILTER_PARAMS.map(p => (
          <div key={p}>
            <label htmlFor={`dl-filter-${p}`} className="block text-2xs text-ink-3 mb-1">
              {t(`deadLetters.filters.${p === 'tenant_id' ? 'tenant' : p}`)}
            </label>
            <input
              id={`dl-filter-${p}`} value={draft[p]} onChange={e => setDraft(d => ({ ...d, [p]: e.target.value }))}
              className="w-56 px-2.5 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink outline-none focus:ring-1 focus:ring-focus"
            />
          </div>
        ))}
        <button type="submit" className="px-2.5 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover">
          {t('deadLetters.filters.apply')}
        </button>
        {filtered && (
          <button type="button" onClick={clearFilters} className="px-2.5 py-1.5 rounded-lg bg-fill text-ink-3 text-xs hover:text-ink-2">
            {t('deadLetters.filters.clear')}
          </button>
        )}
      </form>

      {(page.error || totals.error) && <p className="text-xs text-danger">{page.error || totals.error}</p>}

      {page.loading ? (
        <p className="text-xs text-ink-4">{t('common:status.loading')}</p>
      ) : items.length === 0 ? (
        !page.error && <p className="text-xs text-ink-4">{filtered ? t('deadLetters.emptyFiltered') : t('deadLetters.empty')}</p>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full text-xs">
            <thead>
              <tr className="text-left text-ink-4 border-b border-line">
                <th className="py-2 pr-3 font-medium">{t('deadLetters.columns.seq')}</th>
                <th className="py-2 pr-3 font-medium">{t('deadLetters.columns.subject')}</th>
                <th className="py-2 pr-3 font-medium">{t('deadLetters.columns.consumer')}</th>
                <th className="py-2 pr-3 font-medium">{t('deadLetters.columns.tenant')}</th>
                <th className="py-2 pr-3 font-medium">{t('deadLetters.columns.error')}</th>
                <th className="py-2 pr-3 font-medium">{t('deadLetters.columns.waitingSince')}</th>
              </tr>
            </thead>
            <tbody>
              {items.map(d => (
                <Row key={d.sequence} d={d} waiting={formatDateTime(waitingSince(d))} onOpen={() => open(deadLetterPath(d.stream, d.sequence))} />
              ))}
            </tbody>
          </table>
        </div>
      )}

      {(cursor || page.data?.next_cursor) && (
        <div className="flex gap-2">
          {cursor && (
            <button type="button" onClick={() => writeParams(next => next.delete('cursor'))} className="px-2.5 py-1.5 rounded-lg bg-fill text-ink-3 text-xs hover:text-ink-2">
              {t('deadLetters.firstPage')}
            </button>
          )}
          {page.data?.next_cursor && (
            <button
              type="button" onClick={() => writeParams(next => next.set('cursor', String(page.data?.next_cursor)))}
              className="px-2.5 py-1.5 rounded-lg bg-fill text-ink-3 text-xs hover:text-ink-2"
            >
              {t('deadLetters.nextPage')}
            </button>
          )}
        </div>
      )}
    </div>
  )
}

function Row({ d, waiting, onOpen }: { d: DeadLetterSummary; waiting: string; onOpen: () => void }) {
  const { t } = useTranslation('settings')
  return (
    <tr className="border-b border-line last:border-0 align-top hover:bg-hover cursor-pointer" onClick={onOpen}>
      <td className="py-2 pr-3">
        <a
          href={deadLetterPath(d.stream, d.sequence)}
          onClick={e => { e.preventDefault(); e.stopPropagation(); onOpen() }}
          className="text-accent hover:underline tabular-nums"
        >
          {d.sequence}
        </a>
      </td>
      <td className="py-2 pr-3 text-ink-2 font-mono">{d.subject}</td>
      <td className="py-2 pr-3 text-ink-3">
        {d.consumer_known ? d.consumer : <span className="italic text-ink-4" title={t('deadLetters.unknownConsumerHint')}>{t('deadLetters.unknownConsumer')}</span>}
      </td>
      <td className="py-2 pr-3 text-ink-3 font-mono">{d.tenant_id || '—'}</td>
      <td className="py-2 pr-3 text-danger max-w-[320px] truncate" title={d.error}>{d.error || '—'}</td>
      <td className="py-2 pr-3 text-ink-3 whitespace-nowrap">{waiting}</td>
    </tr>
  )
}
