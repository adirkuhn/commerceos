import { useCallback } from 'react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'

export default function SlowMovingProductsPage() {
  const { token } = useAuth()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getSlowMovingProductsReport = useCallback(reportsApi.getSlowMovingProductsReport, [])
  const { data: slowMovingProducts, loading, error } = useSingleReport(token ?? '', period, getSlowMovingProductsReport)
  const slowMovers = [...(slowMovingProducts?.rows ?? [])].sort((a, b) => b.stock_on_hand - a.stock_on_hand)

  return (
    <ReportPageShell title="Slow-Moving Products" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title="Slow-Moving Products" description="In stock with zero units sold in the selected period">
        {slowMovers.length === 0 ? (
          <EmptyState>No slow-moving products in this period</EmptyState>
        ) : (
          <table className="w-full text-xs">
            <thead>
              <tr className="dark:bg-[#111827]/50 bg-gray-50">
                {['Product', 'SKU', 'Stock on Hand'].map(h => (
                  <th key={h} className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
              {slowMovers.map(p => (
                <tr key={p.sku_id} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                  <td className="px-4 py-2 dark:text-gray-200 text-gray-800 font-medium">{p.product_name || p.sku_code}</td>
                  <td className="px-4 py-2 font-mono dark:text-gray-500 text-gray-400">{p.sku_code}</td>
                  <td className="px-4 py-2 dark:text-amber-400 text-amber-600 font-medium">{p.stock_on_hand}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
