// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Packing from '../Packing'
import type { SalesOrder } from '../../oms/types'
import type { PackingSession, PackingSessionDetail, ScanPackItemResult, CompletePackingResult } from '../../scanner/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
}))

vi.mock('../../scanner/api', () => ({
  startOrResumePacking: vi.fn(),
  getPackingSession: vi.fn(),
  scanPackItem: vi.fn(),
  deletePackScan: vi.fn(),
  touchPackingSession: vi.fn(),
  resetPackingSession: vi.fn(),
  completePacking: vi.fn(),
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
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(useConnections).mockReturnValue({ connections: [], loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
  vi.mocked(shippingApi.lookupAddress).mockResolvedValue(null)
})

describe('Packing page', () => {
  it('lists orders ready to pack, fulfilled and not yet packed', async () => {
    render(<MemoryRouter><Packing /></MemoryRouter>)

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(vi.mocked(omsApi.listSalesOrders)).toHaveBeenCalledWith('test-token', { fulfilled: true, packed: false, perPage: 100 })
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
      .mockResolvedValueOnce(detail({ scans: [{ id: 'scan-1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }] }))
    const scanResult: ScanPackItemResult = { scan: { id: 'scan-1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' } }
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
      .mockResolvedValueOnce(detail({ scans: [{ id: 'scan-1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }] }))
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
    await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))

    expect(await screen.findByText('#1001 packed')).toBeInTheDocument()
    expect(screen.getByText('Widget: 0/2')).toBeInTheDocument()
    expect(packingApi.completePacking).toHaveBeenCalledWith('test-token', 'pack-1')
  })

  it('deep-links via ?order=<id> straight into a packing session for that order', async () => {
    vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session())
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())

    render(<MemoryRouter initialEntries={['/packing?order=so-1']}><Packing /></MemoryRouter>)

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(packingApi.startOrResumePacking).toHaveBeenCalledWith('test-token', 'so-1')
  })
})
