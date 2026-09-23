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
              <tr className="dark:bg-night-900/50 bg-gray-50">
                {['Product', 'SKU', 'Stock on hand'].map(h => (
                  <th key={h} className="px-4 py-2 text-left text-2xs font-medium text-ink-4 uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line/50">
              {slowMovers.map(p => (
                <tr key={p.sku_id} className="hover:bg-hover/30">
                  <td className="px-4 py-2 text-ink font-medium">{p.product_name || p.sku_code}</td>
                  <td className="px-4 py-2 font-mono text-ink-4">{p.sku_code}</td>
                  <td className="px-4 py-2 text-warning font-medium">{p.stock_on_hand}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
