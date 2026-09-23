// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PickListPrintView } from '../PickListPrintView'
import type { PickList } from '../../oms/types'

vi.mock('../../oms/api', () => ({ getPickList: vi.fn() }))
import * as api from '../../oms/api'

const pickList: PickList = {
  items: [
    {
      sku_id: 'sku-a', sku_code: 'A-1', barcode: '8710001', name: 'Apple', total_quantity: 3,
      orders: [
        { sales_order_id: 'so-1', order_number: '1001', line_item_id: 'li-1', quantity: 2 },
        { sales_order_id: 'so-2', order_number: '1002', line_item_id: 'li-2', quantity: 1 },
      ],
    },
    {
      sku_id: null, sku_code: '', barcode: '', name: 'Gift wrap', total_quantity: 1,
      orders: [{ sales_order_id: 'so-2', order_number: '1002', line_item_id: 'li-3', quantity: 1 }],
    },
  ],
}

beforeEach(() => {
  vi.clearAllMocks()
  window.print = vi.fn()
})

describe('PickListPrintView', () => {
  it('shows one row per SKU with total and per-order breakdown', async () => {
    vi.mocked(api.getPickList).mockResolvedValue(pickList)

    render(<PickListPrintView token="tok" orderIds={['so-1', 'so-2']} onClose={vi.fn()} />)

    const apple = (await screen.findByText('Apple')).closest('tr') as HTMLElement
    expect(within(apple).getByText('A-1')).toBeInTheDocument()
    expect(within(apple).getByText('8710001')).toBeInTheDocument()
    expect(within(apple).getByText('3')).toBeInTheDocument()
    expect(within(apple).getByText('#1001 × 2')).toBeInTheDocument()
    expect(within(apple).getByText('#1002 × 1')).toBeInTheDocument()
    expect(screen.getByText('2 orders · 4 items')).toBeInTheDocument()
  })

  it('flags lines not linked to a catalog SKU', async () => {
    vi.mocked(api.getPickList).mockResolvedValue(pickList)

    render(<PickListPrintView token="tok" orderIds={['so-1', 'so-2']} onClose={vi.fn()} />)

    const wrap = (await screen.findByText('Gift wrap')).closest('tr') as HTMLElement
    expect(within(wrap).getByText('No SKU')).toBeInTheDocument()
  })

  it('prints on demand and closes', async () => {
    vi.mocked(api.getPickList).mockResolvedValue(pickList)
    const onClose = vi.fn()
    const user = userEvent.setup()

    render(<PickListPrintView token="tok" orderIds={['so-1']} onClose={onClose} />)
    await screen.findByText('Apple')
    await user.click(screen.getByRole('button', { name: 'Print' }))
    await user.click(screen.getByRole('button', { name: 'Close' }))

    expect(window.print).toHaveBeenCalledTimes(1)
    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('shows the API error', async () => {
    vi.mocked(api.getPickList).mockRejectedValue(new Error('sales order not found'))

    render(<PickListPrintView token="tok" orderIds={['missing']} onClose={vi.fn()} />)

    expect(await screen.findByRole('alert')).toHaveTextContent('sales order not found')
  })
})
