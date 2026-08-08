// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import ScannerApp from '../ScannerApp'
import type { Booking, BookingDetail } from '../types'
import type { PurchaseOrder } from '../../procurement/types'
import type { Supplier } from '../../suppliers/types'
import type { Warehouse } from '../../inventory/types'

vi.mock('../api', () => ({
  claimPairingCode: vi.fn(),
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
}))

vi.mock('../../suppliers/api', () => ({
  listSuppliers: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
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

beforeEach(() => {
  vi.clearAllMocks()
  localStorage.clear()
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
})

describe('ScannerApp pairing', () => {
  it('shows the claim screen and pairs on valid code', async () => {
    vi.mocked(api.claimPairingCode).mockResolvedValue({ token: makeJWT(Math.floor(Date.now() / 1000) + 3600) })
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])

    render(<ScannerApp />)
    expect(screen.getByText('Pair this scanner')).toBeInTheDocument()

    await userEvent.type(screen.getByPlaceholderText('000000'), '123456')
    await userEvent.click(screen.getByRole('button', { name: 'Pair' }))

    expect(await screen.findByText('Select a purchase order')).toBeInTheDocument()
    expect(api.claimPairingCode).toHaveBeenCalledWith('123456')
  })

  it('shows an error on an invalid code', async () => {
    vi.mocked(api.claimPairingCode).mockRejectedValue(new Error('pairing code is invalid or expired'))

    render(<ScannerApp />)
    await userEvent.type(screen.getByPlaceholderText('000000'), '000000')
    await userEvent.click(screen.getByRole('button', { name: 'Pair' }))

    expect(await screen.findByText('pairing code is invalid or expired')).toBeInTheDocument()
  })

  it('skips the claim screen when a valid token is already stored', async () => {
    localStorage.setItem('commerceos_scanner_jwt', makeJWT(Math.floor(Date.now() / 1000) + 3600))
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])

    render(<ScannerApp />)

    expect(await screen.findByText('Select a purchase order')).toBeInTheDocument()
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

    expect(await screen.findByText('Acme Foods')).toBeInTheDocument()
    // draft POs aren't ready for receiving -- only one card should render.
    expect(screen.getAllByText('Acme Foods')).toHaveLength(1)
  })

  it('starts a booking and moves to the booking screen', async () => {
    vi.mocked(procurementApi.listPurchaseOrders).mockResolvedValue([po()])
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])
    vi.mocked(api.startOrResumeBooking).mockResolvedValue(booking())
    vi.mocked(api.getBooking).mockResolvedValue({ booking: booking(), line_items: [], scans: [] })

    render(<ScannerApp />)
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
  })

  async function enterBookingScreen() {
    render(<ScannerApp />)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await screen.findByText('Booking')
  }

  it('shows expected vs scanned progress per line', async () => {
    const detail: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [{ id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', quantity: 4, scanned_at: '', scanned_by: '' }],
    }
    vi.mocked(api.getBooking).mockResolvedValue(detail)

    await enterBookingScreen()

    expect(await screen.findByText('Widget A')).toBeInTheDocument()
    expect(screen.getByText('4 / 10')).toBeInTheDocument()
  })

  it('scans an item and refreshes progress', async () => {
    const initial: BookingDetail = {
      booking: booking(),
      line_items: [{ id: 'line-1', sku_id: 'sku-1', raw_description: 'Widget A', quantity: 10 }],
      scans: [],
    }
    const afterScan: BookingDetail = {
      ...initial,
      scans: [{ id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', quantity: 5, scanned_at: '', scanned_by: '' }],
    }
    vi.mocked(api.getBooking).mockResolvedValueOnce(initial).mockResolvedValueOnce(afterScan)
    vi.mocked(api.scanBookingItem).mockResolvedValue({
      scan: { id: 'scan-1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', quantity: 5, scanned_at: '', scanned_by: '' },
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

  it('finishes the booking and shows a missing-items report', async () => {
    vi.mocked(api.getBooking).mockResolvedValue({ booking: booking(), line_items: [], scans: [] })
    vi.mocked(api.completeBooking).mockResolvedValue({
      booking: booking({ status: 'completed' }),
      missing_items: [{ line_item_id: 'line-1', raw_description: 'Widget A', expected_qty: 10, scanned_qty: 6 }],
    })

    await enterBookingScreen()
    await userEvent.click(screen.getByRole('button', { name: 'Finish' }))

    expect(await screen.findByText('Booking complete')).toBeInTheDocument()
    expect(screen.getByText(/weren't fully scanned/)).toBeInTheDocument()
    expect(screen.getByText('Widget A')).toBeInTheDocument()
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
