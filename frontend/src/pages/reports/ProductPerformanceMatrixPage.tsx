import { useCallback } from 'react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { ProductPerformanceMatrixRow, ProductPerformanceQuadrant } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState, formatMoney } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'

const QUADRANTS: { id: ProductPerformanceQuadrant; label: string; hint: string }[] = [
  { id: 'stars', label: 'Stars', hint: 'High revenue, low margin — optimize pricing' },
  { id: 'winners', label: 'Winners', hint: 'High revenue, high margin — push harder' },
  { id: 'problems', label: 'Problems', hint: 'Low revenue, low margin — consider discontinuing' },
  { id: 'hidden_gems', label: 'Hidden Gems', hint: 'Low revenue, high margin — promote' },
]

export default function ProductPerformanceMatrixPage() {
  const { token } = useAuth()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getProductPerformanceMatrix = useCallback(reportsApi.getProductPerformanceMatrix, [])
  const { data: matrix, loading, error } = useSingleReport(token ?? '', period, getProductPerformanceMatrix)
  const currency = matrix?.currency_code ?? 'USD'

  return (
    <ReportPageShell title="Product Performance Matrix" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard
        title="Product Performance Matrix"
        description={
          matrix && matrix.rows.length > 0
            ? `Split around this period's own median revenue (${formatMoney(matrix.median_revenue_minor_units, currency)}) and median margin (${(matrix.median_margin_basis_points / 100).toFixed(1)}%)`
            : 'Selected period'
        }
      >
        {!matrix || matrix.rows.length === 0 ? (
          <EmptyState>No SKUs with a known margin in this period</EmptyState>
        ) : (
          <div className="grid grid-cols-2 gap-3 p-4">
            {QUADRANTS.map(q => {
              const rows = matrix.rows.filter(r => r.quadrant === q.id)
              return (
                <div key={q.id} data-testid={`quadrant-${q.id}`} className="rounded-lg border dark:border-[#2D3748] border-gray-200 p-3">
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-xs font-semibold dark:text-gray-100 text-gray-900">{q.label}</p>
                    <span className="text-[10px] dark:text-gray-500 text-gray-400">{rows.length}</span>
                  </div>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400 mb-2">{q.hint}</p>
                  {rows.length === 0 ? (
                    <p className="text-[10px] dark:text-gray-600 text-gray-300">None</p>
                  ) : (
                    <div className="space-y-1">
                      {rows.slice(0, 8).map((r: ProductPerformanceMatrixRow) => (
                        <div key={r.sku_id ?? r.sku_code} className="flex items-center justify-between gap-2 text-[11px]">
                          <span className="dark:text-gray-300 text-gray-700 truncate">{r.product_name || r.sku_code}</span>
                          <span className="dark:text-gray-500 text-gray-400 shrink-0">{formatMoney(r.revenue_minor_units, currency)}</span>
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              )
            })}
          </div>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
