import { useCallback } from 'react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { ReportPageShell, ReportCard, EmptyState, formatMoney } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'

const TOP_CUSTOMERS_LIMIT = 20

export default function CustomerPerformancePage() {
  const { token } = useAuth()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getCustomerPerformanceReport = useCallback(reportsApi.getCustomerPerformanceReport, [])
  const { data: customerPerformance, loading, error } = useSingleReport(token ?? '', period, getCustomerPerformanceReport)
  const currency = customerPerformance?.currency_code ?? 'USD'
  const topCustomers = [...(customerPerformance?.rows ?? [])]
    .sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)
    .slice(0, TOP_CUSTOMERS_LIMIT)

  return (
    <ReportPageShell title="Customer Performance" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title="Top Customers by Revenue" description="Selected period — guest orders excluded">
        {topCustomers.length === 0 ? (
          <EmptyState>No customer orders in this period</EmptyState>
        ) : (
          <div className="divide-y dark:divide-[#2D3748]/50 divide-gray-100">
            {topCustomers.map(c => (
              <div key={c.customer_id} className="flex items-center gap-3 px-4 py-2.5">
                <div className="flex-1 min-w-0">
                  <p className="text-xs dark:text-gray-200 text-gray-800 truncate">{c.customer_name || 'Unnamed customer'}</p>
                  <p className="text-[10px] dark:text-gray-500 text-gray-400">
                    {c.order_count} order{c.order_count === 1 ? '' : 's'} · {formatMoney(c.average_order_value_minor_units, currency)} AOV
                  </p>
                </div>
                <p className="text-xs font-medium dark:text-gray-200 text-gray-800 shrink-0">{formatMoney(c.revenue_minor_units, currency)}</p>
              </div>
            ))}
          </div>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
