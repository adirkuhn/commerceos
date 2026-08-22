// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Picking from '../Picking'
import type { SalesOrder } from '../../oms/types'
import type { PickingSession, PickingSessionDetail, ScanPickItemResult, CompletePickingResult, ResolvePickScanCodeResult } from '../../scanner/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
}))

vi.mock('../../scanner/api', () => ({
  startOrResumePicking: vi.fn(),
  getPickingSession: vi.fn(),
  scanPickItem: vi.fn(),
  resolvePickScanCode: vi.fn(),
  deletePickScan: vi.fn(),
  pausePicking: vi.fn(),
  completePicking: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listProductMedia: vi.fn(),
}))

import * as omsApi from '../../oms/api'
import * as pickingApi from '../../scanner/api'
import * as pmsApi from '../../pms/api'

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
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
})

describe('Picking page', () => {
  it('lists unfulfilled orders and filters by search', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({ id: 'so-1', number: '1001' }), salesOrder({ id: 'so-2', number: '2002', billing: { first_name: 'Bob', last_name: 'Smith' } })],
      total: 2,
    })
    render(<Picking />)

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
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(pickingApi.startOrResumePicking).toHaveBeenCalledWith('test-token', 'so-1')
    expect(screen.getByText('0 / 2 items')).toBeInTheDocument()
  })

  it('scanning a code that matches a line item opens the pick sheet, and confirming records the pick', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession)
      .mockResolvedValueOnce(detail())
      .mockResolvedValueOnce(detail({ scans: [{ id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 2, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }] }))
    const resolved: ResolvePickScanCodeResult = { sku_id: 'sku-1', product_name: 'Widget' }
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue(resolved)
    const scanResult: ScanPickItemResult = { scan: { id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 2, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' } }
    vi.mocked(pickingApi.scanPickItem).mockResolvedValue(scanResult)

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.type(screen.getByPlaceholderText('Scan barcode or type and press Enter...'), '111{Enter}')

    expect(await screen.findByText('Confirm pick')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Confirm pick' }))

    await waitFor(() => expect(pickingApi.scanPickItem).toHaveBeenCalledWith('test-token', 'ps-1', {
      sku_id: 'sku-1', line_item_id: 'li-1', action_type: 'pick', quantity: 2,
    }))
    expect(await screen.findByText('2 / 2 items')).toBeInTheDocument()
  })

  it('substitute mode records a substitute scan for the original line item, using the replacement SKU', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.resolvePickScanCode)
      .mockResolvedValueOnce({ sku_id: 'sku-1', product_name: 'Widget' })
      .mockResolvedValueOnce({ sku_id: 'sku-2', product_name: 'Widget Replacement' })
    vi.mocked(pickingApi.scanPickItem).mockResolvedValue({ scan: { id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-2', action_type: 'substitute', quantity: 2, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' } })

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.type(screen.getByPlaceholderText('Scan barcode or type and press Enter...'), '111{Enter}')
    await screen.findByText('Confirm pick')
    await user.click(screen.getByRole('button', { name: 'Substitute' }))
    await user.type(screen.getByPlaceholderText('Scan or type replacement SKU/barcode'), 'sub-code{Enter}')

    await waitFor(() => expect(pickingApi.scanPickItem).toHaveBeenCalledWith('test-token', 'ps-1', {
      sku_id: 'sku-2', line_item_id: 'li-1', action_type: 'substitute',
    }))
  })

  it('pausing returns to the order list', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.pausePicking).mockResolvedValue(session({ status: 'paused' }))

    const user = userEvent.setup()
    render(<Picking />)
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
      session: session({ status: 'completed' }),
      missing_items: [{ line_item_id: 'li-1', name: 'Widget', expected_qty: 2, picked_qty: 0 }],
    }
    vi.mocked(pickingApi.completePicking).mockResolvedValue(result)

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Complete' }))

    expect(await screen.findByText('#1001 marked fulfilled')).toBeInTheDocument()
    const missing = screen.getByText('Not fully picked').closest('div')!
    expect(within(missing).getByText('Widget: 0/2')).toBeInTheDocument()
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
    render(<Picking />)
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
    render(<Picking />)
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
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await waitFor(() => expect(pmsApi.listProductMedia).toHaveBeenCalledWith('test-token', 'product-1'))
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })
})
