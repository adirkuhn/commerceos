import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'
import { formatMoney } from '../../format/money'

const TOP_CUSTOMERS_LIMIT = 20

export default function CustomerPerformancePage() {
  const { token } = useAuth()
  const { t } = useTranslation('reports')
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getCustomerPerformanceReport = useCallback(reportsApi.getCustomerPerformanceReport, [])
  const { data: customerPerformance, loading, error } = useSingleReport(token ?? '', period, getCustomerPerformanceReport)
  const currency = customerPerformance?.currency_code ?? 'USD'
  const topCustomers = [...(customerPerformance?.rows ?? [])]
    .sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)
    .slice(0, TOP_CUSTOMERS_LIMIT)

  return (
    <ReportPageShell title={t('customerPerformance.title')} period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title={t('customerPerformance.card')} description={t('customerPerformance.desc')}>
        {topCustomers.length === 0 ? (
          <EmptyState>{t('customerPerformance.empty')}</EmptyState>
        ) : (
          <div className="divide-y divide-line/50">
            {topCustomers.map(c => (
              <div key={c.customer_id} className="flex items-center gap-3 px-4 py-2.5">
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-ink truncate">{c.customer_name || t('customerPerformance.unnamed')}</p>
                  <p className="text-2xs text-ink-4">
                    {t('customerPerformance.ordersAov', { count: c.order_count, aov: formatMoney(c.average_order_value_minor_units, currency) })}
                  </p>
                </div>
                <p className="text-xs font-medium text-ink shrink-0">{formatMoney(c.revenue_minor_units, currency)}</p>
              </div>
            ))}
          </div>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
