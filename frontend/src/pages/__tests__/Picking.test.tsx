// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act } from 'react'
import { renderAt } from '../../test/renderAt'
import Picking from '../Picking'
import type { SalesOrder } from '../../oms/types'
import type { PickingSession, PickingSessionDetail, ScanPickItemResult, CompletePickingResult, ResolvePickScanCodeResult } from '../../scanner/types'

let mockUser: { user_id: string; tenant_id: string; role: string; exp: number; iat: number } | null = null
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: mockUser, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../team/api', () => ({ listMembers: vi.fn().mockResolvedValue([]) }))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
  getSalesOrder: vi.fn(),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn().mockResolvedValue({ picking_mode: 'scan' }),
}))

vi.mock('../../scanner/api', () => ({
  startOrResumePicking: vi.fn(),
  getPickingSession: vi.fn(),
  scanPickItem: vi.fn(),
  substituteLineItem: vi.fn(),
  refundLineItem: vi.fn(),
  resolvePickScanCode: vi.fn(),
  deletePickScan: vi.fn(),
  pausePicking: vi.fn(),
  completePicking: vi.fn(),
  touchPickingSession: vi.fn(),
  resetPickingSession: vi.fn(),
  releasePickingSessionLock: vi.fn(),
  takeOverPickingSession: vi.fn(),
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
import * as pickingApi from '../../scanner/api'
import * as pmsApi from '../../pms/api'
import * as shippingApi from '../../shipping/api'
import { useConnections } from '../../integrations/hooks/useConnections'
import { playErrorSound } from '../../scanner/errorSound'

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1',
    number: '1001', order_key: '', status: 'processing', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    fulfilled: false,
    billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
    ...overrides,
  }
}

function session(overrides: Partial<PickingSession> = {}): PickingSession {
  return { id: 'ps-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '2026-08-22T10:00:00Z', completed_by: '', ...overrides }
}

function detail(overrides: Partial<PickingSessionDetail> = {}): PickingSessionDetail {
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
  vi.mocked(pickingApi.releasePickingSessionLock).mockResolvedValue(null)
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(useConnections).mockReturnValue({ connections: [], loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
  vi.mocked(shippingApi.lookupAddress).mockResolvedValue(null)
  vi.mocked(omsApi.getSalesOrder).mockResolvedValue({ sales_order: salesOrder(), line_items: [] })
})

describe('Picking page URLs (refresh + Back)', () => {
  it('selecting an order pushes /picking/:sessionId', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())

    const { router } = renderAt(<Picking />, '/picking')
    await userEvent.click(await screen.findByText('#1001'))

    await waitFor(() => expect(router.state.location.pathname).toBe('/picking/ps-1'))
    expect(await screen.findByText('Widget')).toBeInTheDocument()
  })

  it('/picking/:sessionId reopens that session on load without starting a new one', async () => {
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())

    renderAt(<Picking />, '/picking', '/picking/ps-1')

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(pickingApi.getPickingSession).toHaveBeenCalledWith('test-token', 'ps-1')
    expect(omsApi.getSalesOrder).toHaveBeenCalledWith('test-token', 'so-1')
    expect(pickingApi.startOrResumePicking).not.toHaveBeenCalled()
  })

  it('shows an error when the session in the URL cannot be loaded', async () => {
    vi.mocked(pickingApi.getPickingSession).mockRejectedValue(new Error('picking session not found'))

    renderAt(<Picking />, '/picking', '/picking/missing')

    expect(await screen.findByText('picking session not found')).toBeInTheDocument()
  })

  it('browser Back from a session returns to the plain order list', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())

    const { router } = renderAt(<Picking />, '/picking')
    await userEvent.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await act(() => router.navigate(-1))

    expect(router.state.location.pathname).toBe('/picking')
    expect(await screen.findByText('Select an order to start picking')).toBeInTheDocument()
    expect(screen.queryByText('Widget')).not.toBeInTheDocument()
  })

  it('pausing goes back to /picking', async () => {
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.pausePicking).mockResolvedValue(session({ status: 'paused' }))

    const { router } = renderAt(<Picking />, '/picking', '/picking/ps-1')
    await screen.findByText('Widget')
    await userEvent.click(screen.getByRole('button', { name: /Pause/ }))

    await waitFor(() => expect(router.state.location.pathname).toBe('/picking'))
  })
})

describe('Picking page', () => {
  it('lists unfulfilled orders and filters by search', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({ id: 'so-1', number: '1001' }), salesOrder({ id: 'so-2', number: '2002', billing: { first_name: 'Bob', last_name: 'Smith' } })],
      total: 2,
    })
    render(<MemoryRouter><Picking /></MemoryRouter>)

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(screen.getByText('#2002')).toBeInTheDocument()

    const user = userEvent.setup()
    await user.type(screen.getByPlaceholderText('Search orders...'), 'Bob')

    expect(screen.queryByText('#1001')).not.toBeInTheDocument()
    expect(screen.getByText('#2002')).toBeInTheDocument()
  })

  it('selecting an order starts/resumes a session and shows its line items', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(pickingApi.startOrResumePicking).toHaveBeenCalledWith('test-token', 'so-1')
    expect(screen.getByText('0 / 2 items')).toBeInTheDocument()
  })

  it('scanning a code that matches a line item marks one unit picked immediately, with no sheet', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession)
      .mockResolvedValueOnce(detail())
      .mockResolvedValueOnce(detail({ scans: [{ id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }] }))
    const resolved: ResolvePickScanCodeResult = { sku_id: 'sku-1', product_name: 'Widget' }
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue(resolved)
    const scanResult: ScanPickItemResult = { scan: { id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' } }
    vi.mocked(pickingApi.scanPickItem).mockResolvedValue(scanResult)

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.type(screen.getByPlaceholderText('Scan barcode or type and press Enter...'), '111{Enter}')

    await waitFor(() => expect(pickingApi.scanPickItem).toHaveBeenCalledWith('test-token', 'ps-1', {
      sku_id: 'sku-1', action_type: 'pick', quantity: 1,
    }))
    expect(screen.queryByText('Confirm pick')).not.toBeInTheDocument()
    expect(await screen.findByText('1 / 2 items')).toBeInTheDocument()
  })

  it('scanning a code already fully picked shows a blocking error and stages nothing', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({
      scans: [{ id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 2, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }],
    }))
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget' })

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    const input = screen.getByPlaceholderText('Scan barcode or type and press Enter...')
    await user.type(input, '111{Enter}')

    expect(await screen.findByText('Already fully picked')).toBeInTheDocument()
    expect(pickingApi.scanPickItem).not.toHaveBeenCalled()
    expect(playErrorSound).toHaveBeenCalled()
    expect(input).toBeDisabled()

    await user.click(screen.getByRole('button', { name: 'OK' }))
    expect(screen.queryByText('Already fully picked')).not.toBeInTheDocument()
    expect(input).toBeEnabled()
  })

  it('scanning a code not on the order shows a blocking error instead of opening Add item', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-other', product_name: 'Gizmo' })
    // No line item on this order carries sku-other -- still attempted server-
    // side (it may auto-link an unmatched Woo line by product name), and
    // here the server genuinely rejects it as not belonging to this order.
    vi.mocked(pickingApi.scanPickItem).mockRejectedValue(new Error('scanned item does not match any line item on this order'))

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.type(screen.getByPlaceholderText('Scan barcode or type and press Enter...'), '999{Enter}')

    expect(await screen.findByText('Item not on this order')).toBeInTheDocument()
    expect(pickingApi.scanPickItem).toHaveBeenCalledWith('test-token', 'ps-1', {
      sku_id: 'sku-other', action_type: 'pick', quantity: 1,
    })
    expect(screen.queryByText('Add item not on order')).not.toBeInTheDocument()
    expect(playErrorSound).toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'OK' }))
    expect(screen.queryByText('Item not on this order')).not.toBeInTheDocument()
  })

  // Regression: a code that doesn't resolve to ANY known SKU (resolvePickScanCode
  // itself rejects, e.g. 404 sku not found) used to fall through to a plain
  // non-blocking inline error, letting the picker keep scanning straight past
  // a mis-scan. Must block the same way as "already fully picked"/"item not
  // on this order" above.
  it('scanning a code that resolves to no SKU at all shows a blocking error, not just inline text', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.resolvePickScanCode).mockRejectedValue(new Error('sku not found'))

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    const input = screen.getByPlaceholderText('Scan barcode or type and press Enter...')
    await user.type(input, 'garbage{Enter}')

    expect(await screen.findByText('Barcode not recognized')).toBeInTheDocument()
    expect(pickingApi.scanPickItem).not.toHaveBeenCalled()
    expect(playErrorSound).toHaveBeenCalled()
    expect(input).toBeDisabled()

    await user.click(screen.getByRole('button', { name: 'OK' }))
    expect(screen.queryByText('Barcode not recognized')).not.toBeInTheDocument()
    expect(input).toBeEnabled()
  })

  it('start over discards the session and reloads it fresh', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession)
      .mockResolvedValueOnce(detail({
        scans: [{ id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }],
      }))
      .mockResolvedValueOnce(detail())
    vi.mocked(pickingApi.resetPickingSession).mockResolvedValue(session())

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    expect(await screen.findByText('1 / 2 items')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Start over' }))
    expect(await screen.findByText('Start over?')).toBeInTheDocument()

    const startOverButtons = screen.getAllByRole('button', { name: 'Start over' })
    await user.click(startOverButtons[startOverButtons.length - 1])

    expect(pickingApi.resetPickingSession).toHaveBeenCalledWith('test-token', 'ps-1')
    expect(await screen.findByText('0 / 2 items')).toBeInTheDocument()
  })

  it('scanning a code that auto-links an unmatched line item picks it instead of erroring', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-unmatched-resolved', product_name: 'Carne Seca ~414g' })
    vi.mocked(pickingApi.scanPickItem).mockResolvedValue({ scan: {} as never })

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.type(screen.getByPlaceholderText('Scan barcode or type and press Enter...'), '999{Enter}')

    await waitFor(() => expect(pickingApi.scanPickItem).toHaveBeenCalledWith('test-token', 'ps-1', {
      sku_id: 'sku-unmatched-resolved', action_type: 'pick', quantity: 1,
    }))
    expect(screen.queryByText('Item not on this order')).not.toBeInTheDocument()
  })

  it('shows the shipping address in the header once an order is selected', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({ shipping: { address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL', email: 'jane@example.com' } })],
      total: 1,
    })
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    expect(screen.getByText('Damrak 1, 1012AB Amsterdam, NL')).toBeInTheDocument()
  })

  it('picking only picks -- no substitute, refund or add item', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await user.click(await screen.findByText('Widget'))

    expect(screen.getByRole('button', { name: 'Confirm pick' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Substitute/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Refund/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Add item' })).not.toBeInTheDocument()
    expect(screen.getByText(/Substitutes and refunds happen in packing/)).toBeInTheDocument()
  })

  it('pausing returns to the order list', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.pausePicking).mockResolvedValue(session({ status: 'paused' }))

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: /Pause/ }))

    await waitFor(() => expect(pickingApi.pausePicking).toHaveBeenCalledWith('test-token', 'ps-1'))
    expect(await screen.findByText('Select an order to start picking')).toBeInTheDocument()
  })

  it('completing shows missing items and marks the order fulfilled', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    const result: CompletePickingResult = {
      completed: true, session: session({ status: 'completed' }),
      missing_items: [{ line_item_id: 'li-1', name: 'Widget', expected_qty: 2, picked_qty: 0 }],
    }
    vi.mocked(pickingApi.completePicking).mockResolvedValue(result)

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Complete' }))

    // Complete no longer fires the API directly -- it opens a review step
    // showing the divergences first.
    expect(await screen.findByText('Review before completing')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Confirm & complete' }))

    expect(await screen.findByText('#1001 marked fulfilled')).toBeInTheDocument()
    const missing = screen.getByText('Not fully picked').closest('div')!
    expect(within(missing).getByText('Widget: 0/2')).toBeInTheDocument()
  })

  it('shows a stock-shortfall dialog and resubmits with the chosen resolution', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.completePicking)
      .mockResolvedValueOnce({
        completed: false,
        stock_shortfalls: [{ scan_id: 'scan-1', sku_id: 'sku-1', name: 'Widget', needed: 5, short: 3 }],
      })
      .mockResolvedValueOnce({ completed: true, session: session({ status: 'completed' }), missing_items: [] })

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Complete' }))
    expect(await screen.findByText('Review before completing')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Confirm & complete' }))

    expect(await screen.findByText('Not enough stock')).toBeInTheDocument()
    expect(screen.getByText('Needed 5, 3 short')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Confirm & complete' }))

    expect(await screen.findByText('#1001 marked fulfilled')).toBeInTheDocument()
    expect(pickingApi.completePicking).toHaveBeenLastCalledWith('test-token', 'ps-1', { 'scan-1': 'with_item' })
  })

  it('shows the shipping method below the address on the review screen', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({
        shipping: { address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL', email: 'jane@example.com' },
        shipping_method_title: 'PostNL PakjeGemak',
      })],
      total: 1,
    })
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Complete' }))

    expect(await screen.findByText('Review before completing')).toBeInTheDocument()
    expect(screen.getByText('Shipping: PostNL PakjeGemak')).toBeInTheDocument()
  })

  // Label creation moved out of Picking.tsx entirely once packing became its
  // own screen -- completing picking now only offers "Continue to packing"
  // (navigating to /packing?order=<id>), regardless of MyParcel connection
  // state. See Packing.tsx for the address/label flow's new home.
  it('offers Continue to Packing after completing, and navigates to the packing screen', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.completePicking).mockResolvedValue({ completed: true, session: session({ status: 'completed' }), missing_items: [] })

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Complete' }))
    await screen.findByText('Review before completing')
    await user.click(screen.getByRole('button', { name: 'Confirm & complete' }))

    await screen.findByText('#1001 marked fulfilled')
    expect(screen.queryByRole('button', { name: 'Create shipping label' })).not.toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Continue to packing' })).toBeInTheDocument()
  })

})

describe('Picking page product thumbnail', () => {
  it('shows a product thumbnail for a line item linked to a SKU with media', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({
      line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000, product_id: 'product-1' }],
    }))
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([
      { id: 'media-1', tenant_id: 't1', product_id: 'product-1', url: 'https://example.com/thumb.jpg', type: 'image', position: 0, alt_text: 'Widget photo', created_at: '' },
    ])

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    const img = await screen.findByAltText('Widget photo')
    expect(img).toHaveAttribute('src', 'https://example.com/thumb.jpg')
    expect(pmsApi.listProductMedia).toHaveBeenCalledWith('test-token', 'product-1')
  })

  it('falls back to a placeholder icon when the line has no product_id', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({
      line_items: [{ id: 'li-1', sku_id: null, name: 'Unmatched item', sku_code: '', barcode: '', quantity: 1, price_minor_units: 0, total_minor_units: 0, product_id: null }],
    }))

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))

    await screen.findByText('Unmatched item')
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
    expect(pmsApi.listProductMedia).not.toHaveBeenCalled()
  })

  it('falls back to a placeholder icon when the product has no media', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({
      line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000, product_id: 'product-1' }],
    }))
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await waitFor(() => expect(pmsApi.listProductMedia).toHaveBeenCalledWith('test-token', 'product-1'))
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })
})

describe('Picking page mode', () => {
  it('list mode shows the plain pick list instead of scanning', async () => {
    const settingsApi = await import('../../settings/api')
    vi.mocked(settingsApi.getTenantSettings).mockResolvedValueOnce({ picking_mode: 'list' } as Awaited<ReturnType<typeof settingsApi.getTenantSettings>>)

    renderAt(<Picking />, '/picking')

    expect(await screen.findByText('Select an order to see what to pick')).toBeInTheDocument()
    expect(vi.mocked(omsApi.listSalesOrders)).toHaveBeenCalledWith('test-token', { awaitingPicking: true, perPage: 100 })
  })
})

describe('Picking page locks', () => {
  it('another user\'s session is view only', async () => {
    const locked = session({ locked_by: 'u-9', locked_at: new Date().toISOString() })
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(locked)
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({ session: locked }))

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))

    expect(await screen.findByRole('status')).toHaveTextContent('is picking this order')
    expect(screen.getByPlaceholderText('Scan barcode or type and press Enter...')).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Complete' })).toBeDisabled()
  })

  it('an admin can take it over', async () => {
    mockUser = { user_id: 'me', tenant_id: 't', role: 'customer_admin', exp: 0, iat: 0 }
    const locked = session({ locked_by: 'u-9', locked_at: new Date().toISOString() })
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(locked)
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({ session: locked }))
    vi.mocked(pickingApi.takeOverPickingSession).mockResolvedValue(session({ locked_by: 'me' }))

    const user = userEvent.setup()
    render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await user.click(await screen.findByRole('button', { name: 'Take over' }))

    expect(pickingApi.takeOverPickingSession).toHaveBeenCalledWith('test-token', 'ps-1')
  })

  it('leaving a session you hold releases it', async () => {
    const mine = session({ locked_by: 'me', locked_at: new Date().toISOString() })
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(mine)
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({ session: mine }))

    const user = userEvent.setup()
    const { unmount } = render(<MemoryRouter><Picking /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    unmount()

    expect(pickingApi.releasePickingSessionLock).toHaveBeenCalledWith('test-token', 'ps-1')
  })
})
