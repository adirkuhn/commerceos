import {
  Package, Warehouse, ShoppingCart, AlertTriangle,
  TrendingUp, TrendingDown, XCircle, DollarSign, Calendar,
} from 'lucide-react'
import {
  AreaChart, Area, BarChart, Bar, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, Legend,
} from 'recharts'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useExpiringBatches } from '../inventory/hooks/useExpiringBatches'
import { useDashboardStats } from '../dashboard/hooks/useDashboardStats'
import type { ChannelRevenue } from '../dashboard/types'

function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

const CHANNEL_LABELS: Record<string, string> = { pos: 'POS', manual: 'Manual', woocommerce: 'WooCommerce' }
const CHANNEL_COLORS = ['#2563EB', '#8B5CF6', '#22C55E', '#F59E0B']

function channelLabel(source: string): string {
  return CHANNEL_LABELS[source] ?? source
}

interface StatCardProps {
  label: string
  value?: string
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
      <p className="text-2xl font-semibold dark:text-gray-100 text-gray-900 mb-1">{value ?? '—'}</p>
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
  const { token } = useAuth()
  const { stats } = useDashboardStats(token ?? '')
  const currency = stats?.currencyCode ?? 'USD'

  const stockFlowData = (stats?.stockFlowLast7Days ?? []).map(f => ({ day: f.date.slice(5), in: f.in, out: f.out }))
  const channelData: (ChannelRevenue & { label: string; color: string })[] =
    (stats?.salesByChannelThisMonth ?? []).map((c, i) => ({
      ...c, label: channelLabel(c.source), color: CHANNEL_COLORS[i % CHANNEL_COLORS.length],
    }))

  return (
    <div className="p-5 space-y-5 overflow-auto h-full">
      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-3">
        <StatCard label="Total Products" value={stats?.totalProducts.toLocaleString()} sub="across all channels" icon={Package} color="blue" />
        <StatCard label="Total Stock" value={stats?.totalStockUnits.toLocaleString()} sub="units in all locations" icon={Warehouse} color="green" />
        <StatCard label="Total Orders" value={stats?.totalOrders.toLocaleString()} sub="all time" icon={ShoppingCart} color="blue" />
        <StatCard label="Stock Issues" value={stats?.stockIssues.toLocaleString()} sub="orders need stock resolution" icon={ShoppingCart} color="amber" />
        <StatCard label="Expiring (7d)" value={stats?.expiringSoon.toLocaleString()} sub="batches near expiry" icon={AlertTriangle} color="amber" />
        <StatCard label="Expired" value={stats?.expired.toLocaleString()} sub="need disposal" icon={XCircle} color="red" alert />
        <StatCard
          label="Est. Revenue"
          value={stats ? formatMoney(stats.revenueThisMonthMinorUnits, currency) : undefined}
          sub="this month"
          icon={DollarSign}
          color="green"
        />
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Stock flow */}
        <div className="lg:col-span-2 dark:bg-[#1F2937] bg-white rounded-xl p-4 border dark:border-[#2D3748] border-gray-200">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Stock Flow</h3>
              <p className="text-xs dark:text-gray-500 text-gray-400 mt-0.5">Inbound vs outbound</p>
            </div>
          </div>
          <ResponsiveContainer width="100%" height={180}>
            <AreaChart data={stockFlowData} margin={{ top: 0, right: 0, left: -20, bottom: 0 }}>
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
          {channelData.length === 0 ? (
            <p className="text-xs dark:text-gray-500 text-gray-400">No sales yet this month.</p>
          ) : (
            <>
              <ResponsiveContainer width="100%" height={100}>
                <BarChart data={channelData} layout="vertical" margin={{ top: 0, right: 10, left: 0, bottom: 0 }}>
                  <XAxis type="number" hide />
                  <YAxis type="category" dataKey="label" tick={{ fontSize: 11, fill: '#6B7280' }} axisLine={false} tickLine={false} width={80} />
                  <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number) => [formatMoney(v, currency), 'Revenue']} />
                  <Bar dataKey="total_minor_units" radius={4} fill="#2563EB" />
                </BarChart>
              </ResponsiveContainer>
              <div className="mt-3 space-y-2">
                {channelData.map(c => (
                  <div key={c.source} className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full" style={{ background: c.color }} />
                      <span className="text-xs dark:text-gray-400 text-gray-500">{c.label}</span>
                    </div>
                    <span className="text-xs font-medium dark:text-gray-200 text-gray-700">{formatMoney(c.total_minor_units, currency)}</span>
                  </div>
                ))}
              </div>
            </>
          )}
        </div>
      </div>

      {/* Bottom row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Top products */}
        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Top Products</h3>
          </div>
          {(stats?.topProductsThisMonth.length ?? 0) === 0 ? (
            <p className="px-4 py-4 text-xs dark:text-gray-500 text-gray-400">No products sold yet this month.</p>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="border-b dark:border-[#2D3748] border-gray-100">
                  <th className="text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider px-4 py-2">Product</th>
                  <th className="text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider px-4 py-2">Sold</th>
                  <th className="text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase tracking-wider px-4 py-2">Revenue</th>
                </tr>
              </thead>
              <tbody>
                {stats?.topProductsThisMonth.map(p => (
                  <tr key={p.sku_id ?? p.sku_code} className="border-b dark:border-[#2D3748]/50 border-gray-50 last:border-0 hover:dark:bg-[#2D3748]/30 hover:bg-gray-50 transition-colors">
                    <td className="px-4 py-2.5">
                      <p className="text-xs font-medium dark:text-gray-200 text-gray-800">{p.product_name}</p>
                      <p className="text-[10px] dark:text-gray-500 text-gray-400 font-mono">{p.sku_code}</p>
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <span className="text-xs dark:text-gray-300 text-gray-700">{p.quantity_sold}</span>
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <span className="text-xs font-medium dark:text-gray-200 text-gray-800">{formatMoney(p.revenue_minor_units, currency)}</span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </div>

        <ExpiringSoonWidget />
      </div>
    </div>
  )
}
