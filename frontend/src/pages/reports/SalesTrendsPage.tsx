import { useState, useCallback } from 'react'
import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer } from 'recharts'
import clsx from 'clsx'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { ReportsPeriod } from '../../reports/period'
import type { SalesTrendGranularity } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState, formatMoney, TOOLTIP_STYLE } from './shared'

const TREND_GRANULARITIES: SalesTrendGranularity[] = ['day', 'week', 'month']
const WEEKDAY_LABELS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

export default function SalesTrendsPage() {
  const { token } = useAuth()
  const [period, setPeriod] = useState<ReportsPeriod>('7D')
  const [granularity, setGranularity] = useState<SalesTrendGranularity>('day')
  const getSalesTrendsReport = useCallback(
    (t: string, from: string, to: string) => reportsApi.getSalesTrendsReport(t, from, to, granularity),
    [granularity],
  )
  const { data: salesTrends, loading, error } = useSingleReport(token ?? '', period, getSalesTrendsReport)
  const currency = salesTrends?.currency_code ?? 'USD'
  const trendPoints = (salesTrends?.points ?? []).map(p => ({
    ...p,
    label: new Date(p.period_start).toLocaleDateString(undefined, granularity === 'month' ? { month: 'short', year: '2-digit' } : { month: 'short', day: 'numeric' }),
  }))
  const weekdaySales = [...(salesTrends?.by_weekday ?? [])].sort((a, b) => a.weekday - b.weekday)

  return (
    <ReportPageShell title="Sales Trends & Seasonality" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <div className="grid grid-cols-2 gap-4">
        <ReportCard
          title={
            <div className="flex items-center justify-between">
              <span>Sales Trend</span>
              <div className="flex items-center gap-0.5 dark:bg-[#111827] bg-gray-50 border dark:border-[#2D3748] border-gray-200 rounded-lg p-0.5">
                {TREND_GRANULARITIES.map(g => (
                  <button
                    key={g}
                    onClick={() => setGranularity(g)}
                    className={clsx(
                      'px-2 py-1 rounded-md text-[10px] font-medium capitalize transition-colors',
                      granularity === g
                        ? 'bg-blue-600 text-white'
                        : 'dark:text-gray-400 text-gray-500 hover:dark:text-gray-200 hover:text-gray-700',
                    )}
                  >
                    {g}
                  </button>
                ))}
              </div>
            </div>
          }
          description="Revenue over the selected period"
        >
          {trendPoints.length === 0 ? (
            <EmptyState>No sales in this period</EmptyState>
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <LineChart data={trendPoints} margin={{ top: 10, right: 16, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number) => [formatMoney(v, currency), 'Revenue']} />
                <Line type="monotone" dataKey="revenue_minor_units" stroke="#2563EB" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          )}
        </ReportCard>

        <ReportCard title="Seasonality by Weekday" description="Revenue summed across every occurrence of each weekday">
          {weekdaySales.length === 0 ? (
            <EmptyState>No sales to break down by weekday in this period</EmptyState>
          ) : (
            <ResponsiveContainer width="100%" height={260}>
              <BarChart data={weekdaySales.map(w => ({ ...w, label: WEEKDAY_LABELS[w.weekday] }))} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number) => [formatMoney(v, currency), 'Revenue']} />
                <Bar dataKey="revenue_minor_units" fill="#2563EB" radius={4} />
              </BarChart>
            </ResponsiveContainer>
          )}
        </ReportCard>
      </div>
    </ReportPageShell>
  )
}
