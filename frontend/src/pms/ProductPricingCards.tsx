import { useState, useEffect } from 'react'
import clsx from 'clsx'
import { useAuth } from '../auth/AuthContext'
import * as pmsApi from './api'
import type { Category, SKUMarginInsight, SKUPriceHistoryEntry, SKUPriceHistoryReason, ProductHistoryEntry } from './types'
import { formatPrice, parsePriceToMinorUnits } from './productFormat'
import { FormCard } from './FormCard'
import { useDateFormat } from '../format/DateFormatProvider'

// ── Shared form layout primitives ────────────────────────────────────────────

// MarginInsightCard shows the cost price the daily cron computed, the
// recommended price from the effective target margin, and profit/margin at
// whatever price is currently set -- see application/pms's
// GetSKUMarginInsightUseCase. Fetched fresh whenever skuId changes (e.g.
// switching between products), not tied to the price/salePrice form state --
// this reflects the *saved* SKU, not an in-progress unsaved edit.
// btwVariant computes profit/margin/markup at a given selling price against
// a fixed cost -- used to derive the "with BTW" / "without BTW" pair from
// whichever of the two the stored price already represents (per
// tenant.prices_include_tax), by converting the price with taxRateBasisPoints
// and re-deriving profit from there. Cost never carries tax (what you paid
// a supplier), so only price/profit shift between the two variants.
function btwVariant(priceMinorUnits: number, costMinorUnits: number) {
  const profit = priceMinorUnits - costMinorUnits
  return {
    price: priceMinorUnits,
    profit,
    marginBasisPoints: priceMinorUnits > 0 ? Math.round((profit * 10000) / priceMinorUnits) : 0,
    markupBasisPoints: costMinorUnits > 0 ? Math.round((profit * 10000) / costMinorUnits) : 0,
  }
}

export function MarginInsightCard({ skuId, currency, taxRateBasisPoints, pricesIncludeTax }: {
  skuId: string; currency: string; taxRateBasisPoints: number | null; pricesIncludeTax: boolean
}) {
  const { token } = useAuth()
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
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load margin insight') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }

  useEffect(reload, [token, skuId])

  const money = (minorUnits: number) => formatPrice(minorUnits, currency)
  const pct = (basisPoints: number) => `${(basisPoints / 100).toFixed(1)}%`

  // withBoth derives the excl./incl.-BTW pair for one selling price, given
  // the rate and which side the stored price already represents.
  const withBoth = (sellingPriceMinorUnits: number, costMinorUnits: number) => {
    if (taxRateBasisPoints == null) return null
    const rate = taxRateBasisPoints / 10000
    const exclusivePrice = pricesIncludeTax ? Math.round(sellingPriceMinorUnits / (1 + rate)) : sellingPriceMinorUnits
    const inclusivePrice = pricesIncludeTax ? sellingPriceMinorUnits : Math.round(sellingPriceMinorUnits * (1 + rate))
    return { exclusive: btwVariant(exclusivePrice, costMinorUnits), inclusive: btwVariant(inclusivePrice, costMinorUnits) }
  }

  const profitRow = (label: string, profitMinorUnits: number, marginBps: number, markupBps: number, sellingPriceMinorUnits: number, costMinorUnits: number) => {
    const both = withBoth(sellingPriceMinorUnits, costMinorUnits)
    return (
      <div>
        <div className="text-ink-4">{label}</div>
        <div className={clsx('font-medium', profitMinorUnits < 0 ? 'text-danger' : 'text-ink')}>
          {money(profitMinorUnits)} <span className="font-normal">({pct(marginBps)} margin / {pct(markupBps)} markup)</span>
        </div>
        {both && (
          <div className="text-2xs text-ink-4 mt-0.5">
            Excl. BTW: {money(both.exclusive.profit)} ({pct(both.exclusive.marginBasisPoints)} margin / {pct(both.exclusive.markupBasisPoints)} markup)
            {' · '}
            Incl. BTW: {money(both.inclusive.profit)} ({pct(both.inclusive.marginBasisPoints)} margin / {pct(both.inclusive.markupBasisPoints)} markup)
          </div>
        )}
      </div>
    )
  }

  return (
    <FormCard title="Cost & margin insight" description="Cost price is recomputed daily from purchase order receiving history; not a live figure.">
      {loading && <p className="text-xs text-ink-4">Loading…</p>}
      {!loading && error && <p className="text-xs text-danger">{error}</p>}
      {!loading && !error && insight && !insight.cost_price_available && (
        <p className="text-xs text-ink-4">
          No cost price yet — this SKU hasn't been received against a purchase order, or the daily cost-price job hasn't run since.
        </p>
      )}
      {!loading && !error && insight && insight.cost_price_available && (
        <div className="grid grid-cols-2 gap-x-4 gap-y-3 text-xs">
          <div>
            <div className="text-ink-4">Cost price</div>
            <div className="font-medium text-ink">{money(insight.cost_price_minor_units)}</div>
          </div>
          <div>
            <div className="text-ink-4">Recommended price</div>
            <div className="font-medium text-ink">
              {money(insight.recommended_price_minor_units)}
              <span className="text-ink-4 font-normal"> ({pct(insight.effective_target_markup_basis_points)} target markup)</span>
            </div>
          </div>
          {profitRow(
            'Profit at current price', insight.profit_at_price_minor_units, insight.margin_at_price_basis_points, insight.markup_at_price_basis_points,
            insight.cost_price_minor_units + insight.profit_at_price_minor_units, insight.cost_price_minor_units,
          )}
          {insight.profit_at_sale_price_minor_units != null && insight.margin_at_sale_price_basis_points != null && insight.markup_at_sale_price_basis_points != null && (
            profitRow(
              'Profit at sale price', insight.profit_at_sale_price_minor_units, insight.margin_at_sale_price_basis_points, insight.markup_at_sale_price_basis_points,
              insight.cost_price_minor_units + insight.profit_at_sale_price_minor_units, insight.cost_price_minor_units,
            )
          )}
          <div>
            <div className="text-ink-4">On hand</div>
            <div className="font-medium text-ink">{insight.total_on_hand_quantity} units</div>
          </div>
          <div>
            <div className="text-ink-4">Total profit potential</div>
            <div className="font-medium text-ink">{money(insight.total_profit_potential_minor_units)}</div>
          </div>
          {insight.shrinkage_minor_units > 0 && (
            <div className="col-span-2">
              <div className="text-ink-4">Shrinkage (damaged/lost, last 30 days)</div>
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

const PRICE_HISTORY_REASON_LABEL: Record<SKUPriceHistoryReason, string> = {
  created: 'Created',
  manual_update: 'Manual edit',
  sale_expired: 'Sale expired',
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
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load price history') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, skuId])

  const money = (minorUnits: number) => formatPrice(minorUnits, currency)

  return (
    <FormCard title="Price history" description="An immutable record of every effective price change -- not the same as the daily-computed cost price above.">
      {loading && <p className="text-xs text-ink-4">Loading…</p>}
      {!loading && error && <p className="text-xs text-danger">{error}</p>}
      {!loading && !error && history && history.length === 0 && (
        <p className="text-xs text-ink-4">No price changes recorded yet.</p>
      )}
      {!loading && !error && history && history.length > 0 && (
        <table className="w-full text-xs">
          <thead>
            <tr className="text-ink-4">
              <th className="text-left font-medium pb-1.5 pr-3">Valid from</th>
              <th className="text-left font-medium pb-1.5 pr-3">Valid until</th>
              <th className="text-left font-medium pb-1.5 pr-3">Price</th>
              <th className="text-left font-medium pb-1.5 pr-3">Sale price</th>
              <th className="text-left font-medium pb-1.5">Reason</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {history.map(entry => (
              <tr key={entry.id}>
                <td className="py-1.5 pr-3 text-ink-2">{formatDateTime(entry.valid_from)}</td>
                <td className="py-1.5 pr-3 text-ink-2">
                  {entry.valid_until ? formatDateTime(entry.valid_until) : <span className="text-success font-medium">Current</span>}
                </td>
                <td className="py-1.5 pr-3 text-ink font-medium">{money(entry.price_minor_units)}</td>
                <td className="py-1.5 pr-3 text-ink">
                  {entry.sale_price_minor_units != null ? money(entry.sale_price_minor_units) : '—'}
                </td>
                <td className="py-1.5 text-ink-3">{PRICE_HISTORY_REASON_LABEL[entry.reason]}</td>
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
      .catch(err => { if (!cancelled) setError(err instanceof Error ? err.message : 'Failed to load change history') })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, productId])

  return (
    <FormCard title="Change history" description="Name and status changes -- an immutable audit trail.">
      {loading && <p className="text-xs text-ink-4">Loading…</p>}
      {!loading && error && <p className="text-xs text-danger">{error}</p>}
      {!loading && !error && history && history.length === 0 && (
        <p className="text-xs text-ink-4">No changes recorded yet.</p>
      )}
      {!loading && !error && history && history.length > 0 && (
        <table className="w-full text-xs">
          <thead>
            <tr className="text-ink-4">
              <th className="text-left font-medium pb-1.5 pr-3">When</th>
              <th className="text-left font-medium pb-1.5 pr-3">Field</th>
              <th className="text-left font-medium pb-1.5 pr-3">From</th>
              <th className="text-left font-medium pb-1.5">To</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-line/60">
            {history.map(entry => (
              <tr key={entry.id}>
                <td className="py-1.5 pr-3 text-ink-2">{formatDateTime(entry.changed_at)}</td>
                <td className="py-1.5 pr-3 text-ink-3 capitalize">{entry.field}</td>
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
// SKU the daily cron has no basis to compute at all (e.g. a purchase order
// marked received without ever creating a batch), or to correct a value
// staff know is wrong. Setting one sticks until "Revert to automatic" is
// pressed -- the cron skips any SKU with a manual override entirely (see
// application/pms's RecalculateCostPricesUseCase).
function ManualCostPriceEditor({ skuId, insight, currency, onChanged }: { skuId: string; insight: SKUMarginInsight; currency: string; onChanged: () => void }) {
  const { token } = useAuth()
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
      setError(err instanceof Error ? err.message : 'Failed to set cost price')
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
      setError(err instanceof Error ? err.message : 'Failed to revert cost price')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="pt-3 border-t border-line">
      {isManual ? (
        <div className="flex items-center gap-2 text-xs">
          <span className="text-ink-4">
            Cost price set manually ({currency} {(insight.cost_price_minor_units / 100).toFixed(2)}) -- the daily job won't overwrite it.
          </span>
          <button
            type="button" onClick={handleClear} disabled={saving}
            className="px-2 py-1 rounded-lg dark:bg-night-800 bg-gray-100 text-2xs font-medium text-ink-2 hover:bg-hover transition-colors disabled:opacity-50 shrink-0"
          >
            {saving ? 'Reverting…' : 'Revert to automatic'}
          </button>
        </div>
      ) : (
        <div className="flex items-center gap-1.5">
          <span className="text-2xs text-ink-4 shrink-0">
            {insight.cost_price_available ? 'Override cost price' : 'Set cost price manually'}
          </span>
          <input
            type="number" step="0.01" min="0" placeholder={currency}
            value={value} onChange={e => setValue(e.target.value)}
            className="w-24 px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
          />
          <button
            type="button" onClick={handleSet} disabled={saving || value.trim() === ''}
            className="px-2 py-1 rounded-lg dark:bg-night-800 bg-gray-100 text-2xs font-medium text-ink-2 hover:bg-hover transition-colors disabled:opacity-50"
          >
            {saving ? 'Saving…' : 'Set'}
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
        type="number" step="0.1" min="0" placeholder="Tenant default"
        value={value} onChange={e => { setValue(e.target.value); setSavedMsg(false) }}
        className="w-28 px-3 py-1.5 text-xs rounded-lg bg-raised border border-line text-ink-2 outline-none focus-visible:ring-2 focus-visible:ring-focus/60"
      />
      <span className="text-2xs text-ink-4">% target markup for this category</span>
      <button type="button" onClick={handleSave} disabled={saving} className="px-2 py-1 rounded-lg dark:bg-night-800 bg-gray-100 text-2xs font-medium text-ink-2 hover:bg-hover transition-colors disabled:opacity-50">
        {saving ? 'Saving…' : 'Save'}
      </button>
      {savedMsg && <span className="text-2xs text-success">Saved</span>}
    </div>
  )
}
