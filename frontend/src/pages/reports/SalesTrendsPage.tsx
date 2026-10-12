import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import { seriesColor, useChartTheme } from '../../charts/chartTheme'
import clsx from 'clsx'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { SalesTrendGranularity } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'
import { useQueryEnum } from '../../routing/useQueryState'
import { useMoneyFormat } from '../../format/amountVisibility'
import { useDateFormat } from '../../format/DateFormatProvider'

const TREND_GRANULARITIES = ['day', 'week', 'month'] as const satisfies readonly SalesTrendGranularity[]

export default function SalesTrendsPage() {
  const { formatMonthDay, formatMonthYear } = useDateFormat()
  const chart = useChartTheme()
  const { token } = useAuth()
  const { t } = useTranslation('reports')
  const formatMoney = useMoneyFormat()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const [granularity, setGranularity] = useQueryEnum<SalesTrendGranularity>('granularity', TREND_GRANULARITIES, 'day')
  const getSalesTrendsReport = useCallback(
    (t: string, from: string, to: string) => reportsApi.getSalesTrendsReport(t, from, to, granularity),
    [granularity],
  )
  const { data: salesTrends, loading, error } = useSingleReport(token ?? '', period, getSalesTrendsReport)
  const currency = salesTrends?.currency_code ?? 'USD'
  const trendPoints = (salesTrends?.points ?? []).map(p => ({
    ...p,
    label: granularity === 'month' ? formatMonthYear(p.period_start, { shortYear: true }) : formatMonthDay(p.period_start),
  }))
  const weekdayLabels = [t('salesTrends.weekdays.sun'), t('salesTrends.weekdays.mon'), t('salesTrends.weekdays.tue'), t('salesTrends.weekdays.wed'), t('salesTrends.weekdays.thu'), t('salesTrends.weekdays.fri'), t('salesTrends.weekdays.sat')]
  const granularityLabel = { day: t('salesTrends.granularity.day'), week: t('salesTrends.granularity.week'), month: t('salesTrends.granularity.month') }
  const weekdaySales = [...(salesTrends?.by_weekday ?? [])].sort((a, b) => a.weekday - b.weekday)

  return (
    <ReportPageShell title={t('salesTrends.title')} period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <div className="grid grid-cols-2 gap-4">
        <ReportCard
          title={
            <div className="flex items-center justify-between">
              <span>{t('salesTrends.trend')}</span>
              <div className="flex items-center gap-0.5 bg-sunken border border-line rounded-lg p-0.5">
                {TREND_GRANULARITIES.map(g => (
                  <button aria-pressed={granularity === g}
                    key={g}
                    onClick={() => setGranularity(g)}
                    className={clsx(
                      'px-2 py-1 rounded-md text-2xs font-medium capitalize transition-colors',
                      granularity === g
                        ? 'bg-accent-solid text-white'
                        : 'text-ink-3 hover:text-ink-2',
                    )}
                  >
                    {granularityLabel[g]}
                  </button>
                ))}
              </div>
            </div>
          }
          description={t('salesTrends.trendDesc')}
        >
          {trendPoints.length === 0 ? (
            <EmptyState>{t('salesTrends.empty')}</EmptyState>
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <LineChart data={trendPoints} margin={{ top: 10, right: 16, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                <Tooltip contentStyle={chart.tooltip} formatter={(v: number) => [formatMoney(v, currency), t('col.revenue')]} />
                <Line type="monotone" dataKey="revenue_minor_units" stroke={seriesColor(chart, 0)} strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          )}
        </ReportCard>

        <ReportCard title={t('salesTrends.weekday')} description={t('salesTrends.weekdayDesc')}>
          {weekdaySales.length === 0 ? (
            <EmptyState>{t('salesTrends.weekdayEmpty')}</EmptyState>
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <BarChart data={weekdaySales.map(w => ({ ...w, label: weekdayLabels[w.weekday] }))} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
                <XAxis dataKey="label" tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                <Tooltip contentStyle={chart.tooltip} formatter={(v: number) => [formatMoney(v, currency), t('col.revenue')]} />
                <Bar dataKey="revenue_minor_units" fill={seriesColor(chart, 0)} radius={4} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </ReportCard>
      </div>
    </ReportPageShell>
  )
}
