import { useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'
import { formatMoney } from '../../format/money'

export default function CategoryPerformancePage() {
  const { token } = useAuth()
  const { t } = useTranslation('reports')
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getCategoryPerformanceReport = useCallback(reportsApi.getCategoryPerformanceReport, [])
  const { data: categoryPerformance, loading, error } = useSingleReport(token ?? '', period, getCategoryPerformanceReport)
  const currency = categoryPerformance?.currency_code ?? 'USD'
  const categories = [...(categoryPerformance?.rows ?? [])].sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)

  return (
    <ReportPageShell title={t('category.title')} period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title={t('category.card')} description={t('selectedPeriod')}>
        {categories.length === 0 ? (
          <EmptyState>{t('category.empty')}</EmptyState>
        ) : (
          <table className="w-full text-xs">
            <thead>
              <tr className="dark:bg-night-900/50 bg-gray-50">
                {[t('col.category'), t('col.units'), t('col.orders'), t('col.revenue')].map(h => (
                  <th key={h} className="px-4 py-2 text-left text-2xs font-medium text-ink-4 uppercase">{h}</th>
                ))}
              </tr>
            </thead>
            <tbody className="divide-y divide-line/50">
              {categories.map(cat => (
                <tr key={cat.category_id ?? 'uncategorized'} className="hover:bg-hover/30">
                  <td className="px-4 py-2 text-ink font-medium">{cat.category_name}</td>
                  <td className="px-4 py-2 text-ink-3">{cat.quantity_sold}</td>
                  <td className="px-4 py-2 text-ink-3">{cat.order_count}</td>
                  <td className="px-4 py-2 text-ink font-medium">{formatMoney(cat.revenue_minor_units, currency)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </ReportCard>
    </ReportPageShell>
  )
}
