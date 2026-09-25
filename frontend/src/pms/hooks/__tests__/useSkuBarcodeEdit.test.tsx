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
})
