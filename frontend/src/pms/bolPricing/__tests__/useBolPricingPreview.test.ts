// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { act, renderHook } from '@testing-library/react'
import type { BolPricingDraft, BolPricingPreview } from '../../../integrations/types'

vi.mock('../../../integrations/api', () => ({ previewBolPricing: vi.fn() }))

import * as integrationsApi from '../../../integrations/api'
import { useBolPricingPreview } from '../useBolPricingPreview'

const draft: BolPricingDraft = { custom_price_minor_units: null, commission_override_basis_points: null, bundle_pricing_enabled: false, bundle_tiers: [] }
const preview: BolPricingPreview = { ean: '0630509892495', regular_price_minor_units: 2500, product_vat_basis_points: 900, shipping_minor_units: 495, levels: [] }

async function flush(ms: number) {
  await act(async () => { await vi.advanceTimersByTimeAsync(ms) })
}

beforeEach(() => {
  vi.useFakeTimers()
  vi.mocked(integrationsApi.previewBolPricing).mockReset()
})
afterEach(() => vi.useRealTimers())

describe('useBolPricingPreview', () => {
  it('fetches the preview for the draft after a short pause', async () => {
    vi.mocked(integrationsApi.previewBolPricing).mockResolvedValue(preview)
    const { result } = renderHook(() => useBolPricingPreview('tok', 'conn-1', 'prod-1', 'sku-1', draft))

    expect(integrationsApi.previewBolPricing).not.toHaveBeenCalled()
    await flush(400)

    expect(integrationsApi.previewBolPricing).toHaveBeenCalledWith('tok', 'conn-1', 'prod-1', 'sku-1', draft)
    expect(result.current.preview).toEqual(preview)
    expect(result.current.error).toBeNull()
  })

  it('waits until typing stops: only the last draft is fetched', async () => {
    vi.mocked(integrationsApi.previewBolPricing).mockResolvedValue(preview)
    const { rerender } = renderHook(({ d }) => useBolPricingPreview('tok', 'conn-1', 'prod-1', 'sku-1', d), { initialProps: { d: draft } })

    await flush(200)
    const typed = { ...draft, custom_price_minor_units: 3000 }
    rerender({ d: typed })
    await flush(400)

    expect(integrationsApi.previewBolPricing).toHaveBeenCalledTimes(1)
    expect(integrationsApi.previewBolPricing).toHaveBeenCalledWith('tok', 'conn-1', 'prod-1', 'sku-1', typed)
  })

  it('shows the server error and drops stale prices', async () => {
    vi.mocked(integrationsApi.previewBolPricing).mockRejectedValue(new Error('each Bol bulk price must be at least 5% below the price before it'))
    const { result } = renderHook(() => useBolPricingPreview('tok', 'conn-1', 'prod-1', 'sku-1', draft))

    await flush(400)

    expect(result.current.preview).toBeNull()
    expect(result.current.error).toContain('5%')
  })

  it('does nothing without a SKU', async () => {
    renderHook(() => useBolPricingPreview('tok', 'conn-1', 'prod-1', null, draft))
    await flush(400)
    expect(integrationsApi.previewBolPricing).not.toHaveBeenCalled()
  })
})
