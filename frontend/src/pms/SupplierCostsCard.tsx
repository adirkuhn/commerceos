import { useEffect, useState } from 'react'
import { Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import { apiErrorText } from '../api/apiErrorText'
import { Money } from '../format/amountVisibility'
import { useDateFormat } from '../format/DateFormatProvider'
import { FormCard } from './FormCard'
import * as pmsApi from './api'
import type { SKUSupplierCosts, SupplierCostEntry, SupplierCostKind } from './types'

const PAGE_SIZE = 20

const KIND_STYLE: Record<SupplierCostKind, string> = {
  receipt: 'bg-fill text-ink-3',
  reversal: 'bg-red-500/15 text-danger',
  correction: 'bg-amber-500/15 text-warning',
}

// SupplierCostsCard is a SKU's cost history by supplier (COM-11): the last
// receipt's landed cost -- what the cost price follows --, the average over
// everything received, each supplier's last and average cost, and the
// history rows themselves (newest first, one supplier's on click).
// inventoryCostMinorUnits is what the stock on hand actually cost (moving
// average), shown alongside for comparison.
export function SupplierCostsCard({ skuId, inventoryCostMinorUnits }: { skuId: string; inventoryCostMinorUnits?: number | null }) {
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])
  const { formatDate } = useDateFormat()
  const [data, setData] = useState<SKUSupplierCosts | null>(null)
  const [items, setItems] = useState<SupplierCostEntry[]>([])
  const [supplierFilter, setSupplierFilter] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [loadingMore, setLoadingMore] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError('')
    pmsApi.listSKUSupplierCosts(token, skuId, { supplierId: supplierFilter ?? undefined, limit: PAGE_SIZE })
      .then(result => { if (!cancelled) { setData(result); setItems(result.items) } })
      .catch(err => { if (!cancelled) setError(apiErrorText(err) ?? t('supplierCosts.loadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, skuId, supplierFilter]) // eslint-disable-line react-hooks/exhaustive-deps

  const loadMore = async () => {
    if (!token || data?.next_cursor == null) return
    setLoadingMore(true)
    setError('')
    try {
      const page = await pmsApi.listSKUSupplierCosts(token, skuId, { supplierId: supplierFilter ?? undefined, cursor: data.next_cursor, limit: PAGE_SIZE })
      setData(page)
      setItems(prev => [...prev, ...page.items])
    } catch (err) {
      setError(apiErrorText(err) ?? t('supplierCosts.loadFailed'))
    } finally {
      setLoadingMore(false)
    }
  }

  const currency = data?.currency_code ?? ''
  const money = (minor: number) => <Money minor={minor} currency={currency} />
  const supplierLabel = (name: string | null, id: string | null) => name ?? (id ? t('supplierCosts.deletedSupplier') : t('supplierCosts.noSupplier'))
  const filterName = supplierFilter && data?.by_supplier.find(s => s.supplier_id === supplierFilter)?.supplier_name

  return (
    <FormCard title={t('supplierCosts.title')} description={t('supplierCosts.description')}>
      {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
      {!loading && error && <p className="text-xs text-danger">{error}</p>}
      {!loading && !error && data && data.total === 0 && !supplierFilter && (
        <p className="text-xs text-ink-4">{t('supplierCosts.empty')}</p>
      )}
      {!loading && data && (data.total > 0 || supplierFilter) && (
        <div className="flex flex-col gap-4">
          <div className="grid grid-cols-2 sm:grid-cols-3 gap-x-4 gap-y-3 text-xs">
            <div data-testid="last-cost">
              <div className="text-ink-4">{t('supplierCosts.last')}</div>
              {data.last ? (
                <>
                  <div className="font-medium text-ink">{money(data.last.landed_unit_cost_minor_units)}</div>
                  <div className="text-2xs text-ink-4 mt-0.5">
                    {supplierLabel(data.last.supplier_name, data.last.supplier_id)} · {formatDate(data.last.received_at)} ·{' '}
                    <Link to={`/procurement/${data.last.purchase_order_id}`} className="text-accent hover:underline">{t('supplierCosts.purchaseOrder')}</Link>
                  </div>
                </>
              ) : <div className="text-ink-4">—</div>}
            </div>
            <div data-testid="average-cost">
              <div className="text-ink-4">{t('supplierCosts.average')}</div>
              <div className="font-medium text-ink">{data.average_cost_minor_units != null ? money(data.average_cost_minor_units) : '—'}</div>
            </div>
            {inventoryCostMinorUnits != null && (
              <div data-testid="inventory-cost">
                <div className="text-ink-4">{t('supplierCosts.inventoryCost')}</div>
                <div className="font-medium text-ink">{money(inventoryCostMinorUnits)}</div>
              </div>
            )}
          </div>

          {data.excluded_other_currency > 0 && (
            <p className="text-2xs text-warning">{t('supplierCosts.otherCurrency', { count: data.excluded_other_currency, currency })}</p>
          )}

          {data.by_supplier.length > 0 && (
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-ink-4 text-left">
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.supplier')}</th>
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.lastCost')}</th>
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.averageCost')}</th>
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.quantity')}</th>
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.receipts')}</th>
                    <th className="font-medium pb-1.5">{t('supplierCosts.lastReceived')}</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {data.by_supplier.map(s => (
                    <tr key={s.supplier_id ?? 'none'} data-testid="supplier-row">
                      <td className="py-1.5 pr-3">
                        {s.supplier_id ? (
                          <button
                            type="button" onClick={() => setSupplierFilter(s.supplier_id)} title={t('supplierCosts.filterHint')}
                            className={clsx('text-accent hover:underline', supplierFilter === s.supplier_id && 'font-semibold')}
                          >
                            {supplierLabel(s.supplier_name, s.supplier_id)}
                          </button>
                        ) : <span className="text-ink-3 italic">{supplierLabel(null, null)}</span>}
                      </td>
                      <td className="py-1.5 pr-3 text-ink">{s.last_landed_unit_cost_minor_units != null ? money(s.last_landed_unit_cost_minor_units) : '—'}</td>
                      <td className="py-1.5 pr-3 text-ink">{money(s.average_cost_minor_units)}</td>
                      <td className="py-1.5 pr-3 text-ink-2 tabular-nums">{s.quantity}</td>
                      <td className="py-1.5 pr-3 text-ink-2 tabular-nums">{s.receipts}</td>
                      <td className="py-1.5 text-ink-3 whitespace-nowrap">{formatDate(s.last_received_at)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <div>
            <div className="flex items-center gap-2 mb-1.5">
              <h4 className="text-xs font-semibold text-ink-2">{t('supplierCosts.history')}</h4>
              {supplierFilter && (
                <button type="button" onClick={() => setSupplierFilter(null)} className="text-2xs text-accent hover:underline">
                  {t('supplierCosts.showAll', { supplier: filterName ?? t('supplierCosts.deletedSupplier') })}
                </button>
              )}
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr className="text-ink-4 text-left">
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.date')}</th>
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.supplier')}</th>
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.kind')}</th>
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.quantity')}</th>
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.price')}</th>
                    <th className="font-medium pb-1.5 pr-3">{t('supplierCosts.landed')}</th>
                    <th className="font-medium pb-1.5" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-line/60">
                  {items.map((e, i) => (
                    <tr key={`${e.purchase_order_line_item_id}-${e.kind}-${e.received_at}-${i}`} data-testid="history-row">
                      <td className="py-1.5 pr-3 text-ink-2 whitespace-nowrap">{formatDate(e.received_at)}</td>
                      <td className="py-1.5 pr-3 text-ink-2">{supplierLabel(e.supplier_name, e.supplier_id)}</td>
                      <td className="py-1.5 pr-3">
                        <span className={clsx('px-1.5 py-0.5 rounded text-2xs', KIND_STYLE[e.kind])}>{t(`supplierCosts.kinds.${e.kind}`)}</span>
                      </td>
                      <td className={clsx('py-1.5 pr-3 tabular-nums', e.quantity < 0 ? 'text-danger' : 'text-ink-2')}>{e.quantity}</td>
                      <td className="py-1.5 pr-3 text-ink-2">
                        {e.currency_code === currency ? money(e.unit_cost_minor_units) : <Money minor={e.unit_cost_minor_units} currency={e.currency_code} />}
                      </td>
                      <td className="py-1.5 pr-3 text-ink font-medium">
                        {e.currency_code === currency ? money(e.landed_unit_cost_minor_units) : <Money minor={e.landed_unit_cost_minor_units} currency={e.currency_code} />}
                      </td>
                      <td className="py-1.5">
                        <Link to={`/procurement/${e.purchase_order_id}`} className="text-accent hover:underline">{t('supplierCosts.purchaseOrder')}</Link>
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            {data.next_cursor != null && (
              <button
                type="button" onClick={loadMore} disabled={loadingMore}
                className="mt-2 px-2.5 py-1 rounded-lg bg-fill text-2xs font-medium text-ink-2 hover:bg-hover disabled:opacity-50"
              >
                {loadingMore ? t('common:status.loading') : t('supplierCosts.loadMore', { shown: items.length, total: data.total })}
              </button>
            )}
          </div>
        </div>
      )}
    </FormCard>
  )
}
