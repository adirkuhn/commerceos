import { useState, useCallback, useMemo } from 'react'
import { AlertTriangle, Info } from 'lucide-react'
import clsx from 'clsx'
import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, LabelList, ResponsiveContainer } from 'recharts'
import { channelSeriesColor, seriesColor, useChartTheme } from '../../charts/chartTheme'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useReport } from '../../reports/hooks/useSingleReport'
import { dateRange, customDateRange, REPORTS_PERIODS, DEFAULT_REPORTS_PERIOD, type ReportsPeriod } from '../../reports/period'
import { useQueryEnum, useQueryString } from '../../routing/useQueryState'
import type { ChannelRevenuePoint, ProfitabilityPoint } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState, DateRangeControl, ViewToggle, ChangeBadge, type ReportsRangeMode } from './shared'
import { formatMoney } from '../../format/money'
import type { DateInput } from '../../format/date'
import { useDateFormat } from '../../format/DateFormatProvider'

function isoDateToday(): string {
  return new Date().toISOString().slice(0, 10)
}

function isoDateDaysAgo(days: number): string {
  return new Date(Date.now() - days * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
}

const RANGE_MODES = [...REPORTS_PERIODS, 'custom'] as const satisfies readonly ReportsRangeMode[]
const OVERVIEW_VIEWS = ['summary', 'daily'] as const

function isIsoDate(value: string | null): value is string {
  return value !== null && /^\d{4}-\d{2}-\d{2}$/.test(value) && !Number.isNaN(new Date(`${value}T00:00:00Z`).getTime())
}

export default function OverviewPage() {
  const chart = useChartTheme()
  const { token } = useAuth()
  const { formatDate, formatMonthDay } = useDateFormat()
  // Range (?period=, plus ?from=&to= for a custom range) and the
  // summary/daily view (?view=daily) live in the URL -- a refresh or a
  // shared link shows the same report. Malformed dates fall back to the
  // default custom range.
  const [mode, setMode] = useQueryEnum<ReportsRangeMode>('period', RANGE_MODES, DEFAULT_REPORTS_PERIOD)
  const [fromParam, setCustomFrom] = useQueryString('from')
  const [toParam, setCustomTo] = useQueryString('to')
  const customFrom = isIsoDate(fromParam) ? fromParam : isoDateDaysAgo(7)
  const customTo = isIsoDate(toParam) ? toParam : isoDateToday()
  const [view, setView] = useQueryEnum('view', OVERVIEW_VIEWS, 'summary')
  const showDaily = view === 'daily'
  const setShowDaily = (daily: boolean) => setView(daily ? 'daily' : 'summary')
  const [hiddenChannels, setHiddenChannels] = useState<Set<string>>(new Set())

  const toggleChannel = useCallback((source: string) => {
    setHiddenChannels(prev => {
      const next = new Set(prev)
      if (next.has(source)) next.delete(source)
      else next.add(source)
      return next
    })
  }, [])

  const { from, to } = useMemo(() => {
    if (mode === 'custom') return customDateRange(customFrom, customTo)
    return dateRange(mode as ReportsPeriod, new Date())
  }, [mode, customFrom, customTo])

  const getSalesSummaryReport = useCallback(reportsApi.getSalesSummaryReport, [])
  const getBusinessInsights = useCallback(reportsApi.getBusinessInsights, [])
  const getProfitabilityReport = useCallback(reportsApi.getProfitabilityReport, [])
  const getProfitabilityTrendReport = useCallback(
    (t: string, f: string, to2: string) => reportsApi.getProfitabilityTrendReport(t, f, to2, 'day'),
    [],
  )
  const getRevenueByChannelReport = useCallback(
    (t: string, f: string, to2: string) => reportsApi.getRevenueByChannelReport(t, f, to2, 'day'),
    [],
  )

  const { data: salesSummary, loading: summaryLoading, error: summaryError } = useReport(token ?? '', from, to, getSalesSummaryReport)
  const { data: businessInsights, loading: insightsLoading, error: insightsError } = useReport(token ?? '', from, to, getBusinessInsights)
  const { data: profitability, loading: profitLoading, error: profitError } = useReport(token ?? '', from, to, getProfitabilityReport)
  const { data: profitabilityTrend, loading: profitTrendLoading, error: profitTrendError } = useReport(token ?? '', from, to, getProfitabilityTrendReport)
  const { data: revenueByChannel, loading: channelLoading, error: channelError } = useReport(token ?? '', from, to, getRevenueByChannelReport)

  const currency = salesSummary?.currency_code ?? 'USD'
  const kpis = [
    { label: 'Total revenue', value: salesSummary ? formatMoney(salesSummary.current.revenue_minor_units, currency) : '—', change: salesSummary?.revenue_change_percent ?? null },
    { label: 'Total orders', value: salesSummary ? String(salesSummary.current.order_count) : '—', change: salesSummary?.order_count_change_percent ?? null },
    { label: 'Units sold', value: salesSummary ? String(salesSummary.current.units_sold) : '—', change: salesSummary?.units_sold_change_percent ?? null },
    { label: 'Avg order value', value: salesSummary ? formatMoney(salesSummary.current.average_order_value_minor_units, currency) : '—', change: salesSummary?.aov_change_percent ?? null },
  ]

  const profitabilityTrendPoints = (profitabilityTrend?.points ?? []).map((p: ProfitabilityPoint) => ({ ...p, label: formatMonthDay(p.period_start) }))

  const channelSources = Array.from(new Set((revenueByChannel?.points ?? []).map(p => p.source)))
  const visibleChannelSources = channelSources.filter(s => !hiddenChannels.has(s))
  const channelTrendRows = buildChannelTrendRows(revenueByChannel?.points ?? [], channelSources, formatMonthDay)
  // dayTotal sums only the currently visible channels -- it matches the bar
  // height on screen, not necessarily every channel that exists.
  const channelTrendRowsWithTotal = channelTrendRows.map(row => ({
    ...row,
    dayTotal: visibleChannelSources.reduce((sum, source) => sum + Number(row[source] ?? 0), 0),
  }))
  const channelTotalBySource = Object.fromEntries((revenueByChannel?.channels ?? []).map(c => [c.source, c.total_minor_units]))
  const totalAllChannelsMinorUnits = (revenueByChannel?.channels ?? []).reduce((sum, c) => sum + c.total_minor_units, 0)

  return (
    <ReportPageShell
      title="Overview"
      subtitle={`${formatDate(from)} – ${formatDate(to)}`}
      loading={summaryLoading || insightsLoading || profitLoading || channelLoading || (showDaily && profitTrendLoading)}
      error={summaryError ?? insightsError ?? profitError ?? channelError ?? (showDaily ? profitTrendError : null)}
    >
      <div className="flex items-center justify-between gap-3 flex-wrap">
        <DateRangeControl
          mode={mode}
          onModeChange={setMode}
          customFrom={customFrom}
          customTo={customTo}
          onCustomFromChange={setCustomFrom}
          onCustomToChange={setCustomTo}
        />
        <ViewToggle showDaily={showDaily} onChange={setShowDaily} />
      </div>

      <div className="grid grid-cols-4 gap-3">
        {kpis.map(kpi => (
          <div key={kpi.label} className="bg-raised rounded-xl p-4 border border-line">
            <p className="text-xs text-ink-4 mb-2">{kpi.label}</p>
            <p className="text-2xl font-semibold text-ink mb-1">{kpi.value}</p>
            <ChangeBadge percent={kpi.change} />
          </div>
        ))}
      </div>

      {businessInsights && businessInsights.insights.length > 0 && (
        <div data-testid="business-insights" className="space-y-2">
          {businessInsights.insights.map(insight => (
            <div
              key={insight.code}
              className={clsx(
                'flex items-start gap-2 rounded-lg px-3 py-2 text-xs',
                insight.severity === 'warning'
                  ? 'bg-amber-50 dark:bg-amber-500/10 text-warning'
                  : 'bg-accent/10 text-accent',
              )}
            >
              {insight.severity === 'warning' ? <AlertTriangle size={14} className="shrink-0 mt-0.5" /> : <Info size={14} className="shrink-0 mt-0.5" />}
              <span>{insight.message}</span>
            </div>
          ))}
        </div>
      )}

      <ReportCard title="Profitability" description="Profit/margin cover only the revenue with a known SKU cost price">
        {showDaily ? (
          profitabilityTrendPoints.length === 0 ? (
            <EmptyState>No cost-known revenue in this period</EmptyState>
          ) : (
            <div className="px-3 pb-3 pt-1">
              <ResponsiveContainer width="100%" height={260}>
                <LineChart data={profitabilityTrendPoints} margin={{ top: 10, right: 16, left: 0, bottom: 8 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
                  <XAxis dataKey="label" tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} tickMargin={8} />
                  <YAxis width={72} tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                  <Tooltip contentStyle={chart.tooltip} formatter={(v: number, name: string) => [formatMoney(v, currency), name]} />
                  <Legend wrapperStyle={{ fontSize: 11, paddingTop: 12, color: chart.label }} />
                  <Line type="monotone" dataKey="revenue_minor_units" name="Revenue" stroke={seriesColor(chart, 0)} strokeWidth={2} dot={false} />
                  <Line type="monotone" dataKey="profit_minor_units" name="Profit" stroke={seriesColor(chart, 1)} strokeWidth={2} dot={false} />
                </LineChart>
              </ResponsiveContainer>
            </div>
          )
        ) : !profitability || profitability.summary.cost_known_revenue_minor_units === 0 ? (
          <EmptyState>No cost-known revenue in this period</EmptyState>
        ) : (
          <div data-testid="profitability-summary" className="grid grid-cols-4 gap-3 p-4">
            <div>
              <p className="text-xs text-ink-4 mb-1">Revenue</p>
              <p className="text-sm font-semibold text-ink">{formatMoney(profitability.summary.revenue_minor_units, currency)}</p>
              {profitability.summary.cost_known_revenue_minor_units < profitability.summary.revenue_minor_units && (
                <p className="text-2xs text-ink-4 mt-0.5">
                  {formatMoney(profitability.summary.cost_known_revenue_minor_units, currency)} with known cost
                </p>
              )}
            </div>
            <div>
              <p className="text-xs text-ink-4 mb-1">COGS</p>
              <p className="text-sm font-semibold text-ink">{formatMoney(profitability.summary.cost_minor_units, currency)}</p>
            </div>
            <div>
              <p className="text-xs text-ink-4 mb-1">Profit</p>
              <p className="text-sm font-semibold text-ink">{formatMoney(profitability.summary.profit_minor_units, currency)}</p>
            </div>
            <div>
              <p className="text-xs text-ink-4 mb-1">Margin</p>
              <p className="text-sm font-semibold text-ink">{(profitability.summary.margin_basis_points / 100).toFixed(1)}%</p>
            </div>
          </div>
        )}
      </ReportCard>

      <ReportCard title="Revenue by channel" description={showDaily ? 'Daily revenue per sales channel' : 'Total revenue per sales channel'}>
        {showDaily ? (
          channelTrendRows.length === 0 ? (
            <EmptyState>No revenue in this period</EmptyState>
          ) : (
            <>
              <div className="flex items-center justify-between px-4 pt-3">
                <p data-testid="channel-total-all" className="text-xs text-ink-3">
                  Total, all channels: <span className="font-semibold text-ink">{formatMoney(totalAllChannelsMinorUnits, currency)}</span>
                </p>
              </div>
              <div className="flex flex-wrap gap-1.5 px-4 pt-2">
                {channelSources.map(source => {
                  const visible = !hiddenChannels.has(source)
                  const color = channelSeriesColor(chart, source)
                  return (
                    <button
                      key={source}
                      onClick={() => toggleChannel(source)}
                      aria-pressed={visible}
                      className={clsx(
                        'flex items-center gap-1.5 px-2 py-1 rounded-md text-2xs font-medium border transition-colors',
                        visible ? 'border-line bg-raised text-ink-2' : 'border-line text-ink-4 opacity-60',
                      )}
                      title={visible ? `Hide ${source}` : `Show ${source}`}
                    >
                      <span className="w-2.5 h-2.5 rounded-sm shrink-0" style={{ backgroundColor: visible ? color : 'transparent', border: `1.5px solid ${color}` }} aria-hidden />
                      {source} · {formatMoney(channelTotalBySource[source] ?? 0, currency)}
                    </button>
                  )
                })}
              </div>
              {visibleChannelSources.length === 0 ? (
                <EmptyState>Select at least one channel to show</EmptyState>
              ) : (
                <div className="px-3 pb-3 pt-1">
                  <ResponsiveContainer width="100%" height={280}>
                    <BarChart data={channelTrendRowsWithTotal} margin={{ top: 28, right: 16, left: 0, bottom: 8 }}>
                      <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
                      <XAxis dataKey="label" tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} tickMargin={8} />
                      <YAxis width={72} tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                      <Tooltip contentStyle={chart.tooltip} formatter={(v: number, name: string) => [formatMoney(v, currency), name === 'dayTotal' ? 'Total' : name]} />
                      <Legend wrapperStyle={{ fontSize: 11, paddingTop: 12, color: chart.label }} />
                      {visibleChannelSources.map((source, i) => (
                        <Bar
                          key={source} dataKey={source} name={source} stackId="revenue"
                          fill={channelSeriesColor(chart, source)} stroke={chart.surface} strokeWidth={1}
                        >
                          {i === visibleChannelSources.length - 1 && (
                            <LabelList dataKey="dayTotal" position="top" formatter={(v: number) => formatMoney(v, currency)} style={{ fontSize: 11, fill: chart.label }} />
                          )}
                        </Bar>
                      ))}
                    </BarChart>
                  </ResponsiveContainer>
                </div>
              )}
            </>
          )
        ) : !revenueByChannel || revenueByChannel.channels.length === 0 ? (
          <EmptyState>No revenue in this period</EmptyState>
        ) : (
          <>
            <div className="px-3 pb-3 pt-1">
              <ResponsiveContainer width="100%" height={200}>
                <BarChart data={revenueByChannel.channels} margin={{ top: 10, right: 10, left: 0, bottom: 4 }}>
                  <CartesianGrid strokeDasharray="3 3" stroke={chart.grid} vertical={false} />
                  <XAxis dataKey="source" tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} tickMargin={8} />
                  <YAxis width={72} tick={{ fontSize: 11, fill: chart.axis }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                  <Tooltip contentStyle={chart.tooltip} formatter={(v: number) => [formatMoney(v, currency), 'Revenue']} />
                  <Bar dataKey="total_minor_units" fill={seriesColor(chart, 0)} radius={4} />
                </BarChart>
              </ResponsiveContainer>
            </div>
            <table className="w-full text-xs border-t border-line">
              <thead>
                <tr className="dark:bg-night-900/50 bg-gray-50">
                  {['Channel', 'Revenue'].map(h => (
                    <th key={h} className="px-3 py-2 text-left text-2xs font-medium text-ink-4 uppercase">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-line/50">
                {revenueByChannel.channels.map(c => (
                  <tr key={c.source} className="hover:bg-hover/30">
                    <td className="px-3 py-2 text-ink-2 font-medium">{c.source}</td>
                    <td className="px-3 py-2 text-ink font-medium">{formatMoney(c.total_minor_units, currency)}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}

// buildChannelTrendRows pivots the per-(period, source) point list into one
// row per period with each source's revenue as its own column, so recharts
// can render one <Line> per channel against a shared X axis. A source
// missing a bucket (no sales that day) defaults to 0, not undefined --
// undefined would break the line rather than dip to zero.
function buildChannelTrendRows(
  points: ChannelRevenuePoint[], sources: string[], dayLabel: (value: DateInput) => string,
): Array<Record<string, string | number>> {
  const byPeriod = new Map<string, Record<string, string | number>>()
  for (const p of points) {
    let row = byPeriod.get(p.period_start)
    if (!row) {
      row = { label: dayLabel(p.period_start), periodStart: p.period_start }
      for (const source of sources) row[source] = 0
      byPeriod.set(p.period_start, row)
    }
    row[p.source] = p.revenue_minor_units
  }
  return Array.from(byPeriod.values()).sort((a, b) => String(a.periodStart).localeCompare(String(b.periodStart)))
}
