import { useState, useCallback } from 'react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { ReportsPeriod } from '../../reports/period'
import { ReportPageShell, ReportCard, EmptyState, formatMoney } from './shared'

export default function CategoryPerformancePage() {
  const { token } = useAuth()
  const [period, setPeriod] = useState<ReportsPeriod>('7D')
  const getCategoryPerformanceReport = useCallback(reportsApi.getCategoryPerformanceReport, [])
  const { data: categoryPerformance, loading, error } = useSingleReport(token ?? '', period, getCategoryPerformanceReport)
  const currency = categoryPerformance?.currency_code ?? 'USD'
  const categories = [...(categoryPerformance?.rows ?? [])].sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)

  return (
    <ReportPageShell title="Category Performance" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title="Revenue by Category" description="Selected period">
        {categories.length === 0 ? (
          <EmptyState>No category sales in this period</EmptyState>
        ) : (
          <table className="w-full text-xs">
            <thead>
              <tr className="dark:bg-[#111827]/50 bg-gray-50">
                {['Category', 'Units', 'Orders', 'Revenue'].map(h => (
                  <th key={h} className="px-4 py-2 text-left text-[10px] font-medium dark:text-gray-500 text-gray-400 uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
              {categories.map(cat => (
                <tr key={cat.category_id ?? 'uncategorized'} className="hover:dark:bg-[#2D3748]/30 hover:bg-gray-50">
                  <td className="px-4 py-2 dark:text-gray-200 text-gray-800 font-medium">{cat.category_name}</td>
                  <td className="px-4 py-2 dark:text-gray-400 text-gray-500">{cat.quantity_sold}</td>
                  <td className="px-4 py-2 dark:text-gray-400 text-gray-500">{cat.order_count}</td>
                  <td className="px-4 py-2 dark:text-gray-200 text-gray-800 font-medium">{formatMoney(cat.revenue_minor_units, currency)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
