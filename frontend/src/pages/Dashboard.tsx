import {
  Package, Warehouse, ShoppingCart, AlertTriangle,
  TrendingUp, TrendingDown, XCircle, DollarSign, ArrowUpRight, Calendar,
} from 'lucide-react'
import {
  AreaChart, Area, BarChart, Bar, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, Legend,
} from 'recharts'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useExpiringBatches } from '../inventory/hooks/useExpiringBatches'

const STOCK_FLOW = [
  { day: 'Mon', in: 142, out: 89 },
  { day: 'Tue', in: 98, out: 134 },
  { day: 'Wed', in: 203, out: 167 },
  { day: 'Thu', in: 87, out: 112 },
  { day: 'Fri', in: 310, out: 245 },
  { day: 'Sat', in: 56, out: 78 },
  { day: 'Sun', in: 32, out: 41 },
]

const CHANNEL_SALES = [
  { name: 'Shopify', value: 48200, color: '#2563EB' },
  { name: 'WooCommerce', value: 31500, color: '#8B5CF6' },
  { name: 'Bol', value: 22100, color: '#22C55E' },
  { name: 'Lightspeed', value: 14300, color: '#F59E0B' },
]

const TOP_PRODUCTS = [
  { name: 'Wireless Headphones Pro', sku: 'WHP-001', sold: 234, revenue: 58500, trend: 'up' },
  { name: 'USB-C Hub 7-in-1', sku: 'UCH-007', sold: 198, revenue: 19800, trend: 'up' },
  { name: 'Mechanical Keyboard TKL', sku: 'MKT-TKL', sold: 156, revenue: 46800, trend: 'down' },
  { name: 'Webcam 4K Ultra', sku: 'WCM-4K', sold: 143, revenue: 42900, trend: 'up' },
  { name: 'Laptop Stand Pro', sku: 'LSP-002', sold: 127, revenue: 12700, trend: 'down' },
]

interface StatCardProps {
  label: string
  value: string
  sub?: string
  icon: React.ElementType
  trend?: 'up' | 'down'
  trendValue?: string
  color?: string
  alert?: boolean
}

function StatCard({ label, value, sub, icon: Icon, trend, trendValue, color = 'blue', alert }: StatCardProps) {
  return (
    <div className={clsx(
      'dark:bg-[#1F2937] bg-white rounded-xl p-4 border',
      alert
        ? 'dark:border-red-500/30 border-red-200'
        : 'dark:border-[#2D3748] border-gray-200',
    )}>
      <div className="flex items-start justify-between mb-3">
        <span className="text-xs dark:text-gray-400 text-gray-500 font-medium">{label}</span>
        <div className={clsx(
          'w-7 h-7 rounded-lg flex items-center justify-center',
          color === 'blue' && 'bg-blue-500/15',
          color === 'green' && 'bg-green-500/15',
          color === 'amber' && 'bg-amber-500/15',
          color === 'red' && 'bg-red-500/15',
        )}>
          <Icon size={14} className={clsx(
            color === 'blue' && 'text-blue-400',
            color === 'green' && 'text-green-400',
            color === 'amber' && 'text-amber-400',
            color === 'red' && 'text-red-400',
          )} />
        </div>
      </div>
      <p className="text-2xl font-semibold dark:text-gray-100 text-gray-900 mb-1">{value}</p>
      {sub && <p className="text-xs dark:text-gray-500 text-gray-400">{sub}</p>}
      {trend && trendValue && (
        <div className={clsx(
          'flex items-center gap-1 mt-2 text-xs font-medium',
          trend === 'up' ? 'text-green-400' : 'text-red-400',
        )}>
          {trend === 'up' ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
          {trendValue} vs last week
        </div>
      )}
    </div>
  )
}

// The only real (non-mock) data on this page — everything else here is
// still placeholder content pending a proper reporting/analytics backend.
function ExpiringSoonWidget() {
  const { token } = useAuth()
  const { batches, loading, error } = useExpiringBatches(token ?? '', 30)

  return (
    <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
        <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Expiring Soon</h3>
        <span className="text-[10px] bg-amber-500/20 text-amber-400 px-2 py-0.5 rounded-full font-medium">
          {batches.length} within 30 days
        </span>
      </div>
      {loading && <p className="px-4 py-4 text-xs dark:text-gray-500 text-gray-400">Loading…</p>}
      {error && <p className="px-4 py-4 text-xs text-red-400">{error}</p>}
      {!loading && !error && batches.length === 0 && (
        <p className="px-4 py-4 text-xs dark:text-gray-500 text-gray-400">Nothing expiring in the next 30 days.</p>
      )}
      {!loading && !error && batches.length > 0 && (
        <div className="divide-y dark:divide-[#2D3748] divide-gray-100 max-h-64 overflow-auto">
          {batches.map(b => {
            const expired = new Date(b.expiry_date).getTime() < Date.now()
            return (
              <div key={b.batch_id} className="flex items-start gap-3 px-4 py-3 hover:dark:bg-[#2D3748]/30 hover:bg-gray-50 transition-colors">
                <Calendar size={13} className={clsx('mt-0.5 shrink-0', expired ? 'text-red-400' : 'text-amber-400')} />
                <div className="flex-1 min-w-0">
                  <p className="text-xs dark:text-gray-200 text-gray-700 truncate">{b.product_name}</p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 font-mono">
                    {b.sku_code} · lot {b.lot_number || '—'} · {b.warehouse_name}
                  </p>
                </div>
                <div className="text-right shrink-0">
                  <p className="text-xs font-medium dark:text-gray-200 text-gray-800">{b.quantity} units</p>
                  <p className={clsx('text-[10px] font-mono', expired ? 'text-red-400' : 'text-amber-400')}>
                    {new Date(b.expiry_date).toLocaleDateString()}
                  </p>
                </div>
              </div>
            )
          })}
        </div>
      )}
    </div>
  )
}

const TOOLTIP_STYLE = {
  backgroundColor: '#1F2937',
  border: '1px solid #2D3748',
  borderRadius: '8px',
  fontSize: '12px',
  color: '#F9FAFB',
}

export default function Dashboard() {
  return (
    <div className="p-5 space-y-5 overflow-auto h-full">
      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-3">
        <StatCard label="Total Products" value="2,418" sub="across all channels" icon={Package} trend="up" trendValue="+4.2%" color="blue" />
        <StatCard label="Total Stock" value="84,293" sub="units in all locations" icon={Warehouse} trend="up" trendValue="+1.8%" color="green" />
        <StatCard label="Orders Today" value="143" sub="38 pending action" icon={ShoppingCart} trend="up" trendValue="+22%" color="blue" />
        <StatCard label="Pending" value="38" sub="need fulfillment" icon={ShoppingCart} color="amber" />
        <StatCard label="Expiring (7d)" value="8" sub="products near expiry" icon={AlertTriangle} color="amber" />
        <StatCard label="Expired" value="3" sub="need disposal" icon={XCircle} color="red" alert />
        <StatCard label="Est. Revenue" value="€116k" sub="this month" icon={DollarSign} trend="up" trendValue="+8.3%" color="green" />
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Stock flow */}
        <div className="lg:col-span-2 dark:bg-[#1F2937] bg-white rounded-xl p-4 border dark:border-[#2D3748] border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Stock Flow</h3>
              <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Inbound vs outbound — last 7 days</p>
            </div>
          </div>
          <ResponsiveContainer width="100%" height={180}>
            <AreaChart data={STOCK_FLOW} margin={{ top: 0, right: 0, left: -20, bottom: 0 }}>
              <defs>
                <linearGradient id="inGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#2563EB" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#2563EB" stopOpacity={0} />
                </linearGradient>
                <linearGradient id="outGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#22C55E" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#22C55E" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
              <XAxis dataKey="day" tick={{ fontSize: 11, fill: '#6B7280' }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 11, fill: '#6B7280' }} axisLine={false} tickLine={false} />
              <Tooltip contentStyle={TOOLTIP_STYLE} />
              <Legend iconSize={10} wrapperStyle={{ fontSize: '11px' }} />
              <Area type="monotone" dataKey="in" name="Inbound" stroke="#2563EB" strokeWidth={2} fill="url(#inGrad)" />
              <Area type="monotone" dataKey="out" name="Outbound" stroke="#22C55E" strokeWidth={2} fill="url(#outGrad)" />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Channel sales */}
        <div className="dark:bg-[#1F2937] bg-white rounded-xl p-4 border dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Sales by Channel</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mb-4">This month</p>
          <ResponsiveContainer width="100%" height={100}>
            <BarChart data={CHANNEL_SALES} layout="vertical" margin={{ top: 0, right: 10, left: 0, bottom: 0 }}>
              <XAxis type="number" hide />
              <YAxis type="category" dataKey="name" tick={{ fontSize: 11, fill: '#6B7280' }} axisLine={false} tickLine={false} width={80} />
              <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number) => [`€${v.toLocaleString()}`, 'Revenue']} />
              <Bar dataKey="value" radius={4} fill="#2563EB" />
            </BarChart>
          </ResponsiveContainer>
          <div className="mt-3 space-y-2">
            {CHANNEL_SALES.map(c => (
              <div key={c.name} className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-2 h-2 rounded-full" style={{ background: c.color }} />
                  <span className="text-xs dark:text-gray-400 text-gray-500">{c.name}</span>
                </div>
                <span className="text-xs font-medium dark:text-gray-200 text-gray-700">€{c.value.toLocaleString()}</span>
              </div>
            ))}
          </div>
        </div>
      </div>

      {/* Bottom row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Top products */}
        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Top Products</h3>
            <button className="text-xs text-blue-400 hover:text-blue-300 transition-colors flex items-center gap-1">
              View all <ArrowUpRight size={12} />
            </button>
          </div>
          <table className="w-full">
            <thead>
              <tr className="border-b dark:border-[#2D3748] border-gray-100">
                <th className="text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider px-4 py-2">Product</th>
                <th className="text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider px-4 py-2">Sold</th>
                <th className="text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider px-4 py-2">Revenue</th>
              </tr>
            </thead>
            <tbody>
              {TOP_PRODUCTS.map(p => (
                <tr key={p.sku} className="border-b dark:border-[#2D3748]/50 border-gray-50 last:border-0 hover:dark:bg-[#2D3748]/30 hover:bg-gray-50 transition-colors">
                  <td className="px-4 py-2.5">
                    <p className="text-xs font-medium dark:text-gray-200 text-gray-800">{p.name}</p>
                    <p className="text-[10px] dark:text-gray-500 text-gray-400 font-mono">{p.sku}</p>
                  </td>
                  <td className="px-4 py-2.5 text-right">
                    <span className="text-xs dark:text-gray-300 text-gray-700">{p.sold}</span>
                  </td>
                  <td className="px-4 py-2.5 text-right">
                    <div className="flex items-center justify-end gap-1">
                      <span className="text-xs font-medium dark:text-gray-200 text-gray-800">€{p.revenue.toLocaleString()}</span>
                      {p.trend === 'up'
                        ? <TrendingUp size={11} className="text-green-400" />
                        : <TrendingDown size={11} className="text-red-400" />
                      }
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {/* Expiring soon — the one real (non-mock) widget on this page */}
        <ExpiringSoonWidget />
      </div>
    </div>
  )
}
