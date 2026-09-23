import { useCallback } from 'react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'

export default function StockoutRiskPage() {
  const { token } = useAuth()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getStockoutRiskReport = useCallback(reportsApi.getStockoutRiskReport, [])
  const { data: stockoutRisk, loading, error } = useSingleReport(token ?? '', period, getStockoutRiskReport)
  const atRisk = (stockoutRisk?.rows ?? [])
    .filter(r => r.at_risk)
    .sort((a, b) => (a.days_of_stock_remaining ?? 0) - (b.days_of_stock_remaining ?? 0))

  return (
    <ReportPageShell title="Stockout Risk" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard
        title="Stockout Risk"
        description="SKUs projected to run out before a reorder placed today (14-day lead time) could arrive, ranked soonest-first"
      >
        {atRisk.length === 0 ? (
          <EmptyState>No SKUs at risk of stocking out</EmptyState>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-xs">
              <thead>
                <tr className="dark:bg-[#111827]/50 bg-gray-50">
                  <th className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Product</th>
                  <th className="px-4 py-2 text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Stock</th>
                  <th className="px-4 py-2 text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Days Left</th>
                  <th className="px-4 py-2 text-right text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">Suggested Reorder</th>
                </tr>
              </thead>
              <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
                {atRisk.map(row => (
                  <tr key={row.sku_id} data-testid="stockout-risk-row" className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                    <td className="px-4 py-2 dark:text-gray-200 text-gray-800">{row.product_name || row.sku_code}</td>
                    <td className="px-4 py-2 text-right dark:text-gray-400 text-gray-500">{row.stock_on_hand}</td>
                    <td className="px-4 py-2 text-right text-red-500 font-medium">{row.days_of_stock_remaining?.toFixed(1) ?? '—'}</td>
                    <td className="px-4 py-2 text-right dark:text-gray-400 text-gray-500">{row.suggested_reorder_quantity}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
