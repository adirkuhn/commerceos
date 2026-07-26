import { useState } from 'react'
import { Search, Filter, Plus, RefreshCw, ExternalLink, ChevronDown } from 'lucide-react'
import clsx from 'clsx'

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
  shopify: 'bg-green-500/15 text-green-400',
  woocommerce: 'bg-purple-500/15 text-purple-400',
  bol: 'bg-blue-500/15 text-blue-400',
  lightspeed: 'bg-orange-500/15 text-orange-400',
  manual: 'bg-gray-500/15 text-gray-400',
}

const ORDER_STATUS_FLOW = ['new', 'reserved', 'picking', 'packed', 'shipped', 'delivered', 'cancelled'] as const

const STATUS_COLORS: Record<string, string> = {
  new: 'bg-blue-500/15 text-blue-400',
  reserved: 'bg-indigo-500/15 text-indigo-400',
  picking: 'bg-amber-500/15 text-amber-400',
  packed: 'bg-purple-500/15 text-purple-400',
  shipped: 'bg-cyan-500/15 text-cyan-400',
  delivered: 'bg-green-500/15 text-green-400',
  cancelled: 'bg-red-500/15 text-red-400',
}

const PAYMENT_COLORS: Record<string, string> = {
  paid: 'text-green-400',
  pending: 'text-amber-400',
  failed: 'text-red-400',
}

const FILTER_TABS = ['All', 'New', 'Reserved', 'Picking', 'Packed', 'Shipped', 'Delivered', 'Cancelled']

export default function Orders() {
  const [search, setSearch] = useState('')
  const [activeTab, setActiveTab] = useState('All')
  const [selectedOrder, setSelectedOrder] = useState<Order | null>(null)

  const filtered = ORDERS.filter(o => {
    const matchSearch = o.id.toLowerCase().includes(search.toLowerCase()) ||
      o.customer.toLowerCase().includes(search.toLowerCase())
    const matchTab = activeTab === 'All' || o.status === activeTab.toLowerCase()
    return matchSearch && matchTab
  })

  return (
    <div className="flex h-full overflow-hidden">
      {/* Main list */}
      <div className={clsx('flex flex-col', selectedOrder ? 'flex-1' : 'w-full')}>
        {/* Toolbar */}
        <div className="flex items-center gap-3 px-5 py-3 border-b dark:border-[#2D3748] border-gray-200 shrink-0">
          <div className="relative flex-1 max-w-sm">
            <Search size={13} className="absolute left-3 top-1/2 -translate-y-1/2 dark:text-gray-500 text-gray-400" />
            <input
              value={search}
              onChange={e => setSearch(e.target.value)}
              placeholder="Search order ID, customer..."
              className="w-full pl-8 pr-3 py-1.5 text-xs rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-200 text-gray-800 dark:placeholder-gray-500 placeholder-gray-400 outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
            <Filter size={13} /> Filters
          </button>
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 text-xs dark:text-gray-300 text-gray-600 hover:dark:bg-[#2D3748] hover:bg-gray-100">
            <RefreshCw size={13} /> Sync
          </button>
          <button className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-blue-600 text-white text-xs hover:bg-blue-700 ml-auto font-medium">
            <Plus size={13} /> Manual Order
          </button>
        </div>

        {/* Status tabs */}
        <div className="flex items-center gap-0.5 px-5 py-2 border-b dark:border-[#2D3748] border-gray-200 overflow-x-auto scrollbar-none shrink-0">
          {FILTER_TABS.map(tab => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={clsx(
                'px-3 py-1.5 rounded-lg text-xs font-medium whitespace-nowrap transition-colors',
                activeTab === tab
                  ? 'dark:bg-[#1F2937] bg-gray-100 dark:text-gray-100 text-gray-800'
                  : 'dark:text-gray-500 text-gray-400 hover:dark:text-gray-300 hover:text-gray-600',
              )}
            >
              {tab}
              {tab !== 'All' && (
                <span className="ml-1.5 dark:text-gray-600 text-gray-400">
                  {ORDERS.filter(o => o.status === tab.toLowerCase()).length}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Table */}
        <div className="flex-1 overflow-auto">
          <table className="w-full text-xs">
            <thead className="sticky top-0 z-10 dark:bg-[#111827] bg-gray-50 border-b dark:border-[#2D3748] border-gray-200">
              <tr>
                <th className="px-4 py-2.5 w-8">
                  <input type="checkbox" className="rounded accent-blue-600 w-3 h-3" />
                </th>
                {['Order ID', 'Channel', 'Customer', 'Items', 'Total', 'Payment', 'Status', 'Fulfillment', 'Date'].map(h => (
                  <th key={h} className="px-3 py-2.5 text-left font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider whitespace-nowrap">
                    {h}
                  </th>
                ))}
                <th className="px-3 py-2.5 w-8" />
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/60 divide-gray-100">
              {filtered.map(o => (
                <tr
                  key={o.id}
                  onClick={() => setSelectedOrder(selectedOrder?.id === o.id ? null : o)}
                  className={clsx(
                    'cursor-pointer hover:dark:bg-[#1F2937]/60 hover:bg-gray-50 transition-colors group',
                    selectedOrder?.id === o.id && 'dark:bg-blue-600/5 bg-blue-50',
                  )}
                >
                  <td className="px-4 py-2.5" onClick={e => e.stopPropagation()}>
                    <input type="checkbox" className="rounded accent-blue-600 w-3 h-3" />
                  </td>
                  <td className="px-3 py-2.5">
                    <span className="font-mono font-medium dark:text-gray-200 text-gray-800">{o.id}</span>
                  </td>
                  <td className="px-3 py-2.5">
                    <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', CHANNEL_COLORS[o.channel])}>
                      {o.channel}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 dark:text-gray-300 text-gray-700">{o.customer}</td>
                  <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500">{o.items}</td>
                  <td className="px-3 py-2.5 font-medium dark:text-gray-200 text-gray-800">€{o.total.toFixed(2)}</td>
                  <td className="px-3 py-2.5">
                    <span className={clsx('font-medium capitalize', PAYMENT_COLORS[o.paymentStatus])}>
                      {o.paymentStatus}
                    </span>
                  </td>
                  <td className="px-3 py-2.5">
                    <span className={clsx('px-2 py-0.5 rounded-full text-[10px] font-medium capitalize', STATUS_COLORS[o.status])}>
                      {o.status}
                    </span>
                  </td>
                  <td className="px-3 py-2.5 dark:text-gray-400 text-gray-500 capitalize">{o.fulfillmentStatus}</td>
                  <td className="px-3 py-2.5 dark:text-gray-500 text-gray-400 font-mono whitespace-nowrap">{o.date}</td>
                  <td className="px-3 py-2.5">
                    <ExternalLink size={12} className="opacity-0 group-hover:opacity-100 dark:text-gray-500 text-gray-400" />
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </div>

      {/* Order detail panel */}
      {selectedOrder && (
        <div className="w-80 border-l dark:border-[#2D3748] border-gray-200 dark:bg-[#111827] bg-gray-50 flex flex-col overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h2 className="text-sm font-semibold dark:text-gray-100 text-gray-900">{selectedOrder.id}</h2>
            <button onClick={() => setSelectedOrder(null)} className="text-xs dark:text-gray-500 text-gray-400 hover:dark:text-gray-300">✕</button>
          </div>

          {/* Status flow */}
          <div className="px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <p className="text-[10px] dark:text-gray-500 text-gray-400 uppercase tracking-wider mb-2">Order Flow</p>
            <div className="flex items-center gap-1 flex-wrap">
              {ORDER_STATUS_FLOW.filter(s => s !== 'cancelled').map((s, i) => {
                const steps = ORDER_STATUS_FLOW.filter(x => x !== 'cancelled')
                const currentIdx = steps.indexOf(selectedOrder.status as typeof steps[number])
                const stepIdx = steps.indexOf(s)
                const done = stepIdx <= currentIdx
                return (
                  <div key={s} className="flex items-center gap-1">
                    <div className={clsx(
                      'w-5 h-5 rounded-full text-[9px] font-bold flex items-center justify-center',
                      done ? 'bg-blue-600 text-white' : 'dark:bg-[#2D3748] bg-gray-200 dark:text-gray-500 text-gray-400',
                    )}>
                      {i + 1}
                    </div>
                    {i < steps.length - 1 && (
                      <div className={clsx('h-0.5 w-3', done ? 'bg-blue-600' : 'dark:bg-[#2D3748] bg-gray-200')} />
                    )}
                  </div>
                )
              })}
            </div>
            <p className="text-xs mt-2 dark:text-gray-300 text-gray-700 capitalize font-medium">{selectedOrder.status}</p>
          </div>

          {/* Info */}
          <div className="px-4 py-3 space-y-2.5 border-b dark:border-[#2D3748] border-gray-200">
            {[
              ['Customer', selectedOrder.customer],
              ['Address', selectedOrder.address],
              ['Channel', selectedOrder.channel],
              ['Items', selectedOrder.items.toString()],
              ['Total', `€${selectedOrder.total.toFixed(2)}`],
              ['Payment', selectedOrder.paymentStatus],
              ['Date', selectedOrder.date],
            ].map(([k, v]) => (
              <div key={k} className="flex justify-between items-center">
                <span className="text-xs dark:text-gray-500 text-gray-400">{k}</span>
                <span className="text-xs dark:text-gray-200 text-gray-800 font-medium capitalize">{v}</span>
              </div>
            ))}
          </div>

          {/* Actions */}
          <div className="px-4 py-3 space-y-2 mt-auto">
            <button className="w-full py-2 rounded-lg bg-blue-600 text-white text-xs font-medium hover:bg-blue-700 transition-colors">
              Start Picking
            </button>
            <button className="w-full py-2 rounded-lg dark:bg-[#1F2937] bg-white border dark:border-[#2D3748] border-gray-200 dark:text-gray-300 text-gray-700 text-xs hover:dark:bg-[#2D3748] hover:bg-gray-100 transition-colors flex items-center justify-center gap-2">
              Change Status <ChevronDown size={12} />
            </button>
          </div>
        </div>
      )}
    </div>
  )
}
