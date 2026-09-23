import { useCallback } from 'react'
import { useAuth } from '../../auth/AuthContext'
import * as reportsApi from '../../reports/api'
import { useSingleReport } from '../../reports/hooks/useSingleReport'
import type { ProductLifecycleStage } from '../../reports/types'
import { ReportPageShell, ReportCard, EmptyState } from './shared'
import { useReportPeriod } from '../../reports/hooks/useReportPeriod'

const LIFECYCLE_STAGES: { id: ProductLifecycleStage; label: string; hint: string }[] = [
  { id: 'new', label: 'New', hint: 'Sold now, not in the previous period' },
  { id: 'growing', label: 'Growing', hint: 'Sold more than the previous period' },
  { id: 'stable', label: 'Stable', hint: 'Sold about the same as the previous period' },
  { id: 'declining', label: 'Declining', hint: 'Sold less than the previous period' },
  { id: 'dormant', label: 'Dormant', hint: 'Sold last period, nothing now' },
]

export default function ProductLifecyclePage() {
  const { token } = useAuth()
  // In ?period= -- refresh and shared links keep the chosen range.
  const [period, setPeriod] = useReportPeriod()
  const getProductLifecycleReport = useCallback(reportsApi.getProductLifecycleReport, [])
  const { data: productLifecycle, loading, error } = useSingleReport(token ?? '', period, getProductLifecycleReport)

  return (
    <ReportPageShell title="Product Lifecycle" period={period} onPeriodChange={setPeriod} loading={loading} error={error}>
      <ReportCard title="Product Lifecycle" description="Compared against the immediately preceding period of equal length">
        {!productLifecycle || productLifecycle.rows.length === 0 ? (
          <EmptyState>No sales in this or the previous period</EmptyState>
        ) : (
          <div className="grid grid-cols-3 gap-3 p-4">
            {LIFECYCLE_STAGES.map(stage => {
              const rows = productLifecycle.rows.filter(r => r.stage === stage.id)
              return (
                <div key={stage.id} data-testid={`lifecycle-${stage.id}`} className="rounded-lg border border-line p-3">
                  <div className="flex items-center justify-between mb-1.5">
                    <p className="text-xs font-semibold text-ink">{stage.label}</p>
                    <span className="text-[10px] text-ink-4">{rows.length}</span>
                  </div>
                  <p className="text-[10px] text-ink-4 mb-2">{stage.hint}</p>
                  {rows.length === 0 ? (
                    <p className="text-[10px] text-ink-4">None</p>
                  ) : (
                    <div className="space-y-1">
                      {rows.slice(0, 8).map(r => (
                        <div key={r.sku_id ?? r.sku_code} className="flex items-center justify-between gap-2 text-[11px]">
                          <span className="text-ink-2 truncate">{r.product_name || r.sku_code}</span>
                          <span className="text-ink-4 shrink-0">{r.current_quantity_sold} sold</span>
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
