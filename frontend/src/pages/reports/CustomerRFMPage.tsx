import { useCallback } from 'react'
import i18next from 'i18next'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { RFMSegment } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'
import { Money } from '../../format/amountVisibility'

function rfmSegments(): { id: RFMSegment; label: string; hint: string }[] {
  return [
    { id: 'champions', label: i18next.t('reports:rfm.champions'), hint: i18next.t('reports:rfm.championsHint') },
    { id: 'loyal_customers', label: i18next.t('reports:rfm.loyal'), hint: i18next.t('reports:rfm.loyalHint') },
    { id: 'potential_loyalists', label: i18next.t('reports:rfm.potential'), hint: i18next.t('reports:rfm.potentialHint') },
    { id: 'at_risk', label: i18next.t('reports:rfm.atRisk'), hint: i18next.t('reports:rfm.atRiskHint') },
    { id: 'lost', label: i18next.t('reports:rfm.lost'), hint: i18next.t('reports:rfm.lostHint') },
    { id: 'needs_attention', label: i18next.t('reports:rfm.needsAttention'), hint: i18next.t('reports:rfm.needsAttentionHint') },
  ]
}

export default function CustomerRFMPage() {
  const { token } = useAuth()
  const { t } = useTranslation('reports')
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getCustomerRFMReport = useCallback(reportsApi.getCustomerRFMReport, [])
  const { data: customerRFM, loading, error } = useSingleReport(token ?? '', period, getCustomerRFMReport)
  const currency = customerRFM?.currency_code ?? 'USD'

  return (
    <ReportPageShell title={t('rfm.title')} period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard
        title={t('rfm.title')}
        description={t('rfm.desc')}
      >
        {!customerRFM || customerRFM.rows.length === 0 ? (
          <EmptyState>{t('rfm.empty')}</EmptyState>
        ) : (
          <div className="grid grid-cols-3 gap-3 p-4">
            {rfmSegments().map(segment => {
              const rows = customerRFM.rows.filter(r => r.segment === segment.id)
              return (
                <div key={segment.id} data-testid={`rfm-${segment.id}`} className="rounded-lg border border-line p-3">
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-xs font-semibold text-ink">{segment.label}</p>
                    <span className="text-2xs text-ink-4">{rows.length}</span>
                  </div>
                  <p className="text-2xs text-ink-4 mb-2">{segment.hint}</p>
                  {rows.length === 0 ? (
                    <p className="text-2xs text-ink-4">{t('none')}</p>
                  ) : (
                    <div className="space-y-1">
                      {rows.slice(0, 8).map(r => (
                        <div key={r.customer_id} className="flex items-center justify-between gap-2 text-2xs">
                          <span className="text-ink-2 truncate">{r.customer_name || t('unnamedCustomer')}</span>
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
