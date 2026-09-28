// @vitest-environment jsdom
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act } from 'react'
import { renderAt } from '../../test/renderAt'
import { PickingListMode } from '../PickingListMode'
import type { SalesOrder, SalesOrderDetail } from '../../oms/types'

vi.mock('../../oms/api', () => ({ listPickingQueue: vi.fn(), getSalesOrder: vi.fn(), markOrderPicked: vi.fn() }))
vi.mock('../../team/api', () => ({ listMembers: vi.fn().mockResolvedValue([]) }))
import * as omsApi from '../../oms/api'

function order(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 't', source: 'woocommerce', external_order_id: 'ext-1', number: '1001', order_key: '',
    status: 'processing', currency_code: 'EUR', warehouse_id: 'wh-1', billing: { first_name: 'Ann', last_name: 'Lee' },
    total_minor_units: 1250, fulfilled: false, packed: false,
    ...overrides,
  } as SalesOrder
}

function queue(...orders: SalesOrder[]) {
  return { entries: orders.map(o => ({ sales_order: o, lock: null, in_packing: false })), total: orders.length }
}

const detail: SalesOrderDetail = {
  sales_order: order(),
  line_items: [
    { id: 'li-1', name: 'Apple', sku_code: 'A-1', barcode: '871', quantity: 3, refunded_quantity: 1, price_minor_units: 100, sold_expiry_date: '2027-03-09' } as SalesOrderDetail['line_items'][number],
    { id: 'li-2', name: 'Gift wrap', sku_code: '', barcode: '', quantity: 1, price_minor_units: 0 } as SalesOrderDetail['line_items'][number],
    { id: 'li-3', name: 'Refunded thing', sku_code: 'R', barcode: '', quantity: 1, refunded_quantity: 1, price_minor_units: 0 } as SalesOrderDetail['line_items'][number],
  ],
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.listPickingQueue).mockResolvedValue(queue(order(), order({ id: 'so-2', number: '1002' })))
  vi.mocked(omsApi.getSalesOrder).mockResolvedValue(detail)
})

describe('PickingListMode', () => {
  it('lists the picking queue (unpicked, pickable status, no POS -- backend rule)', async () => {
    renderAt(<PickingListMode token="tok" />, '/picking')

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(omsApi.listPickingQueue).toHaveBeenCalledWith('tok', { perPage: 100 })
  })

  it('opening an order shows what to pick, in the URL', async () => {
    const user = userEvent.setup()
    const { router } = renderAt(<PickingListMode token="tok" />, '/picking')

    await user.click(await screen.findByRole('button', { name: /#1001/ }))

    const items = await screen.findByRole('list', { name: 'Items to pick' })
    expect(within(items).getByText('Apple')).toBeInTheDocument()
    expect(within(items).getByText('× 2')).toBeInTheDocument()
    expect(within(items).getByText('Gift wrap')).toBeInTheDocument()
    expect(within(items).queryByText('Refunded thing')).not.toBeInTheDocument()
    expect(within(items).getByText('Sold exp. 09/03/2027')).toBeInTheDocument()
    expect(within(items).getAllByText(/Sold exp\./)).toHaveLength(1)
    expect(router.state.location.search).toBe('?order=so-1')
  })

  it('mark as picked removes the order from the list and says where it went', async () => {
    vi.mocked(omsApi.markOrderPicked).mockResolvedValue(null)
    const user = userEvent.setup()
    const { router } = renderAt(<PickingListMode token="tok" />, '/picking', '/picking?order=so-1')
    vi.mocked(omsApi.listPickingQueue).mockResolvedValue(queue(order({ id: 'so-2', number: '1002' })))

    await user.click(await screen.findByRole('button', { name: 'Mark as picked' }))

    expect(omsApi.markOrderPicked).toHaveBeenCalledWith('tok', 'so-1')
    expect(await screen.findByRole('status')).toHaveTextContent('#1001 marked as picked')
    expect(screen.queryByRole('button', { name: /#1001/ })).not.toBeInTheDocument()
    expect(router.state.location.search).toBe('')
  })

  it('shows why an order could not be marked picked', async () => {
    vi.mocked(omsApi.markOrderPicked).mockRejectedValue(new Error('sales order has a packing session in progress'))
    const user = userEvent.setup()
    renderAt(<PickingListMode token="tok" />, '/picking', '/picking?order=so-1')

    await user.click(await screen.findByRole('button', { name: 'Mark as picked' }))

    expect(await screen.findByRole('alert')).toHaveTextContent('sales order has a packing session in progress')
  })

  it('Back closes the open order', async () => {
    const user = userEvent.setup()
    const { router } = renderAt(<PickingListMode token="tok" />, '/picking')
    await user.click(await screen.findByRole('button', { name: /#1001/ }))
    await screen.findByRole('list', { name: 'Items to pick' })

    await act(async () => { await router.navigate(-1) })

    expect(screen.queryByRole('list', { name: 'Items to pick' })).not.toBeInTheDocument()
  })

  it('an order packing has opens as a read-only item list, without Mark as picked', async () => {
    vi.mocked(omsApi.listPickingQueue).mockResolvedValue({ entries: [
      { sales_order: order(), lock: { activity: 'packing', user_id: 'u-9', since: new Date().toISOString() }, in_packing: true },
    ], total: 1 })
    const user = userEvent.setup()
    renderAt(<PickingListMode token="tok" />, '/picking')

    const row = await screen.findByRole('button', { name: /#1001/ })
    expect(row).toHaveTextContent('Packing')
    await user.click(row)

    const items = await screen.findByRole('list', { name: 'Items to pick' })
    expect(within(items).getByText('Sold exp. 09/03/2027')).toBeInTheDocument()
    expect(screen.getByRole('status')).toHaveTextContent('view only')
    expect(screen.queryByRole('button', { name: 'Mark as picked' })).not.toBeInTheDocument()
  })
})
