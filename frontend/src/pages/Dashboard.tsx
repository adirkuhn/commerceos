import type { ReactNode } from 'react'
import {
  AlertCircle,
  Package, Warehouse, ShoppingCart, AlertTriangle,
  TrendingUp, TrendingDown, XCircle, DollarSign,
} from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useDashboardStats } from '../dashboard/hooks/useDashboardStats'
import { ChannelSalesPanel, ExpiringSoonWidget, StockFlowPanel, TopProductsPanel } from '../dashboard/widgets'
import { Money } from '../format/amountVisibility'

interface StatCardProps {
  label: string
  value?: ReactNode
  sub?: string
  icon: React.ElementType
  trend?: 'up' | 'down'
  trendValue?: string
  color?: string
  alert?: boolean
}

function StatCard({ label, value, sub, icon: Icon, trend, trendValue, color = 'blue', alert }: StatCardProps) {
  const { t } = useTranslation('dashboard')
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
          <AlertCircle size={12} aria-hidden /> {t('needsAttention')}
        </p>
      )}
      {trend && trendValue && (
        <div className={clsx(
          'flex items-center gap-1 mt-2 text-xs font-medium',
          trend === 'up' ? 'text-success' : 'text-danger',
        )}>
          {trend === 'up' ? <TrendingUp size={12} /> : <TrendingDown size={12} />}
          {t('vsLastWeek', { value: trendValue })}
        </div>
      )}
    </div>
  )
}

export default function Dashboard() {
  const { token } = useAuth()
  const { stats } = useDashboardStats(token ?? '')
  const { t } = useTranslation(['dashboard', 'common'])
  const currency = stats?.currencyCode ?? 'USD'

  return (
    <div className="p-5 space-y-5 overflow-auto h-full">
      {/* Stats */}
      <div className="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-3">
        <StatCard label={t('stats.products')} value={stats?.totalProducts.toLocaleString()} sub={t('stats.productsSub')} icon={Package} color="blue" />
        <StatCard label={t('stats.stock')} value={stats?.totalStockUnits.toLocaleString()} sub={t('stats.stockSub')} icon={Warehouse} color="green" />
        <StatCard label={t('stats.orders')} value={stats?.totalOrders.toLocaleString()} sub={t('stats.ordersSub')} icon={ShoppingCart} color="blue" />
        <StatCard label={t('stats.stockIssues')} value={stats?.stockIssues.toLocaleString()} sub={t('stats.stockIssuesSub')} icon={ShoppingCart} color="amber" />
        <StatCard label={t('stats.expiring')} value={stats?.expiringSoon.toLocaleString()} sub={t('stats.expiringSub')} icon={AlertTriangle} color="amber" />
        <StatCard label={t('stats.expired')} value={stats?.expired.toLocaleString()} sub={t('stats.expiredSub')} icon={XCircle} color="red" alert={(stats?.expired ?? 0) > 0} />
        <StatCard
          label={t('stats.revenue')}
          value={stats ? <Money minor={stats.revenueThisMonthMinorUnits} currency={currency} /> : undefined}
          sub={t('stats.revenueSub')}
          icon={DollarSign}
          color="green"
        />
      </div>

      {/* Charts row */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <StockFlowPanel flow={stats?.stockFlowLast7Days ?? []} className="lg:col-span-2" />
        <ChannelSalesPanel channels={stats?.salesByChannelThisMonth ?? []} currency={currency} />
      </div>

      {/* Bottom row */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <TopProductsPanel products={stats?.topProductsThisMonth ?? []} currency={currency} />
        <ExpiringSoonWidget />
      </div>
    </div>
  )
}
