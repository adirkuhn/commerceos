import {
  Package, Warehouse, ShoppingCart, AlertTriangle,
  TrendingUp, TrendingDown, XCircle, DollarSign, Calendar,
} from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useExpiringBatches } from '../inventory/hooks/useExpiringBatches'
import { useDashboardStats } from '../dashboard/hooks/useDashboardStats'
import { ComingSoon } from '../dashboard/components/ComingSoon'

interface StatCardProps {
  label: string
  value?: string
  sub?: string
  icon: React.ElementType
  trend?: 'up' | 'down'
  trendValue?: string
  color?: string
  alert?: boolean
  comingSoon?: boolean
}

function StatCard({ label, value, sub, icon: Icon, trend, trendValue, color = 'blue', alert, comingSoon }: StatCardProps) {
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
      {comingSoon ? (
        <p className="text-sm italic dark:text-gray-500 text-gray-400 mt-1">Coming soon</p>
      ) : (
        <>
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
        </>
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

export default function Dashboard() {
  const { token } = useAuth()
  const { stats } = useDashboardStats(token ?? '')

  return (
    <div className="p-5 space-y-5 overflow-auto h-full">
      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-3">
        <StatCard label="Total Products" value={stats?.totalProducts.toLocaleString()} sub="across all channels" icon={Package} color="blue" />
        <StatCard label="Total Stock" icon={Warehouse} color="green" comingSoon />
        <StatCard label="Total Orders" value={stats?.totalOrders.toLocaleString()} sub="all time" icon={ShoppingCart} color="blue" />
        <StatCard label="Stock Issues" value={stats?.stockIssues.toLocaleString()} sub="orders need stock resolution" icon={ShoppingCart} color="amber" />
        <StatCard label="Expiring (7d)" value={stats?.expiringSoon.toLocaleString()} sub="batches near expiry" icon={AlertTriangle} color="amber" />
        <StatCard label="Expired" value={stats?.expired.toLocaleString()} sub="need disposal" icon={XCircle} color="red" alert />
        <StatCard label="Est. Revenue" icon={DollarSign} color="green" comingSoon />
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
          <ComingSoon label="Stock Flow" sub="Needs a tenant-wide movement aggregate" />
        </div>

        {/* Channel sales */}
        <div className="dark:bg-[#1F2937] bg-white rounded-xl p-4 border dark:border-[#2D3748] border-gray-200">
          <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900 mb-1">Sales by Channel</h3>
          <p className="text-xs dark:text-gray-500 text-gray-400 mb-4">This month</p>
          <ComingSoon label="Sales by Channel" sub="Needs a channel-revenue aggregate" height={140} />
        </div>
      </div>

      {/* Bottom row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        {/* Top products */}
        <div className="dark:bg-[#1F2937] bg-white rounded-xl border dark:border-[#2D3748] border-gray-200 overflow-hidden">
          <div className="flex items-center justify-between px-4 py-3 border-b dark:border-[#2D3748] border-gray-200">
            <h3 className="text-sm font-semibold dark:text-gray-100 text-gray-900">Top Products</h3>
          </div>
          <ComingSoon label="Top Products" sub="Needs a product-sales aggregate" className="m-4" height={140} />
        </div>

        <ExpiringSoonWidget />
      </div>
    </div>
  )
}
