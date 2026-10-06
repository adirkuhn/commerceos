import { RefreshCw } from 'lucide-react'
import clsx from 'clsx'
import { useTranslation } from 'react-i18next'
import { useAuth } from '../auth/AuthContext'
import { useQueryString } from '../routing/useQueryState'
import { useSuppliers } from '../suppliers/hooks/useSuppliers'
import { useReorderRecommendations } from '../purchaseplanning/hooks/useReorderRecommendations'
import { SupplierRecommendationCard } from '../purchaseplanning/SupplierRecommendationCard'
import { COVERAGE, SAFETY, dayOptions, parseDays, type DayRange } from '../purchaseplanning/planningWindow'

const SELECT = 'px-2 py-1.5 rounded-lg border border-line bg-raised text-xs text-ink focus:outline-none focus:ring-1 focus:ring-accent'

export default function PurchasePlanner() {
  const { token } = useAuth()
  const { t } = useTranslation('purchasePlanning')
  const { suppliers } = useSuppliers(token ?? '')

  // The planning window lives in the URL (?coverage=&safety=&supplier=)
  // so refresh, Back and shared links keep it.
  const [coverageRaw, setCoverage] = useQueryString('coverage')
  const [safetyRaw, setSafety] = useQueryString('safety')
  const [supplierId, setSupplierId] = useQueryString('supplier')
  const coverage = parseDays(coverageRaw, COVERAGE)
  const safety = parseDays(safetyRaw, SAFETY)

  const { data, loading, error, refresh } = useReorderRecommendations(token ?? '', {
    coverage_days: coverage, safety_days: safety, supplier_id: supplierId,
  })

  return (
    <div className="flex flex-col h-full">
      <div className="flex flex-wrap items-end gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <DaysPicker id="planner-coverage" label={t('window.coverage')} range={COVERAGE} value={coverage}
          onChange={v => setCoverage(v === COVERAGE.fallback ? null : String(v))} />
        <DaysPicker id="planner-safety" label={t('window.safety')} range={SAFETY} value={safety}
          onChange={v => setSafety(v === SAFETY.fallback ? null : String(v))} />
        <label className="flex flex-col gap-1 text-2xs text-ink-4" htmlFor="planner-supplier">
          {t('window.supplier')}
          <select id="planner-supplier" className={SELECT} value={supplierId ?? ''} onChange={e => setSupplierId(e.target.value || null)}>
            <option value="">{t('window.allSuppliers')}</option>
            {suppliers.map(s => <option key={s.id} value={s.id}>{s.name}</option>)}
          </select>
        </label>
        <button
          onClick={refresh}
          disabled={loading}
          className="ml-auto flex items-center gap-1.5 px-3 py-1.5 rounded-lg border border-line bg-raised text-xs text-ink-2 hover:bg-hover transition-colors font-medium disabled:opacity-50"
        >
          <RefreshCw size={13} className={clsx(loading && 'animate-spin')} /> {t('refresh')}
        </button>
      </div>

      <div className="flex-1 overflow-auto px-5 py-4 space-y-4">
        <p className="text-xs text-ink-3 max-w-3xl">{t('intro')}</p>
        {error && <p role="alert" className="text-xs text-danger">{error}</p>}
        {loading && !data && <p className="text-xs text-ink-4">{t('loading')}</p>}
        {data && data.suppliers.length === 0 && !error && (
          <p className="py-10 text-center text-xs text-ink-4">{t('empty')}</p>
        )}
        {data?.suppliers.map(group => (
          <SupplierRecommendationCard key={group.supplier_id ?? 'unsourced'} group={group} asOf={data.as_of} />
        ))}
      </div>
    </div>
  )
}

interface DaysPickerProps {
  id: string
  label: string
  range: DayRange
  value: number
  onChange: (days: number) => void
}

function DaysPicker({ id, label, range, value, onChange }: DaysPickerProps) {
  const { t } = useTranslation('purchasePlanning')
  return (
    <label className="flex flex-col gap-1 text-2xs text-ink-4" htmlFor={id}>
      {label}
      <select id={id} className={SELECT} value={value} onChange={e => onChange(Number(e.target.value))}>
        {dayOptions(range, value).map(d => <option key={d} value={d}>{t('window.days', { count: d })}</option>)}
      </select>
    </label>
  )
}
