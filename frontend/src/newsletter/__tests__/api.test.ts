import { describe, it, expect, vi, beforeEach } from 'vitest'
import { eraseSubscriber, exportSubscribersCSV, listSubscribers } from '../api'

const fetchMock = vi.fn<typeof fetch>()

beforeEach(() => {
  fetchMock.mockReset()
  vi.stubGlobal('fetch', fetchMock)
  vi.stubEnv('VITE_API_BASE_URL', 'https://api.test/api/v1')
})

describe('newsletter api', () => {
  it('lists with filters and bearer auth', async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({ data: [{ id: 's1' }], total: 7, error: null })))
    const res = await listSubscribers('tok', { status: 'pending', search: 'ann', page: 2, perPage: 50 })
    expect(res).toEqual({ items: [{ id: 's1' }], total: 7 })
    const [url, init] = fetchMock.mock.calls[0]
    expect(String(url)).toBe('https://api.test/api/v1/newsletter/subscribers?page=2&per_page=50&status=pending&search=ann')
    expect(new Headers(init?.headers).get('Authorization')).toBe('Bearer tok')
  })

  it('downloads the CSV export', async () => {
    fetchMock.mockResolvedValue(new Response('email,confirmed_at\n', { headers: { 'Content-Type': 'text/csv' } }))
    const blob = await exportSubscribersCSV('tok')
    expect(await blob.text()).toBe('email,confirmed_at\n')
    expect(String(fetchMock.mock.calls[0][0])).toBe('https://api.test/api/v1/newsletter/subscribers/export')
  })

  it('erases by id and surfaces API errors', async () => {
    fetchMock.mockResolvedValue(new Response(JSON.stringify({ data: null, error: { code: 'NOT_FOUND', message: 'subscriber not found' } }), { status: 404 }))
    await expect(eraseSubscriber('tok', 's1')).rejects.toThrow('subscriber not found')
    expect(fetchMock.mock.calls[0][1]?.method).toBe('DELETE')
  })
})
