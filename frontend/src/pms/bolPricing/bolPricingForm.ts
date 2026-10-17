import type { BolBundleTier, BolPricingDraft, BolProductOverride } from '../../integrations/types'
import type { UpsertBolProductOverridePayload } from '../../integrations/api'

// The Bol tab's price settings as typed: money as text so a half-typed
// value survives re-renders; an empty price means "automatic".
export interface BolTierRow {
  quantity: string
  price: string
}

export interface BolPricingForm {
  customPrice: string
  // Bol fee % the merchant assumes instead of Bol's live fee; '' = live.
  commissionPercent: string
  bundleEnabled: boolean
  tiers: BolTierRow[]
}

const DEFAULT_TIER_QUANTITIES = [2, 4, 6]
// Bol takes at most four bundle prices, quantity 1 included.
export const BOL_MAX_TIERS = 3
const TIER_QUANTITY_STEP = 2

export function emptyBolPricingForm(): BolPricingForm {
  return { customPrice: '', commissionPercent: '', bundleEnabled: false, tiers: [] }
}

export function bolPricingFormFrom(o: BolProductOverride | null): BolPricingForm {
  if (!o) return emptyBolPricingForm()
  return {
    customPrice: o.custom_price_minor_units != null ? toInput(o.custom_price_minor_units) : '',
    commissionPercent: o.commission_override_basis_points != null ? toInput(o.commission_override_basis_points) : '',
    bundleEnabled: o.bundle_pricing_enabled,
    tiers: o.bundle_tiers.map(t => ({
      quantity: String(t.quantity),
      price: t.unit_price_minor_units != null ? toInput(t.unit_price_minor_units) : '',
    })),
  }
}

// Switching bulk prices on with no tiers starts with automatic 2/4/6;
// switching off keeps the tiers for later.
export function setBundleEnabled(form: BolPricingForm, enabled: boolean): BolPricingForm {
  const tiers = enabled && form.tiers.length === 0
    ? DEFAULT_TIER_QUANTITIES.map(q => ({ quantity: String(q), price: '' }))
    : form.tiers
  return { ...form, bundleEnabled: enabled, tiers }
}

export function canAddTierRow(form: BolPricingForm): boolean {
  return form.tiers.length < BOL_MAX_TIERS
}

export function addTierRow(form: BolPricingForm): BolPricingForm {
  const last = parseInt(form.tiers[form.tiers.length - 1]?.quantity ?? '', 10)
  const quantity = Number.isNaN(last) ? DEFAULT_TIER_QUANTITIES[0] : last + TIER_QUANTITY_STEP
  return { ...form, tiers: [...form.tiers, { quantity: String(quantity), price: '' }] }
}

export function bolPricingDraft(form: BolPricingForm): BolPricingDraft {
  return {
    custom_price_minor_units: parseMinor(form.customPrice),
    commission_override_basis_points: parseMinor(form.commissionPercent),
    bundle_pricing_enabled: form.bundleEnabled,
    bundle_tiers: toTiers(form.tiers),
  }
}

export function bolPricingPayload(form: BolPricingForm): Pick<UpsertBolProductOverridePayload,
  'customPriceMinorUnits' | 'commissionOverrideBasisPoints' | 'bundlePricingEnabled' | 'bundleTiers'> {
  const draft = bolPricingDraft(form)
  return {
    customPriceMinorUnits: draft.custom_price_minor_units,
    commissionOverrideBasisPoints: draft.commission_override_basis_points,
    bundlePricingEnabled: draft.bundle_pricing_enabled,
    bundleTiers: draft.bundle_tiers,
  }
}

function toTiers(rows: BolTierRow[]): BolBundleTier[] {
  return rows.map(r => ({ quantity: parseInt(r.quantity, 10) || 0, unit_price_minor_units: parseMinor(r.price) }))
}

// "30,00" / "30.00" -> 3000; blank or not a number -> null. Also used for
// percentages (12.5% -> 1250 basis points): same two-decimal shape.
function parseMinor(text: string): number | null {
  const trimmed = text.trim().replace(',', '.')
  if (trimmed === '') return null
  const n = parseFloat(trimmed)
  return Number.isNaN(n) ? null : Math.round(n * 100)
}

function toInput(minor: number): string {
  return (minor / 100).toFixed(2)
}
