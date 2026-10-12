import { Minus, TrendingDown, TrendingUp } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { Money } from '../format/amountVisibility'
import { useQueryEnum } from '../routing/useQueryState'
import { channelSeriesColor, useChartTheme } from '../charts/chartTheme'
import { useSalesComparison } from './hooks/useSalesComparison'
import type { SalesComparison, SalesPeriod } from './types'
import { channelLabel } from './widgets'

// The dashboard's sales card, in both designs: revenue for today, this
// week or this month so far, against the whole previous period, overall
// and per channel. The period lives in ?sales= so a refresh keeps it.

const PERIOD_PARAMS = ['today', 'week', 'month'] as const
type PeriodParam = typeof PERIOD_PARAMS[number]

const PERIOD_OF: Record<PeriodParam, SalesPeriod> = { today: 'today', week: 'this_week', month: 'this_month' }

interface ChannelRow {
  source: string
  current: number
  previous: number
}

// Every channel that sold in either period -- one that stopped selling
// still shows, at 0 -- busiest now first.
function channelRows(c: SalesComparison): ChannelRow[] {
  const rows = new Map<string, ChannelRow>()
  for (const ch of c.current.by_channel) rows.set(ch.source, { source: ch.source, current: ch.total_minor_units, previous: 0 })
  for (const ch of c.previous.by_channel) {
    const row = rows.get(ch.source) ?? { source: ch.source, current: 0, previous: 0 }
    rows.set(ch.source, { ...row, previous: ch.total_minor_units })
  }
  return [...rows.values()].sort((a, b) => b.current - a.current || b.previous - a.previous)
}

function Change({ percent, period }: { percent: number | null; period: SalesPeriod }) {
  const { t, i18n } = useTranslation('dashboard')
  if (percent === null) {
    return <p className="mt-2 text-xs text-ink-4">{t(`sales.noPrevious.${period}`)}</p>
  }
  const rounded = Math.round(percent)
  const Icon = rounded > 0 ? TrendingUp : rounded < 0 ? TrendingDown : Minus
  const value = new Intl.NumberFormat(i18n.language, { style: 'percent', signDisplay: 'exceptZero', maximumFractionDigits: 0 }).format(rounded / 100)
  return (
    <p className={clsx(
      'mt-2 flex items-center gap-1 text-sm font-medium',
      rounded > 0 ? 'text-success' : rounded < 0 ? 'text-danger' : 'text-ink-3',
    )}>
      <Icon size={14} aria-hidden />
      {t(`sales.vs.${period}`, { value })}
    </p>
  )
}

function Channels({ comparison }: { comparison: SalesComparison }) {
  const { t } = useTranslation('dashboard')
  const chart = useChartTheme()
  const rows = channelRows(comparison)
  const { period, currency_code: currency } = comparison
  if (rows.length === 0) return <p className="text-xs text-ink-4">{t(`sales.empty.${period}`)}</p>
  const max = Math.max(...rows.map(r => r.current), 1)
  return (
    <ul className="space-y-3">
      {rows.map(r => {
        const label = channelLabel(r.source)
        return (
          <li key={r.source} aria-label={label}>
            <div className="flex items-center justify-between gap-3">
              <span className="flex items-center gap-2 text-xs text-ink-2">
                <span className="w-2 h-2 rounded-full" style={{ background: channelSeriesColor(chart, r.source) }} aria-hidden />
                {label}
              </span>
              <Money minor={r.current} currency={currency} className="text-xs font-medium text-ink tabular-nums" />
            </div>
            <div className="mt-1.5 h-1.5 rounded-full bg-hover overflow-hidden" aria-hidden>
              <div className="h-full rounded-full" style={{ width: `${(r.current / max) * 100}%`, background: channelSeriesColor(chart, r.source) }} />
            </div>
            <p className="mt-1 text-2xs text-ink-4">
              {t(`sales.previous.${period}`)}: <Money minor={r.previous} currency={currency} />
            </p>
          </li>
        )
      })}
    </ul>
  )
}

export function SalesPeriodCard({ className }: { className?: string }) {
  const { token } = useAuth()
  const { t } = useTranslation('dashboard')
  const [param, setParam] = useQueryEnum<PeriodParam>('sales', PERIOD_PARAMS, 'month')
  const period = PERIOD_OF[param]
  const { data, error } = useSalesComparison(token ?? '', period)
  const shown = data?.period === period ? data : null

  return (
    <section aria-labelledby="sales-period-title" className={clsx('bg-raised rounded-xl p-4 border border-line', className)}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h3 id="sales-period-title" className="text-sm font-semibold text-ink">{t('sales.title')}</h3>
        <div role="group" aria-label={t('sales.periodLabel')} className="inline-flex rounded-lg border border-line p-0.5">
          {PERIOD_PARAMS.map(p => (
            <button
              key={p}
              type="button"
              aria-pressed={param === p}
              onClick={() => setParam(p)}
              className={clsx(
                'px-3 py-1 rounded-md text-xs font-medium transition-colors outline-none focus-visible:ring-2 focus-visible:ring-focus/60',
                param === p ? 'bg-accent-solid text-white' : 'text-ink-3 hover:text-ink hover:bg-hover',
              )}
            >
              {t(`sales.period.${PERIOD_OF[p]}`)}
            </button>
          ))}
        </div>
      </div>

      {error && <p role="alert" className="mt-3 text-sm text-danger">{error}</p>}

      <div className="mt-4 grid grid-cols-1 md:grid-cols-[minmax(0,1fr)_minmax(0,1.4fr)] gap-6">
        <div>
          <p className="font-data text-3xl font-semibold text-ink tabular-nums">
            {shown ? <Money minor={shown.current.revenue_minor_units} currency={shown.currency_code} /> : '–'}
          </p>
          {shown && (
            <>
              <Change percent={shown.change_percent} period={period} />
              <p className="mt-3 text-xs text-ink-3">{t('sales.orders', { count: shown.current.order_count })}</p>
              <p className="mt-1 text-xs text-ink-4">
                {t(`sales.previous.${period}`)}: <Money minor={shown.previous.revenue_minor_units} currency={shown.currency_code} />
              </p>
            </>
          )}
          <p className="mt-3 text-2xs text-ink-4">{t('sales.excludes')}</p>
        </div>
        <div>
          <h4 className="text-xs font-medium text-ink-3 mb-3">{t('sales.byChannel')}</h4>
          {shown && <Channels comparison={shown} />}
        </div>
      </div>
    </section>
  )
}
