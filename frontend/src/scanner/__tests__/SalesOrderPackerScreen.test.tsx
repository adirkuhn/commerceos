// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SalesOrderPackerScreen } from '../SalesOrderPackerScreen'
import type { SalesOrder } from '../../oms/types'

vi.mock('../../oms/api', () => ({ listPackingQueue: vi.fn() }))
vi.mock('../../team/api', () => ({ listMembers: vi.fn() }))
vi.mock('../api', () => ({ startOrResumePacking: vi.fn() }))
import * as omsApi from '../../oms/api'
import * as api from '../api'
import * as teamApi from '../../team/api'
import type { PackingQueueEntry } from '../../oms/types'

function order(overrides: Partial<SalesOrder>): SalesOrder {
  return {
    id: 'so-1', tenant_id: 't', source: 'woocommerce', external_order_id: 'ext-1', number: '1001', order_key: '',
    status: 'processing', currency_code: 'EUR', warehouse_id: 'wh-1', billing: { first_name: 'Ann', last_name: 'Lee' },
    total_minor_units: 1250, fulfilled: true, packed: false,
    ...overrides,
  } as SalesOrder
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(teamApi.listMembers).mockResolvedValue([])
})

function queue(...orders: SalesOrder[]) {
  return { entries: orders.map((o): PackingQueueEntry => ({ sales_order: o, lock: null })), total: orders.length }
}

// A scanner token for "me" as a regular user.
const TOKEN = `x.${btoa(JSON.stringify({ user_id: 'me', role: 'customer_user' }))}.y`

describe('SalesOrderPackerScreen', () => {
  it('shows the packing queue', async () => {
    vi.mocked(omsApi.listPackingQueue).mockResolvedValue(queue(order({})))

    render(<SalesOrderPackerScreen token={TOKEN} onPackingStarted={vi.fn()} onLogout={vi.fn()} onBack={vi.fn()} />)

    expect(await screen.findByText('Ann Lee')).toBeInTheDocument()
    expect(omsApi.listPackingQueue).toHaveBeenCalledWith(TOKEN, { perPage: 100 })
  })

  it('marks an order that skipped picking', async () => {
    vi.mocked(omsApi.listPackingQueue).mockResolvedValue(queue(order({ id: 'so-1', fulfilled: true }), order({ id: 'so-2', number: '1002', fulfilled: false })))

    render(<SalesOrderPackerScreen token={TOKEN} onPackingStarted={vi.fn()} onLogout={vi.fn()} onBack={vi.fn()} />)

    expect(await screen.findAllByText('Not picked')).toHaveLength(1)
  })

  it('selecting an order starts packing it', async () => {
    const o = order({ fulfilled: false })
    vi.mocked(omsApi.listPackingQueue).mockResolvedValue(queue(o))
    vi.mocked(api.startOrResumePacking).mockResolvedValue({ id: 'pack-1' } as Awaited<ReturnType<typeof api.startOrResumePacking>>)
    const onPackingStarted = vi.fn()
    const user = userEvent.setup()

    render(<SalesOrderPackerScreen token={TOKEN} onPackingStarted={onPackingStarted} onLogout={vi.fn()} onBack={vi.fn()} />)
    await user.click(await screen.findByText('Ann Lee'))

    expect(api.startOrResumePacking).toHaveBeenCalledWith(TOKEN, 'so-1')
    expect(onPackingStarted).toHaveBeenCalledWith('pack-1', o)
  })

  it('shows the start error', async () => {
    vi.mocked(omsApi.listPackingQueue).mockResolvedValue(queue(order({})))
    vi.mocked(api.startOrResumePacking).mockRejectedValue(new Error('sales order has a picking session in progress'))
    const user = userEvent.setup()

    render(<SalesOrderPackerScreen token={TOKEN} onPackingStarted={vi.fn()} onLogout={vi.fn()} onBack={vi.fn()} />)
    await user.click(await screen.findByText('Ann Lee'))

    expect(await screen.findByText('sales order has a picking session in progress')).toBeInTheDocument()
  })

  it('an order someone else holds is read-only and says who and for how long', async () => {
    const o = order({})
    vi.mocked(omsApi.listPackingQueue).mockResolvedValue({
      entries: [{ sales_order: o, lock: { activity: 'picking', user_id: '3f2a9c1e-0000', since: new Date(Date.now() - 3 * 60_000).toISOString() } }], total: 1,
    })
    const user = userEvent.setup()

    render(<SalesOrderPackerScreen token={TOKEN} onPackingStarted={vi.fn()} onLogout={vi.fn()} onBack={vi.fn()} />)
    await user.click(await screen.findByText('Ann Lee'))

    expect(screen.getByText('Picking · User 3f2a9c · 3 min')).toBeInTheDocument()
    expect(api.startOrResumePacking).not.toHaveBeenCalled()
  })

  it('an admin may take over an order someone else holds', async () => {
    const o = order({})
    vi.mocked(omsApi.listPackingQueue).mockResolvedValue({
      entries: [{ sales_order: o, lock: { activity: 'packing', user_id: 'other', since: new Date().toISOString() } }], total: 1,
    })
    vi.mocked(api.startOrResumePacking).mockResolvedValue({ id: 'pack-1' } as Awaited<ReturnType<typeof api.startOrResumePacking>>)
    const adminToken = `x.${btoa(JSON.stringify({ user_id: 'me', role: 'customer_owner' }))}.y`
    const onPackingStarted = vi.fn()
    const user = userEvent.setup()

    render(<SalesOrderPackerScreen token={adminToken} onPackingStarted={onPackingStarted} onLogout={vi.fn()} onBack={vi.fn()} />)
    expect(await screen.findByText('· Take over')).toBeInTheDocument()
    await user.click(screen.getByText('Ann Lee'))

    expect(api.startOrResumePacking).toHaveBeenCalledWith(adminToken, 'so-1')
    expect(onPackingStarted).toHaveBeenCalledWith('pack-1', o)
  })
})
