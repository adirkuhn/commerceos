import { useState } from 'react'
import { ArrowLeft, Search } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../../auth/AuthContext'
import { useQueryPage, useQueryString } from '../../routing/useQueryState'
import { useSalesChannelProducts } from '../hooks/useSalesChannelProducts'
import type { SalesChannel, EnrollmentMode } from '../types'
import type { UpdateSalesChannelPayload } from '../api'
import Pagination from '../../components/common/Pagination'
import { ENROLLMENT_LABELS, INTERNAL_TYPE } from './salesChannelConstants'
import { PricingSection } from './PricingSection'

const PER_PAGE = 20

interface ChannelProductsPanelProps {
  channel: SalesChannel
  onClose: () => void
  update: (id: string, payload: UpdateSalesChannelPayload) => Promise<SalesChannel>
  categories: { id: string; name: string }[]
}

// The "add all" / "add by tag" / "add by category" bulk actions are just
// this channel's default enrollment rule changing -- evaluated live against
// the whole catalog, no per-product rows written. See SalesChannel's
// EnrollmentMode doc comment in the backend.
// `channel` is always the current row from useSalesChannels -- its update()
// replaces that row, so a save here re-renders with fresh data without
// this panel holding its own copy.
export function ChannelProductsPanel({ channel, onClose, update, categories }: ChannelProductsPanelProps) {
  const { token } = useAuth()
  const [page, setPage] = useQueryPage()
  const [searchParam, setSearch] = useQueryString('q', { resets: ['page'] })
  const search = searchParam ?? ''
  const { items, total, loading, error, setOverride } = useSalesChannelProducts(token ?? '', channel.id, { page, perPage: PER_PAGE, search })

  const [bulkCategoryId, setBulkCategoryId] = useState('')
  const [bulkTag, setBulkTag] = useState('')
  const [bulkSaving, setBulkSaving] = useState(false)
  const [bulkError, setBulkError] = useState('')

  const applyRule = async (mode: EnrollmentMode, categoryId: string | null, tag: string | null) => {
    setBulkError('')
    setBulkSaving(true)
    try {
      await update(channel.id, {
        name: channel.name,
        connectionId: channel.connection_id,
        enrollmentMode: mode,
        enrollmentCategoryId: categoryId,
        enrollmentTag: tag,
        priceAdjustmentBasisPoints: channel.price_adjustment_basis_points ?? null,
        quantityBreaks: channel.quantity_breaks ?? [],
        active: channel.active,
      })
    } catch (err) {
      setBulkError(err instanceof Error ? err.message : 'Failed to update enrollment rule')
    } finally {
      setBulkSaving(false)
    }
  }

  return (
    <div className="flex flex-col h-full overflow-hidden">
      <div className="flex items-center justify-between px-6 py-4 border-b border-line bg-sunken shrink-0">
        <div>
          <button
            onClick={onClose}
            className="flex items-center gap-1 text-2xs font-medium text-ink-4 hover:text-ink-2 transition-colors mb-1"
          >
            <ArrowLeft size={11} /> Sales channels
          </button>
          <h1 className="text-xl font-semibold text-ink leading-tight">{channel.name}</h1>
        </div>
        <span className="text-2xs px-2.5 py-1 rounded-full bg-raised border border-line text-ink-3">
          {ENROLLMENT_LABELS[channel.enrollment_mode]}
        </span>
      </div>

      <div className="p-5 border-b border-line flex flex-wrap items-center gap-2 shrink-0">
        <button
          onClick={() => applyRule('all', null, null)}
          disabled={bulkSaving}
          className="px-3 py-1.5 rounded-lg bg-raised border border-line text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
        >
          Add all
        </button>
        <div className="flex items-center gap-1.5">
          <select
            aria-label="Bulk add by category"
            value={bulkCategoryId}
            onChange={e => setBulkCategoryId(e.target.value)}
            className="px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
          >
            <option value="">By category…</option>
            {categories.map(cat => <option key={cat.id} value={cat.id}>{cat.name}</option>)}
          </select>
          <button
            onClick={() => bulkCategoryId && applyRule('category', bulkCategoryId, null)}
            disabled={bulkSaving || !bulkCategoryId}
            className="px-3 py-1.5 rounded-lg bg-raised border border-line text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
          >
            Apply
          </button>
        </div>
        <div className="flex items-center gap-1.5">
          <input
            type="text"
            aria-label="Bulk add by tag"
            value={bulkTag}
            onChange={e => setBulkTag(e.target.value)}
            placeholder="By tag…"
            className="w-28 px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
          />
          <button
            onClick={() => bulkTag && applyRule('tag', null, bulkTag)}
            disabled={bulkSaving || !bulkTag}
            className="px-3 py-1.5 rounded-lg bg-raised border border-line text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
          >
            Apply
          </button>
        </div>
        <button
          onClick={() => applyRule('manual', null, null)}
          disabled={bulkSaving}
          className="px-3 py-1.5 rounded-lg bg-raised border border-line text-xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50 transition-colors"
        >
          Manual only
        </button>
      </div>
      {bulkError && <p className="px-5 pt-2 text-xs text-danger shrink-0">{bulkError}</p>}

      {channel.type === INTERNAL_TYPE && (
        <PricingSection channel={channel} update={update} />
      )}

      <div className="px-5 pt-4 shrink-0">
        <div className="relative max-w-xs">
          <Search size={13} className="absolute left-2.5 top-1/2 -translate-y-1/2 text-ink-4" />
          <input
            type="text"
            value={search}
            onChange={e => setSearch(e.target.value)}
            placeholder="Search products…"
            className="w-full pl-8 pr-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
          />
        </div>
      </div>

      {error && <p className="px-5 pt-3 text-xs text-danger shrink-0">{error}</p>}

      <div className="flex-1 overflow-y-auto px-5 py-4">
        {!loading && items.length === 0 ? (
          <p className="text-xs text-ink-4 text-center py-8">No products found.</p>
        ) : (
          <div className="bg-raised rounded-xl border border-line divide-y divide-line">
            {items.map(({ product, enrolled }) => (
              <div key={product.id} className="flex items-center justify-between px-4 py-2.5">
                <div className="min-w-0">
                  <p className="text-xs font-medium text-ink truncate">{product.name}</p>
                  {(product.tags ?? []).length > 0 && (
                    <p className="text-2xs text-ink-4 truncate">{(product.tags ?? []).join(', ')}</p>
                  )}
                </div>
                <button
                  onClick={() => setOverride(product.id, !enrolled)}
                  className={clsx(
                    'px-3 py-1 rounded-lg text-2xs font-medium transition-colors shrink-0',
                    enrolled
                      ? 'bg-green-500/10 text-success hover:bg-red-500/10 hover:text-danger'
                      : 'bg-fill text-ink-3 hover:text-accent',
                  )}
                >
                  {enrolled ? 'Enrolled — remove' : 'Add'}
                </button>
              </div>
            ))}
          </div>
        )}
      </div>

      <Pagination page={page} pageSize={PER_PAGE} totalItems={total} onPageChange={setPage} />
    </div>
  )
}
