import { Search, Filter, Plus, RefreshCw, ExternalLink, ChevronDown } from 'lucide-react'
import clsx from 'clsx'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { Money, useMoneyFormat } from '../format/amountVisibility'
import { useLocation } from 'react-router-dom'
import { idFromPath } from '../routes'
import { useStackNavigation } from '../routing/useStackNavigation'
import { useQueryEnum, useQueryString } from '../routing/useQueryState'

interface Order {
  id: string
  channel: 'shopify' | 'woocommerce' | 'bol' | 'lightspeed' | 'manual'
  customer: string
  address: string
  status: 'new' | 'reserved' | 'picking' | 'packed' | 'shipped' | 'delivered' | 'cancelled'
  total: number
  paymentStatus: 'paid' | 'pending' | 'failed'
  fulfillmentStatus: 'unfulfilled' | 'partial' | 'fulfilled'
  date: string
  items: number
}

const ORDERS: Order[] = [
  { id: 'ORD-10042', channel: 'shopify', customer: 'Lena Müller', address: 'Berlin, DE', status: 'new', total: 349.98, paymentStatus: 'paid', fulfillmentStatus: 'unfulfilled', date: '2026-05-16 14:32', items: 2 },
  { id: 'ORD-10041', channel: 'bol', customer: 'Jan Bakker', address: 'Amsterdam, NL', status: 'reserved', total: 99.99, paymentStatus: 'paid', fulfillmentStatus: 'unfulfilled', date: '2026-05-16 13:58', items: 1 },
  { id: 'ORD-10040', channel: 'woocommerce', customer: 'Sophie Martin', address: 'Paris, FR', status: 'picking', total: 589.97, paymentStatus: 'paid', fulfillmentStatus: 'partial', date: '2026-05-16 12:20', items: 3 },
  { id: 'ORD-10039', channel: 'shopify', customer: 'Marco Rossi', address: 'Milan, IT', status: 'packed', total: 249.99, paymentStatus: 'paid', fulfillmentStatus: 'fulfilled', date: '2026-05-16 11:05', items: 1 },
  { id: 'ORD-10038', channel: 'lightspeed', customer: 'Emma Wilson', address: 'London, GB', status: 'shipped', total: 174.98, paymentStatus: 'paid', fulfillmentStatus: 'fulfilled', date: '2026-05-15 16:44', items: 2 },
  { id: 'ORD-10037', channel: 'bol', customer: 'Pieter Vos', address: 'Utrecht, NL', status: 'delivered', total: 39.99, paymentStatus: 'paid', fulfillmentStatus: 'fulfilled', date: '2026-05-14 09:12', items: 1 },
  { id: 'ORD-10036', channel: 'shopify', customer: 'Ana Garcia', address: 'Madrid, ES', status: 'cancelled', total: 299.99, paymentStatus: 'failed', fulfillmentStatus: 'unfulfilled', date: '2026-05-14 08:30', items: 1 },
  { id: 'ORD-10035', channel: 'woocommerce', customer: 'Lars Jensen', address: 'Copenhagen, DK', status: 'new', total: 449.97, paymentStatus: 'pending', fulfillmentStatus: 'unfulfilled', date: '2026-05-16 14:10', items: 3 },
  { id: 'ORD-10034', channel: 'shopify', customer: 'Mia Schmidt', address: 'Hamburg, DE', status: 'shipped', total: 129.99, paymentStatus: 'paid', fulfillmentStatus: 'fulfilled', date: '2026-05-15 13:22', items: 1 },
  { id: 'ORD-10033', channel: 'bol', customer: 'Kevin Smit', address: 'Rotterdam, NL', status: 'reserved', total: 79.98, paymentStatus: 'paid', fulfillmentStatus: 'unfulfilled', date: '2026-05-16 10:48', items: 2 },
]

const CHANNEL_COLORS: Record<string, string> = {
  shopify: 'bg-success/10 text-success',
  woocommerce: 'bg-figure/10 text-figure',
  bol: 'bg-accent/15 text-accent',
  lightspeed: 'bg-warning/10 text-warning',
  manual: 'bg-fill text-ink-4',
}

const ORDER_STATUS_FLOW = ['new', 'reserved', 'picking', 'packed', 'shipped', 'delivered', 'cancelled'] as const

const STATUS_COLORS: Record<string, string> = {
  new: 'bg-accent/15 text-accent',
  reserved: 'bg-accent/10 text-accent',
  picking: 'bg-warning/10 text-warning',
  packed: 'bg-figure/10 text-figure',
  shipped: 'bg-cyan-500/15 text-cyan-400',
  delivered: 'bg-success/10 text-success',
  cancelled: 'bg-danger/10 text-danger',
}

const PAYMENT_COLORS: Record<string, string> = {
  paid: 'text-success',
  pending: 'text-warning',
  failed: 'text-danger',
}

const FILTER_TABS = ['all', 'new', 'reserved', 'picking', 'packed', 'shipped', 'delivered', 'cancelled'] as const

// Demo statuses in the admin language (unknown ones as-is).
function statusLabel(s: string): string {
  switch (s) {
    case 'all': return i18next.t('orders:demo.status.all')
    case 'new': return i18next.t('orders:demo.status.new')
    case 'reserved': return i18next.t('orders:demo.status.reserved')
    case 'picking': return i18next.t('orders:demo.status.picking')
    case 'packed': return i18next.t('orders:demo.status.packed')
    case 'shipped': return i18next.t('orders:demo.status.shipped')
    case 'delivered': return i18next.t('orders:demo.status.delivered')
    case 'cancelled': return i18next.t('orders:demo.status.cancelled')
    default: return s
  }
}

function paymentLabel(s: string): string {
  switch (s) {
    case 'paid': return i18next.t('orders:demo.paymentStatus.paid')
    case 'pending': return i18next.t('orders:demo.paymentStatus.pending')
    case 'failed': return i18next.t('orders:demo.paymentStatus.failed')
    default: return s
  }
}

function fulfillmentLabel(s: string): string {
  switch (s) {
    case 'unfulfilled': return i18next.t('orders:demo.fulfillmentStatus.unfulfilled')
    case 'partial': return i18next.t('orders:demo.fulfillmentStatus.partial')
    case 'fulfilled': return i18next.t('orders:demo.fulfillmentStatus.fulfilled')
    default: return s
  }
}
const STATUS_FILTERS = ['all', ...ORDER_STATUS_FLOW] as const
type StatusFilter = typeof STATUS_FILTERS[number]
const ORDERS_PATH = '/orders'

export default function Orders() {
  // Search (?q=), status tab (?status=) and the open order (/orders/<id>)
  // live in the URL, so refresh and Back keep them.
  const [searchParam, setSearch] = useQueryString('q')
  const search = searchParam ?? ''
  const [statusFilter, setStatusFilter] = useQueryEnum<StatusFilter>('status', STATUS_FILTERS, 'all')
  const { t } = useTranslation(['orders', 'common'])
  const formatMoney = useMoneyFormat()
  const activeTab = FILTER_TABS.find(tab => tab === statusFilter) ?? 'all'
  const setActiveTab = (tab: string) => setStatusFilter(tab as StatusFilter)
  const location = useLocation()
  const { open, exitToOpener } = useStackNavigation()
  const selectedId = idFromPath(location.pathname, ORDERS_PATH)
  const selectedOrder = selectedId ? ORDERS.find(o => o.id === selectedId) ?? null : null
  const closeOrder = () => exitToOpener(ORDERS_PATH)
  const toggleOrder = (o: Order) => (selectedOrder?.id === o.id ? closeOrder() : open(`${ORDERS_PATH}/${encodeURIComponent(o.id)}`))

  const filtered = ORDERS.filter(o => {
    const matchSearch = o.id.toLowerCase().includes(search.toLowerCase()) ||
      o.customer.toLowerCase().includes(search.toLowerCase())
    const matchTab = activeTab === 'all' || o.status === activeTab
    return matchSearch && matchTab
  })

  return (
    <div className="flex h-full overflow-hidden">
      {/* Main list */}
      <div className={clsx('flex flex-col', selectedOrder ? 'flex-1' : 'w-full')}>
        {/* Toolbar */}
        <div className="flex items-center gap-3 px-5 py-3 border-b border-line shrink-0">
          <div className="relative flex-1 max-w-sm">
            <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 text-ink-4" />
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder={t('demo.searchPlaceholder')}
              className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink placeholder-ink-4 outline-none focus:ring-1 focus:ring-focus"
            />
          </div>
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-raised border border-line text-xs text-ink-2 hover:bg-hover">
            <Filter size={13} /> {t('demo.filters')}
          </button>
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-raised border border-line text-xs text-ink-2 hover:bg-hover">
            <RefreshCw size={13} /> {t('demo.sync')}
          </button>
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-accent-solid text-white text-xs hover:bg-accent-solid-hover ml-auto font-medium">
            <Plus size={13} /> {t('demo.manualOrder')}
          </button>
        </div>

        {/* Status tabs */}
        <div className="flex items-center gap-0.5 px-5 py-2 border-b border-line overflow-x-auto scrollbar-none shrink-0">
          {FILTER_TABS.map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition-colors',
                activeTab === tab
                  ? 'bg-fill text-ink'
                  : 'text-ink-4 hover:text-ink-2',
              )}
            >
              {statusLabel(tab)}
              {tab !== 'all' && (
                <span className="ml-1.5 text-ink-4">
                  {ORDERS.filter(o => o.status === tab).length}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Table */}
        <div className="flex-1 overflow-auto">
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 bg-sunken border-b border-line">
              <tr>
                <th className="px-4 py-2.5 w-8">
                  <input type="checkbox" className="rounded accent-accent-solid w-3 h-3" />
                </th>
                {[t('demo.orderId'), t('demo.channel'), t('common:fields.customer'), t('demo.items'), t('common:fields.total'), t('demo.payment'), t('common:fields.status'), t('demo.fulfillment'), t('common:fields.date')].map(h => (
                  <th key={h} className="px-3 py-2.5 text-left font-medium text-ink-4 caps-label whitespace-nowrap">
                    {h}
                  </th>
                ))}
                <th className="px-3 py-2.5 w-8" />
              </tr>
            </thead>
            <tbody className="divide-y divide-line/60">
              {filtered.map(o => (
                <tr
                  key={o.id}
                  onClick={() => toggleOrder(o)}
                  className={clsx(
                    'cursor-pointer hover:bg-hover/60 transition-colors group',
                    selectedOrder?.id === o.id && 'bg-accent/10',
                  )}
                >
                  <td className="px-4 py-2.5" onClick={e => e.stopPropagation()}>
                    <input type="checkbox" className="rounded accent-accent-solid w-3 h-3" />
                  </td>
                  <td className="px-3 py-2.5">
                    <span className="font-mono font-medium text-ink">{o.id}</span>
                  </td>
                  <td className="px-3 py-2.5">
                    <span className={clsx('px-2 py-0.5 rounded-sm text-2xs font-semibold capitalize', CHANNEL_COLORS[o.channel])}>
                      {o.channel}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 text-ink-2">{o.customer}</td>
                  <td className="px-3 py-2.5 text-ink-3">{o.items}</td>
                  <td className="px-3 py-2.5 font-medium text-ink"><Money minor={Math.round(o.total * 100)} currency="EUR" /></td>
                  <td className="px-3 py-2.5">
                    <span className={clsx('font-medium capitalize', PAYMENT_COLORS[o.paymentStatus])}>
                      {paymentLabel(o.paymentStatus)}
                    </span>
                  </td>
                  <td className="px-3 py-2.5">
                    <span className={clsx('px-2 py-0.5 rounded-sm text-2xs font-semibold capitalize', STATUS_COLORS[o.status])}>
                      {statusLabel(o.status)}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 text-ink-3 capitalize">{fulfillmentLabel(o.fulfillmentStatus)}</td>
                  <td className="px-3 py-2.5 text-ink-4 font-mono whitespace-nowrap">{o.date}</td>
                  <td className="px-3 py-2.5">
                    <ExternalLink size={12} className="opacity-0 group-hover:opacity-100 focus-within:opacity-100 text-ink-4" />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Order detail panel */}
      {selectedOrder && (
        <div className="w-80 border-l border-line bg-sunken flex flex-col overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-line">
            <h2 className="text-sm font-semibold text-ink">{selectedOrder.id}</h2>
            <button onClick={closeOrder} aria-label={t('demo.close')} className="text-xs text-ink-4 dark:hover:text-ink-2">✕</button>
          </div>

          {/* Status flow */}
          <div className="px-4 py-3 border-b border-line">
            <p className="text-2xs text-ink-4 caps-label mb-2">{t('demo.orderFlow')}</p>
            <div className="flex items-center gap-1 flex-wrap">
              {ORDER_STATUS_FLOW.filter(s => s !== 'cancelled').map((s, i) => {
                const steps = ORDER_STATUS_FLOW.filter(x => x !== 'cancelled')
                const currentIdx = steps.indexOf(selectedOrder.status as typeof steps[number])
                const stepIdx = steps.indexOf(s)
                const done = stepIdx <= currentIdx
                return (
                  <div key={s} className="flex items-center gap-1">
                    <div className={clsx(
                      'w-5 h-5 rounded-full text-2xs font-bold flex items-center justify-center',
                      done ? 'bg-accent-solid text-white' : 'bg-fill text-ink-4',
                    )}>
                      {i + 1}
                    </div>
                    {i < steps.length - 1 && (
                      <div className={clsx('h-0.5 w-3', done ? 'bg-accent-solid' : 'bg-fill')} />
                    )}
                  </div>
                )
              })}
            </div>
            <p className="text-xs mt-2 text-ink-2 capitalize font-medium">{statusLabel(selectedOrder.status)}</p>
          </div>

          {/* Info */}
          <div className="px-4 py-3 space-y-2.5 border-b border-line">
            {[
              [t('common:fields.customer'), selectedOrder.customer],
              [t('demo.address'), selectedOrder.address],
              [t('demo.channel'), selectedOrder.channel],
              [t('demo.items'), selectedOrder.items.toString()],
              [t('common:fields.total'), formatMoney(Math.round(selectedOrder.total * 100), 'EUR')],
              [t('demo.payment'), paymentLabel(selectedOrder.paymentStatus)],
              [t('common:fields.date'), selectedOrder.date],
            ].map(([k, v]) => (
              <div key={k} className="flex justify-between items-center">
                <span className="text-xs text-ink-4">{k}</span>
                <span className="text-xs text-ink font-medium capitalize">{v}</span>
              </div>
            ))}
          </div>

          {/* Actions */}
          <div className="px-4 py-3 space-y-2 mt-auto">
            <button className="w-full py-2 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover transition-colors">
              {t('demo.startPicking')}
            </button>
            <button className="w-full py-2 rounded-lg bg-raised border border-line text-ink-2 text-xs hover:bg-hover transition-colors flex items-center justify-center gap-2">
              {t('demo.changeStatus')} <ChevronDown size={12} />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
