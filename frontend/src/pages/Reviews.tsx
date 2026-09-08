import { useState } from 'react'
import { Search, Star, Trash2, BadgeCheck } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useReviews } from '../reviews/hooks/useReviews'
import type { Review } from '../reviews/types'

const PAGE_SIZE = 20

function formatDate(value: string): string {
  return new Date(value).toLocaleDateString()
}

function Stars({ rating }: { rating: number }) {
  return (
    <span className="flex items-center gap-0.5" aria-label={`${rating} out of 5 stars`}>
      {[1, 2, 3, 4, 5].map(n => (
        <Star key={n} size={12} className={n <= rating ? 'fill-amber-400 text-amber-400' : 'dark:text-gray-600 text-gray-300'} />
      ))}
    </span>
  )
}

export default function Reviews() {
  const { token } = useAuth()
  const { reviews, loading, error, remove } = useReviews(token ?? '')

  const [search, setSearch] = useState('')
  const [page, setPage] = useState(1)
  const [actionError, setActionError] = useState('')

  function handleSearchChange(value: string) {
    setSearch(value)
    setPage(1)
  }

  const q = search.toLowerCase()
  const filtered = reviews.filter(r =>
    r.customer_name.toLowerCase().includes(q) || r.body.toLowerCase().includes(q) || r.product_id.toLowerCase().includes(q),
  )
  const paged = filtered.slice((page - 1) * PAGE_SIZE, page * PAGE_SIZE)

  async function handleDelete(r: Review) {
    if (!window.confirm(`Delete this review from "${r.customer_name}"? This cannot be undone.`)) return
    setActionError('')
    try {
      await remove(r.id)
    } catch (err) {
      setActionError(err instanceof Error ? err.message : 'Unknown error')
    }
  }

  return (
    <div className="flex flex-col h-full">
      <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 shrink-0">
        <div className="relative flex-1 max-w-sm">
          <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
          <input
            value={search}
            onChange={e => handleSearchChange(e.target.value)}
            placeholder="Search by customer, product, or text..."
            className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white dark:border-[#2D3748] border-gray-200 border dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
          />
        </div>
      </div>

      <div className="flex items-center gap-6 px-5 py-2.5 border-b dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-white shrink-0">
        <span className="text-xs dark:text-gray-500 text-gray-400">{filtered.length} review{filtered.length === 1 ? '' : 's'}</span>
        {(error || actionError) && <span className="text-xs text-red-400 ml-auto">{error || actionError}</span>}
      </div>

      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
            <tr>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Customer</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Rating</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Review</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Product ID</th>
              <th className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider">Date</th>
              <th className="px-3 py-2.5 w-10" />
            </tr>
          </thead>
          <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
            {loading && (
              <tr><td colSpan={6} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">Loading…</td></tr>
            )}
            {!loading && filtered.length === 0 && (
              <tr><td colSpan={6} className="px-4 py-6 text-center dark:text-gray-500 text-gray-400">No reviews yet</td></tr>
            )}
            {paged.map(r => (
              <tr key={r.id} className="group hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors">
                <td className="px-3 py-2.5 max-w-[160px]">
                  <div className="flex items-center gap-1.5">
                    <p className="font-medium dark:text-gray-100 text-gray-900 truncate">{r.customer_name}</p>
                    {r.verified && (
                      <span title="Verified purchase">
                        <BadgeCheck size={13} className="text-green-500 shrink-0" />
                      </span>
                    )}
                  </div>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 truncate">{r.customer_email}</p>
                </td>
                <td className="px-3 py-2.5"><Stars rating={r.rating} /></td>
                <td className="px-3 py-2.5 max-w-[360px]">
                  {r.title && <p className="font-medium dark:text-gray-200 text-gray-800 truncate">{r.title}</p>}
                  <p className={clsx('dark:text-gray-400 text-gray-500', r.title ? 'truncate' : 'line-clamp-2')}>{r.body}</p>
                </td>
                <td className="px-3 py-2.5 font-mono text-[10px] dark:text-gray-500 text-gray-400">{r.product_id}</td>
                <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{formatDate(r.created_at)}</td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      onClick={() => handleDelete(r)}
                      className="w-6 h-6 flex items-center justify-center rounded dark:hover:bg-[#2D3748] hover:bg-gray-200 transition-colors"
                      title="Delete"
                    >
                      <Trash2 size={12} className="dark:text-gray-400 text-gray-500 hover:text-red-400" />
                    </button>
                  </div>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <Pagination page={page} pageSize={PAGE_SIZE} totalItems={filtered.length} onPageChange={setPage} />
    </div>
  )
}
