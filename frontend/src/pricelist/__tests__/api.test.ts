// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { listPriceList, setAutomaticCostPrice } from '../api'
import type { PriceListRow } from '../types'

const row: PriceListRow = {
  product_id: 'p-1', product_name: 'Feijão', variant_name: '', sku_id: 's-1', sku_code: 'FEI-1', unit_type: 'unit',
  currency_code: 'EUR', price_minor_units: 999, available_quantity: 3, tax_rate_basis_points: 900, target_markup_basis_points: 0,
}

function mockFetch(body: unknown, status = 200) {
  const fetchMock = vi.fn().mockResolvedValue({ ok: status < 300, status, json: () => Promise.resolve(body) })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

describe('listPriceList', () => {
  it('sends paging, search and product ids with the bearer token', async () => {
    const fetchMock = mockFetch({ data: [row], meta: { total: 4, prices_include_tax: true }, error: null })

    const out = await listPriceList('tok', { page: 2, perPage: 25, search: 'fei', productIds: ['p-1', 'p-2'] })

    expect(out).toEqual({ rows: [row], total: 4, pricesIncludeTax: true })
    const [url, init] = fetchMock.mock.calls[0]
    expect(url).toMatch(/\/pms\/price-list\?page=2&per_page=25&search=fei&product_ids=p-1%2Cp-2$/)
    expect(init.headers.Authorization).toBe('Bearer tok')
  })

  it('omits empty params', async () => {
    const fetchMock = mockFetch({ data: [], meta: { total: 0, prices_include_tax: false }, error: null })

    await listPriceList('tok', {})

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/pms\/price-list$/)
  })

  it('throws the API error', async () => {
    mockFetch({ data: null, error: { code: 'INVALID_REQUEST', message: 'bad product_ids' } }, 400)

    await expect(listPriceList('tok', {})).rejects.toThrow('bad product_ids')
  })
})

describe('setAutomaticCostPrice', () => {
  it('sets a SKU cost that receipts keep updating', async () => {
    const fetchMock = mockFetch({ data: { id: 's-1' }, error: null })

    await setAutomaticCostPrice('tok', 's-1', 450)

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/pms\/skus\/s-1\/cost-price\/automatic$/)
    expect(fetchMock.mock.calls[0][1]).toMatchObject({ method: 'PUT', body: JSON.stringify({ cost_price_minor_units: 450 }) })
  })
})
