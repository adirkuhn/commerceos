import clsx from 'clsx'
import type { TFunction } from 'i18next'
import { useTranslation } from 'react-i18next'
import { useDateFormat } from '../format/DateFormatProvider'
import { Money, useMoneyFormat } from '../format/amountVisibility'
import type { Supplier } from '../suppliers/types'
import { useOrderQuantities } from './hooks/useOrderQuantities'
import { OrderActions } from './OrderActions'
import { formatPurchaseQuantity, formatStockQuantity, orderUrgency, purchaseUnitSuffix, type OrderUrgency } from './quantity'
import { urgencyLabels } from './labels'
import type { PlanOrderLine, RecommendationLine, SupplierRecommendation } from './types'

const URGENCY_STYLE: Record<OrderUrgency, string> = {
  overdue: 'text-danger font-semibold',
  today: 'text-danger font-medium',
  soon: 'text-warning font-medium',
  later: 'text-ink-3',
  none: 'text-ink-4',
}

interface Props {
  group: SupplierRecommendation
  // The plan's "today" (YYYY-MM-DD), for how urgent each order-by date is.
  asOf: string
  // With onCreateOrder, quantities are editable and the card turns them
  // into a draft PO (suppliers: who to order unsourced products from).
  suppliers?: Pick<Supplier, 'id' | 'name'>[]
  onCreateOrder?: (supplierId: string, lines: PlanOrderLine[]) => Promise<void>
}

// One supplier's share of the plan: what to order from them, most urgent first.
export function SupplierRecommendationCard({ group, asOf, suppliers = [], onCreateOrder }: Props) {
  const { t } = useTranslation('purchasePlanning')
  const formatMoney = useMoneyFormat()
  const { formatDate } = useDateFormat()
  const unsourced = !group.supplier_id
  const order = useOrderQuantities(group.lines)
  const editable = Boolean(onCreateOrder)
  const total = editable ? editedTotal(group, order.quantity) : group.estimated_cost_minor_units

  return (
    <section className="rounded-xl border border-line bg-raised overflow-hidden">
      <header className="flex flex-wrap items-baseline gap-x-4 gap-y-1 px-4 py-3 border-b border-line bg-sunken">
        <h2 className="text-sm font-semibold text-ink">{unsourced ? t('group.noSupplier') : group.supplier_name}</h2>
        <span className="text-xs text-ink-3">
          {t('group.leadTime', { count: group.lead_time_days })}
          {group.lead_time_assumed && (
            <span className="ml-1.5 px-1.5 py-0.5 rounded bg-hover text-2xs text-ink-4">{t('group.leadTimeAssumed')}</span>
          )}
        </span>
        <span className="text-xs text-ink-4">{t('group.products', { count: group.lines.length })}</span>
        <span className="ml-auto flex items-baseline gap-4 text-xs">
          {group.earliest_order_by_date && (
            <span className={URGENCY_STYLE[orderUrgency(group.earliest_order_by_date, asOf)]}>
              {t('group.orderBy', { date: formatDate(group.earliest_order_by_date) })}
            </span>
          )}
          {total != null && (
            <span className="font-medium text-ink">
              {t('group.estimatedTotal', { amount: formatMoney(total, group.currency_code) })}
            </span>
          )}
        </span>
        {unsourced && <p className="basis-full text-2xs text-ink-4">{t('group.noSupplierHint')}</p>}
        {onCreateOrder && (
          <OrderActions supplierId={group.supplier_id ?? null} suppliers={suppliers} lines={order.orderLines} onCreateOrder={onCreateOrder} />
        )}
      </header>
      <div className="overflow-x-auto">
        <table className="w-full text-xs">
          <thead>
            <tr className="text-2xs font-medium text-ink-4 uppercase">
              <th className="px-4 py-2 text-left">{t('col.product')}</th>
              <th className="px-4 py-2 text-right">{t('col.onHand')}</th>
              <th className="px-4 py-2 text-right">{t('col.onOrder')}</th>
              <th className="px-4 py-2 text-left">{t('col.forecast')}</th>
              <th className="px-4 py-2 text-right">{t('col.order')}</th>
              <th className="px-4 py-2 text-right">{t('col.cost')}</th>
              <th className="px-4 py-2 text-right">{t('col.orderBy')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line/50">
            {group.lines.map(line => (
              <RecommendationRow
                key={line.sku_id} line={line} asOf={asOf}
                order={editable ? { typed: order.typed(line.sku_id), quantity: order.quantity(line.sku_id), onChange: v => order.setTyped(line.sku_id, v) } : null}
              />
            ))}
          </tbody>
        </table>
      </div>
    </section>
  )
}

interface RowOrder {
  typed: string
  quantity: number
  onChange: (value: string) => void
}

function RecommendationRow({ line, asOf, order }: { line: RecommendationLine; asOf: string; order: RowOrder | null }) {
  const { t } = useTranslation('purchasePlanning')
  const { formatDate } = useDateFormat()
  const qty = (units: number) => formatStockQuantity(units, line.is_weight_pool)
  const urgency = orderUrgency(line.order_by_date, asOf)
  const onOrder = line.on_order.sent + line.on_order.draft

  return (
    <tr data-testid="recommendation-row" className="hover:bg-hover/30 align-top">
      <td className="px-4 py-2">
        <div className="text-ink">{line.product_name}</div>
        <div className="text-2xs text-ink-4">{line.sku_code}</div>
      </td>
      <td className="px-4 py-2 text-right text-ink-3">{qty(line.on_hand)}</td>
      <td className="px-4 py-2 text-right text-ink-3">
        <div>{qty(onOrder)}</div>
        {line.on_order.draft > 0 && (
          <div className="text-2xs text-ink-4">{t('onOrderDraft', { amount: qty(line.on_order.draft) })}</div>
        )}
      </td>
      <td className="px-4 py-2">
        <div className="text-ink">{qty(line.forecast.units)}</div>
        <div className="text-2xs text-ink-4">{t('forecast.days', { count: line.forecast.days })}</div>
        <div className="text-2xs text-ink-4">{forecastReason(line, qty, t)}</div>
        {line.pre_season && line.pre_season.units > 0 && (
          <div className="text-2xs text-ink-4">{t('season.preSeason', { amount: qty(line.pre_season.units) })}</div>
        )}
        {line.seasonal_lift != null && (
          <div className="text-2xs font-medium text-accent">{t('season.lift', { lift: Math.round(line.seasonal_lift * 10) / 10 })}</div>
        )}
      </td>
      <td className="px-4 py-2 text-right font-semibold text-ink whitespace-nowrap">
        {order ? (
          <>
            <input
              type="number" min={0} step={1} value={order.typed} onChange={e => order.onChange(e.target.value)}
              aria-label={t('order.quantityFor', { product: line.product_name })}
              className="w-20 px-2 py-1 rounded-md border border-line bg-surface text-right text-xs text-ink"
            />
            {purchaseUnitSuffix(line) && <span className="ml-1 text-2xs font-normal text-ink-4">{purchaseUnitSuffix(line)}</span>}
          </>
        ) : formatPurchaseQuantity(line)}
      </td>
      <td className="px-4 py-2 text-right text-ink-3">{lineCost(line, order)}</td>
      <td className="px-4 py-2 text-right">
        <div className={URGENCY_STYLE[urgency]}>{line.order_by_date ? formatDate(line.order_by_date) : '—'}</div>
        {urgency !== 'none' && <div className={clsx('text-2xs', URGENCY_STYLE[urgency])}>{urgencyLabels(t)[urgency]}</div>}
      </td>
    </tr>
  )
}

function lineCost(line: RecommendationLine, order: RowOrder | null) {
  if (line.unit_cost_minor_units == null) return '—'
  const cost = order ? line.unit_cost_minor_units * order.quantity : line.estimated_cost_minor_units
  return cost != null ? <Money minor={cost} currency={line.currency_code} /> : '—'
}

// The edited order's total; null unless every line has a cost in the group's currency.
function editedTotal(group: SupplierRecommendation, quantity: (skuId: string) => number): number | null {
  let total = 0
  for (const l of group.lines) {
    if (l.unit_cost_minor_units == null || l.currency_code !== group.currency_code) return null
    total += l.unit_cost_minor_units * quantity(l.sku_id)
  }
  return total
}

function forecastReason(line: RecommendationLine, qty: (n: number) => string, t: TFunction<'purchasePlanning'>): string {
  switch (line.forecast.method) {
    case 'seasonal':
      return t('forecast.seasonal', {
        lastYear: qty(line.forecast.last_year_units ?? 0),
        growth: (Math.round(line.forecast.growth_factor * 100) / 100).toString(),
      })
    case 'recent_velocity':
      return t('forecast.recentVelocity', { recent: qty(line.forecast.recent_units ?? 0) })
    default:
      return t('forecast.noDemand')
  }
}
