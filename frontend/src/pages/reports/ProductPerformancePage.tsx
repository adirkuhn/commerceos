import { useState, useCallback } from 'react'
import clsx from 'clsx'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { ReportsPeriod } from '../../reports/period'
import { ReportPageShell, ReportCard, EmptyState, formatMoney } from './shared'

const TOP_PRODUCTS_LIMIT = 20

export default function ProductPerformancePage() {
  const { token } = useAuth()
  const [period, setPeriod] = useState<ReportsPeriod>('7D')
  const getProductPerformanceReport = useCallback(reportsApi.getProductPerformanceReport, [])
  const { data: productPerformance, loading, error } = useSingleReport(token ?? '', period, getProductPerformanceReport)
  const currency = productPerformance?.currency_code ?? 'USD'
  const topProducts = [...(productPerformance?.rows ?? [])]
    .sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)
    .slice(0, TOP_PRODUCTS_LIMIT)

  return (
    <ReportPageShell title="Product Performance" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title="Top Products by Revenue" description="Selected period">
        {topProducts.length === 0 ? (
          <EmptyState>No sales in this period</EmptyState>
        ) : (
          <div className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
            {topProducts.map(p => (
              <div key={p.sku_id ?? p.sku_code} className="flex items-center gap-3 px-4 py-2.5">
                <div className="flex-1 min-w-0">
                  <p className="text-xs dark:text-gray-200 text-gray-800 truncate">{p.product_name || p.sku_code}</p>
                  <p className="text-[10px] font-mono dark:text-gray-500 text-gray-400">{p.sku_code}</p>
                </div>
                <div className="text-right shrink-0">
                  <p className="text-xs font-medium dark:text-gray-200 text-gray-800">{formatMoney(p.revenue_minor_units, currency)}</p>
                  <p className={clsx('text-[10px] font-medium', p.margin_basis_points === null ? 'dark:text-gray-500 text-gray-400' : p.margin_basis_points >= 5000 ? 'text-green-400' : 'text-amber-400')}>
                    {p.margin_basis_points === null ? 'margin n/a' : `${(p.margin_basis_points / 100).toFixed(1)}%`}
                  </p>
                </div>
              </div>
            ))}
          </div>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
