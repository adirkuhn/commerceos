// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ReadOnlyOrderItems } from '../ReadOnlyOrderItems'
import type { SalesOrder, SalesOrderDetail } from '../../oms/types'

vi.mock('../../oms/api', () => ({ getSalesOrder: vi.fn() }))
import * as omsApi from '../../oms/api'

const detail: SalesOrderDetail = {
  sales_order: { id: 'so-1', number: '1001', external_order_id: 'e', billing: { first_name: 'Ann', last_name: 'Lee' } } as SalesOrder,
  line_items: [
    { id: 'li-1', name: 'Apple', sku_code: 'A-1', barcode: '871', quantity: 3, refunded_quantity: 1, price_minor_units: 1 } as SalesOrderDetail['line_items'][number],
  ],
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail)
})

describe('ReadOnlyOrderItems', () => {
  it('shows the order items with who is packing it, and no actions', async () => {
    const since = new Date(Date.now() - 5 * 60_000).toISOString()
    render(<ReadOnlyOrderItems token="tok" orderId="so-1" lock={{ activity: 'packing', user_id: 'u-9', since }} nameFor={() => 'Dana Box'} />)

    const items = await screen.findByRole('list', { name: 'Items to pick' })
    expect(within(items).getByText('Apple')).toBeInTheDocument()
    expect(within(items).getByText('× 2')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('Packing · Dana Box · 5 min -- view only')
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('says the order is in packing when nobody holds it right now', async () => {
    render(<ReadOnlyOrderItems token="tok" orderId="so-1" lock={null} nameFor={id => id} />)

    expect(await screen.findByRole('status')).toHaveTextContent('This order is in packing -- view only')
  })

  it('shows a load error', async () => {
    vi.mocked(omsApi.getSalesOrder).mockRejectedValue(new Error('sales order not found'))
    render(<ReadOnlyOrderItems token="tok" orderId="x" lock={null} nameFor={id => id} />)

    expect(await screen.findByRole('alert')).toHaveTextContent('sales order not found')
  })
})
