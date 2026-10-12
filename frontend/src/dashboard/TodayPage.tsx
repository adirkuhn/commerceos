import type { ElementType } from 'react'
import { useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { AlertTriangle, CalendarClock, CalendarX, CheckCircle2, PackageCheck, PackageSearch, ScanBarcode } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { useDateFormat } from '../format/DateFormatProvider'
import { Money } from '../format/amountVisibility'
import { inventoryPaths } from '../inventory/inventoryRoutes'
import { navItemToPath } from '../routes'
import { useToday } from './hooks/useToday'
import type { TodayQueue } from './types'
import { ChannelSalesPanel, StockFlowPanel, TopProductsPanel } from './widgets'

// The depot design's home: what the shop has to do now. A dispatch board
// with the two daily queues, then only the problems that exist right now
// -- each with the action that clears it -- then this month's figures.

function todayHeading(language: string, timeZone: string | undefined): string {
  const options: Intl.DateTimeFormatOptions = { weekday: 'long', day: 'numeric', month: 'long' }
  try {
    return new Intl.DateTimeFormat(language, { ...options, timeZone }).format(new Date())
  } catch {
    return new Intl.DateTimeFormat(language, options).format(new Date())
  }
}

interface QueueCounterProps {
  count: number | undefined
  label: string
  action: string
  icon: ElementType
  onStart: () => void
}

function QueueCounter({ count, label, action, icon: Icon, onStart }: QueueCounterProps) {
  const idle = count === 0
  return (
    <div className="flex items-end justify-between gap-4 px-6 py-5">
      <div>
        <p className="font-data text-6xl font-semibold leading-none tabular-nums text-depot-ink">{count ?? '–'}</p>
        <p className="mt-2 text-sm text-depot-ink-2">{label}</p>
      </div>
      <button
        onClick={onStart}
        className={clsx(
          'shrink-0 inline-flex items-center gap-2 px-3.5 py-2 rounded-lg text-sm font-medium transition-colors focus-visible:outline-depot-mark',
          idle ? 'text-depot-ink-2 hover:bg-depot-hover/10' : 'bg-white text-depot hover:bg-depot-ink',
        )}
      >
        <Icon size={16} aria-hidden />
        {action}
      </button>
    </div>
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
      <h2 className="font-data text-3xl font-semibold tracking-tight text-ink first-letter:uppercase">
        {todayHeading(i18n.language, settings.timeZone)}
      </h2>

      {error && <p role="alert" className="text-sm text-danger">{error}</p>}

      <section aria-label={t('today.workQueue')} className="rounded-xl bg-depot grid grid-cols-1 md:grid-cols-2 md:divide-x divide-depot-hover/10">
        <QueueCounter
          count={queue?.to_pick}
          label={t('today.toPick', { count: queue?.to_pick ?? 0 })}
          action={t('today.startPicking')}
          icon={ScanBarcode}
          onStart={() => navigate(navItemToPath('picking'))}
        />
        <QueueCounter
          count={queue?.to_pack}
          label={t('today.toPack', { count: queue?.to_pack ?? 0 })}
          action={t('today.startPacking')}
          icon={PackageCheck}
          onStart={() => navigate(navItemToPath('packing'))}
        />
      </section>

      <section aria-labelledby="today-attention" className="bg-raised rounded-xl border border-line">
        <h3 id="today-attention" className="px-5 pt-4 pb-2 text-sm font-semibold text-ink">{t('today.needsAttention')}</h3>
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
                    className="shrink-0 px-3 py-1.5 rounded-lg text-xs font-medium text-accent hover:bg-accent/10 transition-colors"
                  >
                    {item.action}
                  </button>
                </li>
              )
            })}
          </ul>
        )}
      </section>

      <section aria-labelledby="today-month" className="space-y-4">
        <h3 id="today-month" className="text-sm font-semibold text-ink">{t('today.thisMonth')}</h3>
        <div className="grid grid-cols-1 lg:grid-cols-3 gap-4 items-start">
          <div className="bg-raised rounded-xl p-4 border border-line">
            <p className="text-xs text-ink-3">{t('today.revenue')}</p>
            <p className="mt-2 font-data text-3xl font-semibold text-ink">
              {report ? <Money minor={report.revenue_this_month_minor_units} currency={currency} /> : '–'}
            </p>
            <p className="mt-1 text-xs text-ink-4">{t('today.revenueSub')}</p>
          </div>
          <ChannelSalesPanel channels={report?.sales_by_channel_this_month ?? []} currency={currency} className="lg:col-span-2" />
          <TopProductsPanel products={report?.top_products_this_month ?? []} currency={currency} className="lg:col-span-2" />
          <StockFlowPanel flow={report?.stock_flow_last_7_days ?? []} />
        </div>
      </section>
    </div>
  )
}
