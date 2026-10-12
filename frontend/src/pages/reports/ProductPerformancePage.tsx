import { useCallback } from 'react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'
import { Money } from '../../format/amountVisibility'

const TOP_PRODUCTS_LIMIT = 20

export default function ProductPerformancePage() {
  const { token } = useAuth()
  const { t } = useTranslation('reports')
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getProductPerformanceReport = useCallback(reportsApi.getProductPerformanceReport, [])
  const { data: productPerformance, loading, error } = useSingleReport(token ?? '', period, getProductPerformanceReport)
  const currency = productPerformance?.currency_code ?? 'USD'
  const topProducts = [...(productPerformance?.rows ?? [])]
    .sort((a, b) => b.revenue_minor_units - a.revenue_minor_units)
    .slice(0, TOP_PRODUCTS_LIMIT)

  return (
    <ReportPageShell title={t('productPerformance.title')} period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title={t('productPerformance.card')} description={t('selectedPeriod')}>
        {topProducts.length === 0 ? (
          <EmptyState>{t('productPerformance.empty')}</EmptyState>
        ) : (
          <div className="divide-y divide-line/50">
            {topProducts.map(p => (
              <div key={p.sku_id ?? p.sku_code} className="flex items-center gap-3 px-4 py-2.5">
                <div className="flex-1 min-w-0">
                  <p className="text-xs text-ink truncate">{p.product_name || p.sku_code}</p>
                  <p className="text-2xs font-mono text-ink-4">{p.sku_code}</p>
                </div>
                <div className="text-right shrink-0">
                  <p className="text-xs font-medium text-ink"><Money minor={p.revenue_minor_units} currency={currency} /></p>
                  <p className={clsx('text-2xs font-medium', p.margin_basis_points === null ? 'text-ink-4' : p.margin_basis_points >= 5000 ? 'text-success' : 'text-warning')}>
                    {p.margin_basis_points === null ? t('productPerformance.marginNa') : `${(p.margin_basis_points / 100).toFixed(1)}%`}
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
