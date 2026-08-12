// @vitest-environment jsdom
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ScannerApp from '../ScannerApp'
import type { Booking, BookingDetail } from '../types'
import type { PurchaseOrder } from '../../procurement/types'
import type { Supplier } from '../../suppliers/types'
import type { Warehouse, Batch } from '../../inventory/types'

vi.mock('../api', () => ({
  claimPairingCode: vi.fn(),
  claimPairingCodeByToken: vi.fn(),
  generatePairingCode: vi.fn(),
  getPairingCodeStatus: vi.fn(),
  startOrResumeBooking: vi.fn(),
  getBooking: vi.fn(),
  scanBookingItem: vi.fn(),
  pauseBooking: vi.fn(),
  completeBooking: vi.fn(),
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

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
  mockDecodeFromVideoDevice.mockResolvedValue({ stop: vi.fn() })
})

describe('ScannerApp pairing', () => {
  it('defaults to the Scan tab, not Type code', async () => {
    render(<ScannerApp />)

    expect(screen.getByPlaceholderText('Scan QR code or barcode…')).toBeInTheDocument()
    expect(screen.queryByPlaceholderText('000000')).not.toBeInTheDocument()
  })

  it('pairs via the Scan tab using the token path, not the digit-code path', async () => {
    vi.mocked(api.claimPairingCodeByToken).mockResolvedValue({ token: makeJWT(Math.floor(Date.now() / 1000) + 3600) })
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])

    render(<ScannerApp />)
    // A hardware scan-engine keyboard-wedges the decoded token then a
    // terminator keystroke -- typing + Enter simulates that.
    await userEvent.type(screen.getByPlaceholderText('Scan QR code or barcode…'), 'opaque-token-abc{Enter}')

    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
    expect(api.claimPairingCodeByToken).toHaveBeenCalledWith('opaque-token-abc')
    expect(api.claimPairingCode).not.toHaveBeenCalled()
  })

  it('shows an error on an invalid scanned token', async () => {
    vi.mocked(api.claimPairingCodeByToken).mockRejectedValue(new Error('pairing code is invalid or expired'))

    render(<ScannerApp />)
    await userEvent.type(screen.getByPlaceholderText('Scan QR code or barcode…'), 'bad-token{Enter}')

    expect(await screen.findByText('pairing code is invalid or expired')).toBeInTheDocument()
  })

  it('pairs via the camera and auto-submits on decode', async () => {
    vi.mocked(api.claimPairingCodeByToken).mockResolvedValue({ token: makeJWT(Math.floor(Date.now() / 1000) + 3600) })
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])

    render(<ScannerApp />)
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

    render(<ScannerApp />)
    await userEvent.click(screen.getByRole('button', { name: 'Type code' }))
    await userEvent.type(screen.getByPlaceholderText('000000'), '123456')
    await userEvent.click(screen.getByRole('button', { name: 'Pair' }))

    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
    expect(api.claimPairingCode).toHaveBeenCalledWith('123456')
  })

  it('shows an error on an invalid typed code', async () => {
    vi.mocked(api.claimPairingCode).mockRejectedValue(new Error('pairing code is invalid or expired'))

    render(<ScannerApp />)
    await userEvent.click(screen.getByRole('button', { name: 'Type code' }))
    await userEvent.type(screen.getByPlaceholderText('000000'), '000000')
    await userEvent.click(screen.getByRole('button', { name: 'Pair' }))

    expect(await screen.findByText('pairing code is invalid or expired')).toBeInTheDocument()
  })

  it('skips the claim screen when a valid token is already stored', async () => {
    localStorage.setItem('commerceos_scanner_jwt', makeJWT(Math.floor(Date.now() / 1000) + 3600))

    render(<ScannerApp />)

    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
  })
})

describe('ScannerApp launcher', () => {
  beforeEach(() => {
    localStorage.setItem('commerceos_scanner_jwt', makeJWT(Math.floor(Date.now() / 1000) + 3600))
  })

  it('shows every mini-app, with only Pick Order disabled', async () => {
    render(<ScannerApp />)

    expect(await screen.findByText('Receive PO')).toBeInTheDocument()
    expect(screen.getByText('Inventory')).toBeInTheDocument()
    expect(screen.getByText('Price Lookup')).toBeInTheDocument()
    expect(screen.getByText('Pick Order')).toBeInTheDocument()
    expect(screen.getAllByText('Coming soon')).toHaveLength(1)
  })

  it('disabled tiles are not clickable', async () => {
    render(<ScannerApp />)
    await screen.findByText('Receive PO')

    expect(screen.getByRole('button', { name: /Inventory/ })).not.toBeDisabled()
    expect(screen.getByRole('button', { name: /Price Lookup/ })).not.toBeDisabled()
    expect(screen.getByRole('button', { name: /Pick Order/ })).toBeDisabled()
  })

  it('Back from the purchase order picker returns to the launcher', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])

    render(<ScannerApp />)
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

    render(<ScannerApp />)
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

    render(<ScannerApp />)
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

    render(<ScannerApp />)
    await userEvent.click(await screen.findByText('Receive PO'))

    expect(await screen.findByText(new Date('2026-07-01T00:00:00Z').toLocaleDateString())).toBeInTheDocument()
  })

  it('starts a booking and moves to the booking screen', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])
    vi.mocked(api.startOrResumeBooking).mockResolvedValue(booking())
    vi.mocked(api.getBooking).mockResolvedValue({ booking: booking(), line_items: [], scans: [] })

    render(<ScannerApp />)
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

    render(<ScannerApp />)
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
    render(<ScannerApp />)
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

    expect(screen.getByLabelText('Quantity to receive')).toHaveValue('6')
    await userEvent.click(await screen.findByText('LOT-9'))
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

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
    await userEvent.type(screen.getByLabelText('Lot number'), 'LOT-NEW')
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
    await userEvent.click(screen.getByRole('button', { name: 'Remove' }))

    await userEvent.click(await screen.findByText('LOT-9'))
    await userEvent.type(screen.getByLabelText('Quantity to remove'), '3')
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
    await userEvent.click(screen.getByRole('button', { name: 'Damaged/Lost' }))

    await userEvent.click(await screen.findByText('LOT-9'))
    await userEvent.type(screen.getByLabelText('Quantity'), '2')
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

  it('scans an item and refreshes progress', async () => {
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
    vi.mocked(api.scanBookingItem).mockResolvedValue({
      scan: { id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 5, scanned_at: '', scanned_by: '' },
    })

    await enterBookingScreen()
    await screen.findByText('0 / 10')

    await userEvent.type(screen.getByPlaceholderText('Scan or type barcode…'), 'BARCODE-1')
    await userEvent.clear(screen.getByDisplayValue('1'))
    await userEvent.type(screen.getByDisplayValue(''), '5')
    await userEvent.click(screen.getByRole('button', { name: 'Add' }))

    await waitFor(() => expect(api.scanBookingItem).toHaveBeenCalledWith(expect.any(String), 'booking-1', expect.objectContaining({
      code: 'BARCODE-1', quantity: 5,
    })))
    expect(await screen.findByText('5 / 10')).toBeInTheDocument()
  })

  it('scans an item decoded via the camera', async () => {
    const detail: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [],
    }
    vi.mocked(api.getBooking).mockResolvedValue(detail)
    vi.mocked(api.scanBookingItem).mockResolvedValue({
      scan: { id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', write_off_quantity: 0, quantity: 1, scanned_at: '', scanned_by: '' },
    })

    await enterBookingScreen()
    await userEvent.click(screen.getByTitle('Use camera'))
    await waitFor(() => expect(mockDecodeFromVideoDevice).toHaveBeenCalled())
    const callback = mockDecodeFromVideoDevice.mock.calls[0][2]
    act(() => { callback({ getText: () => 'CAMERA-BARCODE-1' }, undefined, { stop: vi.fn() }) })

    await waitFor(() => expect(api.scanBookingItem).toHaveBeenCalledWith(expect.any(String), 'booking-1', expect.objectContaining({
      code: 'CAMERA-BARCODE-1',
    })))
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

    expect(screen.getByText('Booking')).toBeInTheDocument()
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
    await userEvent.click(screen.getByRole('button', { name: 'Confirm & Complete' }))

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
