import {
  AlertCircle,
  Package, Warehouse, ShoppingCart, AlertTriangle,
  TrendingUp, TrendingDown, XCircle, DollarSign, Calendar,
} from 'lucide-react'
import {
  AreaChart, Area, BarChart, Bar, Cell, XAxis, YAxis, CartesianGrid,
  Tooltip, ResponsiveContainer, Legend,
} from 'recharts'
import { channelSeriesColor, seriesColor, useChartTheme } from '../charts/chartTheme'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useExpiringBatches } from '../inventory/hooks/useExpiringBatches'
import { useDashboardStats } from '../dashboard/hooks/useDashboardStats'
import type { ChannelRevenue } from '../dashboard/types'
import { formatMoney } from '../format/money'

const CHANNEL_LABELS: Record<string, string> = { pos: 'POS', manual: 'Manual', woocommerce: 'WooCommerce' }

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
      'bg-raised rounded-xl p-4 border',
      alert
        ? 'border-danger/40'
        : 'border-line',
    )}>
      <div className="flex items-start justify-between mb-3">
        <span className="text-xs text-ink-3 font-medium">{label}</span>
        <div className={clsx(
          'w-7 h-7 rounded-lg flex items-center justify-center',
          color === 'blue' && 'bg-accent/15',
          color === 'green' && 'bg-green-500/15',
          color === 'amber' && 'bg-amber-500/15',
          color === 'red' && 'bg-red-500/15',
        )}>
          <Icon size={14} className={clsx(
            color === 'blue' && 'text-accent',
            color === 'green' && 'text-success',
            color === 'amber' && 'text-warning',
            color === 'red' && 'text-danger',
          )} />
        </div>
      </div>
      <p className="text-2xl font-semibold text-ink mb-1">{value ?? '—'}</p>
      {sub && <p className="text-xs text-ink-4">{sub}</p>}
      {alert && (
        <p className="mt-2 flex items-center gap-1 text-xs font-medium text-danger">
          <AlertCircle size={12} aria-hidden /> Needs attention
        </p>
      )}
      {trend && trendValue && (
        <div className={clsx(
          'flex items-center gap-1 mt-2 text-xs font-medium',
          trend === 'up' ? 'text-success' : 'text-danger',
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
    <div className="bg-raised rounded-xl border border-line overflow-hidden">
      <div className="flex items-center justify-between px-4 py-3 border-b border-line">
        <h3 className="text-sm font-semibold text-ink">Expiring soon</h3>
        <span className={clsx('text-2xs px-2 py-0.5 rounded-full font-medium', batches.length > 0 ? 'bg-amber-500/20 text-warning' : 'bg-fill text-ink-3')}>
          {batches.length} within 30 days
        </span>
      </div>
      {loading && <p className="px-4 py-4 text-xs text-ink-4">Loading…</p>}
      {error && <p className="px-4 py-4 text-xs text-danger">{error}</p>}
      {!loading && !error && batches.length === 0 && (
        <p className="px-4 py-4 text-xs text-ink-4">Nothing expiring in the next 30 days.</p>
      )}
      {!loading && !error && batches.length > 0 && (
        <div className="divide-y divide-line max-h-64 overflow-auto">
          {batches.map(b => {
            const expired = new Date(b.expiry_date).getTime() < Date.now()
            return (
              <div key={b.batch_id} className="flex items-start gap-3 px-4 py-3 hover:bg-hover/30 transition-colors">
                <Calendar size={13} className={clsx('mt-0.5 shrink-0', expired ? 'text-danger' : 'text-warning')} />
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-ink-2 truncate">{b.product_name}</p>
                  <p className="text-2xs text-ink-4 font-mono">
                    {b.sku_code} · lot {b.lot_number || '—'} · {b.warehouse_name}
                  </p>
                </div>
                <div className="text-right shrink-0">
                  <p className="text-xs font-medium text-ink">{b.quantity} units</p>
                  <p className={clsx('text-2xs font-mono', expired ? 'text-danger' : 'text-warning')}>
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

export default function Dashboard() {
  const { token } = useAuth()
  const { stats } = useDashboardStats(token ?? '')
  const currency = stats?.currencyCode ?? 'USD'
  const chart = useChartTheme()

  const stockFlowData = (stats?.stockFlowLast7Days ?? []).map(f => ({ day: f.date.slice(5), in: f.in, out: f.out }))
  const channelData: (ChannelRevenue & { label: string; color: string })[] =
    (stats?.salesByChannelThisMonth ?? []).map(c => ({
      ...c, label: channelLabel(c.source), color: channelSeriesColor(chart, c.source),
    }))

  return (
    <div className="p-5 space-y-5 overflow-auto h-full">
      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-3">
        <StatCard label="Total products" value={stats?.totalProducts.toLocaleString()} sub="across all channels" icon={Package} color="blue" />
        <StatCard label="Total stock" value={stats?.totalStockUnits.toLocaleString()} sub="units in all locations" icon={Warehouse} color="green" />
        <StatCard label="Total orders" value={stats?.totalOrders.toLocaleString()} sub="all time" icon={ShoppingCart} color="blue" />
        <StatCard label="Stock Issues" value={stats?.stockIssues.toLocaleString()} sub="orders need stock resolution" icon={ShoppingCart} color="amber" />
        <StatCard label="Expiring (7d)" value={stats?.expiringSoon.toLocaleString()} sub="batches near expiry" icon={AlertTriangle} color="amber" />
        <StatCard label="Expired" value={stats?.expired.toLocaleString()} sub="need disposal" icon={XCircle} color="red" alert={(stats?.expired ?? 0) > 0} />
        <StatCard
          label="Est. revenue"
          value={stats ? formatMoney(stats.revenueThisMonthMinorUnits, currency) : undefined}
          sub="this month"
          icon={DollarSign}
          color="green"
        />
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        {/* Stock flow */}
        <div className="lg:col-span-2 bg-raised rounded-xl p-4 border border-line">
          <div className="flex items-center justify-between mb-4">
            <div>
              <h3 className="text-sm font-semibold text-ink">Stock flow</h3>
              <p className="text-xs text-ink-4 mt-0.5">Inbound vs outbound</p>
            </div>
          </div>
          <ResponsiveContainer width="100%" height={180}>
            <AreaChart data={stockFlowData} margin={{ top: 0, right: 0, left: -20, bottom: 0 }}>
              <defs>
                <linearGradient id="inGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={seriesColor(chart, 0)} stopOpacity={0.3} />
                  <stop offset="95%" stopColor={seriesColor(chart, 0)} stopOpacity={0} />
                </linearGradient>
                <linearGradient id="outGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor={seriesColor(chart, 1)} stopOpacity={0.3} />
                  <stop offset="95%" stopColor={seriesColor(chart, 1)} stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
              <XAxis dataKey="day" tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} />
              <YAxis tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} />
              <Tooltip contentStyle={chart.tooltip} />
              <Legend iconSize={10} wrapperStyle={{ fontSize: '11px', color: chart.label }} />
              <Area type="monotone" dataKey="in" name="Inbound" stroke={seriesColor(chart, 0)} strokeWidth={2} fill="url(#inGrad)" />
              <Area type="monotone" dataKey="out" name="Outbound" stroke={seriesColor(chart, 1)} strokeWidth={2} fill="url(#outGrad)" />
            </AreaChart>
          </ResponsiveContainer>
        </div>

        {/* Channel sales */}
        <div className="bg-raised rounded-xl p-4 border border-line">
          <h3 className="text-sm font-semibold text-ink mb-1">Sales by channel</h3>
          <p className="text-xs text-ink-4 mb-4">This month</p>
          {channelData.length === 0 ? (
            <p className="text-xs text-ink-4">No sales yet this month.</p>
          ) : (
            <>
              <ResponsiveContainer width="100%" height={100}>
                <BarChart data={channelData} layout="vertical" margin={{ top: 0, right: 10, left: 0, bottom: 0 }}>
                  <XAxis type="number" hide />
                  <YAxis type="category" dataKey="label" tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} width={80} />
                  <Tooltip contentStyle={chart.tooltip} formatter={(v: number) => [formatMoney(v, currency), 'Revenue']} />
                  <Bar dataKey="total_minor_units" radius={4}>
                    {channelData.map(c => <Cell key={c.source} fill={c.color} />)}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
              <div className="mt-3 space-y-2">
                {channelData.map(c => (
                  <div key={c.source} className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <div className="w-2 h-2 rounded-full" style={{ background: c.color }} />
                      <span className="text-xs text-ink-3">{c.label}</span>
                    </div>
                    <span className="text-xs font-medium text-ink-2">{formatMoney(c.total_minor_units, currency)}</span>
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
        <div className="bg-raised rounded-xl border border-line overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b border-line">
            <h3 className="text-sm font-semibold text-ink">Top products</h3>
          </div>
          {(stats?.topProductsThisMonth.length ?? 0) === 0 ? (
            <p className="px-4 py-4 text-xs text-ink-4">No products sold yet this month.</p>
          ) : (
            <table className="w-full">
              <thead>
                <tr className="border-b border-line">
                  <th className="text-left text-2xs font-medium text-ink-4 uppercase tracking-wider px-4 py-2">Product</th>
                  <th className="text-right text-2xs font-medium text-ink-4 uppercase tracking-wider px-4 py-2">Sold</th>
                  <th className="text-right text-2xs font-medium text-ink-4 uppercase tracking-wider px-4 py-2">Revenue</th>
                </tr>
              </thead>
              <tbody>
                {stats?.topProductsThisMonth.map(p => (
                  <tr key={p.sku_id ?? p.sku_code} className="border-b dark:border-night-700/50 border-gray-50 last:border-0 hover:bg-hover/30 transition-colors">
                    <td className="px-4 py-2.5">
                      <p className="text-xs font-medium text-ink">{p.product_name}</p>
                      <p className="text-2xs text-ink-4 font-mono">{p.sku_code}</p>
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <span className="text-xs text-ink-2">{p.quantity_sold}</span>
                    </td>
                    <td className="px-4 py-2.5 text-right">
                      <span className="text-xs font-medium text-ink">{formatMoney(p.revenue_minor_units, currency)}</span>
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
