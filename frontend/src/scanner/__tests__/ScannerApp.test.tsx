// @vitest-environment jsdom
import { screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderAt } from '../../test/renderAt'
import ScannerApp from '../ScannerApp'
import type { Booking, BookingDetail, PickingSession, PickingSessionDetail } from '../types'
import type { PurchaseOrder } from '../../procurement/types'
import type { Supplier } from '../../suppliers/types'
import type { Warehouse, Batch } from '../../inventory/types'
import type { SalesOrder } from '../../oms/types'

vi.mock('../api', () => ({
  claimPairingCode: vi.fn(),
  claimPairingCodeByToken: vi.fn(),
  generatePairingCode: vi.fn(),
  getPairingCodeStatus: vi.fn(),
  startOrResumeBooking: vi.fn(),
  getBooking: vi.fn(),
  scanBookingItem: vi.fn(),
  resolveBookingScanCode: vi.fn(),
  pauseBooking: vi.fn(),
  completeBooking: vi.fn(),
  startOrResumePicking: vi.fn(),
  getPickingSession: vi.fn(),
  scanPickItem: vi.fn(),
  resolvePickScanCode: vi.fn(),
  deletePickScan: vi.fn(),
  pausePicking: vi.fn(),
  completePicking: vi.fn(),
  touchPickingSession: vi.fn(),
  resetPickingSession: vi.fn(),
  refreshScannerSession: vi.fn(),
  setOnTokenActivity: vi.fn(),
}))

vi.mock('../../procurement/api', () => ({
  listPurchaseOrders: vi.fn(),
  getPurchaseOrder: vi.fn(),
  updatePurchaseOrder: vi.fn(),
}))

vi.mock('../../suppliers/api', () => ({
  listSuppliers: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
  listBatches: vi.fn(),
  markDamagedOrLost: vi.fn(),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
  getSalesOrder: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listProducts: vi.fn(),
  getProduct: vi.fn(),
  listProductMedia: vi.fn(),
}))

vi.mock('../errorSound', () => ({
  playErrorSound: vi.fn(),
}))

const mockDecodeFromVideoDevice = vi.fn()
vi.mock('@zxing/browser', () => ({
  BrowserMultiFormatReader: vi.fn().mockImplementation(() => ({
    decodeFromVideoDevice: mockDecodeFromVideoDevice,
  })),
}))

import * as api from '../api'
import * as procurementApi from '../../procurement/api'
import * as suppliersApi from '../../suppliers/api'
import * as inventoryApi from '../../inventory/api'
import * as omsApi from '../../oms/api'
import * as pmsApi from '../../pms/api'
import { playErrorSound } from '../errorSound'

function makeJWT(exp: number): string {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const payload = btoa(JSON.stringify({ user_id: 'user-1', tenant_id: 'tenant-1', role: 'customer_owner', exp }))
  return `${header}.${payload}.signature`
}

function po(overrides: Partial<PurchaseOrder> = {}): PurchaseOrder {
  return {
    id: 'po-1', tenant_id: 'tenant-1', supplier_id: 'supplier-1', warehouse_id: 'warehouse-1',
    status: 'sent', currency_code: 'USD', expected_date: null, notes: '',
    is_potential_duplicate: false, total_minor_units: 0, created_at: '', updated_at: '',
    ...overrides,
  }
}

function supplier(overrides: Partial<Supplier> = {}): Supplier {
  return {
    id: 'supplier-1', tenant_id: 'tenant-1', name: 'Acme Foods',
    email: '', phone: '', country: '', lead_time_days: null, status: 'active', notes: '',
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function booking(overrides: Partial<Booking> = {}): Booking {
  return { id: 'booking-1', tenant_id: 'tenant-1', purchase_order_id: 'po-1', status: 'in_progress', started_at: '', completed_by: '', ...overrides }
}

function warehouse(overrides: Partial<Warehouse> = {}): Warehouse {
  return {
    id: 'warehouse-1', tenant_id: 'tenant-1', name: 'Main Warehouse',
    address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function batch(overrides: Partial<Batch> = {}): Batch {
  return {
    id: 'batch-1', tenant_id: 'tenant-1', sku_id: 'sku-1', warehouse_id: 'warehouse-1',
    lot_number: 'LOT-1', expiry_date: null, received_at: '', created_at: '', updated_at: '',
    quantity: 10, is_general_ledger: false,
    ...overrides,
  }
}

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'order-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1',
    number: '1001', order_key: 'key-1', status: 'processing', warehouse_id: 'warehouse-1',
    billing: { first_name: 'Jane', last_name: 'Doe' },
    shipping: { address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' },
    currency_code: 'USD', discount_total_minor_units: 0, shipping_total_minor_units: 0,
    tax_total_minor_units: 0, total_minor_units: 2999,
    occurred_at: '2026-08-05T00:00:00Z', created_at: '2026-08-05T00:00:00Z',
    fulfilled: false,
    ...overrides,
  }
}

function pickingSession(overrides: Partial<PickingSession> = {}): PickingSession {
  return { id: 'session-1', sales_order_id: 'order-1', status: 'in_progress', started_at: '', completed_by: '', ...overrides }
}

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  // Picking/packing screens are URL-addressed and reload their order by id.
  vi.mocked(omsApi.getSalesOrder).mockResolvedValue({ sales_order: salesOrder(), line_items: [] })
  mockDecodeFromVideoDevice.mockResolvedValue({ stop: vi.fn() })
})

describe('ScannerApp pairing', () => {
  it('defaults to the Scan tab, not Type code', async () => {
    renderAt(<ScannerApp />, '/scanner')

    expect(screen.getByPlaceholderText('Scan QR code or barcode…')).toBeInTheDocument()
    expect(screen.queryByPlaceholderText('000000')).not.toBeInTheDocument()
  })

  it('pairs via the Scan tab using the token path, not the digit-code path', async () => {
    vi.mocked(api.claimPairingCodeByToken).mockResolvedValue({ token: makeJWT(Math.floor(Date.now() / 1000) + 3600) })
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])

    renderAt(<ScannerApp />, '/scanner')
    // A hardware scan-engine keyboard-wedges the decoded token then a
    // terminator keystroke -- typing + Enter simulates that.
    await userEvent.type(screen.getByPlaceholderText('Scan QR code or barcode…'), 'opaque-token-abc{Enter}')

    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
    expect(api.claimPairingCodeByToken).toHaveBeenCalledWith('opaque-token-abc')
    expect(api.claimPairingCode).not.toHaveBeenCalled()
  })

  it('shows an error on an invalid scanned token', async () => {
    vi.mocked(api.claimPairingCodeByToken).mockRejectedValue(new Error('pairing code is invalid or expired'))

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.type(screen.getByPlaceholderText('Scan QR code or barcode…'), 'bad-token{Enter}')

    expect(await screen.findByText('pairing code is invalid or expired')).toBeInTheDocument()
  })

  it('pairs via the camera and auto-submits on decode', async () => {
    vi.mocked(api.claimPairingCodeByToken).mockResolvedValue({ token: makeJWT(Math.floor(Date.now() / 1000) + 3600) })
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(screen.getByRole('button', { name: /Use camera/ }))
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())
    const callback = mockDecodeFromVideoDevice.mock.calls[0][2]
    act(() => { callback({ getText: () => 'camera-token-xyz' }, undefined, { stop: vi.fn() }) })

    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
    expect(api.claimPairingCodeByToken).toHaveBeenCalledWith('camera-token-xyz')
  })

  it('pairs on valid code from the Type code tab', async () => {
    vi.mocked(api.claimPairingCode).mockResolvedValue({ token: makeJWT(Math.floor(Date.now() / 1000) + 3600) })
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(screen.getByRole('button', { name: 'Type code' }))
    await userEvent.type(await screen.findByPlaceholderText('000000'), '123456')
    await userEvent.click(screen.getByRole('button', { name: 'Pair' }))

    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
    expect(api.claimPairingCode).toHaveBeenCalledWith('123456')
  })

  it('shows an error on an invalid typed code', async () => {
    vi.mocked(api.claimPairingCode).mockRejectedValue(new Error('pairing code is invalid or expired'))

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(screen.getByRole('button', { name: 'Type code' }))
    await userEvent.type(await screen.findByPlaceholderText('000000'), '000000')
    await userEvent.click(screen.getByRole('button', { name: 'Pair' }))

    expect(await screen.findByText('pairing code is invalid or expired')).toBeInTheDocument()
  })

  it('skips the claim screen when a valid token is already stored', async () => {
    localStorage.setItem('commerceos_scanner_jwt', makeJWT(Math.floor(Date.now() / 1000) + 3600))

    renderAt(<ScannerApp />, '/scanner')

    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
  })
})

// ScannerApp's Google-login option -- an alternative to pairing off a
// desktop admin session that signs the device itself in via lm2, the same
// flow ../auth/LoginPage's Google button uses (see LoginPage.test.tsx).
// This only proves the redirect fires correctly; the resulting long-lived
// session is useScannerAuth's own concern (useScannerAuth.test.ts).
describe('ScannerApp pairing — Google sign-in', () => {
  function mockGoogleUrlFetch(data: unknown, ok = true) {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
      ok, json: () => Promise.resolve({ data, error: ok ? null : { message: 'Google sign-in is unavailable' } }),
    }))
  }

  beforeEach(() => {
    vi.stubEnv('VITE_LM2_LOGIN_URL', 'https://lm2.example.com/login')
    delete (window as unknown as { location: unknown }).location
    // @ts-expect-error -- test-only stub so we can assert on the redirect target
    window.location = { href: '', origin: 'https://scanner.example.com' }
  })

  it('fetches the google login url from lm2, targeting the scanner callback path, and redirects to it', async () => {
    mockGoogleUrlFetch({ url: 'https://accounts.google.com/o/oauth2/v2/auth?state=abc' })

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(screen.getByRole('button', { name: /sign in with google/i }))

    await waitFor(() => expect(window.location.href).toBe('https://accounts.google.com/o/oauth2/v2/auth?state=abc'))
    const [url] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('https://lm2.example.com/api/v1/auth/app/google/url')
    expect(url).toContain('app=commerceos')
    expect(url).toContain(encodeURIComponent('https://scanner.example.com/scanner/auth/callback'))
  })

  it('shows an error when lm2 has no Google configured', async () => {
    mockGoogleUrlFetch(null, false)

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(screen.getByRole('button', { name: /sign in with google/i }))

    expect(await screen.findByText('Google sign-in is unavailable')).toBeInTheDocument()
  })
})

describe('ScannerApp launcher', () => {
  beforeEach(() => {
    localStorage.setItem('commerceos_scanner_jwt', makeJWT(Math.floor(Date.now() / 1000) + 3600))
  })

  it('shows every mini-app, all enabled', async () => {
    renderAt(<ScannerApp />, '/scanner')

    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
    expect(screen.getByText('Inventory')).toBeInTheDocument()
    expect(screen.getByText('Price Lookup')).toBeInTheDocument()
    expect(screen.getByText('Discard')).toBeInTheDocument()
    expect(screen.getByText('Pick Order')).toBeInTheDocument()
    expect(screen.queryByText('Coming soon')).not.toBeInTheDocument()
  })

  it('always renders in the dark appearance, whatever the device setting', async () => {
    renderAt(<ScannerApp />, '/scanner')
    const tile = await screen.findByText('Receive PO')
    expect(tile.closest('.dark')).not.toBeNull()
  })

  it('no tiles are disabled', async () => {
    renderAt(<ScannerApp />, '/scanner')
    await screen.findByText('Receive PO')

    expect(screen.getByRole('button', { name: /Inventory/ })).not.toBeDisabled()
    expect(screen.getByRole('button', { name: /Price Lookup/ })).not.toBeDisabled()
    expect(screen.getByRole('button', { name: 'Discard' })).not.toBeDisabled()
    expect(screen.getByRole('button', { name: /Pick Order/ })).not.toBeDisabled()
  })

  it('Discard tile opens the Discard app scan screen', async () => {
    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Discard'))

    expect(await screen.findByPlaceholderText('Scan or type SKU / barcode…')).toBeInTheDocument()
  })

  it('Back from the purchase order picker returns to the launcher', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Receive PO'))
    await screen.findByText('Select a purchase order')

    await userEvent.click(screen.getByTitle('Back to apps'))

    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
  })
})

describe('ScannerApp purchase order picker', () => {
  beforeEach(() => {
    localStorage.setItem('commerceos_scanner_jwt', makeJWT(Math.floor(Date.now() / 1000) + 3600))
  })

  it('lists sent/partially_received purchase orders by supplier name', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po(), po({ id: 'po-2', status: 'draft' })])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Receive PO'))

    expect(await screen.findByText('Acme Foods')).toBeInTheDocument()
    // draft POs aren't ready for receiving -- only one card should render.
    expect(screen.getAllByText('Acme Foods')).toHaveLength(1)
  })

  it('shows the id, expected date, and order value for each PO', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([
      po({ id: 'a1b2c3d4-0000-0000-0000-000000000000', expected_date: '2026-08-20', total_minor_units: 154999, currency_code: 'USD' }),
    ])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Receive PO'))

    expect(await screen.findByText('#A1B2C3D4')).toBeInTheDocument()
    expect(screen.getByText('USD 1549.99')).toBeInTheDocument()
    expect(screen.getByText(new Date('2026-08-20').toLocaleDateString())).toBeInTheDocument()
  })

  it('falls back to the created date when no expected date is set', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([
      po({ expected_date: null, created_at: '2026-07-01T00:00:00Z' }),
    ])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Receive PO'))

    expect(await screen.findByText(new Date('2026-07-01T00:00:00Z').toLocaleDateString())).toBeInTheDocument()
  })

  it('starts a booking and moves to the booking screen', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])
    vi.mocked(api.startOrResumeBooking).mockResolvedValue(booking())
    vi.mocked(api.getBooking).mockResolvedValue({ booking: booking(), line_items: [], scans: [] })

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Receive PO'))
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('Booking')).toBeInTheDocument()
    expect(api.startOrResumeBooking).toHaveBeenCalledWith(expect.any(String), 'po-1', undefined)
  })

  it('prompts for a warehouse when the PO has none yet, then starts the booking with it', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po({ warehouse_id: null })])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    vi.mocked(api.startOrResumeBooking).mockResolvedValue(booking())
    vi.mocked(api.getBooking).mockResolvedValue({ booking: booking(), line_items: [], scans: [] })

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Receive PO'))
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(await screen.findByText('Pick a warehouse')).toBeInTheDocument()
    await userEvent.click(screen.getByText('Main Warehouse'))

    expect(await screen.findByText('Booking')).toBeInTheDocument()
    expect(api.startOrResumeBooking).toHaveBeenCalledWith(expect.any(String), 'po-1', 'warehouse-1')
  })
})

describe('ScannerApp booking screen', () => {
  beforeEach(() => {
    localStorage.setItem('commerceos_scanner_jwt', makeJWT(Math.floor(Date.now() / 1000) + 3600))
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])
    vi.mocked(api.startOrResumeBooking).mockResolvedValue(booking())
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse(), warehouse({ id: 'warehouse-2', name: 'Overflow Warehouse' })])
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue({ purchase_order: po(), line_items: [], documents: [] })
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([])
  })

  async function enterBookingScreen() {
    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Receive PO'))
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Booking')
  }

  it('shows the PO warehouse and lets it be changed mid-booking', async () => {
    vi.mocked(api.getBooking).mockResolvedValue({ booking: booking(), line_items: [], scans: [] })
    vi.mocked(procurementApi.updatePurchaseOrder).mockResolvedValue(po({ warehouse_id: 'warehouse-2' }))

    await enterBookingScreen()

    expect(await screen.findByText('Main Warehouse')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Change' }))
    await userEvent.click(await screen.findByText('Overflow Warehouse'))

    await waitFor(() => expect(procurementApi.updatePurchaseOrder).toHaveBeenCalledWith(expect.any(String), 'po-1', expect.objectContaining({
      warehouse_id: 'warehouse-2',
    })))
    expect(await screen.findByText('Overflow Warehouse')).toBeInTheDocument()
  })

  it('shows expected vs scanned progress per line', async () => {
    const detail: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [{ id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 4, scanned_at: '', scanned_by: '' }],
    }
    vi.mocked(api.getBooking).mockResolvedValue(detail)

    await enterBookingScreen()

    expect(await screen.findByText('Widget A')).toBeInTheDocument()
    expect(screen.getByText('4 / 10')).toBeInTheDocument()
  })

  it('Receive mode (default) is prefilled with the remaining quantity and can pick an existing batch', async () => {
    const detail: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [{ id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 4, scanned_at: '', scanned_by: '' }],
    }
    vi.mocked(api.getBooking).mockResolvedValue(detail)
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch({ id: 'batch-9', lot_number: 'LOT-9' })])
    vi.mocked(api.scanBookingItem).mockResolvedValue({
      scan: { id: 'scan-2', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 6, scanned_at: '', scanned_by: '' },
    })

    await enterBookingScreen()
    await userEvent.click(await screen.findByText('Widget A'))

    expect(await screen.findByLabelText('Quantity to receive')).toHaveValue('6')
    await userEvent.click(await screen.findByText('LOT-9'))
    await userEvent.click(await screen.findByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(api.scanBookingItem).toHaveBeenCalledWith(expect.any(String), 'booking-1', expect.objectContaining({
      sku_id: 'sku-1', action_type: 'receive', quantity: 6, batch_id: 'batch-9',
    })))
  })

  it('Receive mode can create a new batch with a lot number and expiry date', async () => {
    const detail: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [],
    }
    vi.mocked(api.getBooking).mockResolvedValue(detail)
    vi.mocked(api.scanBookingItem).mockResolvedValue({
      scan: { id: 'scan-2', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 10, scanned_at: '', scanned_by: '' },
    })

    await enterBookingScreen()
    await userEvent.click(await screen.findByText('Widget A'))

    await userEvent.click(await screen.findByText('+ New batch'))
    await userEvent.type(await screen.findByLabelText('Lot number'), 'LOT-NEW')
    await userEvent.type(screen.getByLabelText('Expiry date'), '2027-01-01')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(api.scanBookingItem).toHaveBeenCalledWith(expect.any(String), 'booking-1', expect.objectContaining({
      sku_id: 'sku-1', action_type: 'receive', quantity: 10, lot_number: 'LOT-NEW', expiry_date: '2027-01-01',
    })))
  })

  it('Remove mode stages a correction against a chosen batch, with a reason', async () => {
    const detail: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [{ id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 10, scanned_at: '', scanned_by: '' }],
    }
    vi.mocked(api.getBooking).mockResolvedValue(detail)
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch({ id: 'batch-9', lot_number: 'LOT-9', quantity: 10 })])
    vi.mocked(api.scanBookingItem).mockResolvedValue({
      scan: { id: 'scan-2', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'correction', write_off_quantity: 0, quantity: -3, scanned_at: '', scanned_by: '' },
    })

    await enterBookingScreen()
    await userEvent.click(await screen.findByText('Widget A'))
    await userEvent.click(await screen.findByRole('button', { name: 'Remove' }))

    await userEvent.click(await screen.findByText('LOT-9'))
    await userEvent.type(await screen.findByLabelText('Quantity to remove'), '3')
    await userEvent.type(screen.getByLabelText('Reason (optional)'), 'counted wrong')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    // Nothing real happens yet -- this is still just a staged scan. quantity
    // stays a positive magnitude; action_type carries the direction.
    await waitFor(() => expect(api.scanBookingItem).toHaveBeenCalledWith(expect.any(String), 'booking-1', expect.objectContaining({
      sku_id: 'sku-1', action_type: 'correction', quantity: 3, batch_id: 'batch-9', reason: 'counted wrong',
    })))
  })

  it('Damaged/Lost mode stages a write-off via the same scan endpoint, without touching real inventory yet', async () => {
    const detail: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [{ id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 10, scanned_at: '', scanned_by: '' }],
    }
    vi.mocked(api.getBooking).mockResolvedValue(detail)
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch({ id: 'batch-9', lot_number: 'LOT-9', quantity: 10 })])
    vi.mocked(api.scanBookingItem).mockResolvedValue({
      scan: { id: 'scan-2', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'damaged', write_off_quantity: 2, quantity: 0, scanned_at: '', scanned_by: '' },
    })

    await enterBookingScreen()
    await userEvent.click(await screen.findByText('Widget A'))
    await userEvent.click(await screen.findByRole('button', { name: 'Damaged/lost' }))

    await userEvent.click(await screen.findByText('LOT-9'))
    await userEvent.type(await screen.findByLabelText('Quantity'), '2')
    await userEvent.type(screen.getByLabelText('Reason (optional)'), 'arrived crushed')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(api.scanBookingItem).toHaveBeenCalledWith(expect.any(String), 'booking-1', expect.objectContaining({
      sku_id: 'sku-1', action_type: 'damaged', quantity: 2, batch_id: 'batch-9', reason: 'arrived crushed',
    })))
    expect(inventoryApi.markDamagedOrLost).not.toHaveBeenCalled()
  })

  it('does not let an unlinked line item be tapped to book (no SKU to book against)', async () => {
    const detail: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: null, raw_description: 'Widget B (unlinked)', quantity: 5 }],
      scans: [],
    }
    vi.mocked(api.getBooking).mockResolvedValue(detail)

    await enterBookingScreen()
    await userEvent.click(await screen.findByText('Widget B (unlinked)'))

    expect(screen.queryByLabelText('Quantity received')).not.toBeInTheDocument()
  })

  it('scans an item, opens the confirm screen, and refreshes progress once confirmed', async () => {
    const initial: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [],
    }
    const afterScan: BookingDetail = {
      ...initial,
      scans: [{ id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 5, scanned_at: '', scanned_by: '' }],
    }
    vi.mocked(api.getBooking).mockResolvedValueOnce(initial).mockResolvedValueOnce(afterScan)
    vi.mocked(api.resolveBookingScanCode).mockResolvedValueOnce({ sku_id: 'sku-1', product_name: 'Widget A', units_per_scan: 1 })
    vi.mocked(api.scanBookingItem).mockResolvedValue({
      scan: { id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 5, scanned_at: '', scanned_by: '' },
    })

    await enterBookingScreen()
    await screen.findByText('0 / 10')

    await userEvent.type(screen.getByPlaceholderText('Scan or type barcode…'), 'BARCODE-1{Enter}')

    // Resolved, not booked yet -- confirm screen defaults to the full
    // remaining quantity (10), editable before Confirm.
    expect(await screen.findByLabelText('Quantity to receive')).toHaveValue('10')
    await userEvent.clear(screen.getByLabelText('Quantity to receive'))
    await userEvent.type(screen.getByLabelText('Quantity to receive'), '5')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(api.scanBookingItem).toHaveBeenCalledWith(expect.any(String), 'booking-1', expect.objectContaining({
      sku_id: 'sku-1', quantity: 5,
    })))
    expect(await screen.findByText('5 / 10')).toBeInTheDocument()
  })

  it('scans an item decoded via the camera and opens the confirm screen', async () => {
    const detail: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [],
    }
    vi.mocked(api.getBooking).mockResolvedValue(detail)
    vi.mocked(api.resolveBookingScanCode).mockResolvedValueOnce({ sku_id: 'sku-1', product_name: 'Widget A', units_per_scan: 1 })
    vi.mocked(api.scanBookingItem).mockResolvedValue({
      scan: { id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 1, scanned_at: '', scanned_by: '' },
    })

    await enterBookingScreen()
    await userEvent.click(screen.getByTitle('Use camera'))
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())
    const callback = mockDecodeFromVideoDevice.mock.calls[0][2]
    act(() => { callback({ getText: () => 'CAMERA-BARCODE-1' }, undefined, { stop: vi.fn() }) })

    await waitFor(() => expect(api.resolveBookingScanCode).toHaveBeenCalledWith(expect.any(String), 'booking-1', 'CAMERA-BARCODE-1'))
    expect(await screen.findByRole('button', { name: 'Confirm' })).toBeInTheDocument()
    expect(api.scanBookingItem).not.toHaveBeenCalled()
  })

  it('Finish shows a preview of what is still missing, computed locally -- no completion yet', async () => {
    vi.mocked(api.getBooking).mockResolvedValue({
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [{ id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 6, scanned_at: '', scanned_by: '' }],
    })

    await enterBookingScreen()
    await userEvent.click(screen.getByRole('button', { name: 'Finish' }))

    expect(await screen.findByText('Review before finishing')).toBeInTheDocument()
    expect(screen.getByText('Widget A')).toBeInTheDocument()
    expect(screen.getByText('expected 10 · scanned 6')).toBeInTheDocument()
    expect(api.completeBooking).not.toHaveBeenCalled()
  })

  it('Back from the preview returns to the booking screen without completing', async () => {
    vi.mocked(api.getBooking).mockResolvedValue({
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [],
    })

    await enterBookingScreen()
    await userEvent.click(screen.getByRole('button', { name: 'Finish' }))
    await screen.findByText('Review before finishing')
    await userEvent.click(screen.getByRole('button', { name: 'Back, scan more' }))

    expect(await screen.findByText('Booking')).toBeInTheDocument()
    expect(api.completeBooking).not.toHaveBeenCalled()
  })

  it('Confirm & Complete on the preview actually finishes the booking', async () => {
    vi.mocked(api.getBooking).mockResolvedValue({
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [{ id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 6, scanned_at: '', scanned_by: '' }],
    })
    vi.mocked(api.completeBooking).mockResolvedValue({
      booking: booking({ status: 'completed' }),
      missing_items: [{ line_item_id: 'line-1', raw_description: 'Widget A', expected_qty: 10, scanned_qty: 6 }],
    })

    await enterBookingScreen()
    await userEvent.click(screen.getByRole('button', { name: 'Finish' }))
    await screen.findByText('Review before finishing')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm & complete' }))

    expect(await screen.findByText('Booking complete')).toBeInTheDocument()
    expect(screen.getByText(/weren't fully scanned/)).toBeInTheDocument()
    expect(screen.getByText('Widget A')).toBeInTheDocument()
    expect(api.completeBooking).toHaveBeenCalledWith(expect.any(String), 'booking-1')
  })

  it('pauses the booking and returns to the purchase order picker', async () => {
    vi.mocked(api.getBooking).mockResolvedValue({ booking: booking(), line_items: [], scans: [] })
    vi.mocked(api.pauseBooking).mockResolvedValue(booking({ status: 'paused' }))

    await enterBookingScreen()
    await userEvent.click(screen.getByTitle('Pause and go back'))

    expect(await screen.findByText('Select a purchase order')).toBeInTheDocument()
    expect(api.pauseBooking).toHaveBeenCalledWith(expect.any(String), 'booking-1')
  })
})

describe('ScannerApp sales order picker', () => {
  beforeEach(() => {
    localStorage.setItem('commerceos_scanner_jwt', makeJWT(Math.floor(Date.now() / 1000) + 3600))
  })

  it('lists the picking queue by customer name and order number', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Pick Order'))

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument()
    expect(screen.getByText('#1001')).toBeInTheDocument()
    expect(omsApi.listSalesOrders).toHaveBeenCalledWith(expect.any(String), expect.objectContaining({ awaitingPicking: true }))
  })

  it('starts a picking session and moves to the picking screen', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(api.startOrResumePicking).mockResolvedValue(pickingSession())
    vi.mocked(api.getPickingSession).mockResolvedValue({ session: pickingSession(), line_items: [], scans: [] })

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Pick Order'))
    await userEvent.click(await screen.findByText('Jane Doe'))

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(screen.getByText('Damrak 1, 1012AB Amsterdam, NL')).toBeInTheDocument()
    expect(api.startOrResumePicking).toHaveBeenCalledWith(expect.any(String), 'order-1')
  })

  it('Back from the sales order picker returns to the launcher', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [], total: 0 })

    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Pick Order'))
    await screen.findByText('Select an order to pick')

    await userEvent.click(screen.getByTitle('Back to apps'))

    expect(await screen.findByText('Pick Order')).toBeInTheDocument()
  })
})

describe('ScannerApp picking screen', () => {
  beforeEach(() => {
    localStorage.setItem('commerceos_scanner_jwt', makeJWT(Math.floor(Date.now() / 1000) + 3600))
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
    vi.mocked(api.startOrResumePicking).mockResolvedValue(pickingSession())
  })

  async function enterPickingScreen() {
    renderAt(<ScannerApp />, '/scanner')
    await userEvent.click(await screen.findByText('Pick Order'))
    await userEvent.click(await screen.findByText('Jane Doe'))
    await screen.findByText('#1001')
  }

  const lineItem: PickingSessionDetail['line_items'][number] = {
    id: 'li-1', sku_id: 'sku-1', name: 'Widget A', sku_code: 'WID-A', barcode: '', quantity: 10,
    price_minor_units: 100, total_minor_units: 1000,
  }

  it('shows expected vs picked progress per line', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue({
      session: pickingSession(), line_items: [lineItem],
      scans: [{ id: 'scan-1', session_id: 'session-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 4, scanned_at: '', scanned_by: '' }],
    })

    await enterPickingScreen()

    expect(await screen.findByText('Widget A')).toBeInTheDocument()
    expect(screen.getByText('4 / 10')).toBeInTheDocument()
  })

  it('tapping a line item and confirming Pick stages a pick scan for the remaining quantity', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue({ session: pickingSession(), line_items: [lineItem], scans: [] })
    vi.mocked(api.scanPickItem).mockResolvedValue({
      scan: { id: 'scan-1', session_id: 'session-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 10, scanned_at: '', scanned_by: '' },
    })

    await enterPickingScreen()
    await userEvent.click(await screen.findByText('Widget A'))

    expect(await screen.findByLabelText('Quantity to pick')).toHaveValue('10')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm pick' }))

    await waitFor(() => expect(api.scanPickItem).toHaveBeenCalledWith(expect.any(String), 'session-1', expect.objectContaining({
      sku_id: 'sku-1', action_type: 'pick', quantity: 10,
    })))
  })

  it('scanning a code that matches a line item marks one unit picked immediately, with no sheet', async () => {
    vi.mocked(api.getPickingSession)
      // first load: the /scanner/pick/:sessionId route resolving the session's order
      .mockResolvedValueOnce({ session: pickingSession(), line_items: [lineItem], scans: [] })
      .mockResolvedValueOnce({ session: pickingSession(), line_items: [lineItem], scans: [] })
      .mockResolvedValueOnce({
        session: pickingSession(), line_items: [lineItem],
        scans: [{ id: 'scan-1', session_id: 'session-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 1, scanned_at: '', scanned_by: '' }],
      })
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget A' })
    vi.mocked(api.scanPickItem).mockResolvedValue({
      scan: { id: 'scan-1', session_id: 'session-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 1, scanned_at: '', scanned_by: '' },
    })

    await enterPickingScreen()
    await userEvent.type(screen.getByPlaceholderText('Scan or type barcode…'), 'WID-A-BARCODE{Enter}')

    await waitFor(() => expect(api.scanPickItem).toHaveBeenCalledWith(expect.any(String), 'session-1', expect.objectContaining({
      sku_id: 'sku-1', action_type: 'pick', quantity: 1,
    })))
    expect(screen.queryByLabelText('Quantity to pick')).not.toBeInTheDocument()
    expect(await screen.findByText('1 / 10')).toBeInTheDocument()
  })

  it('scanning a code already fully picked shows a blocking error and stages nothing', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue({
      session: pickingSession(), line_items: [lineItem],
      scans: [{ id: 'scan-1', session_id: 'session-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 10, scanned_at: '', scanned_by: '' }],
    })
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget A' })

    await enterPickingScreen()
    const input = screen.getByPlaceholderText('Scan or type barcode…')
    await userEvent.type(input, 'WID-A-BARCODE{Enter}')

    expect(await screen.findByText('Already fully picked')).toBeInTheDocument()
    expect(api.scanPickItem).not.toHaveBeenCalled()
    expect(playErrorSound).toHaveBeenCalled()
    expect(input).toBeDisabled()

    await userEvent.click(screen.getByRole('button', { name: 'OK' }))
    expect(screen.queryByText('Already fully picked')).not.toBeInTheDocument()
    expect(input).toBeEnabled()
  })

  it('scanning a code not on the order shows a blocking error instead of opening the add-item sheet', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue({ session: pickingSession(), line_items: [lineItem], scans: [] })
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-other', product_name: 'Gizmo' })
    // No line item on this order carries sku-other -- still attempted
    // server-side (it may auto-link an unmatched Woo line by product name),
    // and here the server genuinely rejects it as not belonging to this
    // order.
    vi.mocked(api.scanPickItem).mockRejectedValue(new Error('scanned item does not match any line item on this order'))

    await enterPickingScreen()
    await userEvent.type(screen.getByPlaceholderText('Scan or type barcode…'), 'GIZMO-BARCODE{Enter}')

    expect(await screen.findByText('Item not on this order')).toBeInTheDocument()
    expect(api.scanPickItem).toHaveBeenCalledWith(expect.any(String), 'session-1', {
      sku_id: 'sku-other', action_type: 'pick', quantity: 1,
    })
    expect(screen.queryByRole('heading', { name: 'Add freebie or new item' })).not.toBeInTheDocument()
    expect(playErrorSound).toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'OK' }))
    expect(screen.queryByText('Item not on this order')).not.toBeInTheDocument()
  })

  it('picking only picks -- the line sheet has no substitute or refund, and there is no add-item', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue({ session: pickingSession(), line_items: [lineItem], scans: [] })

    await enterPickingScreen()
    await userEvent.click(await screen.findByText(lineItem.name))

    expect(await screen.findByRole('button', { name: 'Confirm pick' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Substitute/ })).not.toBeInTheDocument()
    expect(screen.queryByRole('button', { name: /Refund/ })).not.toBeInTheDocument()
    expect(screen.queryByText('Add freebie or new item')).not.toBeInTheDocument()
    expect(screen.getByText(/Substitutes and refunds happen in packing/)).toBeInTheDocument()
  })

  it('Finish shows a preview of what is still missing, computed locally -- no completion yet', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue({
      session: pickingSession(), line_items: [lineItem],
      scans: [{ id: 'scan-1', session_id: 'session-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 6, scanned_at: '', scanned_by: '' }],
    })

    await enterPickingScreen()
    await userEvent.click(screen.getByRole('button', { name: 'Finish' }))

    expect(await screen.findByText('Review before finishing')).toBeInTheDocument()
    expect(screen.getByText('Widget A')).toBeInTheDocument()
    expect(screen.getByText('expected 10 · picked 6')).toBeInTheDocument()
    expect(api.completePicking).not.toHaveBeenCalled()
  })

  it('Confirm & Complete finishes the session and shows the missing-items result', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue({
      session: pickingSession(), line_items: [lineItem],
      scans: [{ id: 'scan-1', session_id: 'session-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 6, scanned_at: '', scanned_by: '' }],
    })
    vi.mocked(api.completePicking).mockResolvedValue({
      completed: true,
      session: pickingSession({ status: 'completed' }),
      missing_items: [{ line_item_id: 'li-1', name: 'Widget A', expected_qty: 10, picked_qty: 6 }],
    })

    await enterPickingScreen()
    await userEvent.click(screen.getByRole('button', { name: 'Finish' }))
    await screen.findByText('Review before finishing')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm & complete' }))

    expect(await screen.findByText('Order picked')).toBeInTheDocument()
    expect(screen.getByText(/weren't fully picked/)).toBeInTheDocument()
    expect(api.completePicking).toHaveBeenCalledWith(expect.any(String), 'session-1', undefined)
  })

  it('pauses the picking session and returns to the sales order picker', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue({ session: pickingSession(), line_items: [], scans: [] })
    vi.mocked(api.pausePicking).mockResolvedValue(pickingSession({ status: 'paused' }))

    await enterPickingScreen()
    await userEvent.click(screen.getByTitle('Pause and go back'))

    expect(await screen.findByText('Select an order to pick')).toBeInTheDocument()
    expect(api.pausePicking).toHaveBeenCalledWith(expect.any(String), 'session-1')
  })

  it('shows a product thumbnail for a line item linked to a SKU with media', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue({
      session: pickingSession(), line_items: [{ ...lineItem, product_id: 'product-1' }], scans: [],
    })
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([
      { id: 'media-1', tenant_id: 't1', product_id: 'product-1', url: 'https://example.com/thumb.jpg', type: 'image', position: 0, alt_text: 'Widget A photo', created_at: '' },
    ])

    await enterPickingScreen()

    const img = await screen.findByAltText('Widget A photo')
    expect(img).toHaveAttribute('src', 'https://example.com/thumb.jpg')
    expect(pmsApi.listProductMedia).toHaveBeenCalledWith(expect.any(String), 'product-1')
  })

  it('falls back to a placeholder icon when the line has no product_id', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue({
      session: pickingSession(), line_items: [{ ...lineItem, product_id: null }], scans: [],
    })

    await enterPickingScreen()

    expect(await screen.findByText('Widget A')).toBeInTheDocument()
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
    expect(pmsApi.listProductMedia).not.toHaveBeenCalled()
  })
})
