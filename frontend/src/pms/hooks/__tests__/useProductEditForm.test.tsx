// @vitest-environment jsdom
import { act, renderHook, waitFor } from '@testing-library/react'
import type { FormEvent } from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useProductEditForm } from '../useProductEditForm'
import type { Product, ProductDetail, SKU } from '../../types'
import type { TenantSettings } from '../../../settings/types'

vi.mock('../../api', () => ({
  getProduct: vi.fn(),
  createVariant: vi.fn(),
  createSKU: vi.fn(),
  updateSKU: vi.fn(),
}))
import * as pmsApi from '../../api'

const product: Product = {
  id: 'prod-1', tenant_id: 't1', name: 'Tee', slug: 'tee', description: 'd', category_id: 'cat-1',
  status: 'active', has_variants: false, tags: ['a'], created_at: '', updated_at: '',
}
const sku: SKU = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'var-1', sku_code: 'TEE-1', barcode: '123', price_minor_units: 1999,
  currency_code: 'EUR', is_virtual: false, tracks_expiry: true, status: 'active', created_at: '', updated_at: '',
}
const variant = { id: 'var-1', tenant_id: 't1', product_id: 'prod-1', name: 'Tee', position: 0, created_at: '', updated_at: '' }
const settings = { default_currency: 'GBP' } as TenantSettings
const submit = { preventDefault: () => {} } as FormEvent

function setup(productId: string | null, tenantSettings: TenantSettings | null = null) {
  const onCreate = vi.fn((_: unknown) => Promise.resolve({ ...product, id: 'new-1', name: 'New' }))
  const onUpdate = vi.fn((_id: string, _p: unknown) => Promise.resolve({ ...product, slug: 'tee-2' }))
  const hook = renderHook(() => useProductEditForm({ token: 'tok', productId, tenantSettings, onCreate, onUpdate }))
  return { ...hook, onCreate, onUpdate }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(pmsApi.createVariant).mockResolvedValue({ ...variant, id: 'new-var' })
  vi.mocked(pmsApi.createSKU).mockResolvedValue(sku)
  vi.mocked(pmsApi.updateSKU).mockResolvedValue(sku)
})

describe('useProductEditForm', () => {
  it('hydrates product fields and the flat SKU form from a single-SKU product', async () => {
    const detail: ProductDetail = { product, variants: [{ ...variant, skus: [sku] }] }
    vi.mocked(pmsApi.getProduct).mockResolvedValue(detail)
    const { result } = setup('prod-1')

    await waitFor(() => expect(result.current.fields.name).toBe('Tee'))
    expect(result.current.fields).toEqual({ name: 'Tee', slug: 'tee', description: 'd', categoryId: 'cat-1', status: 'active', tags: ['a'] })
    expect(result.current.sku).toMatchObject({ skuCode: 'TEE-1', barcode: '123', price: '19.99', currency: 'EUR' })
    expect(result.current.currentSku?.id).toBe('sku-1')
    expect(result.current.showFlatSKU).toBe(true)
  })

  it('hides the flat form for a has_variants product', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: { ...product, has_variants: true }, variants: [{ ...variant, skus: [sku] }] })
    const { result } = setup('prod-1')
    await waitFor(() => expect(result.current.variantsMode).toBe(true))
    expect(result.current.showFlatSKU).toBe(false)
    expect(result.current.currentSku).toBeNull()
  })

  it('seeds a new product\'s currency from tenant settings', () => {
    const { result } = setup(null, settings)
    expect(result.current.sku.currency).toBe('GBP')
  })

  it('creating a flat product writes product, default variant and SKU, then switches to the saved id', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product: { ...product, id: 'new-1' }, variants: [] })
    const { result, onCreate } = setup(null)
    act(() => {
      result.current.setField('name', 'New')
      result.current.setSkuField('skuCode', 'NEW-1')
      result.current.setSkuField('price', '5')
    })
    await act(() => result.current.handleSave(submit))

    expect(onCreate).toHaveBeenCalledWith(expect.objectContaining({ name: 'New', has_variants: false }))
    expect(onCreate.mock.calls[0][0]).not.toHaveProperty('slug')
    expect(pmsApi.createVariant).toHaveBeenCalledWith('tok', 'new-1', { name: 'New' })
    expect(pmsApi.createSKU).toHaveBeenCalledWith('tok', 'new-var', expect.objectContaining({ sku_code: 'NEW-1', price_minor_units: 500, unit_type: 'unit' }))
    expect(result.current.savedProductId).toBe('new-1')
  })

  it('updating sends only a changed slug, patches fields from the response and updates the SKU', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product, variants: [{ ...variant, skus: [sku] }] })
    const { result, onUpdate } = setup('prod-1')
    await waitFor(() => expect(result.current.currentSku).not.toBeNull())

    await act(() => result.current.handleSave(submit))
    expect(onUpdate.mock.calls[0][1]).not.toHaveProperty('slug')
    expect(result.current.fields.slug).toBe('tee-2')
    expect(pmsApi.updateSKU).toHaveBeenCalledWith('tok', 'sku-1', expect.objectContaining({ sku_code: 'TEE-1', tracks_expiry: true }))

    act(() => result.current.setField('slug', 'fixed'))
    await act(() => result.current.handleSave(submit))
    expect(onUpdate.mock.calls[1][1]).toMatchObject({ slug: 'fixed' })
  })

  it('updating a product with no variants yet creates its variant + SKU', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product, variants: [] })
    const { result } = setup('prod-1')
    await waitFor(() => expect(result.current.fields.name).toBe('Tee'))
    act(() => result.current.setSkuField('skuCode', 'TEE-9'))

    await act(() => result.current.handleSave(submit))
    expect(pmsApi.createVariant).toHaveBeenCalledWith('tok', 'prod-1', { name: 'Tee' })
    expect(pmsApi.createSKU).toHaveBeenCalledWith('tok', 'new-var', expect.objectContaining({ sku_code: 'TEE-9' }))
  })

  it('surfaces a save failure as formError', async () => {
    const { result, onCreate } = setup(null)
    onCreate.mockRejectedValueOnce(new Error('nope'))
    await act(() => result.current.handleSave(submit))
    expect(result.current.formError).toBe('nope')
    expect(result.current.saving).toBe(false)
  })

  it('enabling variants on a new product just flips the mode', async () => {
    const { result, onUpdate } = setup(null)
    await act(() => result.current.handleEnableVariants())
    expect(result.current.variantsMode).toBe(true)
    expect(onUpdate).not.toHaveBeenCalled()
  })

  it('enabling variants on a saved product persists has_variants first, and reports failure', async () => {
    vi.mocked(pmsApi.getProduct).mockResolvedValue({ product, variants: [] })
    const { result, onUpdate } = setup('prod-1')
    await waitFor(() => expect(result.current.fields.name).toBe('Tee'))

    onUpdate.mockRejectedValueOnce(new Error('invariant'))
    await act(() => result.current.handleEnableVariants())
    expect(result.current.enableVariantsError).toBe('invariant')
    expect(result.current.variantsMode).toBe(false)

    await act(() => result.current.handleEnableVariants())
    expect(onUpdate).toHaveBeenLastCalledWith('prod-1', expect.objectContaining({ has_variants: true, name: 'Tee' }))
    expect(result.current.variantsMode).toBe(true)
  })
})
