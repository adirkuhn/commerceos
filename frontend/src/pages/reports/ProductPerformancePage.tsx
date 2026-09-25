import { useCallback } from 'react'
import clsx from 'clsx'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'
import { formatMoney } from '../../format/money'

const TOP_PRODUCTS_LIMIT = 20

export default function ProductPerformancePage() {
  const { token } = useAuth()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getProductPerformanceReport = useCallback(reportsApi.getProductPerformanceReport, [])
  const { data: productPerformance, loading, error } = useSingleReport(token ?? '', period, getProductPerformanceReport)
  const currency = productPerformance?.currency_code ?? 'USD'
  const topProducts = [...(productPerformance?.rows ?? [])]
    .sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)
    .slice(0, TOP_PRODUCTS_LIMIT)

  return (
    <ReportPageShell title="Product Performance" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title="Top products by revenue" description="Selected period">
        {topProducts.length === 0 ? (
          <EmptyState>No sales in this period</EmptyState>
        ) : (
          <div className="divide-y divide-line/50">
            {topProducts.map(p => (
              <div key={p.sku_id ?? p.sku_code} className="flex items-center gap-3 px-4 py-2.5">
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-ink truncate">{p.product_name || p.sku_code}</p>
                  <p className="text-2xs font-mono text-ink-4">{p.sku_code}</p>
                </div>
                <div className="text-right shrink-0">
                  <p className="text-xs font-medium text-ink">{formatMoney(p.revenue_minor_units, currency)}</p>
                  <p className={clsx('text-2xs font-medium', p.margin_basis_points === null ? 'text-ink-4' : p.margin_basis_points >= 5000 ? 'text-success' : 'text-warning')}>
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
