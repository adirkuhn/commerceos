// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { act } from 'react'
import { createMemoryRouter, RouterProvider } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ScannerApp from '../ScannerApp'
import { scannerPaths } from '../scannerRoutes'
import type { PickingSessionDetail, PackingSessionDetail } from '../types'
import type { SalesOrder } from '../../oms/types'

vi.mock('../api', () => ({
  claimPairingCode: vi.fn(),
  claimPairingCodeByToken: vi.fn(),
  startOrResumeBooking: vi.fn(),
  getBooking: vi.fn(),
  startOrResumePicking: vi.fn(),
  getPickingSession: vi.fn(),
  touchPickingSession: vi.fn(),
  startOrResumePacking: vi.fn(),
  getPackingSession: vi.fn(),
  touchPackingSession: vi.fn(),
  refreshScannerSession: vi.fn(),
  setOnTokenActivity: vi.fn(),
}))
vi.mock('../../procurement/api', () => ({ listPurchaseOrders: vi.fn(), getPurchaseOrder: vi.fn() }))
vi.mock('../../suppliers/api', () => ({ listSuppliers: vi.fn() }))
vi.mock('../../inventory/api', () => ({ listWarehouses: vi.fn(), listBatches: vi.fn() }))
vi.mock('../../oms/api', () => ({ listSalesOrders: vi.fn(), getSalesOrder: vi.fn() }))
vi.mock('../../pms/api', () => ({ resolveBarcode: vi.fn(), listProducts: vi.fn(), getProduct: vi.fn(), listProductMedia: vi.fn() }))
vi.mock('../../shipping/api', () => ({ listShipmentsForOrder: vi.fn().mockResolvedValue([]) }))
vi.mock('../../integrations/api', () => ({ listConnections: vi.fn().mockResolvedValue([]) }))
vi.mock('../errorSound', () => ({ playErrorSound: vi.fn() }))
vi.mock('@zxing/browser', () => ({
  BrowserMultiFormatReader: vi.fn().mockImplementation(() => ({ decodeFromVideoDevice: vi.fn().mockResolvedValue({ stop: vi.fn() }) })),
}))

import * as api from '../api'
import * as procurementApi from '../../procurement/api'
import * as suppliersApi from '../../suppliers/api'
import * as omsApi from '../../oms/api'
import * as pmsApi from '../../pms/api'
import * as inventoryApi from '../../inventory/api'

function makeJWT(exp: number): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ user_id: 'user-1', tenant_id: 'tenant-1', role: 'customer_owner', exp }))
  return `${header}.${payload}.signature`
}

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'order-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1',
    number: '1001', order_key: 'key-1', status: 'processing', warehouse_id: 'warehouse-1',
    billing: { first_name: 'Jane', last_name: 'Doe' }, shipping: {},
    currency_code: 'USD', discount_total_minor_units: 0, shipping_total_minor_units: 0,
    tax_total_minor_units: 0, total_minor_units: 2999,
    occurred_at: '2026-08-05T00:00:00Z', created_at: '2026-08-05T00:00:00Z', fulfilled: false,
    ...overrides,
  }
}

const pickingDetail: PickingSessionDetail = {
  session: { id: 'session-1', sales_order_id: 'order-1', status: 'in_progress', started_at: '', completed_by: '' },
  line_items: [], scans: [],
} as unknown as PickingSessionDetail

const packingDetail: PackingSessionDetail = {
  session: { id: 'pack-1', sales_order_id: 'order-1', status: 'in_progress', started_at: '', completed_by: '' },
  line_items: [], scans: [],
} as unknown as PackingSessionDetail

function renderAt(url: string) {
  const router = createMemoryRouter([{ path: '/scanner/*', element: <ScannerApp /> }], { initialEntries: [url] })
  render(<RouterProvider router={router} />)
  return router
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  localStorage.setItem('commerceos_scanner_jwt', makeJWT(Math.floor(Date.now() / 1000) + 3600))
  vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
  vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 } as never)
  vi.mocked(omsApi.getSalesOrder).mockResolvedValue({ sales_order: salesOrder(), line_items: [] })
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
  vi.mocked(inventoryApi.listBatches).mockResolvedValue([])
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
})

describe('scannerPaths', () => {
  it('builds per-resource URLs', () => {
    expect(scannerPaths.launcher).toBe('/scanner')
    expect(scannerPaths.pickSession('s 1')).toBe('/scanner/pick/s%201')
    expect(scannerPaths.packSession('p1')).toBe('/scanner/pack/p1')
    expect(scannerPaths.booking('b1')).toBe('/scanner/receive/b1')
    expect(scannerPaths.inventorySku('SKU/1')).toBe('/scanner/inventory/SKU%2F1')
  })
})

describe('ScannerApp deep links (refresh lands on the same screen)', () => {
  it('/scanner/receive opens the purchase order picker directly', async () => {
    renderAt('/scanner/receive')
    expect(await screen.findByText('Select a purchase order')).toBeInTheDocument()
  })

  it('/scanner/receive/:bookingId reopens that booking without starting a new one', async () => {
    vi.mocked(api.getBooking).mockResolvedValue({
      booking: { id: 'booking-9', tenant_id: 't', purchase_order_id: 'po-1', status: 'in_progress', started_at: '', completed_by: '' },
      line_items: [], scans: [],
    } as never)
    renderAt('/scanner/receive/booking-9')
    expect(await screen.findByText('Booking')).toBeInTheDocument()
    expect(api.getBooking).toHaveBeenCalledWith(expect.any(String), 'booking-9')
    expect(api.startOrResumeBooking).not.toHaveBeenCalled()
  })

  it('/scanner/pick opens the sales order picker directly', async () => {
    renderAt('/scanner/pick')
    expect(await screen.findByText('Select an order to pick')).toBeInTheDocument()
  })

  it('/scanner/pick/:sessionId reloads the session and its order, without starting a new session', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(pickingDetail)
    renderAt('/scanner/pick/session-1')
    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(api.getPickingSession).toHaveBeenCalledWith(expect.any(String), 'session-1')
    expect(omsApi.getSalesOrder).toHaveBeenCalledWith(expect.any(String), 'order-1')
    expect(api.startOrResumePicking).not.toHaveBeenCalled()
  })

  it('/scanner/pick/:sessionId shows an error with a way back when the session cannot be loaded', async () => {
    vi.mocked(api.getPickingSession).mockRejectedValue(new Error('picking session not found'))
    const router = renderAt('/scanner/pick/missing')
    expect(await screen.findByText('picking session not found')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: /Back/ }))
    expect(router.state.location.pathname).toBe('/scanner/pick')
  })

  it('/scanner/pack/:sessionId reloads the packing session and its order', async () => {
    vi.mocked(api.getPackingSession).mockResolvedValue(packingDetail)
    renderAt('/scanner/pack/pack-1')
    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(api.getPackingSession).toHaveBeenCalledWith(expect.any(String), 'pack-1')
    expect(api.startOrResumePacking).not.toHaveBeenCalled()
  })

  it('/scanner/inventory/:code re-resolves the SKU and opens its screen', async () => {
    vi.mocked(pmsApi.resolveBarcode).mockResolvedValue({
      sku: { id: 'sku-1', sku_code: 'SKU-1', price_minor_units: 100, currency_code: 'EUR' },
      product_name: 'Widget A',
    } as never)
    renderAt('/scanner/inventory/SKU-1')
    expect(await screen.findByText('Widget A')).toBeInTheDocument()
    expect(pmsApi.resolveBarcode).toHaveBeenCalledWith(expect.any(String), 'SKU-1')
  })

  it('an unknown /scanner path falls back to the launcher', async () => {
    const router = renderAt('/scanner/nope')
    expect(await screen.findByText('Pick Order')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/scanner')
  })

  it('without a paired token, a deep link shows the claim screen and keeps the URL', async () => {
    localStorage.clear()
    const router = renderAt('/scanner/pick/session-1')
    expect(await screen.findByPlaceholderText('Scan QR code or barcode…')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/scanner/pick/session-1')
  })
})

describe('ScannerApp navigation writes history (Back works)', () => {
  it('launcher tile pushes the app URL and Back returns to the launcher', async () => {
    const router = renderAt('/scanner')
    await userEvent.click(await screen.findByText('Pick Order'))
    expect(router.state.location.pathname).toBe('/scanner/pick')
    await screen.findByText('Select an order to pick')

    await act(() => router.navigate(-1))
    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
    expect(router.state.location.pathname).toBe('/scanner')
  })

  it('selecting an order pushes the session URL', async () => {
    vi.mocked(api.startOrResumePicking).mockResolvedValue(pickingDetail.session)
    vi.mocked(api.getPickingSession).mockResolvedValue(pickingDetail)
    const router = renderAt('/scanner/pick')
    await userEvent.click(await screen.findByText('Jane Doe'))
    await waitFor(() => expect(router.state.location.pathname).toBe('/scanner/pick/session-1'))
    expect(await screen.findByText('#1001')).toBeInTheDocument()
  })

  it('in-app Back arrow from the order picker goes to the launcher URL', async () => {
    const router = renderAt('/scanner/pick')
    await userEvent.click(await screen.findByTitle('Back to apps'))
    expect(router.state.location.pathname).toBe('/scanner')
  })
})
