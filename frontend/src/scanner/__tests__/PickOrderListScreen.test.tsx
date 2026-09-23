// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PickOrderListScreen } from '../PickOrderListScreen'
import type { SalesOrder, SalesOrderDetail } from '../../oms/types'

vi.mock('../../oms/api', () => ({ getSalesOrder: vi.fn(), markOrderPicked: vi.fn() }))
import * as omsApi from '../../oms/api'

const detail: SalesOrderDetail = {
  sales_order: { id: 'so-1', number: '1001', external_order_id: 'e', billing: { first_name: 'Ann', last_name: 'Lee' } } as SalesOrder,
  line_items: [{ id: 'li-1', name: 'Apple', sku_code: 'A-1', barcode: '', quantity: 2, price_minor_units: 1 } as SalesOrderDetail['line_items'][number]],
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail)
})

describe('PickOrderListScreen', () => {
  it('shows the order items to pick', async () => {
    render(<PickOrderListScreen token="tok" orderId="so-1" onDone={vi.fn()} onBack={vi.fn()} />)

    const items = await screen.findByRole('list', { name: 'Items to pick' })
    expect(within(items).getByText('Apple')).toBeInTheDocument()
    expect(within(items).getByText('× 2')).toBeInTheDocument()
    expect(screen.getByText('#1001')).toBeInTheDocument()
  })

  it('mark as picked finishes the order', async () => {
    vi.mocked(omsApi.markOrderPicked).mockResolvedValue(null)
    const onDone = vi.fn()
    const user = userEvent.setup()
    render(<PickOrderListScreen token="tok" orderId="so-1" onDone={onDone} onBack={vi.fn()} />)

    await user.click(await screen.findByRole('button', { name: 'Mark as picked' }))

    expect(omsApi.markOrderPicked).toHaveBeenCalledWith('tok', 'so-1')
    expect(onDone).toHaveBeenCalled()
  })

  it('shows the error when marking fails', async () => {
    vi.mocked(omsApi.markOrderPicked).mockRejectedValue(new Error('sales order is already fulfilled'))
    const user = userEvent.setup()
    render(<PickOrderListScreen token="tok" orderId="so-1" onDone={vi.fn()} onBack={vi.fn()} />)

    await user.click(await screen.findByRole('button', { name: 'Mark as picked' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('sales order is already fulfilled')
  })

  it('shows a load error', async () => {
    vi.mocked(omsApi.getSalesOrder).mockRejectedValue(new Error('sales order not found'))
    render(<PickOrderListScreen token="tok" orderId="missing" onDone={vi.fn()} onBack={vi.fn()} />)

    expect(await screen.findByRole('alert')).toHaveTextContent('sales order not found')
  })
})
