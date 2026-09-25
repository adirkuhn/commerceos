import { useState } from 'react'
import { X } from 'lucide-react'
import type { SalesChannel, QuantityBreak } from '../types'
import type { UpdateSalesChannelPayload } from '../api'

// priceAdjustmentBasisPointsOrError converts the form's percent string to
// basis points (e.g. "20" -> 2000), or throws a message-carrying Error for
// an out-of-range value -- caught by handleSubmit the same way every other
// save error is.
function priceAdjustmentBasisPointsOrError(percent: string): number | null {
  if (percent.trim() === '') return null
  const value = Math.round(parseFloat(percent) * 100)
  if (!Number.isFinite(value) || value < 0 || value > 10000) {
    throw new Error('Discount must be between 0 and 100%.')
  }
  return value
}

// quantityBreaksOrError mirrors priceAdjustmentBasisPointsOrError's
// convert-and-validate-at-submit convention for the repeatable tier rows
// -- a positive integer minimum quantity, a 0-100% discount, and no two
// rows sharing the same minimum (which one would even apply is undefined
// otherwise, same rule the backend enforces).
function quantityBreaksOrError(rows: { minQuantity: string; discountPercent: string }[]): QuantityBreak[] {
  const seen = new Set<number>()
  return rows.map(row => {
    const minQuantity = parseInt(row.minQuantity, 10)
    if (!Number.isFinite(minQuantity) || minQuantity <= 0) {
      throw new Error('Each quantity tier needs a minimum quantity of at least 1.')
    }
    if (seen.has(minQuantity)) {
      throw new Error(`Two quantity tiers both start at ${minQuantity} -- each minimum quantity must be unique.`)
    }
    seen.add(minQuantity)
    const discountBasisPoints = Math.round(parseFloat(row.discountPercent) * 100)
    if (!Number.isFinite(discountBasisPoints) || discountBasisPoints < 0 || discountBasisPoints > 10000) {
      throw new Error("Each quantity tier's discount must be between 0 and 100%.")
    }
    return { min_quantity: minQuantity, discount_basis_points: discountBasisPoints }
  })
}

interface PricingSectionProps {
  channel: SalesChannel
  update: (id: string, payload: UpdateSalesChannelPayload) => Promise<SalesChannel>
}

// PricingSection lives on the Products tab, not the create/edit channel
// form -- a discount/quantity tier is a pricing-on-this-catalog concern,
// same place a merchant already manages which products are even on this
// channel. Every save is a full-state update (see UpdateSalesChannelPayload
// itself, and applyRule's identical pattern above) -- name/enrollment/
// active are resent unchanged from the channel prop, only pricing fields
// come from this section's own local state.
export function PricingSection({ channel, update }: PricingSectionProps) {
  const [percent, setPercent] = useState(
    channel.price_adjustment_basis_points ? (channel.price_adjustment_basis_points / 100).toString() : '',
  )
  const [quantityBreaks, setQuantityBreaks] = useState(
    (channel.quantity_breaks ?? []).map(b => ({ minQuantity: b.min_quantity.toString(), discountPercent: (b.discount_basis_points / 100).toString() })),
  )
  const [dirty, setDirty] = useState(false)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState('')

  const handleSave = async () => {
    setError('')
    setSaving(true)
    try {
      await update(channel.id, {
        name: channel.name,
        connectionId: channel.connection_id,
        enrollmentMode: channel.enrollment_mode,
        enrollmentCategoryId: channel.enrollment_category_id ?? null,
        enrollmentTag: channel.enrollment_tag ?? null,
        priceAdjustmentBasisPoints: priceAdjustmentBasisPointsOrError(percent),
        quantityBreaks: quantityBreaksOrError(quantityBreaks),
        active: channel.active,
      })
      setDirty(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save pricing')
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="mx-5 mt-4 p-4 rounded-xl border border-line bg-raised shrink-0">
      <h3 className="text-xs font-semibold text-ink mb-3">Pricing</h3>
      <div>
        <label htmlFor="pricing-discount" className="block text-2xs text-ink-4 uppercase tracking-wider mb-1.5">
          Discount off list price (%)
        </label>
        <input
          id="pricing-discount"
          type="number" step={0.01}
          value={percent}
          onChange={e => { setPercent(e.target.value); setDirty(true) }}
          placeholder="0"
          className="w-full max-w-xs px-3 py-2 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
        />
        <p className="mt-1.5 text-2xs text-ink-4">
          Applied to every product's price on this storefront -- e.g. 20 for a wholesale channel selling at 20% off retail.
        </p>
      </div>
      <div className="mt-4">
        <div className="flex items-center justify-between mb-1.5">
          <label className="block text-2xs text-ink-4 uppercase tracking-wider">Quantity tiers</label>
          <button
            type="button"
            onClick={() => { setQuantityBreaks(b => [...b, { minQuantity: '', discountPercent: '' }]); setDirty(true) }}
            className="text-2xs text-accent hover:text-accent"
          >
            + Add tier
          </button>
        </div>
        {quantityBreaks.length === 0 && (
          <p className="text-2xs text-ink-4">No quantity tiers -- every order uses the flat discount above.</p>
        )}
        <div className="space-y-2">
          {quantityBreaks.map((row, i) => (
            <div key={i} className="flex items-center gap-2">
              <input
                type="number" min={1} step={1}
                aria-label={`Quantity tier ${i + 1} minimum quantity`}
                value={row.minQuantity}
                onChange={e => {
                  setQuantityBreaks(qb => qb.map((r, ri) => (ri === i ? { ...r, minQuantity: e.target.value } : r)))
                  setDirty(true)
                }}
                placeholder="Qty"
                className="w-20 px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
              />
              <span className="text-2xs text-ink-4">units or more →</span>
              <input
                type="number" step={0.01}
                aria-label={`Quantity tier ${i + 1} discount percent`}
                value={row.discountPercent}
                onChange={e => {
                  setQuantityBreaks(qb => qb.map((r, ri) => (ri === i ? { ...r, discountPercent: e.target.value } : r)))
                  setDirty(true)
                }}
                placeholder="0"
                className="w-20 px-2 py-1.5 text-xs rounded-lg bg-surface border border-line text-ink-2 outline-none focus:ring-1 focus:ring-focus"
              />
              <span className="text-2xs text-ink-4">% off</span>
              <button
                type="button"
                onClick={() => { setQuantityBreaks(qb => qb.filter((_, ri) => ri !== i)); setDirty(true) }}
                aria-label={`Remove quantity tier ${i + 1}`}
                className="ml-auto text-ink-4 hover:text-danger"
              >
                <X size={12} />
              </button>
            </div>
          ))}
        </div>
        <p className="mt-1.5 text-2xs text-ink-4">
          Buy this many or more of one SKU in a single cart line and get this bigger discount instead of the flat rate above.
        </p>
      </div>
      {error && <p className="mt-3 text-xs text-danger bg-red-50 dark:bg-red-500/10 rounded-lg px-2 py-1.5">{error}</p>}
      <button
        onClick={handleSave}
        disabled={saving || !dirty}
        className="mt-3 px-4 py-1.5 rounded-lg bg-accent-solid text-white text-xs font-medium hover:bg-accent-solid-hover disabled:opacity-50 transition-colors"
      >
        {saving ? 'Saving…' : 'Save pricing'}
      </button>
    </div>
  )
}
