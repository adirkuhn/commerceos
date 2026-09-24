// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act } from 'react'
import { renderAt } from '../../test/renderAt'
import Packing from '../Packing'
import type { SalesOrder } from '../../oms/types'
import type { PackingSession, PackingSessionDetail, ScanPackItemResult, CompletePackingResult } from '../../scanner/types'

let mockUser: { user_id: string; tenant_id: string; role: string; exp: number; iat: number } | null = null
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: mockUser, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../oms/api', () => ({
  listPackingQueue: vi.fn(),
  getSalesOrder: vi.fn(),
  getPickList: vi.fn(),
}))

vi.mock('../../team/api', () => ({
  listMembers: vi.fn(),
}))

vi.mock('../../scanner/api', () => ({
  startOrResumePacking: vi.fn(),
  getPackingSession: vi.fn(),
  scanPackItem: vi.fn(),
  deletePackScan: vi.fn(),
  touchPackingSession: vi.fn(),
  resetPackingSession: vi.fn(),
  completePacking: vi.fn(),
  releasePackingSessionLock: vi.fn(),
  takeOverPackingSession: vi.fn(),
  substitutePackLine: vi.fn(),
  refundPackLine: vi.fn(),
  resolvePackScanCode: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listProductMedia: vi.fn(),
}))

vi.mock('../../integrations/hooks/useConnections', () => ({
  useConnections: vi.fn(),
}))

vi.mock('../../shipping/api', () => ({
  createShipment: vi.fn(),
  getShippingSettings: vi.fn(),
  getShipmentLabel: vi.fn(),
  listShipmentsForOrder: vi.fn(),
  lookupAddress: vi.fn(),
}))

vi.mock('../../scanner/errorSound', () => ({
  playErrorSound: vi.fn(),
}))

import * as omsApi from '../../oms/api'
import * as teamApi from '../../team/api'
import * as packingApi from '../../scanner/api'
import * as pmsApi from '../../pms/api'
import * as shippingApi from '../../shipping/api'
import { useConnections } from '../../integrations/hooks/useConnections'

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1',
    number: '1001', order_key: '', status: 'processing', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    fulfilled: true, packed: false,
    billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
    ...overrides,
  }
}

function session(overrides: Partial<PackingSession> = {}): PackingSession {
  return { id: 'pack-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '2026-08-22T10:00:00Z', completed_by: '', ...overrides }
}

function detail(overrides: Partial<PackingSessionDetail> = {}): PackingSessionDetail {
  return {
    session: session(),
    line_items: [
      { id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000 },
    ],
    scans: [],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  mockUser = { user_id: 'me', tenant_id: 't', role: 'customer_user', exp: 0, iat: 0 }
  vi.mocked(packingApi.releasePackingSessionLock).mockResolvedValue(null)
  vi.mocked(packingApi.touchPackingSession).mockResolvedValue(null)
  vi.mocked(omsApi.listPackingQueue).mockResolvedValue({ entries: [{ sales_order: salesOrder(), lock: null, in_packing: false }], total: 1 })
  vi.mocked(teamApi.listMembers).mockResolvedValue([])
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(useConnections).mockReturnValue({ connections: [], loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
  vi.mocked(shippingApi.lookupAddress).mockResolvedValue(null)
  vi.mocked(omsApi.getSalesOrder).mockResolvedValue({ sales_order: salesOrder(), line_items: [] })
})

describe('Packing page URLs (refresh + Back)', () => {
  it('selecting an order pushes /packing/:sessionId', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())

    const { router } = renderAt(<Packing />, '/packing')
    await userEvent.click(await screen.findByText('#1001'))

    await waitFor(() => expect(router.state.location.pathname).toBe('/packing/pack-1'))
    expect(await screen.findByText('Widget')).toBeInTheDocument()
  })

  it('/packing/:sessionId reopens that session on load without starting a new one', async () => {
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())

    renderAt(<Packing />, '/packing', '/packing/pack-1')

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(packingApi.getPackingSession).toHaveBeenCalledWith('test-token', 'pack-1')
    expect(packingApi.startOrResumePacking).not.toHaveBeenCalled()
  })

  it('shows an error when the session in the URL cannot be loaded', async () => {
    vi.mocked(packingApi.getPackingSession).mockRejectedValue(new Error('packing session not found'))

    renderAt(<Packing />, '/packing', '/packing/missing')

    expect(await screen.findByText('packing session not found')).toBeInTheDocument()
  })

  it('browser Back from a session returns to the plain order list', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())

    const { router } = renderAt(<Packing />, '/packing')
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await act(() => router.navigate(-1))

    expect(router.state.location.pathname).toBe('/packing')
    expect(await screen.findByText('Select an order to start packing')).toBeInTheDocument()
  })

  it('?order=<id> starts/resumes that order and replaces the URL with the session URL', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())

    const { router } = renderAt(<Packing />, '/packing', '/packing?order=so-1')

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(packingApi.startOrResumePacking).toHaveBeenCalledWith('test-token', 'so-1')
    expect(router.state.location.pathname).toBe('/packing/pack-1')
    expect(router.state.location.search).toBe('')
    expect(router.state.historyAction).toBe('REPLACE')
  })
})

describe('Packing page', () => {
  it('lists the packing queue -- every unpacked order, the backend decides', async () => {
    render(<MemoryRouter><Packing /></MemoryRouter>)

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(vi.mocked(omsApi.listPackingQueue)).toHaveBeenCalledWith('test-token', { perPage: 100 })
  })

  it('an order another user is packing is read-only, naming them and how long', async () => {
    vi.mocked(omsApi.listPackingQueue).mockResolvedValue({
      entries: [{ sales_order: salesOrder(), lock: { activity: 'packing', user_id: 'u-9', since: new Date(Date.now() - 5 * 60_000).toISOString() }, in_packing: false }],
      total: 1,
    })
    vi.mocked(teamApi.listMembers).mockResolvedValue([
      { membership_id: 'm', user_id: 'u-9', email: 'x@y.z', first_name: 'Dana', last_name: 'Box', role: 'customer_user', status: 'active', last_login_at: null },
    ])
    render(<MemoryRouter><Packing /></MemoryRouter>)

    expect(await screen.findByText('Packing · Dana Box · 5 min')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /#1001/ })).not.toBeInTheDocument()
  })

  it('ticking orders and opening the pick list puts it in the URL (Back closes it)', async () => {
    vi.mocked(omsApi.getPickList).mockResolvedValue({
      items: [{ sku_id: 'sku-1', sku_code: 'W-1', barcode: '', name: 'Widget', total_quantity: 2, orders: [] }],
    })
    const user = userEvent.setup()
    const { router } = renderAt(<Packing />, '/packing')

    await user.click(await screen.findByRole('checkbox', { name: 'Add #1001 to pick list' }))
    await user.click(screen.getByRole('button', { name: /Pick list/ }))

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(router.state.location.search).toBe('?pick=so-1')
    expect(omsApi.getPickList).toHaveBeenCalledWith('test-token', ['so-1'])

    await act(async () => { await router.navigate(-1) })
    expect(screen.queryByText('Widget')).not.toBeInTheDocument()
  })

  it('a pick list URL opens straight into the pick list (refresh-safe)', async () => {
    vi.mocked(omsApi.getPickList).mockResolvedValue({ items: [] })
    renderAt(<Packing />, '/packing', '/packing?pick=so-1,so-2')

    expect(await screen.findByRole('heading', { name: 'Pick list' })).toBeInTheDocument()
    expect(omsApi.getPickList).toHaveBeenCalledWith('test-token', ['so-1', 'so-2'])
  })

  it('selecting an order starts/resumes a packing session and shows its line items', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(packingApi.startOrResumePacking).toHaveBeenCalledWith('test-token', 'so-1')
    expect(screen.getByText('0 / 2 items')).toBeInTheDocument()
  })

  it('scanning a code that matches a line item packs it immediately', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession)
      .mockResolvedValueOnce(detail())
      .mockResolvedValueOnce(detail({ scans: [{ id: 'scan-1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pack', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }] }))
    const scanResult: ScanPackItemResult = { scan: { id: 'scan-1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pack', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' } }
    vi.mocked(packingApi.scanPackItem).mockResolvedValue(scanResult)

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.type(screen.getByPlaceholderText('Scan barcode or type and press Enter...'), '111{Enter}')

    await waitFor(() => expect(packingApi.scanPackItem).toHaveBeenCalledWith('test-token', 'pack-1', { code: '111', quantity: 1 }))
    expect(await screen.findByText('1 / 2 items')).toBeInTheDocument()
  })

  it('a scan that fails (SKU not on order, or over-pack) shows a blocking error', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())
    vi.mocked(packingApi.scanPackItem).mockRejectedValue(new Error('scanned item does not match any line item on this order'))

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    const input = screen.getByPlaceholderText('Scan barcode or type and press Enter...')
    await user.type(input, '999{Enter}')

    expect(await screen.findByText('scanned item does not match any line item on this order')).toBeInTheDocument()
    expect(input).toBeDisabled()

    await user.click(screen.getByRole('button', { name: 'OK' }))
    expect(input).not.toBeDisabled()
  })

  it('start over discards the session and reloads it fresh', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession)
      .mockResolvedValueOnce(detail({ scans: [{ id: 'scan-1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pack', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }] }))
      .mockResolvedValueOnce(detail())
    vi.mocked(packingApi.resetPackingSession).mockResolvedValue(session())

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    expect(screen.getByText('1 / 2 items')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Start over' }))
    const confirmButtons = await screen.findAllByRole('button', { name: 'Start over' })
    await user.click(confirmButtons[confirmButtons.length - 1])

    expect(packingApi.resetPackingSession).toHaveBeenCalledWith('test-token', 'pack-1')
    expect(await screen.findByText('0 / 2 items')).toBeInTheDocument()
  })

  it('completing shows missing items and marks the order packed', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())
    const result: CompletePackingResult = {
      completed: true,
      session: session({ status: 'completed' }),
      missing_items: [{ line_item_id: 'li-1', name: 'Widget', expected_qty: 2, packed_qty: 0 }],
    }
    vi.mocked(packingApi.completePacking).mockResolvedValue(result)

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Complete' }))
    await screen.findByText('Review before completing')
    await user.click(screen.getByRole('button', { name: 'Confirm & complete' }))

    expect(await screen.findByText('#1001 packed')).toBeInTheDocument()
    expect(screen.getByText('Widget: 0/2')).toBeInTheDocument()
    expect(packingApi.completePacking).toHaveBeenCalledWith('test-token', 'pack-1', undefined)
  })

  it('a stock shortfall at finish asks per item, then finishes with the decisions', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())
    vi.mocked(packingApi.completePacking)
      .mockResolvedValueOnce({ completed: false, stock_shortfalls: [{ scan_id: 'p9', sku_id: 'sku-9', name: 'Jam jar', needed: 3, short: 2 }] })
      .mockResolvedValueOnce({ completed: true, session: session({ status: 'completed' }), missing_items: [] })

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Complete' }))
    await user.click(await screen.findByRole('button', { name: 'Confirm & complete' }))

    expect(await screen.findByText('Not enough stock')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Skip it' }))
    const confirms = screen.getAllByRole('button', { name: 'Confirm & complete' })
    await user.click(confirms[confirms.length - 1])

    expect(await screen.findByText('#1001 packed')).toBeInTheDocument()
    expect(packingApi.completePacking).toHaveBeenLastCalledWith('test-token', 'pack-1', { p9: 'without_item' })
  })

  it('replacing a line resolves the replacement code and stages it', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())
    vi.mocked(packingApi.resolvePackScanCode).mockResolvedValue({ sku_id: 'sku-sub', product_name: 'Other widget' })
    vi.mocked(packingApi.substitutePackLine).mockResolvedValue({ scans: [] })

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await user.click(await screen.findByText('Widget'))
    await user.click(screen.getByRole('button', { name: /Replace/ }))
    await user.type(screen.getByPlaceholderText('Scan or type replacement SKU/barcode'), 'SUB-1')
    await user.type(screen.getByLabelText('Covers (of original)'), '2')
    await user.click(screen.getByRole('button', { name: 'Confirm replace' }))

    await waitFor(() => expect(packingApi.substitutePackLine).toHaveBeenCalledWith('test-token', 'pack-1', {
      line_item_id: 'li-1',
      lines: [{ sku_id: 'sku-sub', substitute_quantity: 1, original_quantity_covered: 2, return_to_stock: false }],
    }))
    expect(packingApi.resolvePackScanCode).toHaveBeenCalledWith('test-token', 'pack-1', 'SUB-1')
  })

  it('refunding part of a line stages the refund', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())
    vi.mocked(packingApi.refundPackLine).mockResolvedValue({ scan: { id: 'r1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'refund', quantity: 1, scanned_at: '', scanned_by: '' } })

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await user.click(await screen.findByText('Widget'))
    await user.click(screen.getByRole('button', { name: /Refund/ }))
    await user.clear(screen.getByLabelText('Quantity to refund'))
    await user.type(screen.getByLabelText('Quantity to refund'), '1')
    await user.click(screen.getByRole('button', { name: 'Confirm refund' }))

    await waitFor(() => expect(packingApi.refundPackLine).toHaveBeenCalledWith('test-token', 'pack-1', { line_item_id: 'li-1', quantity: 1, reason: undefined }))
  })

  it('adding an extra item stages a freebie or new item', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())
    vi.mocked(packingApi.resolvePackScanCode).mockResolvedValue({ sku_id: 'sku-9', product_name: 'Jam jar' })
    vi.mocked(packingApi.scanPackItem).mockResolvedValue({ scan: { id: 'x', session_id: 'pack-1', line_item_id: null, sku_id: 'sku-9', action_type: 'freebie', quantity: 1, scanned_at: '', scanned_by: '' } })

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Add item' }))
    await user.type(screen.getByPlaceholderText('Scan or type SKU/barcode'), 'JAM-1{Enter}')
    await screen.findByText('Jam jar')
    await user.click(screen.getByRole('button', { name: 'Add to order' }))

    await waitFor(() => expect(packingApi.scanPackItem).toHaveBeenCalledWith('test-token', 'pack-1', {
      sku_id: 'sku-9', quantity: 1, action_type: 'freebie', price_minor_units: undefined, reason: 'Jam jar',
    }))
  })

  it('the session log names each order change', async () => {
    const covered = 1
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail({ scans: [
      { id: 's1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-sub', action_type: 'substitute', quantity: 2, original_quantity_covered: covered, scanned_at: '', scanned_by: '' },
      { id: 's2', session_id: 'pack-1', line_item_id: null, sku_id: 'sku-9', action_type: 'new_item', quantity: 1, reason: 'Jam jar', scanned_at: '', scanned_by: '' },
    ] }))

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Substituted · qty 2')).toBeInTheDocument()
    expect(screen.getByText('New item · Jam jar · qty 1')).toBeInTheDocument()
    expect(screen.getByText('1 / 2 items')).toBeInTheDocument()
  })

  it('deep-links via ?order=<id> straight into a packing session for that order', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())

    render(<MemoryRouter initialEntries={['/packing?order=so-1']}><Packing /></MemoryRouter>)

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(packingApi.startOrResumePacking).toHaveBeenCalledWith('test-token', 'so-1')
  })
})

describe('Packing page locks and labels', () => {
  const recent = () => new Date(Date.now() - 3 * 60_000 + 60_000).toISOString()

  it('another user\'s open session is read-only with who has it', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session({ locked_by: 'u-9', locked_at: new Date().toISOString(), locked_since: recent() }))
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail({ session: session({ locked_by: 'u-9', locked_at: new Date().toISOString(), locked_since: recent() }) }))
    vi.mocked(teamApi.listMembers).mockResolvedValue([
      { membership_id: 'm', user_id: 'u-9', email: 'd@x.io', first_name: 'Dana', last_name: 'Box', role: 'customer_user', status: 'active', last_login_at: null },
    ])

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))

    expect(await screen.findByRole('status')).toHaveTextContent('Dana Box is packing this order')
    expect(screen.getByPlaceholderText('Scan barcode or type and press Enter...')).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Complete' })).toBeDisabled()
    expect(screen.queryByRole('button', { name: 'Take over' })).not.toBeInTheDocument()
  })

  it('an admin can take over a session someone else is packing', async () => {
    mockUser = { user_id: 'me', tenant_id: 't', role: 'customer_owner', exp: 0, iat: 0 }
    const locked = session({ locked_by: 'u-9', locked_at: new Date().toISOString() })
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(locked)
    vi.mocked(packingApi.getPackingSession)
      .mockResolvedValueOnce(detail({ session: locked }))
      .mockResolvedValue(detail({ session: session({ locked_by: 'me', locked_at: new Date().toISOString() }) }))
    vi.mocked(packingApi.takeOverPackingSession).mockResolvedValue(session({ locked_by: 'me' }))

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await user.click(await screen.findByRole('button', { name: 'Take over' }))

    expect(packingApi.takeOverPackingSession).toHaveBeenCalledWith('test-token', 'pack-1')
    await waitFor(() => expect(screen.getByPlaceholderText('Scan barcode or type and press Enter...')).not.toBeDisabled())
  })

  it('leaving an order you were working releases its lock', async () => {
    const mine = session({ locked_by: 'me', locked_at: new Date().toISOString() })
    vi.mocked(omsApi.listPackingQueue).mockResolvedValue({ entries: [
      { sales_order: salesOrder(), lock: null, in_packing: false },
      { sales_order: salesOrder({ id: 'so-2', number: '1002' }), lock: null, in_packing: false },
    ], total: 2 })
    vi.mocked(packingApi.startOrResumePacking)
      .mockResolvedValueOnce(mine)
      .mockResolvedValueOnce(session({ id: 'pack-2', sales_order_id: 'so-2' }))
    vi.mocked(packingApi.getPackingSession)
      .mockResolvedValueOnce(detail({ session: mine }))
      .mockResolvedValue(detail({ session: session({ id: 'pack-2', sales_order_id: 'so-2' }) }))
    vi.mocked(packingApi.releasePackingSessionLock).mockResolvedValue(null)

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByText('#1002'))

    await waitFor(() => expect(packingApi.releasePackingSessionLock).toHaveBeenCalledWith('test-token', 'pack-1'))
  })

  it('just looking at an order does not release anything', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    const { unmount } = render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    unmount()

    expect(packingApi.releasePackingSessionLock).not.toHaveBeenCalled()
  })

  it('a label can be created before finishing, claiming the order first', async () => {
    vi.mocked(useConnections).mockReturnValue({ connections: [{ id: 'c1', provider: 'myparcel' } as ReturnType<typeof useConnections>['connections'][number]], loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
    vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: null } as unknown as Awaited<ReturnType<typeof shippingApi.getShippingSettings>>)
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([])
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())
    vi.mocked(packingApi.touchPackingSession).mockResolvedValue(null)

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Create label' }))

    expect(packingApi.touchPackingSession).toHaveBeenCalledWith('test-token', 'pack-1')
    expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()
  })

  it('the review screen offers the label before completing', async () => {
    vi.mocked(useConnections).mockReturnValue({ connections: [{ id: 'c1', provider: 'myparcel' } as ReturnType<typeof useConnections>['connections'][number]], loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
    vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: null } as unknown as Awaited<ReturnType<typeof shippingApi.getShippingSettings>>)
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([])
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())
    vi.mocked(packingApi.touchPackingSession).mockResolvedValue(null)

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Complete' }))
    await screen.findByText('Review before completing')
    await user.click(screen.getByRole('button', { name: 'Create shipping label' }))

    expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()
    expect(packingApi.completePacking).not.toHaveBeenCalled()
  })

  it('a fully packed line still opens, to replace or refund it', async () => {
    const packed = { id: 'scan-1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pack' as const, quantity: 2, scanned_at: '', scanned_by: '' }
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail({ scans: [packed] }))

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await user.click(await screen.findByText('Widget'))

    expect(await screen.findByRole('button', { name: /Replace/ })).toBeInTheDocument()
    expect(screen.getByText(/Remove what's recorded to replace it/)).toBeInTheDocument()
  })

  it('after completing, the label is created without re-claiming the finished session', async () => {
    vi.mocked(useConnections).mockReturnValue({ connections: [{ id: 'c1', provider: 'myparcel' } as ReturnType<typeof useConnections>['connections'][number]], loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
    vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: null } as unknown as Awaited<ReturnType<typeof shippingApi.getShippingSettings>>)
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([])
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())
    vi.mocked(packingApi.completePacking).mockResolvedValue({ completed: true, session: session({ status: 'completed' }), missing_items: [] })
    vi.mocked(packingApi.touchPackingSession).mockRejectedValue(new Error('touchPackingSession: packing session is not active'))

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Complete' }))
    await user.click(await screen.findByRole('button', { name: 'Confirm & complete' }))
    await screen.findByText('#1001 packed')
    await user.click(screen.getByRole('button', { name: 'Create shipping label' }))

    expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()
    expect(packingApi.touchPackingSession).not.toHaveBeenCalled()
  })
})
