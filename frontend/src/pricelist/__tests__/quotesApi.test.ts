// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { createQuote, deleteQuote, getQuote, listQuotes, setAutomaticCostPrice, updateQuote } from '../quotesApi'
import type { QuoteInput, SavedQuote } from '../types'

function mockFetch(body: unknown, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue({ ok: status < 300, status, json: () => Promise.resolve(body) })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

const quote: SavedQuote = {
  id: 'q-1', number: 7, reference: 'Q-0007', name: 'Sabor', currency_code: 'EUR', discount_basis_points: 3000,
  line_count: 0, total_minor_units: 0, created_at: '', updated_at: '', lines: [],
}
const input: QuoteInput = { name: 'Sabor', discount_basis_points: 3000, currency_code: 'EUR', lines: [] }

describe('quotes API', () => {
  it('lists with paging and search', async () => {
    const fetchMock = mockFetch({ data: [quote], meta: { total: 9 }, error: null })

    expect(await listQuotes('tok', { page: 2, perPage: 20, search: 'sab' })).toEqual({ quotes: [quote], total: 9 })
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/quotes\?page=2&per_page=20&search=sab$/)
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe('Bearer tok')
  })

  it('creates, reads and saves over', async () => {
    let fetchMock = mockFetch({ data: quote, error: null }, 201)
    expect(await createQuote('tok', input)).toEqual(quote)
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/quotes$/)
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'POST', body: JSON.stringify(input) })

    fetchMock = mockFetch({ data: quote, error: null })
    await getQuote('tok', 'q-1')
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/quotes\/q-1$/)

    fetchMock = mockFetch({ data: quote, error: null })
    await updateQuote('tok', 'q-1', input)
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'PUT', body: JSON.stringify(input) })
  })

  it('deletes (204, no body)', async () => {
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204, json: () => Promise.reject(new Error('no body')) })
    vi.stubGlobal('fetch', fetchMock)

    await deleteQuote('tok', 'q-1')

    expect(fetchMock.mock.calls[0][1].method).toBe('DELETE')
  })

  it('throws the coded API error', async () => {
    mockFetch({ data: null, error: { code: 'QUOTE_NOT_FOUND', message: 'quote not found' } }, 404)
    await expect(getQuote('tok', 'q-x')).rejects.toMatchObject({ code: 'QUOTE_NOT_FOUND' })
  })

  it('sets a SKU cost that receipts keep updating', async () => {
    const fetchMock = mockFetch({ data: { id: 's-1' }, error: null })

    await setAutomaticCostPrice('tok', 's-1', 450)

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/pms\/skus\/s-1\/cost-price\/automatic$/)
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'PUT', body: JSON.stringify({ cost_price_minor_units: 450 }) })
  })
})
