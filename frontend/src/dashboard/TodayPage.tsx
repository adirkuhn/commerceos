import type { ElementType } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, CalendarClock, CalendarX, CheckCircle2, PackageCheck, PackageSearch, ScanBarcode, Truck } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useDateFormat } from '../format/DateFormatProvider'
import { formatLongDay } from '../format/date'
import { inventoryPaths } from '../inventory/inventoryRoutes'
import { navItemToPath } from '../routes'
import { useToday } from './hooks/useToday'
import type { TodayQueue } from './types'
import { SalesPeriodCard } from './SalesPeriodCard'
import { StockFlowPanel, TopProductsPanel } from './widgets'
import { Coupon, CouponHead } from '../components/common/Coupon'

// Home ("/"): what the shop has to do now. A dispatch board
// with the three daily queues (pick, pack, ship today), then only the problems that exist right now
// -- each with the action that clears it -- then sales (today, this week or
// this month against the previous one) and this month's figures.

interface QueueCounterProps {
  count: number | undefined
  stub: string
  label: string
  action: string
  icon: ElementType
  onStart: () => void
}

// One daily queue as a coupon: the count on the stub, what it is and the
// action that works it. An empty queue waits in outline.
function QueueCounter({ count, stub, label, action, icon: Icon, onStart }: QueueCounterProps) {
  const idle = !count
  return (
    <Coupon tone={idle ? 'waiting' : 'plain'} stubLabel={stub} stubValue={count ?? '–'} stubWidth="wide" className="min-h-[6.5rem]">
      <CouponHead rule={idle ? 'none' : 'mark'} muted={idle}>{label}</CouponHead>
      <button
        onClick={onStart}
        className={clsx(
          'self-start inline-flex items-center gap-2 h-8 px-3 rounded-md text-xs font-semibold transition-colors',
          idle ? 'border border-line text-ink-3 hover:bg-hover' : 'bg-accent-solid text-white hover:bg-accent-solid-hover',
        )}
      >
        <Icon size={15} aria-hidden />
        {action}
      </button>
    </Coupon>
  )
}

interface AttentionItem {
  key: string
  text: string
  action: string
  icon: ElementType
  tone: 'warning' | 'danger'
  to: string
}

function attentionItems(q: TodayQueue, t: (key: string, opts?: Record<string, unknown>) => string): AttentionItem[] {
  const items: AttentionItem[] = []
  if (q.expired_batches > 0) items.push({
    key: 'expired', text: t('today.expired', { count: q.expired_batches }), action: t('today.writeOffExpired'),
    icon: CalendarX, tone: 'danger', to: `${inventoryPaths.expiry}?tab=expired`,
  })
  if (q.stock_issues > 0) items.push({
    key: 'stock', text: t('today.stockIssues', { count: q.stock_issues }), action: t('today.resolveStock'),
    icon: AlertTriangle, tone: 'danger', to: navItemToPath('stock-issues'),
  })
  if (q.expiring_batches > 0) items.push({
    key: 'expiring', text: t('today.expiring', { count: q.expiring_batches }), action: t('today.reviewExpiring'),
    icon: CalendarClock, tone: 'warning', to: `${inventoryPaths.expiry}?tab=expiring`,
  })
  if (q.unmatched_skus > 0) items.push({
    key: 'unmatched', text: t('today.unmatched', { count: q.unmatched_skus }), action: t('today.matchSkus'),
    icon: PackageSearch, tone: 'warning', to: navItemToPath('unmatched-skus'),
  })
  return items
}

export default function TodayPage() {
  const { token } = useAuth()
  const { queue, report, error } = useToday(token ?? '')
  const { t, i18n } = useTranslation('dashboard')
  const { settings } = useDateFormat()
  const navigate = useNavigate()
  const currency = report?.currency_code ?? 'USD'
  const items = queue ? attentionItems(queue, t as (key: string, opts?: Record<string, unknown>) => string) : []

  return (
    <div className="p-6 space-y-6 max-w-6xl">
      <h2 className="text-3xl font-bold tracking-tight text-ink first-letter:uppercase">
        {formatLongDay(new Date(), i18n.language, settings)}
      </h2>

      {error && <p role="alert" className="text-sm text-danger">{error}</p>}

      <section aria-label={t('today.workQueue')} className="grid grid-cols-1 md:grid-cols-3 gap-4">
        <QueueCounter
          count={queue?.to_pick}
          stub={t('today.stubPick')}
          label={t('today.toPick', { count: queue?.to_pick ?? 0 })}
          action={t('today.startPicking')}
          icon={ScanBarcode}
          onStart={() => navigate(navItemToPath('picking'))}
        />
        <QueueCounter
          count={queue?.to_pack}
          stub={t('today.stubPack')}
          label={t('today.toPack', { count: queue?.to_pack ?? 0 })}
          action={t('today.startPacking')}
          icon={PackageCheck}
          onStart={() => navigate(navItemToPath('packing'))}
        />
        <QueueCounter
          count={queue?.ship_today}
          stub={t('today.stubShip')}
          label={t('today.toShip', { count: queue?.ship_today ?? 0 })}
          action={t('today.viewToShip')}
          icon={Truck}
          onStart={() => navigate(`${navItemToPath('sales-orders')}?ship_due=true`)}
        />
      </section>

      <section aria-labelledby="today-attention" className="bg-raised rounded-lg border border-line shadow-panel">
        <h3 id="today-attention" className="px-5 pt-4 pb-2 caps-label-widest text-2xs font-bold text-ink">{t('today.needsAttention')}</h3>
        {queue && items.length === 0 && (
          <p className="flex items-center gap-2 px-5 pb-4 text-sm text-ink-3">
            <CheckCircle2 size={16} className="text-success" aria-hidden />
            {t('today.allClear')}
          </p>
        )}
        {items.length > 0 && (
          <ul className="divide-y divide-line">
            {items.map(item => {
              const Icon = item.icon
              return (
                <li key={item.key} className="flex items-center gap-3 px-5 py-3">
                  <Icon size={16} className={clsx('shrink-0', item.tone === 'danger' ? 'text-danger' : 'text-warning')} aria-hidden />
                  <span className="flex-1 text-sm text-ink">{item.text}</span>
                  <button
                    onClick={() => navigate(item.to)}
                    className="shrink-0 h-8 px-3 rounded-md border border-line text-xs font-semibold text-ink-2 hover:bg-hover hover:text-ink transition-colors"
                  >
                    {item.action}
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </section>

      <SalesPeriodCard />

      <section aria-labelledby="today-month" className="space-y-4">
        <h3 id="today-month" className="caps-label-widest text-2xs font-bold text-ink">{t('today.thisMonth')}</h3>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 items-start">
          <TopProductsPanel products={report?.top_products_this_month ?? []} currency={currency} className="lg:col-span-2" />
          <StockFlowPanel flow={report?.stock_flow_last_7_days ?? []} />
        </div>
      </section>
    </div>
  )
}
