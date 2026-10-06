import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../format/DateFormatProvider'
import { addDays } from '../format/calendarDay'
import { useQueryString } from '../routing/useQueryState'
import type { Supplier } from '../suppliers/types'
import { useSeasonEvents, useSeasonPlan } from './hooks/useSeasonPlan'
import { seasonEventLabels } from './labels'
import { SAFETY, parseDays } from './planningWindow'
import { DaysPicker, Field, PlanResult, RefreshButton, SELECT, SupplierFilter } from './PlanControls'
import { SupplierRecommendationCard } from './SupplierRecommendationCard'
import type { DateWindow, SeasonEventKey } from './types'

const CUSTOM = 'custom'

// "What do I need for a season": ?season=<event>|custom (&from=&to=, both
// inclusive days), &safety=, &supplier=, &all=1. Without ?season the next
// upcoming holiday is planned.
export function SeasonPlanTab({ token, suppliers }: { token: string; suppliers: Supplier[] }) {
  const { t } = useTranslation('purchasePlanning')
  const { formatDate } = useDateFormat()
  const labels = seasonEventLabels(t)
  const [seasonRaw, setSeason] = useQueryString('season')
  const [from, setFrom] = useQueryString('from')
  const [to, setTo] = useQueryString('to')
  const [safetyRaw, setSafety] = useQueryString('safety')
  const [supplierId, setSupplierId] = useQueryString('supplier')
  const [allRaw, setAll] = useQueryString('all')
  const safety = parseDays(safetyRaw, SAFETY)
  const allProducts = allRaw === '1'

  const events = useSeasonEvents(token)
  const eventList = events.data ?? []
  const season = seasonRaw ?? eventList[0]?.key ?? null
  const custom = season === CUSTOM
  const event = !custom && season && season in labels ? (season as SeasonEventKey) : null

  const plan = useSeasonPlan(token, {
    event,
    from: custom ? from : null,
    // The API's range end is exclusive; the picker's "to" is the last day.
    to: custom && to ? addDays(to, 1) : null,
    safety_days: safety, supplier_id: supplierId, all_products: allProducts,
  })
  const data = plan.data
  const showWindow = (w: DateWindow) => t('season.range', { start: formatDate(w.start), end: formatDate(addDays(w.end, -1)) })

  return (
    <>
      <div className="flex flex-wrap items-end gap-3 px-5 py-3 border-b border-line bg-sunken shrink-0">
        <Field id="planner-season" label={t('season.picker')}>
          <select id="planner-season" className={SELECT} value={season ?? ''} onChange={e => setSeason(e.target.value || null)}>
            {eventList.map(e => (
              <option key={e.key} value={e.key}>{t('season.option', { name: labels[e.key], date: formatDate(e.date) })}</option>
            ))}
            <option value={CUSTOM}>{t('season.custom')}</option>
          </select>
        </Field>
        {custom && (
          <>
            <Field id="planner-from" label={t('season.from')}>
              <input id="planner-from" type="date" className={SELECT} value={from ?? ''} onChange={e => setFrom(e.target.value || null)} />
            </Field>
            <Field id="planner-to" label={t('season.to')}>
              <input id="planner-to" type="date" className={SELECT} value={to ?? ''} min={from ?? undefined} onChange={e => setTo(e.target.value || null)} />
            </Field>
          </>
        )}
        <DaysPicker id="planner-season-safety" label={t('window.safety')} range={SAFETY} value={safety}
          onChange={v => setSafety(v === SAFETY.fallback ? null : String(v))} />
        <SupplierFilter id="planner-season-supplier" suppliers={suppliers} value={supplierId} onChange={setSupplierId} />
        <label className="flex items-center gap-1.5 pb-1.5 text-xs text-ink-3">
          <input type="checkbox" checked={allProducts} onChange={e => setAll(e.target.checked ? '1' : null)} />
          {t('season.allProducts')}
        </label>
        <RefreshButton loading={plan.loading} onClick={plan.refresh} />
      </div>
      <div className="flex-1 overflow-auto px-5 py-4 space-y-4">
        <p className="text-xs text-ink-3 max-w-3xl">{t('season.intro')}</p>
        {events.error && <p role="alert" className="text-xs text-danger">{events.error}</p>}
        {data && !plan.error && (
          <p className="text-sm text-ink">
            <span className="font-semibold">
              {data.event ? t('season.title', { name: labels[data.event], year: data.year }) : t('season.custom')}
            </span>
            {' · '}{showWindow(data.window)}
            <span className="text-xs text-ink-4"> · {t('season.comparedWith', { range: showWindow(data.last_year_window) })}</span>
          </p>
        )}
        <PlanResult loading={plan.loading} error={plan.error} empty={!data || data.suppliers.length === 0} emptyText={t('season.empty')}>
          {data?.suppliers.map(group => (
            <SupplierRecommendationCard key={group.supplier_id ?? 'unsourced'} group={group} asOf={data.as_of} />
          ))}
        </PlanResult>
      </div>
    </>
  )
}
