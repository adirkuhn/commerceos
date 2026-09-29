import { useCallback } from 'react'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { ProductLifecycleStage } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'

function lifecycleStages(): { id: ProductLifecycleStage; label: string; hint: string }[] {
  return [
    { id: 'new', label: i18next.t('reports:lifecycle.new'), hint: i18next.t('reports:lifecycle.newHint') },
    { id: 'growing', label: i18next.t('reports:lifecycle.growing'), hint: i18next.t('reports:lifecycle.growingHint') },
    { id: 'stable', label: i18next.t('reports:lifecycle.stable'), hint: i18next.t('reports:lifecycle.stableHint') },
    { id: 'declining', label: i18next.t('reports:lifecycle.declining'), hint: i18next.t('reports:lifecycle.decliningHint') },
    { id: 'dormant', label: i18next.t('reports:lifecycle.dormant'), hint: i18next.t('reports:lifecycle.dormantHint') },
  ]
}

export default function ProductLifecyclePage() {
  const { token } = useAuth()
  const { t } = useTranslation('reports')
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getProductLifecycleReport = useCallback(reportsApi.getProductLifecycleReport, [])
  const { data: productLifecycle, loading, error } = useSingleReport(token ?? '', period, getProductLifecycleReport)

  return (
    <ReportPageShell title={t('lifecycle.title')} period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title={t('lifecycle.title')} description={t('lifecycle.desc')}>
        {!productLifecycle || productLifecycle.rows.length === 0 ? (
          <EmptyState>{t('lifecycle.empty')}</EmptyState>
        ) : (
          <div className="grid grid-cols-3 gap-3 p-4">
            {lifecycleStages().map(stage => {
              const rows = productLifecycle.rows.filter(r => r.stage === stage.id)
              return (
                <div key={stage.id} data-testid={`lifecycle-${stage.id}`} className="rounded-lg border border-line p-3">
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-xs font-semibold text-ink">{stage.label}</p>
                    <span className="text-2xs text-ink-4">{rows.length}</span>
                  </div>
                  <p className="text-2xs text-ink-4 mb-2">{stage.hint}</p>
                  {rows.length === 0 ? (
                    <p className="text-2xs text-ink-4">{t('none')}</p>
                  ) : (
                    <div className="space-y-1">
                      {rows.slice(0, 8).map(r => (
                        <div key={r.sku_id ?? r.sku_code} className="flex items-center justify-between gap-2 text-2xs">
                          <span className="text-ink-2 truncate">{r.product_name || r.sku_code}</span>
                          <span className="text-ink-4 shrink-0">{t('lifecycle.sold', { count: r.current_quantity_sold })}</span>
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
