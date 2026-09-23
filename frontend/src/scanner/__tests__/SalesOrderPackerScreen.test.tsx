// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SalesOrderPackerScreen } from '../SalesOrderPackerScreen'
import type { SalesOrder } from '../../oms/types'

vi.mock('../../oms/api', () => ({ listSalesOrders: vi.fn() }))
vi.mock('../api', () => ({ startOrResumePacking: vi.fn() }))
import * as omsApi from '../../oms/api'
import * as api from '../api'

function order(overrides: Partial<SalesOrder>): SalesOrder {
  return {
    id: 'so-1', tenant_id: 't', source: 'woocommerce', external_order_id: 'ext-1', number: '1001', order_key: '',
    status: 'processing', currency_code: 'EUR', warehouse_id: 'wh-1', billing: { first_name: 'Ann', last_name: 'Lee' },
    total_minor_units: 1250, fulfilled: true, packed: false,
    ...overrides,
  } as SalesOrder
}

beforeEach(() => vi.clearAllMocks())

describe('SalesOrderPackerScreen', () => {
  it('asks the backend for orders ready to pack, picked or not', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [order({})], total: 1 })

    render(<SalesOrderPackerScreen token="tok" onPackingStarted={vi.fn()} onLogout={vi.fn()} onBack={vi.fn()} />)

    expect(await screen.findByText('Ann Lee')).toBeInTheDocument()
    expect(omsApi.listSalesOrders).toHaveBeenCalledWith('tok', { readyToPack: true, perPage: 100 })
  })

  it('marks an order that skipped picking', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [order({ id: 'so-1', fulfilled: true }), order({ id: 'so-2', number: '1002', fulfilled: false })], total: 2,
    })

    render(<SalesOrderPackerScreen token="tok" onPackingStarted={vi.fn()} onLogout={vi.fn()} onBack={vi.fn()} />)

    expect(await screen.findAllByText('Not picked')).toHaveLength(1)
  })

  it('selecting an order starts packing it', async () => {
    const o = order({ fulfilled: false })
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [o], total: 1 })
    vi.mocked(api.startOrResumePacking).mockResolvedValue({ id: 'pack-1' } as Awaited<ReturnType<typeof api.startOrResumePacking>>)
    const onPackingStarted = vi.fn()
    const user = userEvent.setup()

    render(<SalesOrderPackerScreen token="tok" onPackingStarted={onPackingStarted} onLogout={vi.fn()} onBack={vi.fn()} />)
    await user.click(await screen.findByText('Ann Lee'))

    expect(api.startOrResumePacking).toHaveBeenCalledWith('tok', 'so-1')
    expect(onPackingStarted).toHaveBeenCalledWith('pack-1', o)
  })

  it('shows the start error', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [order({})], total: 1 })
    vi.mocked(api.startOrResumePacking).mockRejectedValue(new Error('sales order has a picking session in progress'))
    const user = userEvent.setup()

    render(<SalesOrderPackerScreen token="tok" onPackingStarted={vi.fn()} onLogout={vi.fn()} onBack={vi.fn()} />)
    await user.click(await screen.findByText('Ann Lee'))

    expect(await screen.findByText('sales order has a picking session in progress')).toBeInTheDocument()
  })
})
