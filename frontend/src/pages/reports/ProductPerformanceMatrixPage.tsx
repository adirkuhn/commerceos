import { useCallback } from 'react'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { ProductPerformanceMatrixRow, ProductPerformanceQuadrant } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'
import { Money, useMoneyFormat } from '../../format/amountVisibility'

function quadrants(): { id: ProductPerformanceQuadrant; label: string; hint: string }[] {
  return [
    { id: 'stars', label: i18next.t('reports:matrix.stars'), hint: i18next.t('reports:matrix.starsHint') },
    { id: 'winners', label: i18next.t('reports:matrix.winners'), hint: i18next.t('reports:matrix.winnersHint') },
    { id: 'problems', label: i18next.t('reports:matrix.problems'), hint: i18next.t('reports:matrix.problemsHint') },
    { id: 'hidden_gems', label: i18next.t('reports:matrix.hiddenGems'), hint: i18next.t('reports:matrix.hiddenGemsHint') },
  ]
}

export default function ProductPerformanceMatrixPage() {
  const { token } = useAuth()
  const { t } = useTranslation('reports')
  const formatMoney = useMoneyFormat()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getProductPerformanceMatrix = useCallback(reportsApi.getProductPerformanceMatrix, [])
  const { data: matrix, loading, error } = useSingleReport(token ?? '', period, getProductPerformanceMatrix)
  const currency = matrix?.currency_code ?? 'USD'

  return (
    <ReportPageShell title={t('matrix.title')} period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard
        title={t('matrix.title')}
        description={
          matrix && matrix.rows.length > 0
            ? t('matrix.split', { revenue: formatMoney(matrix.median_revenue_minor_units, currency), margin: `${(matrix.median_margin_basis_points / 100).toFixed(1)}%` })
            : t('selectedPeriod')
        }
      >
        {!matrix || matrix.rows.length === 0 ? (
          <EmptyState>{t('matrix.empty')}</EmptyState>
        ) : (
          <div className="grid grid-cols-2 gap-3 p-4">
            {quadrants().map(q => {
              const rows = matrix.rows.filter(r => r.quadrant === q.id)
              return (
                <div key={q.id} data-testid={`quadrant-${q.id}`} className="rounded-lg border border-line p-3">
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-xs font-semibold text-ink">{q.label}</p>
                    <span className="text-2xs text-ink-4">{rows.length}</span>
                  </div>
                  <p className="text-2xs text-ink-4 mb-2">{q.hint}</p>
                  {rows.length === 0 ? (
                    <p className="text-2xs text-ink-4">{t('none')}</p>
                  ) : (
                    <div className="space-y-1">
                      {rows.slice(0, 8).map((r: ProductPerformanceMatrixRow) => (
                        <div key={r.sku_id ?? r.sku_code} className="flex items-center justify-between gap-2 text-2xs">
                          <span className="text-ink-2 truncate">{r.product_name || r.sku_code}</span>
                          <span className="text-ink-4 shrink-0"><Money minor={r.revenue_minor_units} currency={currency} /></span>
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
