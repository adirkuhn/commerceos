import { useState, useCallback, useMemo } from 'react'
import { AlertTriangle, Info } from 'lucide-react'
import clsx from 'clsx'
import { BarChart, Bar, LineChart, Line, XAxis, YAxis, CartesianGrid, Tooltip, Legend, ResponsiveContainer } from 'recharts'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useReport } from '../../reports/hooks/useSingleReport'
import { dateRange, customDateRange, type ReportsPeriod } from '../../reports/period'
import type { ChannelRevenuePoint, ProfitabilityPoint } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState, DateRangeControl, ViewToggle, formatMoney, ChangeBadge, TOOLTIP_STYLE, type ReportsRangeMode } from './shared'

const CHANNEL_COLORS = ['#2563EB', '#16A34A', '#D97706', '#DC2626', '#7C3AED']

function isoDateToday(): string {
  return new Date().toISOString().slice(0, 10)
}

function isoDateDaysAgo(days: number): string {
  return new Date(Date.now() - days * 24 * 60 * 60 * 1000).toISOString().slice(0, 10)
}

function dayLabel(iso: string): string {
  return new Date(iso).toLocaleDateString(undefined, { month: 'short', day: 'numeric' })
}

export default function OverviewPage() {
  const { token } = useAuth()
  const [mode, setMode] = useState<ReportsRangeMode>('7D')
  const [customFrom, setCustomFrom] = useState(() => isoDateDaysAgo(7))
  const [customTo, setCustomTo] = useState(() => isoDateToday())
  const [showDaily, setShowDaily] = useState(false)
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
    { label: 'Total Revenue', value: salesSummary ? formatMoney(salesSummary.current.revenue_minor_units, currency) : '—', change: salesSummary?.revenue_change_percent ?? null },
    { label: 'Total Orders', value: salesSummary ? String(salesSummary.current.order_count) : '—', change: salesSummary?.order_count_change_percent ?? null },
    { label: 'Units Sold', value: salesSummary ? String(salesSummary.current.units_sold) : '—', change: salesSummary?.units_sold_change_percent ?? null },
    { label: 'Avg Order Value', value: salesSummary ? formatMoney(salesSummary.current.average_order_value_minor_units, currency) : '—', change: salesSummary?.aov_change_percent ?? null },
  ]

  const profitabilityTrendPoints = (profitabilityTrend?.points ?? []).map((p: ProfitabilityPoint) => ({ ...p, label: dayLabel(p.period_start) }))

  const channelSources = Array.from(new Set((revenueByChannel?.points ?? []).map(p => p.source)))
  const visibleChannelSources = channelSources.filter(s => !hiddenChannels.has(s))
  const channelTrendRows = buildChannelTrendRows(revenueByChannel?.points ?? [], channelSources)

  return (
    <ReportPageShell
      title="Overview"
      subtitle={`${new Date(from).toLocaleDateString()} – ${new Date(to).toLocaleDateString()}`}
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
          <div key={kpi.label} className="dark:bg-[#1F2937] bg-white rounded-xl p-4 border dark:border-[#2D3748] border-gray-200">
            <p className="text-xs dark:text-gray-500 text-gray-400 mb-2">{kpi.label}</p>
            <p className="text-2xl font-semibold dark:text-gray-100 text-gray-900 mb-1">{kpi.value}</p>
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
                  ? 'bg-amber-50 dark:bg-amber-500/10 text-amber-700 dark:text-amber-400'
                  : 'bg-blue-50 dark:bg-blue-500/10 text-blue-700 dark:text-blue-400',
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
            <ResponsiveContainer width="100%" height={220}>
              <LineChart data={profitabilityTrendPoints} margin={{ top: 10, right: 16, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
                <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number, name: string) => [formatMoney(v, currency), name]} />
                <Legend wrapperStyle={{ fontSize: 11 }} />
                <Line type="monotone" dataKey="revenue_minor_units" name="Revenue" stroke="#2563EB" strokeWidth={2} dot={false} />
                <Line type="monotone" dataKey="profit_minor_units" name="Profit" stroke="#16A34A" strokeWidth={2} dot={false} />
              </LineChart>
            </ResponsiveContainer>
          )
        ) : !profitability || profitability.summary.cost_known_revenue_minor_units === 0 ? (
          <EmptyState>No cost-known revenue in this period</EmptyState>
        ) : (
          <div data-testid="profitability-summary" className="grid grid-cols-4 gap-3 p-4">
            <div>
              <p className="text-xs dark:text-gray-500 text-gray-400 mb-1">Revenue</p>
              <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{formatMoney(profitability.summary.revenue_minor_units, currency)}</p>
              {profitability.summary.cost_known_revenue_minor_units < profitability.summary.revenue_minor_units && (
                <p className="text-[10px] dark:text-gray-500 text-gray-400 mt-0.5">
                  {formatMoney(profitability.summary.cost_known_revenue_minor_units, currency)} with known cost
                </p>
              )}
            </div>
            <div>
              <p className="text-xs dark:text-gray-500 text-gray-400 mb-1">COGS</p>
              <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{formatMoney(profitability.summary.cost_minor_units, currency)}</p>
            </div>
            <div>
              <p className="text-xs dark:text-gray-500 text-gray-400 mb-1">Profit</p>
              <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{formatMoney(profitability.summary.profit_minor_units, currency)}</p>
            </div>
            <div>
              <p className="text-xs dark:text-gray-500 text-gray-400 mb-1">Margin</p>
              <p className="text-sm font-semibold dark:text-gray-100 text-gray-900">{(profitability.summary.margin_basis_points / 100).toFixed(1)}%</p>
            </div>
          </div>
        )}
      </ReportCard>

      <ReportCard title="Revenue by Channel" description={showDaily ? 'Daily revenue per sales channel' : 'Total revenue per sales channel'}>
        {showDaily ? (
          channelTrendRows.length === 0 ? (
            <EmptyState>No revenue in this period</EmptyState>
          ) : (
            <>
              <div className="flex flex-wrap gap-1.5 px-4 pt-3">
                {channelSources.map((source, i) => {
                  const visible = !hiddenChannels.has(source)
                  const color = CHANNEL_COLORS[i % CHANNEL_COLORS.length]
                  return (
                    <button
                      key={source}
                      onClick={() => toggleChannel(source)}
                      className={clsx(
                        'px-2 py-1 rounded-md text-[10px] font-medium border transition-colors',
                        visible ? 'text-white border-transparent' : 'dark:border-[#2D3748] border-gray-200 dark:text-gray-500 text-gray-400 opacity-60',
                      )}
                      style={visible ? { backgroundColor: color } : undefined}
                      title={visible ? `Hide ${source}` : `Show ${source}`}
                    >
                      {source}
                    </button>
                  )
                })}
              </div>
              {visibleChannelSources.length === 0 ? (
                <EmptyState>Select at least one channel to show</EmptyState>
              ) : (
                <ResponsiveContainer width="100%" height={220}>
                  <BarChart data={channelTrendRows} margin={{ top: 10, right: 16, left: -20, bottom: 0 }}>
                    <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
                    <XAxis dataKey="label" tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} />
                    <YAxis tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                    <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number, name: string) => [formatMoney(v, currency), name]} />
                    <Legend wrapperStyle={{ fontSize: 11 }} />
                    {visibleChannelSources.map(source => (
                      <Bar key={source} dataKey={source} name={source} stackId="revenue" fill={CHANNEL_COLORS[channelSources.indexOf(source) % CHANNEL_COLORS.length]} />
                    ))}
                  </BarChart>
                </ResponsiveContainer>
              )}
            </>
          )
        ) : !revenueByChannel || revenueByChannel.channels.length === 0 ? (
          <EmptyState>No revenue in this period</EmptyState>
        ) : (
          <>
            <ResponsiveContainer width="100%" height={180}>
              <BarChart data={revenueByChannel.channels} margin={{ top: 10, right: 10, left: -20, bottom: 0 }}>
                <CartesianGrid strokeDasharray="3 3" stroke="#2D3748" />
                <XAxis dataKey="source" tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} />
                <YAxis tick={{ fontSize: 10, fill: '#6B7280' }} axisLine={false} tickLine={false} tickFormatter={v => formatMoney(v, currency)} />
                <Tooltip contentStyle={TOOLTIP_STYLE} formatter={(v: number) => [formatMoney(v, currency), 'Revenue']} />
                <Bar dataKey="total_minor_units" fill="#2563EB" radius={4} />
              </BarChart>
            </ResponsiveContainer>
            <table className="w-full text-xs border-t dark:border-[#2D3748] border-gray-200">
              <thead>
                <tr className="dark:bg-[#111827]/50 bg-gray-50">
                  {['Channel', 'Revenue'].map(h => (
                    <th key={h} className="px-3 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">{h}</th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
                {revenueByChannel.channels.map(c => (
                  <tr key={c.source} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                    <td className="px-3 py-2 dark:text-gray-300 text-gray-700 font-medium">{c.source}</td>
                    <td className="px-3 py-2 dark:text-gray-200 text-gray-800 font-medium">{formatMoney(c.total_minor_units, currency)}</td>
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
function buildChannelTrendRows(points: ChannelRevenuePoint[], sources: string[]): Array<Record<string, string | number>> {
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
