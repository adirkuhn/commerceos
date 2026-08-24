import { useState, useCallback } from 'react'
import { AlertTriangle, Info } from 'lucide-react'
import clsx from 'clsx'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { ReportsPeriod } from '../../reports/period'
import { ReportPageShell, formatMoney, ChangeBadge } from './shared'

export default function OverviewPage() {
  const { token } = useAuth()
  const [period, setPeriod] = useState<ReportsPeriod>('7D')

  const getSalesSummaryReport = useCallback(reportsApi.getSalesSummaryReport, [])
  const getBusinessInsights = useCallback(reportsApi.getBusinessInsights, [])
  const { data: salesSummary, loading: summaryLoading, error: summaryError } = useSingleReport(token ?? '', period, getSalesSummaryReport)
  const { data: businessInsights, loading: insightsLoading, error: insightsError } = useSingleReport(token ?? '', period, getBusinessInsights)

  const currency = salesSummary?.currency_code ?? 'USD'
  const kpis = [
    { label: 'Total Revenue', value: salesSummary ? formatMoney(salesSummary.current.revenue_minor_units, currency) : '—', change: salesSummary?.revenue_change_percent ?? null },
    { label: 'Total Orders', value: salesSummary ? String(salesSummary.current.order_count) : '—', change: salesSummary?.order_count_change_percent ?? null },
    { label: 'Units Sold', value: salesSummary ? String(salesSummary.current.units_sold) : '—', change: salesSummary?.units_sold_change_percent ?? null },
    { label: 'Avg Order Value', value: salesSummary ? formatMoney(salesSummary.current.average_order_value_minor_units, currency) : '—', change: salesSummary?.aov_change_percent ?? null },
  ]

  return (
    <ReportPageShell
      title="Overview"
      period={period}
      onPeriodChange={setPeriod}
      loading={summaryLoading || insightsLoading}
      error={summaryError ?? insightsError}
    >
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
    </ReportPageShell>
  )
}
