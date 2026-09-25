import { useEffect, useState } from 'react'
import clsx from 'clsx'
import { getPromotionPerformanceReport } from '../reports/api'
import type { PromotionPerformanceReport } from '../reports/types'

function formatMoney(minorUnits: number, currencyCode: string): string {
  try {
    return new Intl.NumberFormat(undefined, { style: 'currency', currency: currencyCode || 'USD' }).format(minorUnits / 100)
  } catch {
    return `${(minorUnits / 100).toFixed(2)} ${currencyCode}`
  }
}

// PromotionPerformanceCard compares the promotion's covered SKUs' revenue/
// units during its active window (capped at "now" if still ongoing)
// against the immediately preceding period of equal length -- the Phase 0
// "Promotion Analytics" report.
export function PromotionPerformanceCard({ token, promotionId }: { token: string; promotionId: string }) {
  const [report, setReport] = useState<PromotionPerformanceReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    getPromotionPerformanceReport(token, promotionId)
      .then(result => { if (!cancelled) setReport(result) })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load performance') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, promotionId])

  const changeLabel = (percent: number | null) => {
    if (percent === null) return '— vs before'
    const up = percent >= 0
    return `${up ? '+' : ''}${percent.toFixed(1)}% vs before`
  }

  return (
    <div className="rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-xs font-semibold text-ink">Performance</h3>
        <p className="text-2xs text-ink-4 mt-0.5">
          Covered SKUs during the active window vs. the immediately preceding period of equal length
        </p>
      </div>
      {loading && <p className="text-xs text-ink-4">Loading…</p>}
      {!loading && error && <p className="text-xs text-danger">{error}</p>}
      {!loading && !error && report && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <p className="text-2xs text-ink-4 mb-1">Revenue (during)</p>
            <p className="text-sm font-semibold text-ink">{formatMoney(report.during.revenue_minor_units, report.currency_code)}</p>
            <p className={clsx('text-2xs font-medium', report.revenue_change_percent === null ? 'text-ink-4' : report.revenue_change_percent >= 0 ? 'text-success' : 'text-danger')}>
              {changeLabel(report.revenue_change_percent)}
            </p>
          </div>
          <div>
            <p className="text-2xs text-ink-4 mb-1">Units sold (during)</p>
            <p className="text-sm font-semibold text-ink">{report.during.units_sold}</p>
            <p className={clsx('text-2xs font-medium', report.units_sold_change_percent === null ? 'text-ink-4' : report.units_sold_change_percent >= 0 ? 'text-success' : 'text-danger')}>
              {changeLabel(report.units_sold_change_percent)}
            </p>
          </div>
          <div>
            <p className="text-2xs text-ink-4 mb-1">Revenue (before)</p>
            <p className="text-xs text-ink-2">{formatMoney(report.before.revenue_minor_units, report.currency_code)}</p>
          </div>
          <div>
            <p className="text-2xs text-ink-4 mb-1">Units sold (before)</p>
            <p className="text-xs text-ink-2">{report.before.units_sold}</p>
          </div>
        </div>
      )}
    </div>
  )
}
