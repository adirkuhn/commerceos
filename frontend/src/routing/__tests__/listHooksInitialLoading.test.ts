// @vitest-environment jsdom
import { renderHook } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { useConnections } from '../../integrations/hooks/useConnections'
import { useSalesChannels } from '../../integrations/hooks/useSalesChannels'
import { useCoupons } from '../../coupons/hooks/useCoupons'
import { useWarehouses } from '../../inventory/hooks/useWarehouses'
import { useTaxClasses } from '../../tax/hooks/useTaxClasses'

vi.mock('../../integrations/api', () => ({ listConnections: vi.fn(() => new Promise(() => {})), listSalesChannels: vi.fn(() => new Promise(() => {})) }))
vi.mock('../../coupons/api', () => ({ listCoupons: vi.fn(() => new Promise(() => {})) }))
vi.mock('../../inventory/api', () => ({ listWarehouses: vi.fn(() => new Promise(() => {})) }))
vi.mock('../../tax/api', () => ({ listTaxClasses: vi.fn(() => new Promise(() => {})) }))

// Pages resolve a record from the URL (/coupons/<id>, ...) against these
// lists and show "not found" when it's missing and the list isn't loading.
// The very first render -- before the fetch effect runs -- must already
// report loading, or a deep link flashes "not found" before the data lands.
describe.each([
  ['useConnections', useConnections],
  ['useSalesChannels', useSalesChannels],
  ['useCoupons', useCoupons],
  ['useWarehouses', useWarehouses],
  ['useTaxClasses', useTaxClasses],
] as const)('%s', (_name, useList) => {
  it('reports loading on the first render when it has a token to load with', () => {
    const renders: boolean[] = []
    renderHook(() => { renders.push(useList('tok').loading) })
    expect(renders[0]).toBe(true)
  })

  it('does not report loading without a token (nothing will load)', () => {
    const { result } = renderHook(() => useList(''))
    expect(result.current.loading).toBe(false)
  })
})
