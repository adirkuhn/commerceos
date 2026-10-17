import { describe, it, expect } from 'vitest'
import type { BolProductOverride } from '../../../integrations/types'
import { addTierRow, canAddTierRow, bolPricingDraft, bolPricingFormFrom, bolPricingPayload, emptyBolPricingForm, setBundleEnabled } from '../bolPricingForm'

const override: BolProductOverride = {
  product_id: 'p', connection_id: 'c', condition_category: 'NEW', fulfilment_method: 'FBR',
  custom_price_minor_units: 3000, commission_override_basis_points: 1250,
  bundle_pricing_enabled: true, bundle_tiers: [{ quantity: 2, unit_price_minor_units: 2500 }, { quantity: 4, unit_price_minor_units: null }],
}

describe('bolPricingFormFrom', () => {
  it('fills the form from a saved override', () => {
    expect(bolPricingFormFrom(override)).toEqual({
      customPrice: '30.00', commissionPercent: '12.50', bundleEnabled: true,
      tiers: [{ quantity: '2', price: '25.00' }, { quantity: '4', price: '' }],
    })
  })

  it('is empty without an override', () => {
    expect(bolPricingFormFrom(null)).toEqual(emptyBolPricingForm())
  })
})

describe('setBundleEnabled', () => {
  it('switching on with no tiers starts with automatic 2/4/6 (Bol allows 3 bulk levels)', () => {
    const form = setBundleEnabled(emptyBolPricingForm(), true)
    expect(form.bundleEnabled).toBe(true)
    expect(form.tiers).toEqual([
      { quantity: '2', price: '' }, { quantity: '4', price: '' }, { quantity: '6', price: '' },
    ])
  })

  it('keeps the tiers when switched off and on again', () => {
    const on = { ...emptyBolPricingForm(), bundleEnabled: true, tiers: [{ quantity: '3', price: '20' }] }
    expect(setBundleEnabled(setBundleEnabled(on, false), true).tiers).toEqual([{ quantity: '3', price: '20' }])
  })
})

describe('addTierRow', () => {
  it('adds the next quantity after the last tier', () => {
    expect(addTierRow({ ...emptyBolPricingForm(), tiers: [{ quantity: '4', price: '' }] }).tiers[1]).toEqual({ quantity: '6', price: '' })
    expect(addTierRow(emptyBolPricingForm()).tiers).toEqual([{ quantity: '2', price: '' }])
  })
})

describe('canAddTierRow', () => {
  it('stops at Bol’s 3 bulk levels', () => {
    const rows = (n: number) => ({ ...emptyBolPricingForm(), tiers: Array.from({ length: n }, (_, i) => ({ quantity: String(i + 2), price: '' })) })
    expect(canAddTierRow(rows(2))).toBe(true)
    expect(canAddTierRow(rows(3))).toBe(false)
  })
})

describe('bolPricingDraft / bolPricingPayload', () => {
  const form = {
    customPrice: '30,00', commissionPercent: '', bundleEnabled: true,
    tiers: [{ quantity: '2', price: '25' }, { quantity: '4', price: ' ' }],
  }

  it('turns the form into the preview draft (comma decimals accepted, blank = automatic)', () => {
    expect(bolPricingDraft(form)).toEqual({
      custom_price_minor_units: 3000, commission_override_basis_points: null, bundle_pricing_enabled: true,
      bundle_tiers: [{ quantity: 2, unit_price_minor_units: 2500 }, { quantity: 4, unit_price_minor_units: null }],
    })
  })

  it('turns the form into the save payload', () => {
    expect(bolPricingPayload({ ...form, commissionPercent: '12.5' })).toEqual({
      customPriceMinorUnits: 3000, commissionOverrideBasisPoints: 1250, bundlePricingEnabled: true,
      bundleTiers: [{ quantity: 2, unit_price_minor_units: 2500 }, { quantity: 4, unit_price_minor_units: null }],
    })
  })

  it('empty custom price means automatic', () => {
    expect(bolPricingDraft({ ...form, customPrice: '' }).custom_price_minor_units).toBeNull()
  })
})
