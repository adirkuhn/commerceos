// @vitest-environment jsdom
import { act, renderHook } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import { useSkuBarcodeEdit } from '../useSkuBarcodeEdit'
import type { SKU, UpdateSKUPayload } from '../../types'

const sku: SKU = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'var-1', sku_code: 'A-1', barcode: '123', price_minor_units: 1999,
  currency_code: 'EUR', is_virtual: false, tracks_expiry: true, tax_class_id: 'tc', status: 'active', created_at: '', updated_at: '',
}

describe('useSkuBarcodeEdit', () => {
  it('prefills from the SKU once it exists', () => {
    const { result, rerender } = renderHook(({ s }: { s: SKU | undefined }) => useSkuBarcodeEdit(s, vi.fn()), { initialProps: { s: undefined as SKU | undefined } })
    expect(result.current.value).toBe('')
    rerender({ s: sku })
    expect(result.current.value).toBe('123')
  })

  it('saves the new barcode with every other field passed through (no tax class)', async () => {
    const updateSKU = vi.fn((_id: string, _p: UpdateSKUPayload) => Promise.resolve())
    const { result } = renderHook(() => useSkuBarcodeEdit(sku, updateSKU))
    act(() => result.current.setValue('999'))
    await act(() => result.current.save())
    expect(updateSKU).toHaveBeenCalledWith('sku-1', expect.objectContaining({ barcode: '999', sku_code: 'A-1', tracks_expiry: true }))
    expect(updateSKU.mock.calls[0][1]).not.toHaveProperty('tax_class_id')
  })

  it('clearing the barcode sends undefined; a failure is reported', async () => {
    const updateSKU = vi.fn(() => Promise.reject(new Error('dup')))
    const { result } = renderHook(() => useSkuBarcodeEdit(sku, updateSKU))
    act(() => result.current.setValue(''))
    await act(() => result.current.save())
    expect(updateSKU).toHaveBeenCalledWith('sku-1', expect.objectContaining({ barcode: undefined }))
    expect(result.current.error).toBe('dup')
    expect(result.current.saving).toBe(false)
  })

  // A variant's SKU can be removed and a new one added without the row
  // unmounting -- the field must then show the NEW SKU's barcode (and its
  // stale error must go), not keep the old one with a stray Save button.
  it('re-prefills when the row gets a different SKU', () => {
    const { result, rerender } = renderHook(({ s }: { s: SKU | undefined }) => useSkuBarcodeEdit(s, vi.fn()), { initialProps: { s: sku as SKU | undefined } })
    act(() => result.current.setValue('typed-but-not-saved'))
    rerender({ s: undefined })
    rerender({ s: { ...sku, id: 'sku-2', barcode: '456' } })
    expect(result.current.value).toBe('456')
  })

  it('keeps what was saved when the same SKU refreshes after a save', async () => {
    const updateSKU = vi.fn(() => Promise.resolve())
    const { result, rerender } = renderHook(({ s }: { s: SKU }) => useSkuBarcodeEdit(s, updateSKU), { initialProps: { s: sku } })
    act(() => result.current.setValue('999'))
    await act(() => result.current.save())
    rerender({ s: { ...sku, barcode: '999', updated_at: 'later' } })
    expect(result.current.value).toBe('999')
  })
})
