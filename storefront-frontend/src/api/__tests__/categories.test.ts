import { describe, it, expect, vi, beforeEach } from 'vitest'
import { listCategories } from '../categories'
import type { APIResponse, StorefrontCategory } from '../../types'

function mockFetch<T>(data: T) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve(body) }))
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listCategories', () => {
  it('returns the tenant\'s category tree', async () => {
    const categories: StorefrontCategory[] = [{ id: 'cat-1', parent_id: null, name: 'Bebidas', slug: 'bebidas', position: 0 }]
    mockFetch(categories)

    expect(await listCategories()).toEqual(categories)
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/categories')
  })
})
