import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { Search, Star, Trash2, BadgeCheck } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useReviews } from '../reviews/hooks/useReviews'
import type { Review } from '../reviews/types'
import { useQueryPage, useQueryString } from '../routing/useQueryState'
import { useConfirm } from '../components/common/useConfirm'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'

const PAGE_SIZE = 20

function Stars({ rating }: { rating: number }) {
  const { t } = useTranslation('reviews')
  return (
    <span className="flex items-center gap-0.5" aria-label={t('list.stars', { rating })}>
      {[1, 2, 3, 4, 5].map(n => (
        <Star key={n} size={12} className={n <= rating ? 'fill-amber-400 text-warning' : 'text-ink-4'} />
      ))}
    </span>
  )
}

export default function Reviews() {
  const { formatDate } = useDateFormat()
  const { t } = useTranslation(['reviews', 'common'])
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  const { reviews, loading, error, remove } = useReviews(token ?? '')

  // Search and page live in the URL (?q=, ?page=) -- refresh and Back keep them.
  const [searchParam, setSearch] = useQueryString('q', { resets: ['page'] })
  const search = searchParam ?? ''
  const [page, setPage] = useQueryPage()
  const [actionError, setActionError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
  }

  const q = search.toLowerCase()
  const filtered = reviews.filter(r =>
    r.customer_name.toLowerCase().includes(q) || r.body.toLowerCase().includes(q) || r.product_id.toLowerCase().includes(q),
  )
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(r: Review) {
    if (!(await confirm({ title: t('list.deleteTitle', { name: r.customer_name }), message: t('list.deleteMessage'), confirmLabel: t('list.deleteConfirm'), destructive: true }))) return
    setActionError('')
    try {
      await remove(r.id)
    } catch (err) {
      setActionError(apiErrorText(err) ?? t('common:status.error'))
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            value={search}
            onChange={e => handleSearchChange(e.target.value)}
            placeholder={t('list.search')}
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border-line border text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4">{t('list.count', { count: filtered.length })}</span>
        {(error || actionError) && <span className="text-xs text-danger ml-auto">{error || actionError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('list.colCustomer')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('list.colRating')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('list.colReview')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('list.colProduct')}</th>
              <th className="px-3 py-2.5 text-left font-medium text-ink-4 uppercase tracking-wider">{t('list.colDate')}</th>
              <th className="px-3 py-2.5 w-10" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center text-ink-4">{t('list.empty')}</td></tr>
            )}
            {paged.map(r => (
              <tr key={r.id} className="group hover:bg-hover/60 transition-colors">
                <td className="px-3 py-2.5 max-w-[160px]">
                  <div className="flex items-center gap-1.5">
                    <p className="font-medium text-ink truncate">{r.customer_name}</p>
                    {r.verified && (
                      <span title={t('list.verified')}>
                        <BadgeCheck size={13} className="text-success shrink-0" />
                      </span>
                    )}
                  </div>
                  <p className="text-2xs text-ink-4 truncate">{r.customer_email}</p>
                </td>
                <td className="px-3 py-2.5"><Stars rating={r.rating} /></td>
                <td className="px-3 py-2.5 max-w-[360px]">
                  {r.title && <p className="font-medium text-ink truncate">{r.title}</p>}
                  <p className={clsx('text-ink-3', r.title ? 'truncate' : 'line-clamp-2')}>{r.body}</p>
                </td>
                <td className="px-3 py-2.5 font-mono text-2xs text-ink-4">{r.product_id}</td>
                <td className="px-3 py-2.5 text-ink-3">{formatDate(r.created_at)}</td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 focus-within:opacity-100 transition-opacity">
                    <button aria-label={t('common:actions.delete')}
                      onClick={() => handleDelete(r)}
                      className="w-6 h-6 flex items-center justify-center rounded hover:bg-hover transition-colors"
                      title={t('common:actions.delete')}
                    >
                      <Trash2 size={12} className="text-ink-3 hover:text-danger" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={filtered.length} onPageChange={setPage} />
      {confirmDialog}
    </div>
  )
}
