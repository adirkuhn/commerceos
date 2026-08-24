import { useState, useCallback } from 'react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { ReportsPeriod } from '../../reports/period'
import { ReportPageShell, ReportCard, EmptyState, formatMoney } from './shared'

export default function ProfitabilityPage() {
  const { token } = useAuth()
  const [period, setPeriod] = useState<ReportsPeriod>('7D')
  const getProfitabilityReport = useCallback(reportsApi.getProfitabilityReport, [])
  const { data: profitability, loading, error } = useSingleReport(token ?? '', period, getProfitabilityReport)
  const currency = profitability?.currency_code ?? 'USD'

  return (
    <ReportPageShell title="Profitability" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title="Profitability" description="Selected period — profit/margin cover only the revenue with a known SKU cost price">
        {!profitability || profitability.summary.cost_known_revenue_minor_units === 0 ? (
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
    </ReportPageShell>
  )
}
