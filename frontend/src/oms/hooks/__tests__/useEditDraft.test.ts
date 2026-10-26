// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'

vi.mock('../../../pricelist/api', () => ({ listPriceList: vi.fn() }))
vi.mock('../../../customers/api', () => ({ getCustomer: vi.fn() }))
import { listPriceList } from '../../../pricelist/api'
import { getCustomer } from '../../../customers/api'
import { useEditDraft } from '../useEditDraft'
import type { SalesOrderDetail } from '../../types'

const order = {
  sales_order: { id: 'so-1', source: 'manual', status: 'pending', number: 'CO-1001', warehouse_id: 'wh-1', customer_id: 'c-9', occurred_at: '2026-10-26T10:00:00Z',
    shipping_total_minor_units: 0, tax_total_minor_units: 0, total_minor_units: 299, discount_total_minor_units: 0 },
  line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Mate', sku_code: 'MATE-100', barcode: '', quantity: 1, price_minor_units: 299,
    subtotal_minor_units: 299, subtotal_tax_minor_units: 0, total_minor_units: 299, total_tax_minor_units: 0 }],
} as unknown as SalesOrderDetail

beforeEach(() => vi.clearAllMocks())

describe('useEditDraft', () => {
  it('builds the draft from the price list and loads the customer', async () => {
    vi.mocked(listPriceList).mockResolvedValue({ total: 1, pricesIncludeTax: true, rows: [{ sku_id: 'sku-1', tax_rate_basis_points: 900, available_quantity: 5 } as never] })
    vi.mocked(getCustomer).mockResolvedValue({ id: 'c-9' } as never)

    const { result } = renderHook(() => useEditDraft('tok', order))

    await waitFor(() => expect(result.current.draft).not.toBeNull())
    expect(listPriceList).toHaveBeenCalledWith('tok', { search: 'MATE-100', perPage: 20 })
    expect(result.current.draft!.lines[0].taxRateBasisPoints).toBe(900)
    expect(result.current.customer).toEqual({ id: 'c-9' })
  })

  it('reports a failed load', async () => {
    vi.mocked(listPriceList).mockRejectedValue(new Error('down'))
    vi.mocked(getCustomer).mockResolvedValue({ id: 'c-9' } as never)

    const { result } = renderHook(() => useEditDraft('tok', order))

    await waitFor(() => expect(result.current.error).toBeInstanceOf(Error))
    expect(result.current.draft).toBeNull()
  })
})
