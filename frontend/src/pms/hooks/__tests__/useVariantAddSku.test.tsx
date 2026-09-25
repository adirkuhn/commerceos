// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { useVariantAddSku } from '../useVariantAddSku'
import type { TenantSettings } from '../../../settings/types'

const settings = { default_currency: 'EUR' } as TenantSettings

describe('useVariantAddSku', () => {
  it('seeds the currency from tenant settings once they arrive', () => {
    const { result, rerender } = renderHook(({ s }: { s: TenantSettings | null }) => useVariantAddSku('var-1', vi.fn(), s), { initialProps: { s: null as TenantSettings | null } })
    expect(result.current.draft.currency).toBe('USD')
    rerender({ s: settings })
    expect(result.current.draft.currency).toBe('EUR')
  })

  it('submits the draft, then closes and resets it', async () => {
    const addSKU = vi.fn(() => Promise.resolve())
    const { result } = renderHook(() => useVariantAddSku('var-1', addSKU, settings))
    act(() => {
      result.current.setOpen(true)
      result.current.setField('skuCode', 'X-1')
      result.current.setField('currency', 'GBP')
    })
    await act(() => result.current.submit())

    expect(addSKU).toHaveBeenCalledWith('var-1', expect.objectContaining({ sku_code: 'X-1', currency_code: 'GBP' }))
    expect(result.current.open).toBe(false)
    expect(result.current.draft.skuCode).toBe('')
    expect(result.current.draft.currency).toBe('EUR')
  })

  it('keeps the draft and shows the error when adding fails', async () => {
    const addSKU = vi.fn(() => Promise.reject(new Error('taken')))
    const { result } = renderHook(() => useVariantAddSku('var-1', addSKU, settings))
    act(() => { result.current.setOpen(true); result.current.setField('skuCode', 'X-1') })
    await act(() => result.current.submit())

    expect(result.current.error).toBe('taken')
    expect(result.current.open).toBe(true)
    expect(result.current.draft.skuCode).toBe('X-1')
    expect(result.current.submitting).toBe(false)
  })

  it('patch merges several fields at once', () => {
    const { result } = renderHook(() => useVariantAddSku('var-1', vi.fn(), null))
    act(() => result.current.patch({ portionedFromSKUID: 'p', portionedFromLabel: 'Pool' }))
    expect(result.current.draft).toMatchObject({ portionedFromSKUID: 'p', portionedFromLabel: 'Pool' })
  })
})
