import { useState, useEffect, useCallback } from 'react'
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { idFromPath } from '../routes'
import { ShoppingCart, Plus, ChevronUp, ChevronDown, X } from 'lucide-react'
import clsx from 'clsx'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { useSalesOrders } from '../oms/hooks/useSalesOrders'
import type { SalesOrderStatus, SalesOrderSource, SalesOrderSortField, SortDirection } from '../oms/types'
import { useConfirm } from '../components/common/useConfirm'
import { STATUS_OPTIONS, customerName } from '../oms/salesOrderDisplay'
import { PaymentStatusBadge } from '../oms/PaymentStatusBadge'
import { salesOrderStatusLabel } from '../oms/statusLabel'
import { CreateSalesOrderPage } from '../oms/CreateSalesOrderPage'
import { SalesOrderDetailPage } from '../oms/SalesOrderDetailPage'
import { formatMoney } from '../format/money'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'

const PAGE_SIZE = 20

// SOURCE_FILTER_OPTIONS: 'manual' reads as "Internal" here -- POS/manual/
// bol/woocommerce is the wire vocabulary (see SalesOrderSource), Internal is
// the label staff actually recognize for a manually-created order.
const sourceFilterOptions = (): { value: SalesOrderSource; label: string }[] => [
  { value: 'pos', label: 'POS' },
  { value: 'woocommerce', label: 'WOO' },
  { value: 'bol', label: 'Bol' },
  { value: 'manual', label: i18next.t('orders:list.sourceInternal') },
  { value: 'whatsapp', label: 'WhatsApp' },
]

const STATUS_COLORS: Record<SalesOrderStatus, string> = {
  pending: 'bg-gray-500/15 text-ink-4',
  processing: 'bg-accent/15 text-accent',
  'on-hold': 'bg-amber-500/15 text-warning',
  shipped: 'bg-indigo-500/15 text-indigo-400',
  completed: 'bg-green-500/15 text-success',
  cancelled: 'bg-gray-500/15 text-ink-4',
  refunded: 'bg-purple-500/15 text-purple-400',
  partially_refunded: 'bg-purple-500/10 text-purple-300',
  failed: 'bg-red-500/15 text-danger',
}

function SortableTh({ field, label, sortBy, sortDir, onSort, align = 'left' }: {
  field: SalesOrderSortField
  label: string
  sortBy: SalesOrderSortField
  sortDir: SortDirection
  onSort: (field: SalesOrderSortField) => void
  align?: 'left' | 'right'
}) {
  const active = field === sortBy
  const Icon = sortDir === 'asc' ? ChevronUp : ChevronDown
  return (
    <th
      className={clsx(
        'px-3 py-2.5 font-medium uppercase tracking-wider cursor-pointer select-none hover:text-ink-2 transition-colors',
        align === 'right' ? 'text-right' : 'text-left',
        active ? 'text-ink-2' : 'text-ink-4',
      )}
      onClick={() => onSort(field)}
    >
      <span className={clsx('inline-flex items-center gap-1', align === 'right' && 'flex-row-reverse')}>
        {label}
        {active && <Icon size={11} />}
      </span>
    </th>
  )
}

export default function SalesOrders() {
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['orders', 'common'])
  const { confirm, confirmDialog } = useConfirm()
  const { token } = useAuth()
  // Filters/sort/page are seeded from the URL on first render and kept in
  // sync with it (see the effect below) -- reloading the page, or sharing
  // the link, reproduces the exact same view instead of resetting it.
  const [searchParams, setSearchParams] = useSearchParams()
  const [page, setPage] = useState(() => {
    const p = Number(searchParams.get('page'))
    return p > 0 ? p : 1
  })
  const [sortBy, setSortBy] = useState<SalesOrderSortField>(() => (searchParams.get('sort_by') as SalesOrderSortField | null) ?? 'occurred_at')
  const [sortDir, setSortDir] = useState<SortDirection>(() => (searchParams.get('sort_dir') as SortDirection | null) ?? 'desc')
  const [sourceFilter, setSourceFilter] = useState<SalesOrderSource[]>(() => searchParams.getAll('source') as SalesOrderSource[])
  const [statusFilter, setStatusFilter] = useState<SalesOrderStatus[]>(() => searchParams.getAll('status') as SalesOrderStatus[])
  // fulfilledFilter: '' = don't filter (all substatuses), otherwise
  // "true"/"false" -- a substatus nested under whatever status is selected
  // above (e.g. Processing + not yet fulfilled).
  const [fulfilledFilter, setFulfilledFilter] = useState<'' | 'true' | 'false'>(() => {
    const f = searchParams.get('fulfilled')
    return f === 'true' || f === 'false' ? f : ''
  })
  // customerId: not a filter button -- arrives via a link from the Customer
  // detail page's "Orders" stat (see Customers.tsx), same URL-seeding
  // convention as the other filters so page/sort changes don't drop it.
  const [customerId, setCustomerId] = useState<string | null>(() => searchParams.get('customer_id'))

  // The open order lives in the URL (/sales-orders/<uuid>, or
  // /sales-orders/new) -- read via location.pathname (not useParams) so it
  // reflects the URL on every render even when navigating here doesn't
  // remount the page. This used to be `useState(() => searchParams.get('id'))`,
  // a lazy initializer that only ran on mount: global search's
  // CommandPalette navigating straight to this order while already on this
  // page changed the URL but never re-ran the initializer, so the click
  // silently did nothing. See routes.ts's idFromPath doc comment.
  const location = useLocation()
  const navigate = useNavigate()
  const selectedId = idFromPath(location.pathname, '/sales-orders')

  const buildListParams = useCallback(() => {
    const params = new URLSearchParams()
    if (page !== 1) params.set('page', String(page))
    if (sortBy !== 'occurred_at') params.set('sort_by', sortBy)
    if (sortDir !== 'desc') params.set('sort_dir', sortDir)
    sourceFilter.forEach(s => params.append('source', s))
    statusFilter.forEach(s => params.append('status', s))
    if (fulfilledFilter) params.set('fulfilled', fulfilledFilter)
    if (customerId) params.set('customer_id', customerId)
    return params
  }, [page, sortBy, sortDir, sourceFilter, statusFilter, fulfilledFilter, customerId])

  useEffect(() => {
    // Don't touch the query string while viewing a record -- it now lives
    // on its own path (/sales-orders/<id>), not here.
    if (selectedId !== null) return
    setSearchParams(buildListParams(), { replace: true })
  }, [selectedId, buildListParams, setSearchParams])

  // Returns to the list, restoring whatever page/sort/filters were active
  // before opening the record -- captured here from current state rather
  // than relying on it having survived the route change.
  function closeDetail() {
    const qs = buildListParams().toString()
    navigate(`/sales-orders${qs ? `?${qs}` : ''}`)
  }

  const hasActiveFilters = sourceFilter.length > 0 || statusFilter.length > 0 || fulfilledFilter !== '' || customerId !== null

  function clearFilters() {
    setSourceFilter([])
    setStatusFilter([])
    setFulfilledFilter('')
    setCustomerId(null)
    setPage(1)
  }

  const { salesOrders, total, loading, error, create, refresh, bulkUpdate } = useSalesOrders(token ?? '', {
    page, perPage: PAGE_SIZE, sortBy, sortDir, source: sourceFilter, status: statusFilter,
    fulfilled: fulfilledFilter === '' ? undefined : fulfilledFilter === 'true',
    customerId: customerId ?? undefined,
  })
  const [checkedIds, setCheckedIds] = useState<Set<string>>(new Set())
  const [bulkStatus, setBulkStatus] = useState('')
  const [bulkFulfilled, setBulkFulfilled] = useState('')
  const [bulkApplying, setBulkApplying] = useState(false)
  const [bulkError, setBulkError] = useState('')

  const allOnPageChecked = salesOrders.length > 0 && salesOrders.every(o => checkedIds.has(o.id))

  function toggleAllOnPage() {
    setCheckedIds(prev => {
      const next = new Set(prev)
      if (allOnPageChecked) {
        salesOrders.forEach(o => next.delete(o.id))
      } else {
        salesOrders.forEach(o => next.add(o.id))
      }
      return next
    })
  }

  function toggleOne(id: string) {
    setCheckedIds(prev => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  async function applyBulkChange() {
    if (!bulkStatus && !bulkFulfilled) return
    const count = checkedIds.size
    if (!(await confirm({ title: t('list.bulkConfirm', { count }), confirmLabel: t('list.bulkApply') }))) return
    setBulkApplying(true)
    setBulkError('')
    try {
      const result = await bulkUpdate({
        orderIds: Array.from(checkedIds),
        status: bulkStatus ? (bulkStatus as SalesOrderStatus) : undefined,
        fulfilled: bulkFulfilled ? bulkFulfilled === 'true' : undefined,
      })
      if (result.failed.length > 0) {
        // Leave the bar (and the still-checked ids) up so the error stays
        // visible -- clearing here would hide which orders failed.
        setBulkError(t('list.bulkPartial', { updated: result.updated.length, failed: result.failed.length, errors: result.failed.map(f => f.error).join('; ') }))
      } else {
        setCheckedIds(new Set())
        setBulkStatus('')
        setBulkFulfilled('')
      }
    } catch (err) {
      setBulkError(apiErrorText(err) ?? t('list.bulkFailed'))
    } finally {
      setBulkApplying(false)
    }
  }

  function toggleSourceFilter(source: SalesOrderSource) {
    setSourceFilter(prev => prev.includes(source) ? prev.filter(s => s !== source) : [...prev, source])
    setPage(1)
  }

  function toggleStatusFilter(status: SalesOrderStatus) {
    setStatusFilter(prev => prev.includes(status) ? prev.filter(s => s !== status) : [...prev, status])
    setPage(1)
  }

  function changeFulfilledFilter(value: '' | 'true' | 'false') {
    setFulfilledFilter(value)
    setPage(1)
  }

  function toggleSort(field: SalesOrderSortField) {
    if (field === sortBy) {
      setSortDir(d => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortBy(field)
      setSortDir('desc')
    }
    setPage(1)
  }

  if (selectedId === 'new') {
    return (
      <CreateSalesOrderPage
        onCreate={create}
        onClose={closeDetail}
      />
    )
  }
  if (selectedId !== null) {
    return <SalesOrderDetailPage salesOrderId={selectedId} onClose={() => { closeDetail(); refresh() }} />
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toolbar */}
      <div className="flex items-center justify-end gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <button
          onClick={() => navigate('/sales-orders/new')}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-medium"
        >
          <Plus size={13} /> {t('list.newOrder')}
        </button>
      </div>

      {/* Stats strip */}
      <div className="flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <span className="text-xs text-ink-4 whitespace-nowrap">{t('list.count', { count: total })}</span>
        <div className="flex items-center gap-1.5" role="group" aria-label={t('list.filterChannel')}>
          {sourceFilterOptions().map(({ value, label }) => (
            <button
              key={value}
              onClick={() => toggleSourceFilter(value)}
              aria-pressed={sourceFilter.includes(value)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-xs font-medium transition-colors',
                sourceFilter.includes(value)
                  ? 'bg-accent-solid text-white'
                  : 'bg-raised text-ink-3 hover:bg-hover',
              )}
            >
              {label}
            </button>
          ))}
        </div>
        {error && <span className="text-xs text-danger ml-auto">{error}</span>}
      </div>

      {/* Status + substatus filter strip */}
      <div className="flex items-center flex-wrap gap-3 px-5 py-2.5 border-b border-line bg-surface shrink-0">
        <div className="flex items-center flex-wrap gap-1.5" role="group" aria-label={t('list.filterStatus')}>
          {STATUS_OPTIONS.map(s => (
            <button
              key={s}
              onClick={() => toggleStatusFilter(s)}
              aria-pressed={statusFilter.includes(s)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-xs font-medium capitalize transition-colors',
                statusFilter.includes(s)
                  ? 'bg-accent-solid text-white'
                  : 'dark:bg-night-800 bg-gray-50 text-ink-3 hover:bg-hover',
              )}
            >
              {salesOrderStatusLabel(s)}
            </button>
          ))}
        </div>
        <select
          aria-label={t('list.filterFulfilled')} value={fulfilledFilter}
          onChange={e => changeFulfilledFilter(e.target.value as '' | 'true' | 'false')}
          className="px-2 py-1.5 text-xs rounded-lg dark:bg-night-800 bg-gray-50 border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
        >
          <option value="">{t('list.anyFulfillment')}</option>
          <option value="true">{t('list.onlyFulfilled')}</option>
          <option value="false">{t('list.onlyUnfulfilled')}</option>
        </select>
        {customerId && (
          <span className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-accent/15 text-accent">
            {t('list.oneCustomer')}
            <button type="button" onClick={() => setCustomerId(null)} aria-label={t('list.clearCustomer')} className="hover:opacity-70">
              <X size={12} />
            </button>
          </span>
        )}
        {hasActiveFilters && (
          <button
            type="button" onClick={clearFilters}
            className="flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium text-ink-3 hover:text-ink transition-colors ml-auto"
          >
            <X size={12} /> {t('list.clearFilters')}
          </button>
        )}
      </div>

      {/* Bulk action bar */}
      {checkedIds.size > 0 && (
        <div className="flex items-center gap-3 px-5 py-2.5 border-b border-line dark:bg-night-800 bg-accent/10 shrink-0">
          <span className="text-xs font-medium text-ink-2">{t('list.selected', { count: checkedIds.size })}</span>
          <select
            aria-label={t('list.bulkStatus')} value={bulkStatus} onChange={e => setBulkStatus(e.target.value)} disabled={bulkApplying}
            className="px-2 py-1 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60 disabled:opacity-50"
          >
            <option value="">{t('list.statusPlaceholder')}</option>
            {STATUS_OPTIONS.map(s => <option key={s} value={s}>{salesOrderStatusLabel(s)}</option>)}
          </select>
          <select
            aria-label={t('list.bulkFulfilled')} value={bulkFulfilled} onChange={e => setBulkFulfilled(e.target.value)} disabled={bulkApplying}
            className="px-2 py-1 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60 disabled:opacity-50"
          >
            <option value="">{t('list.fulfilledPlaceholder')}</option>
            <option value="true">{t('list.fulfilled')}</option>
            <option value="false">{t('list.unfulfilled')}</option>
          </select>
          <button
            onClick={applyBulkChange}
            disabled={bulkApplying || (!bulkStatus && !bulkFulfilled)}
            className="px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
          >
            {bulkApplying ? t('list.applying') : t('common:actions.apply')}
          </button>
          <button
            onClick={() => { setCheckedIds(new Set()); setBulkStatus(''); setBulkFulfilled(''); setBulkError('') }}
            disabled={bulkApplying}
            className="text-xs font-medium text-ink-3 hover:text-ink-2 disabled:opacity-50"
          >
            {t('common:actions.clear')}
          </button>
          {bulkError && <span className="text-xs text-danger ml-auto">{bulkError}</span>}
        </div>
      )}

      {/* Table */}
      <div className="flex-1 overflow-auto">
        <table className="w-full text-xs">
          <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
            <tr>
              <th className="px-3 py-2.5 w-8">
                <input
                  type="checkbox" aria-label={t('list.selectAll')}
                  checked={allOnPageChecked} onChange={toggleAllOnPage}
                  className="rounded border dark:border-night-700 border-gray-300"
                />
              </th>
              <th className="px-2 py-2.5 w-8" />
              <SortableTh field="occurred_at" label={t('list.colOrder')} sortBy={sortBy} sortDir={sortDir} onSort={toggleSort} />
              <SortableTh field="customer" label={t('common:fields.customer')} sortBy={sortBy} sortDir={sortDir} onSort={toggleSort} />
              <SortableTh field="source" label={t('list.colSource')} sortBy={sortBy} sortDir={sortDir} onSort={toggleSort} />
              <SortableTh field="status" label={t('common:fields.status')} sortBy={sortBy} sortDir={sortDir} onSort={toggleSort} />
              <SortableTh field="total" label={t('common:fields.total')} sortBy={sortBy} sortDir={sortDir} onSort={toggleSort} align="right" />
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {loading && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">{t('common:status.loading')}</td></tr>
            )}
            {!loading && salesOrders.length === 0 && (
              <tr><td colSpan={7} className="px-4 py-6 text-center text-ink-4">{t('list.empty')}</td></tr>
            )}
            {salesOrders.map(o => (
              <tr
                key={o.id}
                onClick={() => navigate(`/sales-orders/${o.id}`)}
                className="group hover:bg-hover/60 transition-colors cursor-pointer"
              >
                <td className="px-3 py-2.5" onClick={e => e.stopPropagation()}>
                  <input
                    type="checkbox" aria-label={t('list.selectOrder', { number: o.number || o.external_order_id })}
                    checked={checkedIds.has(o.id)} onChange={() => toggleOne(o.id)}
                    className="rounded border dark:border-night-700 border-gray-300"
                  />
                </td>
                <td className="px-2 py-2.5">
                  <div className="w-6 h-6 rounded dark:bg-night-700 bg-gray-200 flex items-center justify-center">
                    <ShoppingCart size={12} className="text-ink-4" />
                  </div>
                </td>
                <td className="px-3 py-2.5 max-w-[200px]">
                  <p className="font-medium text-ink truncate">#{o.number || o.external_order_id}</p>
                  <p className="text-2xs text-ink-4 truncate">{formatDateTime(o.occurred_at)}</p>
                </td>
                <td className="px-3 py-2.5 text-ink-3 max-w-[220px] truncate">{customerName(o)}</td>
                <td className="px-3 py-2.5 text-ink-3">{o.source}</td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <span className={clsx('px-2 py-0.5 rounded-full text-2xs font-medium', STATUS_COLORS[o.status])}>
                      {salesOrderStatusLabel(o.status)}
                    </span>
                    <PaymentStatusBadge status={o.payment_status} />
                    <span className={clsx(
                      'px-2 py-0.5 rounded-full text-2xs font-medium',
                      o.fulfilled ? 'bg-green-500/15 text-success' : 'bg-gray-500/15 text-ink-4',
                    )}>
                      {o.fulfilled ? t('list.fulfilled') : t('list.unfulfilled')}
                    </span>
                    {(o.needs_stock_review || o.needs_sku_review) && (
                      <span className="px-2 py-0.5 rounded-full text-2xs font-medium bg-amber-500/15 text-warning">
                        {t('list.needsReview')}
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-3 py-2.5 text-right font-medium text-ink">
                  {formatMoney(o.total_minor_units, o.currency_code)}
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
