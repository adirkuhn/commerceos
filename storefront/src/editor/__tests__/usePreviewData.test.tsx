// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { act, renderHook, waitFor } from '@testing-library/react'
import { StorefrontAPI, type ProductDetail } from '../../api/storefront'
import { PreviewStore } from '../previewStore'
import { usePreviewData } from '../usePreviewData'
import type { DocRef } from '../useDraft'

const detail = (slug: string): ProductDetail => ({ id: slug, name: slug, slug, description: '', category_id: null, tags: [], images: [], variants: [] })

function store() {
  const s = new StorefrontAPI({ baseURL: 'x', storeKey: 'k' })
  s.listProducts = vi.fn(async () => ({ items: [{ id: 'a', name: 'A', slug: 'first', category_id: null, tags: [], image_url: null }], total: 30 }))
  s.getProductBySlug = vi.fn(async (slug: string) => detail(slug))
  s.listCollections = vi.fn(async () => [
    { id: 'c1', name: 'Summer', slug: 'summer', description: '', seo_title: '', seo_description: '' },
    { id: 'c2', name: 'Winter', slug: 'winter', description: '', seo_title: '', seo_description: '' },
  ])
  return s
}

describe('usePreviewData', () => {
  it('previews the product template with the first product, then the chosen one', async () => {
    const preview = new PreviewStore()
    const { result } = renderHook(() => usePreviewData(store(), preview, { kind: 'template', key: 'product' }))
    await waitFor(() => expect(preview.snapshot().product?.slug).toBe('first'))

    act(() => result.current.chooseProduct('other'))
    await waitFor(() => expect(preview.snapshot().product?.slug).toBe('other'))
  })

  it('previews collections with a real listing', async () => {
    const preview = new PreviewStore()
    const s = store()
    const { result } = renderHook(() => usePreviewData(s, preview, { kind: 'template', key: 'collection' }))
    await waitFor(() => expect(preview.snapshot().listing?.title).toBe('Summer'))
    expect(preview.snapshot().listing?.pageCount).toBe(3)

    act(() => result.current.chooseCollection('winter'))
    await waitFor(() => expect(preview.snapshot().listing?.title).toBe('Winter'))
    expect(s.listProducts).toHaveBeenLastCalledWith({ collectionId: 'c2', search: undefined, perPage: 12 })
  })

  it('clears preview data on other documents', async () => {
    const preview = new PreviewStore()
    preview.set({ product: detail('x') })
    const doc: DocRef = { kind: 'page', key: 'about' }
    renderHook(() => usePreviewData(store(), preview, doc))
    await waitFor(() => expect(preview.snapshot()).toEqual({}))
  })
})
