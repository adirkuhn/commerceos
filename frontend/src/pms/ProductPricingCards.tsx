import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import * as pmsApi from './api'
import type { Category, SKUMarginInsight, SKUPriceHistoryEntry, SKUPriceHistoryReason, ProductHistoryEntry } from './types'
import { parsePriceToMinorUnits } from './productFormat'
import { Money, useMoneyFormat } from '../format/amountVisibility'
import { FormCard } from './FormCard'
import { useDateFormat } from '../format/DateFormatProvider'
import { apiErrorText } from '../api/apiErrorText'
import { profitExclIncl } from './profit'

// ── Shared form layout primitives ────────────────────────────────────────────

// MarginInsightCard shows the cost price (the last receipt's landed cost, or
// a manual one -- see SupplierCostsCard for where it comes from), the
// recommended price from the effective target margin, and profit/margin at
// whatever price is currently set -- see application/pms's
// GetSKUMarginInsightUseCase. Fetched fresh whenever skuId changes (e.g.
// switching between products), not tied to the price/salePrice form state --
// this reflects the *saved* SKU, not an in-progress unsaved edit.
export function MarginInsightCard({ skuId, currency, taxRateBasisPoints, pricesIncludeTax }: {
  skuId: string; currency: string; taxRateBasisPoints: number | null; pricesIncludeTax: boolean
}) {
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])
  const [insight, setInsight] = useState<SKUMarginInsight | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  const reload = () => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError('')
    pmsApi.getSKUMarginInsight(token, skuId)
      .then(result => { if (!cancelled) setInsight(result) })
      .catch(err => { if (!cancelled) setError(apiErrorText(err) ?? t('insight.loadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }

  useEffect(reload, [token, skuId])

  const formatMoney = useMoneyFormat()
  const money = (minorUnits: number) => <Money minor={minorUnits} currency={currency} />
  const moneyText = (minorUnits: number) => formatMoney(minorUnits, currency)
  const pct = (basisPoints: number) => `${(basisPoints / 100).toFixed(1)}%`

  const withBoth = (sellingPriceMinorUnits: number, costMinorUnits: number) =>
    profitExclIncl(sellingPriceMinorUnits, costMinorUnits, taxRateBasisPoints, pricesIncludeTax)

  const profitRow = (label: string, profitMinorUnits: number, marginBps: number, markupBps: number, sellingPriceMinorUnits: number, costMinorUnits: number) => {
    const both = withBoth(sellingPriceMinorUnits, costMinorUnits)
    return (
      <div>
        <div className="text-ink-4">{label}</div>
        <div className={clsx('font-medium', profitMinorUnits < 0 ? 'text-danger' : 'text-ink')}>
          {money(profitMinorUnits)} <span className="font-normal">{t('insight.marginMarkup', { margin: pct(marginBps), markup: pct(markupBps) })}</span>
        </div>
        {both && (
          <div className="text-2xs text-ink-4 mt-0.5">
            {t('insight.exclBtw', { profit: moneyText(both.exclusive.profit), margin: pct(both.exclusive.marginBasisPoints), markup: pct(both.exclusive.markupBasisPoints) })}
            {' · '}
            {t('insight.inclBtw', { profit: moneyText(both.inclusive.profit), margin: pct(both.inclusive.marginBasisPoints), markup: pct(both.inclusive.markupBasisPoints) })}
          </div>
        )}
      </div>
    )
  }

  return (
    <FormCard title={t('insight.title')} description={t('insight.description')}>
      {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
      {!loading && error && <p className="text-xs text-danger">{error}</p>}
      {!loading && !error && insight && !insight.cost_price_available && (
        <p className="text-xs text-ink-4">
          {t('insight.noCost')}
        </p>
      )}
      {!loading && !error && insight && insight.cost_price_available && (
        <div className="grid grid-cols-2 gap-x-4 gap-y-3 text-xs">
          <div>
            <div className="text-ink-4">{t('insight.costPrice')}</div>
            <div className="font-medium text-ink">{money(insight.cost_price_minor_units)}</div>
          </div>
          <div>
            <div className="text-ink-4">{t('insight.recommended')}</div>
            <div className="font-medium text-ink">
              {money(insight.recommended_price_minor_units)}
              <span className="text-ink-4 font-normal">{t('insight.targetMarkup', { percent: pct(insight.effective_target_markup_basis_points) })}</span>
            </div>
          </div>
          {profitRow(
            t('insight.profitCurrent'), insight.profit_at_price_minor_units, insight.margin_at_price_basis_points, insight.markup_at_price_basis_points,
            insight.cost_price_minor_units + insight.profit_at_price_minor_units, insight.cost_price_minor_units,
          )}
          {insight.profit_at_sale_price_minor_units != null && insight.margin_at_sale_price_basis_points != null && insight.markup_at_sale_price_basis_points != null && (
            profitRow(
              t('insight.profitSale'), insight.profit_at_sale_price_minor_units, insight.margin_at_sale_price_basis_points, insight.markup_at_sale_price_basis_points,
              insight.cost_price_minor_units + insight.profit_at_sale_price_minor_units, insight.cost_price_minor_units,
            )
          )}
          <div>
            <div className="text-ink-4">{t('insight.onHand')}</div>
            <div className="font-medium text-ink">{t('insight.units', { count: insight.total_on_hand_quantity })}</div>
          </div>
          <div>
            <div className="text-ink-4">{t('insight.profitPotential')}</div>
            <div className="font-medium text-ink">{money(insight.total_profit_potential_minor_units)}</div>
          </div>
          {insight.shrinkage_minor_units > 0 && (
            <div className="col-span-2">
              <div className="text-ink-4">{t('insight.shrinkage')}</div>
              <div className="font-medium text-warning">{money(insight.shrinkage_minor_units)}</div>
            </div>
          )}
        </div>
      )}
      {!loading && !error && insight && (
        <ManualCostPriceEditor skuId={skuId} insight={insight} currency={currency} onChanged={reload} />
      )}
    </FormCard>
  )
}


// PriceHistoryCard shows the SKU's immutable effective-dated price/sale
// price intervals -- see domain/pms.SKUPriceHistory. A row is only written
// when price/sale price actually changes, so this reconstructs what a SKU
// actually cost at any past point in time, independent of its current
// PriceMinorUnits/SalePriceMinorUnits. Fetched fresh whenever skuId
// changes, same reasoning as MarginInsightCard.
export function PriceHistoryCard({ skuId, currency }: { skuId: string; currency: string }) {
  const { token } = useAuth()
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['pms', 'common'])
  const reasonLabel: Record<SKUPriceHistoryReason, string> = {
    created: t('priceHistory.created'), manual_update: t('priceHistory.manual_update'), sale_expired: t('priceHistory.sale_expired'),
  }
  const [history, setHistory] = useState<SKUPriceHistoryEntry[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError('')
    pmsApi.listSKUPriceHistory(token, skuId)
      .then(result => { if (!cancelled) setHistory(result) })
      .catch(err => { if (!cancelled) setError(apiErrorText(err) ?? t('priceHistory.loadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, skuId])

  const money = (minorUnits: number) => <Money minor={minorUnits} currency={currency} />

  return (
    <FormCard title={t('priceHistory.title')} description={t('priceHistory.description')}>
      {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
      {!loading && error && <p className="text-xs text-danger">{error}</p>}
      {!loading && !error && history && history.length === 0 && (
        <p className="text-xs text-ink-4">{t('priceHistory.empty')}</p>
      )}
      {!loading && !error && history && history.length > 0 && (
        <table className="w-full text-xs">
          <thead>
            <tr className="text-ink-4">
              <th className="text-left font-medium pb-1.5 pr-3">{t('priceHistory.validFrom')}</th>
              <th className="text-left font-medium pb-1.5 pr-3">{t('priceHistory.validUntil')}</th>
              <th className="text-left font-medium pb-1.5 pr-3">{t('priceHistory.price')}</th>
              <th className="text-left font-medium pb-1.5 pr-3">{t('priceHistory.salePrice')}</th>
              <th className="text-left font-medium pb-1.5">{t('priceHistory.reason')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {history.map(entry => (
              <tr key={entry.id}>
                <td className="py-1.5 pr-3 text-ink-2">{formatDateTime(entry.valid_from)}</td>
                <td className="py-1.5 pr-3 text-ink-2">
                  {entry.valid_until ? formatDateTime(entry.valid_until) : <span className="text-success font-medium">{t('priceHistory.current')}</span>}
                </td>
                <td className="py-1.5 pr-3 text-ink font-medium">{money(entry.price_minor_units)}</td>
                <td className="py-1.5 pr-3 text-ink">
                  {entry.sale_price_minor_units != null ? money(entry.sale_price_minor_units) : '—'}
                </td>
                <td className="py-1.5 text-ink-3">{reasonLabel[entry.reason]}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </FormCard>
  )
}

// ProductHistoryCard shows the product's name/status change audit trail --
// see domain/pms.ProductHistory. A row is only written when a field
// actually changes. v1 scope: name and status only (not a generic diff of
// every field), and product-level only (not per-variant).
export function ProductHistoryCard({ productId }: { productId: string }) {
  const { token } = useAuth()
  const { formatDateTime } = useDateFormat()
  const { t } = useTranslation(['pms', 'common'])
  const fieldLabel: Record<string, string> = { name: t('changeHistory.fieldName'), status: t('changeHistory.fieldStatus') }
  const [history, setHistory] = useState<ProductHistoryEntry[] | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState('')

  useEffect(() => {
    if (!token) return
    let cancelled = false
    setLoading(true)
    setError('')
    pmsApi.listProductHistory(token, productId)
      .then(result => { if (!cancelled) setHistory(result) })
      .catch(err => { if (!cancelled) setError(apiErrorText(err) ?? t('changeHistory.loadFailed')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, productId])

  return (
    <FormCard title={t('changeHistory.title')} description={t('changeHistory.description')}>
      {loading && <p className="text-xs text-ink-4">{t('common:status.loading')}</p>}
      {!loading && error && <p className="text-xs text-danger">{error}</p>}
      {!loading && !error && history && history.length === 0 && (
        <p className="text-xs text-ink-4">{t('changeHistory.empty')}</p>
      )}
      {!loading && !error && history && history.length > 0 && (
        <table className="w-full text-xs">
          <thead>
            <tr className="text-ink-4">
              <th className="text-left font-medium pb-1.5 pr-3">{t('changeHistory.when')}</th>
              <th className="text-left font-medium pb-1.5 pr-3">{t('changeHistory.field')}</th>
              <th className="text-left font-medium pb-1.5 pr-3">{t('changeHistory.from')}</th>
              <th className="text-left font-medium pb-1.5">{t('changeHistory.to')}</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {history.map(entry => (
              <tr key={entry.id}>
                <td className="py-1.5 pr-3 text-ink-2">{formatDateTime(entry.changed_at)}</td>
                <td className="py-1.5 pr-3 text-ink-3 capitalize">{fieldLabel[entry.field] ?? entry.field}</td>
                <td className="py-1.5 pr-3 text-ink-2">{entry.old_value}</td>
                <td className="py-1.5 text-ink font-medium">{entry.new_value}</td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </FormCard>
  )
}

// ManualCostPriceEditor lets staff set a SKU's cost price by hand -- for a
// SKU with no receipt to take it from, or to correct a value staff know is
// wrong. Setting one sticks until "Revert to automatic" is pressed, which
// goes back to the last receipt's cost (receipts never overwrite a manual
// cost price).
function ManualCostPriceEditor({ skuId, insight, currency, onChanged }: { skuId: string; insight: SKUMarginInsight; currency: string; onChanged: () => void }) {
  const { token } = useAuth()
  const { t } = useTranslation(['pms', 'common'])
  const [value, setValue] = useState('')
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const isManual = insight.cost_price_source === 'manual'

  // Prefill from the computed cost price when one exists, so overriding it
  // is a tweak starting from the real value rather than blank re-entry --
  // only a SKU with no cost basis at all starts empty.
  useEffect(() => {
    if (isManual) return
    setValue(insight.cost_price_available ? (insight.cost_price_minor_units / 100).toFixed(2) : '')
  }, [skuId, isManual, insight.cost_price_available, insight.cost_price_minor_units])

  const handleSet = async () => {
    if (!token || value.trim() === '') return
    setSaving(true)
    setError('')
    try {
      await pmsApi.setManualCostPrice(token, skuId, parsePriceToMinorUnits(value))
      setValue('')
      onChanged()
    } catch (err) {
      setError(apiErrorText(err) ?? t('insight.setFailed'))
    } finally {
      setSaving(false)
    }
  }

  const handleClear = async () => {
    if (!token) return
    setSaving(true)
    setError('')
    try {
      await pmsApi.clearCostPriceOverride(token, skuId)
      onChanged()
    } catch (err) {
      setError(apiErrorText(err) ?? t('insight.revertFailed'))
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="pt-3 border-t border-line">
      {isManual ? (
        <div className="flex items-center gap-2 text-xs">
          <span className="text-ink-4">
            {t('insight.manualSet', { currency, amount: (insight.cost_price_minor_units / 100).toFixed(2) })}
          </span>
          <button
            type="button" onClick={handleClear} disabled={saving}
            className="px-2 py-1 rounded-lg bg-fill text-2xs font-medium text-ink-2 hover:bg-hover transition-colors disabled:opacity-50 shrink-0"
          >
            {saving ? t('insight.reverting') : t('insight.revert')}
          </button>
        </div>
      ) : (
        <div className="flex items-center gap-1.5">
          <span className="text-2xs text-ink-4 shrink-0">
            {insight.cost_price_available ? t('insight.override') : t('insight.setManually')}
          </span>
          <input
            type="number" step="0.01" min="0" placeholder={currency}
            value={value} onChange={e => setValue(e.target.value)}
            className="w-24 px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
          />
          <button
            type="button" onClick={handleSet} disabled={saving || value.trim() === ''}
            className="px-2 py-1 rounded-lg bg-fill text-2xs font-medium text-ink-2 hover:bg-hover transition-colors disabled:opacity-50"
          >
            {saving ? t('common:actions.saving') : t('insight.set')}
          </button>
        </div>
      )}
      {error && <p className="text-2xs text-danger mt-1">{error}</p>}
    </div>
  )
}

// CategoryMarginEditor lets staff override the tenant-wide default target
// markup for every product in the selected category (see
// pms.Category.TargetMarkupBasisPoints) -- an empty input means "use the
// tenant default", matching UpdateCategoryInput's full-replace semantics.
export function CategoryMarginEditor({ category, onSave }: { category: Category | null; onSave: (basisPoints: number | null) => Promise<Category> }) {
  const { t } = useTranslation(['pms', 'common'])
  const [value, setValue] = useState('')
  const [saving, setSaving] = useState(false)
  const [savedMsg, setSavedMsg] = useState(false)

  useEffect(() => {
    setValue(category?.target_markup_basis_points != null ? (category.target_markup_basis_points / 100).toString() : '')
    setSavedMsg(false)
  }, [category?.id, category?.target_markup_basis_points])

  if (!category) return null

  const handleSave = async () => {
    setSaving(true)
    try {
      const basisPoints = value.trim() === '' ? null : Math.round(parseFloat(value) * 100)
      await onSave(basisPoints)
      setSavedMsg(true)
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="flex items-center gap-1.5 mt-1.5">
      <input
        type="number" step="0.1" min="0" placeholder={t('insight.tenantDefault')}
        value={value} onChange={e => { setValue(e.target.value); setSavedMsg(false) }}
        className="w-28 px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
      />
      <span className="text-2xs text-ink-4">{t('insight.categoryMarkup')}</span>
      <button type="button" onClick={handleSave} disabled={saving} className="px-2 py-1 rounded-lg bg-fill text-2xs font-medium text-ink-2 hover:bg-hover transition-colors disabled:opacity-50">
        {saving ? t('common:actions.saving') : t('common:actions.save')}
      </button>
      {savedMsg && <span className="text-2xs text-success">{t('common:status.saved')}</span>}
    </div>
  )
}
