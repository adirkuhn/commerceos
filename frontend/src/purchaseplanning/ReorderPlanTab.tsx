import { useTranslation } from 'react-i18next'
import { useQueryString } from '../routing/useQueryState'
import type { Supplier } from '../suppliers/types'
import { useReorderRecommendations } from './hooks/useReorderRecommendations'
import { COVERAGE, SAFETY, parseDays } from './planningWindow'
import { DaysPicker, PlanResult, RefreshButton, SupplierFilter } from './PlanControls'
import { SupplierRecommendationCard } from './SupplierRecommendationCard'
import { useCreatePlanOrder } from './hooks/useCreatePlanOrder'

// "What to reorder now": window in the URL (?coverage=&safety=&supplier=).
export function ReorderPlanTab({ token, suppliers }: { token: string; suppliers: Supplier[] }) {
  const { t } = useTranslation('purchasePlanning')
  const createOrder = useCreatePlanOrder(token)
  const [coverageRaw, setCoverage] = useQueryString('coverage')
  const [safetyRaw, setSafety] = useQueryString('safety')
  const [supplierId, setSupplierId] = useQueryString('supplier')
  const coverage = parseDays(coverageRaw, COVERAGE)
  const safety = parseDays(safetyRaw, SAFETY)
  const { data, loading, error, refresh } = useReorderRecommendations(token, {
    coverage_days: coverage, safety_days: safety, supplier_id: supplierId,
  })

  return (
    <>
      <div className="flex flex-wrap items-end gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <DaysPicker id="planner-coverage" label={t('window.coverage')} range={COVERAGE} value={coverage}
          onChange={v => setCoverage(v === COVERAGE.fallback ? null : String(v))} />
        <DaysPicker id="planner-safety" label={t('window.safety')} range={SAFETY} value={safety}
          onChange={v => setSafety(v === SAFETY.fallback ? null : String(v))} />
        <SupplierFilter id="planner-supplier" suppliers={suppliers} value={supplierId} onChange={setSupplierId} />
        <RefreshButton loading={loading} onClick={refresh} />
      </div>
      <div className="flex-1 overflow-auto px-5 py-4 space-y-4">
        <p className="text-xs text-ink-3 max-w-3xl">{t('intro')}</p>
        <PlanResult loading={loading} error={error} empty={!data || data.suppliers.length === 0} emptyText={t('empty')}>
          {data?.suppliers.map(group => (
            <SupplierRecommendationCard key={group.supplier_id ?? 'unsourced'} group={group} asOf={data.as_of} suppliers={suppliers} onCreateOrder={createOrder} />
          ))}
        </PlanResult>
      </div>
    </>
  )
}
