import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Review, APIResponse } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 404) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  }))
}

const sampleReview: Review = {
  id: 'review-1', sales_channel_id: 'channel-1', product_id: 'product-1',
  customer_name: 'Ada', customer_email: 'ada@example.com', rating: 5,
  title: 'Great', body: 'Loved it', verified: true, created_at: '2026-06-01T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listReviews', () => {
  it('returns array of reviews', async () => {
    mockFetch([sampleReview])
    const result = await api.listReviews(TOKEN)
    expect(result).toEqual([sampleReview])
  })

  it('calls correct endpoint', async () => {
    mockFetch([])
    await api.listReviews(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/reviews',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listReviews(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('deleteReview', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.deleteReview(TOKEN, 'review-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/reviews/review-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'review not found', 404)
    await expect(api.deleteReview(TOKEN, 'missing')).rejects.toThrow('review not found')
  })
})
