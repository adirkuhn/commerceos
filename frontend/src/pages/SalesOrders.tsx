import { useState, useEffect, useCallback } from 'react'
import { useLocation, useNavigate, useSearchParams } from 'react-router-dom'
import { idFromPath, subPathSegments } from '../routes'
import { Plus, ChevronUp, ChevronDown, X } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import Pagination from '../components/common/Pagination'
import { LIST_TOOLBAR, LIST_TOOLBAR_ACTION, LIST_TOOLBAR_INLINE, LIST_TOOLBAR_ROW } from '../components/common/listToolbar'
import { useSalesOrders } from '../oms/hooks/useSalesOrders'
import type { SalesOrderStatus, SalesOrderSource, SalesOrderSortField, SortDirection } from '../oms/types'
import { useConfirm } from '../components/common/useConfirm'
import { FilterChips } from '../components/common/FilterChips'
import { STATUS_OPTIONS, customerName, salesOrderSourceLabel } from '../oms/salesOrderDisplay'
import { PaymentStatusBadge } from '../oms/PaymentStatusBadge'
import { salesOrderStatusLabel } from '../oms/statusLabel'
import { CreateSalesOrderPage } from '../oms/CreateSalesOrderPage'
import { EditSalesOrderPage } from '../oms/EditSalesOrderPage'
import { SalesOrderDetailPage } from '../oms/SalesOrderDetailPage'
import { MiniLegs } from '../oms/FulfilmentLegs'
import { Money } from '../format/amountVisibility'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'

const PAGE_SIZE = 20

// SOURCE_FILTER_OPTIONS: 'manual' reads as "Internal" here -- POS/manual/
// The wire vocabulary (see SalesOrderSource); labels come from
// salesOrderSourceLabel -- "Manual" for an order created by staff,
// "Storefront" for one placed through the built-in storefront ("internal").
const SOURCE_FILTER_VALUES: SalesOrderSource[] = ['pos', 'woocommerce', 'bol', 'manual', 'internal', 'whatsapp']
const sourceFilterOptions = (): { value: SalesOrderSource; label: string }[] =>
  SOURCE_FILTER_VALUES.map(value => ({ value, label: salesOrderSourceLabel(value) }))

const STATUS_COLORS: Record<SalesOrderStatus, string> = {
  pending: 'bg-fill text-ink-3',
  processing: 'bg-accent/10 text-accent',
  'on-hold': 'bg-warning/10 text-warning',
  shipped: 'bg-accent/10 text-accent',
  completed: 'bg-success/10 text-success',
  cancelled: 'bg-fill text-ink-4',
  refunded: 'bg-figure/10 text-figure',
  partially_refunded: 'bg-figure/10 text-figure',
  failed: 'bg-danger/10 text-danger',
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
        'px-3 py-2.5 text-2xs font-semibold caps-label-widest cursor-pointer select-none hover:text-ink-2 transition-colors',
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
  // shipDue: also not a filter button -- arrives via the Today page's
  // "orders to ship today" link (?ship_due=true).
  const [shipDue, setShipDue] = useState(() => searchParams.get('ship_due') === 'true')

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
    if (shipDue) params.set('ship_due', 'true')
    return params
  }, [page, sortBy, sortDir, sourceFilter, statusFilter, fulfilledFilter, customerId, shipDue])

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

  const hasActiveFilters = sourceFilter.length > 0 || statusFilter.length > 0 || fulfilledFilter !== '' || customerId !== null || shipDue

  function clearFilters() {
    setSourceFilter([])
    setStatusFilter([])
    setFulfilledFilter('')
    setCustomerId(null)
    setShipDue(false)
    setPage(1)
  }

  const { salesOrders, total, loading, error, create, refresh, bulkUpdate } = useSalesOrders(token ?? '', {
    page, perPage: PAGE_SIZE, sortBy, sortDir, source: sourceFilter, status: statusFilter,
    fulfilled: fulfilledFilter === '' ? undefined : fulfilledFilter === 'true',
    customerId: customerId ?? undefined,
    shipDue: shipDue || undefined,
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

  const [recordId, view] = subPathSegments(location.pathname, '/sales-orders')
  if (recordId && view === 'edit') {
    return <EditSalesOrderPage salesOrderId={recordId} onClose={() => navigate(`/sales-orders/${encodeURIComponent(recordId)}`)} />
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
      {/* Toolbar -- rows that flow into one band (common/listToolbar) */}
      <div className={LIST_TOOLBAR}>
        <div className={clsx('flex items-center justify-end gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0', LIST_TOOLBAR_ROW)}>
          <button
            onClick={() => navigate('/sales-orders/new')}
            className={clsx('flex items-center gap-1.5 h-8 px-3 rounded-md bg-accent-solid text-white text-xs hover:bg-accent-solid-hover transition-colors font-semibold whitespace-nowrap', LIST_TOOLBAR_ACTION)}
          >
            <Plus size={13} /> {t('list.newOrder')}
          </button>
        </div>

        {/* Stats strip */}
        <div className={clsx('flex items-center gap-6 px-5 py-2.5 border-b border-line bg-surface shrink-0', LIST_TOOLBAR_ROW)}>
          <span className="self-end pb-1.5 text-xs font-semibold text-ink-3 tabular-nums whitespace-nowrap">{t('list.count', { count: total })}</span>
          <FilterChips label={t('list.channel')} options={sourceFilterOptions()} selected={sourceFilter} onToggle={toggleSourceFilter} />
          {error && <span className={clsx('text-xs text-danger ml-auto', LIST_TOOLBAR_INLINE)}>{error}</span>}
        </div>

        {/* Status + substatus filter strip */}
        <div className={clsx('flex items-center flex-wrap gap-3 px-5 py-2.5 border-b border-line bg-surface shrink-0', LIST_TOOLBAR_ROW)}>
          <FilterChips
            label={t('list.status')}
            options={STATUS_OPTIONS.map(value => ({ value, label: salesOrderStatusLabel(value) }))}
            selected={statusFilter}
            onToggle={toggleStatusFilter}
          />
          <label className="flex flex-col gap-1">
            <span aria-hidden className="caps-label-widest text-2xs font-semibold text-ink-4">{t('list.fulfilment')}</span>
            <select
              aria-label={t('list.filterFulfilled')} value={fulfilledFilter}
              onChange={e => changeFulfilledFilter(e.target.value as '' | 'true' | 'false')}
              className="h-7 px-2 text-xs font-semibold rounded-md bg-surface border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
            >
              <option value="">{t('list.anyFulfillment')}</option>
              <option value="true">{t('list.onlyFulfilled')}</option>
              <option value="false">{t('list.onlyUnfulfilled')}</option>
            </select>
          </label>
          {customerId && (
            <span className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-accent/15 text-accent">
              {t('list.oneCustomer')}
              <button type="button" onClick={() => setCustomerId(null)} aria-label={t('list.clearCustomer')} className="hover:opacity-70">
                <X size={12} />
              </button>
            </span>
          )}
          {shipDue && (
            <span className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg text-xs font-medium bg-accent/15 text-accent">
              {t('list.shipDue')}
              <button type="button" onClick={() => { setShipDue(false); setPage(1) }} aria-label={t('list.clearShipDue')} className="hover:opacity-70">
                <X size={12} />
              </button>
            </span>
          )}
          {hasActiveFilters && (
            <button
              type="button" onClick={clearFilters}
              className={clsx('flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs font-medium text-ink-3 hover:text-ink transition-colors ml-auto', LIST_TOOLBAR_INLINE)}
            >
              <X size={12} /> {t('list.clearFilters')}
            </button>
          )}
        </div>
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
                  className="rounded-sm border-line accent-[rgb(var(--accent-solid))]"
                />
              </th>
              <th className="px-2 py-2.5 w-16"><span className="sr-only">{t('legs.label')}</span></th>
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
                    className="rounded-sm border-line accent-[rgb(var(--accent-solid))]"
                  />
                </td>
                <td className="px-2 py-2.5">
                  <MiniLegs order={o} />
                </td>
                <td className="px-3 py-2.5 max-w-[200px]">
                  <p className="font-semibold text-ink truncate tabular-nums">#{o.number || o.external_order_id}</p>
                  <p className="text-2xs text-ink-4 truncate">{formatDateTime(o.occurred_at)}</p>
                </td>
                <td className="px-3 py-2.5 text-ink-3 max-w-[220px] truncate">{customerName(o)}</td>
                <td className="px-3 py-2.5 text-ink-3">{salesOrderSourceLabel(o.source)}</td>
                <td className="px-3 py-2.5">
                  <div className="flex items-center gap-1.5 flex-wrap">
                    <span className={clsx('px-1.5 py-0.5 rounded-sm text-2xs font-semibold', STATUS_COLORS[o.status])}>
                      {salesOrderStatusLabel(o.status)}
                    </span>
                    <PaymentStatusBadge status={o.payment_status} />
                    <span className={clsx(
                      'px-1.5 py-0.5 rounded-sm text-2xs font-semibold',
                      o.fulfilled ? 'bg-success/10 text-success' : 'bg-fill text-ink-4',
                    )}>
                      {o.fulfilled ? t('list.fulfilled') : t('list.unfulfilled')}
                    </span>
                    {(o.needs_stock_review || o.needs_sku_review) && (
                      <span className="px-1.5 py-0.5 rounded-sm text-2xs font-semibold bg-warning/10 text-warning">
                        {t('list.needsReview')}
                      </span>
                    )}
                  </div>
                </td>
                <td className="px-3 py-2.5 text-right font-semibold text-figure tabular-nums">
                  <Money minor={o.total_minor_units} currency={o.currency_code} />
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
