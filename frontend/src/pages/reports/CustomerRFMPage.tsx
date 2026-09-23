import { useCallback } from 'react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { RFMSegment } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState, formatMoney } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'

const RFM_SEGMENTS: { id: RFMSegment; label: string; hint: string }[] = [
  { id: 'champions', label: 'Champions', hint: 'Recent, frequent, highest spend' },
  { id: 'loyal_customers', label: 'Loyal Customers', hint: 'Frequent and recent' },
  { id: 'potential_loyalists', label: 'Potential Loyalists', hint: 'Recent, not yet frequent' },
  { id: 'at_risk', label: 'At Risk', hint: 'Used to order often, hasn’t recently' },
  { id: 'lost', label: 'Lost', hint: 'Neither recent nor frequent' },
  { id: 'needs_attention', label: 'Needs attention', hint: 'Mid-range on every axis' },
]

export default function CustomerRFMPage() {
  const { token } = useAuth()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getCustomerRFMReport = useCallback(reportsApi.getCustomerRFMReport, [])
  const { data: customerRFM, loading, error } = useSingleReport(token ?? '', period, getCustomerRFMReport)
  const currency = customerRFM?.currency_code ?? 'USD'

  return (
    <ReportPageShell title="Customer Segments (RFM)" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard
        title="Customer Segments (RFM)"
        description="Recency/Frequency/Monetary quintiles, relative to this period's own customers — guest orders excluded"
      >
        {!customerRFM || customerRFM.rows.length === 0 ? (
          <EmptyState>No customer orders in this period</EmptyState>
        ) : (
          <div className="grid grid-cols-3 gap-3 p-4">
            {RFM_SEGMENTS.map(segment => {
              const rows = customerRFM.rows.filter(r => r.segment === segment.id)
              return (
                <div key={segment.id} data-testid={`rfm-${segment.id}`} className="rounded-lg border border-line p-3">
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-xs font-semibold text-ink">{segment.label}</p>
                    <span className="text-2xs text-ink-4">{rows.length}</span>
                  </div>
                  <p className="text-2xs text-ink-4 mb-2">{segment.hint}</p>
                  {rows.length === 0 ? (
                    <p className="text-2xs text-ink-4">None</p>
                  ) : (
                    <div className="space-y-1">
                      {rows.slice(0, 8).map(r => (
                        <div key={r.customer_id} className="flex items-center justify-between gap-2 text-2xs">
                          <span className="text-ink-2 truncate">{r.customer_name || 'Unnamed customer'}</span>
                          <span className="text-ink-4 shrink-0">{formatMoney(r.revenue_minor_units, currency)}</span>
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
