import { useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { FolderOpen, Trash2 } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { apiErrorText } from '../api/apiErrorText'
import Pagination from '../components/common/Pagination'
import { useConfirm } from '../components/common/useConfirm'
import { Money } from '../format/amountVisibility'
import { useDateFormat } from '../format/DateFormatProvider'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { useSavedQuotes } from './hooks/useSavedQuotes'
import { deleteQuote, getQuote } from './quotesApi'
import { withSavedQuote, withoutQuote } from './savedQuote'
import { SearchField } from './SearchField'
import type { SavedQuoteSummary } from './types'

const PAGE_SIZE = 50
const TH = 'px-3 py-2.5 font-medium text-ink-4 caps-label'

// Saved quotes: open one in the editor or delete it. ?qs= search, ?qpage= page.
export function SavedQuotesTab({ token }: { token: string }) {
  const { t } = useTranslation(['priceList', 'common'])
  const { formatDateTime } = useDateFormat()
  const { confirm, confirmDialog } = useConfirm()
  const [, setSearchParams] = useSearchParams()
  const [openQuoteId] = useQueryString('quote')
  const [urlSearch, setUrlSearch] = useQueryString('qs', { resets: ['qpage'] })
  const search = urlSearch ?? ''
  const [page, setPage] = useQueryPage('qpage')
  const [searchInput, setSearchInput] = useState(search)
  useEffect(() => {
    if (searchInput === search) return
    const timeout = setTimeout(() => setUrlSearch(searchInput), 300)
    return () => clearTimeout(timeout)
  }, [searchInput, search, setUrlSearch])
  useEffect(() => { setSearchInput(search) }, [search])

  const { data, loading, error, refresh } = useSavedQuotes(token, { page, perPage: PAGE_SIZE, search })
  const [busyId, setBusyId] = useState<string | null>(null)
  const [actionError, setActionError] = useState('')
  const quotes = data?.quotes ?? []

  async function open(id: string) {
    setBusyId(id)
    setActionError('')
    try {
      const q = await getQuote(token, id)
      setSearchParams(prev => withSavedQuote(prev, q))
    } catch (e) {
      setActionError(apiErrorText(e) ?? t('saved.openFailed'))
    } finally {
      setBusyId(null)
    }
  }

  async function remove(q: SavedQuoteSummary) {
    const ok = await confirm({
      title: t('saved.deleteTitle', { reference: q.reference, name: q.name }),
      message: t('saved.deleteMessage'), confirmLabel: t('saved.deleteConfirm'), destructive: true,
    })
    if (!ok) return
    setBusyId(q.id)
    setActionError('')
    try {
      await deleteQuote(token, q.id)
      if (q.id === openQuoteId) setSearchParams(prev => withoutQuote(prev), { replace: true })
      refresh()
    } catch (e) {
      setActionError(apiErrorText(e) ?? t('saved.deleteFailed'))
    } finally {
      setBusyId(null)
    }
  }

  return (
    <>
      {confirmDialog}
      <div className="flex flex-wrap items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="flex-1 max-w-sm">
          <SearchField label={t('saved.search')} value={searchInput} onChange={setSearchInput} />
        </div>
        {(error || actionError) && <span role="alert" className="text-xs text-danger ml-auto">{actionError || error}</span>}
      </div>
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className={`${TH} text-left`}>{t('saved.colReference')}</th>
              <th className={`${TH} text-left`}>{t('saved.colName')}</th>
              <th className={`${TH} text-right`}>{t('saved.colLines')}</th>
              <th className={`${TH} text-right`}>{t('saved.colDiscount')}</th>
              <th className={`${TH} text-right`}>{t('saved.colTotal')}</th>
              <th className={`${TH} text-left`}>{t('saved.colUpdated')}</th>
              <th className="w-20" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && quotes.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && quotes.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">{t('saved.empty')}</td></tr>
            )}
            {quotes.map(q => (
              <tr key={q.id} data-testid={`saved-quote-${q.reference}`} className="hover:bg-hover/60">
                <td className="px-3 py-2 font-mono text-ink-2">{q.reference}</td>
                <td className="px-3 py-2">
                  <button onClick={() => open(q.id)} disabled={busyId !== null} className="font-medium text-ink hover:underline text-left">
                    {q.name}
                  </button>
                </td>
                <td className="px-3 py-2 text-right tabular-nums text-ink-2">{q.line_count}</td>
                <td className="px-3 py-2 text-right tabular-nums text-ink-3">{q.discount_basis_points ? `${q.discount_basis_points / 100}%` : '—'}</td>
                <td className="px-3 py-2 text-right tabular-nums font-medium text-ink">
                  {q.currency_code ? <Money minor={q.total_minor_units} currency={q.currency_code} /> : '—'}
                </td>
                <td className="px-3 py-2 text-ink-3 whitespace-nowrap">{formatDateTime(q.updated_at)}</td>
                <td className="px-2 py-2">
                  <div className="flex items-center justify-end gap-1">
                    <button onClick={() => open(q.id)} disabled={busyId !== null} aria-label={t('saved.open', { reference: q.reference })}
                      title={t('saved.open', { reference: q.reference })}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover text-ink-3 disabled:opacity-40">
                      <FolderOpen size={12} />
                    </button>
                    <button onClick={() => remove(q)} disabled={busyId !== null} aria-label={t('saved.delete', { reference: q.reference })}
                      title={t('saved.delete', { reference: q.reference })}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover text-ink-3 hover:text-danger disabled:opacity-40">
                      <Trash2 size={12} />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={data?.total ?? 0} onPageChange={setPage} />
    </>
  )
}
