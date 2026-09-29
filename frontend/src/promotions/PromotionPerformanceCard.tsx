import { useTranslation } from 'react-i18next'
import { useEffect, useState } from 'react'
import clsx from 'clsx'
import { getPromotionPerformanceReport } from '../reports/api'
import type { PromotionPerformanceReport } from '../reports/types'
import { formatMoney } from '../format/money'

// PromotionPerformanceCard compares the promotion's covered SKUs' revenue/
// units during its active window (capped at "now" if still ongoing)
// against the immediately preceding period of equal length -- the Phase 0
// "Promotion Analytics" report.
export function PromotionPerformanceCard({ token, promotionId }: { token: string; promotionId: string }) {
  const { t } = useTranslation(['promotions', 'common'])
  const [report, setReport] = useState<PromotionPerformanceReport | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError('')
    getPromotionPerformanceReport(token, promotionId)
      .then(result => { if (!cancelled) setReport(result) })
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : t('performance.loadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, promotionId])

  const changeLabel = (percent: number | null) => {
    if (percent === null) return t('performance.noChange')
    const up = percent >= 0
    return t('performance.change', { change: `${up ? '+' : ''}${percent.toFixed(1)}` })
  }

  return (
    <div className="rounded-xl border border-line p-4 space-y-3">
      <div>
        <h3 className="text-xs font-semibold text-ink">{t('performance.title')}</h3>
        <p className="text-2xs text-ink-4 mt-0.5">
          {t('performance.intro')}
        </p>
      </div>
      {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
      {!loading && error && <p className="text-xs text-danger">{error}</p>}
      {!loading && !error && report && (
        <div className="grid grid-cols-2 gap-3">
          <div>
            <p className="text-2xs text-ink-4 mb-1">{t('performance.revenueDuring')}</p>
            <p className="text-sm font-semibold text-ink">{formatMoney(report.during.revenue_minor_units, report.currency_code)}</p>
            <p className={clsx('text-2xs font-medium', report.revenue_change_percent === null ? 'text-ink-4' : report.revenue_change_percent >= 0 ? 'text-success' : 'text-danger')}>
              {changeLabel(report.revenue_change_percent)}
            </p>
          </div>
          <div>
            <p className="text-2xs text-ink-4 mb-1">{t('performance.unitsDuring')}</p>
            <p className="text-sm font-semibold text-ink">{report.during.units_sold}</p>
            <p className={clsx('text-2xs font-medium', report.units_sold_change_percent === null ? 'text-ink-4' : report.units_sold_change_percent >= 0 ? 'text-success' : 'text-danger')}>
              {changeLabel(report.units_sold_change_percent)}
            </p>
          </div>
          <div>
            <p className="text-2xs text-ink-4 mb-1">{t('performance.revenueBefore')}</p>
            <p className="text-xs text-ink-2">{formatMoney(report.before.revenue_minor_units, report.currency_code)}</p>
          </div>
          <div>
            <p className="text-2xs text-ink-4 mb-1">{t('performance.unitsBefore')}</p>
            <p className="text-xs text-ink-2">{report.before.units_sold}</p>
          </div>
        </div>
      )}
    </div>
  )
}
