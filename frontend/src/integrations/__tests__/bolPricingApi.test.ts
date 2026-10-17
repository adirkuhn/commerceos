import { describe, it, expect, vi } from 'vitest'
import * as api from '../api'
import type { APIResponse, BolPricingPreview } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve(body) }))
}

const preview: BolPricingPreview = {
  ean: '0630509892495', regular_price_minor_units: 2500, cost_price_minor_units: 1200,
  product_vat_basis_points: 900, shipping_minor_units: 495, levels: [],
}

describe('upsertBolProductOverride', () => {
  it('sends the bulk price switch and tiers', async () => {
    mockFetch({ product_id: 'p', connection_id: 'c', condition_category: 'NEW', fulfilment_method: 'FBR', bundle_pricing_enabled: true, bundle_tiers: [] })
    await api.upsertBolProductOverride(TOKEN, 'conn-1', 'prod-1', {
      bundlePricingEnabled: true, bundleTiers: [{ quantity: 2, unit_price_minor_units: 2500 }, { quantity: 4, unit_price_minor_units: null }],
    })
    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    const body = JSON.parse(opts.body as string)
    expect(body.bundle_pricing_enabled).toBe(true)
    expect(body.bundle_tiers).toEqual([{ quantity: 2, unit_price_minor_units: 2500 }, { quantity: 4, unit_price_minor_units: null }])
  })
})

describe('previewBolPricing', () => {
  it('posts the draft to the preview endpoint', async () => {
    mockFetch(preview)
    const result = await api.previewBolPricing(TOKEN, 'conn-1', 'prod-1', 'sku-1', {
      custom_price_minor_units: null, bundle_pricing_enabled: true, bundle_tiers: [{ quantity: 2, unit_price_minor_units: null }],
      commission_override_basis_points: null,
    })
    expect(result).toEqual(preview)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/integrations/connections/conn-1/bol-pricing/prod-1/sku-1/preview')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual({
      draft: { custom_price_minor_units: null, bundle_pricing_enabled: true, bundle_tiers: [{ quantity: 2, unit_price_minor_units: null }], commission_override_basis_points: null },
    })
  })
})
