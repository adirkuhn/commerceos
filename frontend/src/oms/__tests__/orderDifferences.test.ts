// @vitest-environment jsdom
import { describe, it, expect, vi, afterEach } from 'vitest'
import { listOrderDifferences, dismissOrderDifference, auditWooOrderDifferences, describeDifference } from '../orderDifferences'
import type { OrderDifference } from '../orderDifferences'

function okFetch(data: unknown) {
  const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 200, json: () => Promise.resolve({ data, error: null }) })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

function difference(overrides: Partial<OrderDifference> = {}): OrderDifference {
  return {
    id: 'd-1', sales_order_id: 'so-1', order_number: '25922', customer_name: 'Ana Silva',
    order_occurred_at: '2026-10-06T10:20:00Z', kind: 'missing', sku_code: '7896015976606',
    name: 'Mistura de Pão de Queijo - Pinduca - 500g', channel_quantity: 1, local_quantity: 0,
    channel_total_minor_units: 0, local_total_minor_units: 0, currency_code: 'EUR', detected_at: '2026-10-09T10:00:00Z',
    ...overrides,
  }
}

afterEach(() => vi.unstubAllGlobals())

describe('orderDifferences api', () => {
  it('lists a page of open differences', async () => {
    const fetchMock = okFetch({ differences: [difference()], total: 1 })

    const page = await listOrderDifferences('tok', 2, 10)

    expect(page.total).toBe(1)
    expect(page.differences[0].id).toBe('d-1')
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/order-differences\?page=2&per_page=10$/)
    expect(fetchMock.mock.calls[0][1].headers.Authorization).toBe('Bearer tok')
  })

  it('dismisses one', async () => {
    const fetchMock = okFetch(null)

    await dismissOrderDifference('tok', 'd-1')

    expect(fetchMock.mock.calls[0][0]).toMatch(/\/order-differences\/d-1\/dismiss$/)
    expect(fetchMock.mock.calls[0][1].method).toBe('POST')
  })

  it('runs one audit batch with the cursor', async () => {
    const fetchMock = okFetch({ checked: 25, errors: [], next_cursor: 'c-2' })

    const result = await auditWooOrderDifferences('tok', 'conn-1', 'c-1')

    expect(result.next_cursor).toBe('c-2')
    expect(fetchMock.mock.calls[0][0]).toMatch(/\/integrations\/connections\/conn-1\/audit-woo-order-differences$/)
    expect(JSON.parse(fetchMock.mock.calls[0][1].body)).toEqual({ cursor: 'c-1' })
  })

  it('throws the API error message', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok: false, status: 404, json: () => Promise.resolve({ data: null, error: { code: 'NOT_FOUND', message: 'order difference not found' } }),
    }))

    await expect(dismissOrderDifference('tok', 'x')).rejects.toThrow('order difference not found')
  })
})

describe('describeDifference', () => {
  it('names the item and both quantities', () => {
    expect(describeDifference(difference())).toEqual({ key: 'missing', values: { name: 'Mistura de Pão de Queijo - Pinduca - 500g', sku: '7896015976606', channel: 1, local: 0 } })
    expect(describeDifference(difference({ kind: 'quantity', channel_quantity: 3, local_quantity: 2 })).key).toBe('quantity')
  })

  it('formats the totals for a total gap', () => {
    const d = describeDifference(difference({ kind: 'total', channel_total_minor_units: 3641, local_total_minor_units: 3294 }))

    expect(d.key).toBe('total')
    expect(d.values.channelTotal).toBe(3641)
    expect(d.values.localTotal).toBe(3294)
  })
})
