// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { BookingScreen } from '../BookingScreen'
import type { BookingDetail } from '../types'
import type { PurchaseOrder } from '../../procurement/types'
import type { Warehouse } from '../../inventory/types'
import type { SKU, Product, ProductDetail } from '../../pms/types'

vi.mock('../api', () => ({
  getBooking: vi.fn(),
  scanBookingItem: vi.fn(),
  resolveBookingScanCode: vi.fn(),
  pauseBooking: vi.fn(),
  completeBooking: vi.fn(),
}))

vi.mock('../../procurement/api', () => ({
  getPurchaseOrder: vi.fn(),
  updatePurchaseOrder: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
  listBatches: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listProducts: vi.fn(),
  getProduct: vi.fn(),
  listProductMedia: vi.fn(),
}))

import * as api from '../api'
import * as procurementApi from '../../procurement/api'
import * as inventoryApi from '../../inventory/api'
import * as pmsApi from '../../pms/api'

const TOKEN = 'test-token'

function sku(overrides: Partial<SKU> = {}): SKU {
  return {
    id: 'sku-1', tenant_id: 't1', variant_id: 'v1', sku_code: 'WID-L',
    price_minor_units: 1999, currency_code: 'USD', is_virtual: false, status: 'active',
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function product(overrides: Partial<Product> = {}): Product {
  return {
    id: 'product-1', tenant_id: 't1', name: 'Widget', slug: 'widget', description: '',
    status: 'active', has_variants: false, created_at: '', updated_at: '',
    ...overrides,
  } as Product
}

function productDetail(p: Product, skus: SKU[]): ProductDetail {
  return {
    product: p,
    variants: [{ id: `v-${p.id}`, tenant_id: 't1', product_id: p.id, name: 'Default', position: 0, created_at: '', updated_at: '', skus }],
  }
}

function warehouse(overrides: Partial<Warehouse> = {}): Warehouse {
  return {
    id: 'wh-1', tenant_id: 't1', name: 'Main Warehouse',
    address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
    created_at: '', updated_at: '',
    ...overrides,
  }
}

function purchaseOrder(overrides: Partial<PurchaseOrder> = {}): PurchaseOrder {
  return {
    id: 'po-1', tenant_id: 't1', warehouse_id: 'wh-1', status: 'sent', currency_code: 'EUR', notes: '',
    is_potential_duplicate: false, total_minor_units: 0, created_at: '', updated_at: '',
    ...overrides,
  }
}

function bookingDetail(overrides: Partial<BookingDetail> = {}): BookingDetail {
  return {
    booking: { id: 'booking-1', tenant_id: 't1', purchase_order_id: 'po-1', status: 'in_progress', started_at: '', completed_by: '' },
    line_items: [
      {
        id: 'li-1', sku_id: 'sku-1', raw_description: 'Farofa Premium', quantity: 36,
        suggested_lot_number: 'PT26004', suggested_expiry_date: '2027-03-01',
      },
    ],
    scans: [],
    ...overrides,
  }
}

describe('BookingScreen / BookLineSheet suggested batch pre-fill', () => {
  it('pre-fills lot number and expiry date from the line item suggestion, without submitting anything', async () => {
    vi.mocked(api.getBooking).mockResolvedValue(bookingDetail())
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue({ purchase_order: purchaseOrder(), line_items: [], documents: [] })
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([])

    render(<BookingScreen token={TOKEN} bookingId="booking-1" onExit={vi.fn()} onAuthError={vi.fn()} />)

    const row = await screen.findByText('Farofa Premium')
    await userEvent.click(row)

    expect(await screen.findByText(/Suggested from the uploaded document/)).toBeInTheDocument()
    expect(screen.getByLabelText('Lot number')).toHaveValue('PT26004')
    expect(screen.getByLabelText('Expiry date')).toHaveValue('2027-03-01')

    // Nothing is booked just from opening the sheet -- still requires
    // hitting Confirm.
    expect(api.scanBookingItem).not.toHaveBeenCalled()
  })

  it('leaves the batch picker on general ledger with no suggestion to pre-fill from', async () => {
    vi.mocked(api.getBooking).mockResolvedValue(bookingDetail({
      line_items: [{ id: 'li-2', sku_id: 'sku-2', raw_description: 'Unlabeled Widget', quantity: 5 }],
    }))
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue({ purchase_order: purchaseOrder(), line_items: [], documents: [] })
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([])

    render(<BookingScreen token={TOKEN} bookingId="booking-1" onExit={vi.fn()} onAuthError={vi.fn()} />)

    const row = await screen.findByText('Unlabeled Widget')
    await userEvent.click(row)

    await waitFor(() => expect(screen.getByText('General ledger').closest('button')).toHaveClass('border-blue-500'))
    expect(screen.queryByLabelText('Lot number')).not.toBeInTheDocument()
  })
})

describe('BookingScreen scan-to-confirm-screen flow', () => {
  it('resolves an exact code match and opens the confirm screen instead of booking instantly', async () => {
    vi.mocked(api.getBooking).mockResolvedValue(bookingDetail())
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue({ purchase_order: purchaseOrder(), line_items: [], documents: [] })
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([])
    vi.mocked(api.resolveBookingScanCode).mockResolvedValueOnce({ sku_id: 'sku-1', product_name: 'Farofa Premium', units_per_scan: 1 })
    vi.mocked(api.scanBookingItem).mockResolvedValueOnce({
      scan: { id: 'scan-1', tenant_id: 't1', booking_id: 'booking-1', sku_id: 'sku-1', action_type: 'receive', quantity: 1, write_off_quantity: 0, scanned_at: '', scanned_by: '' },
    })

    render(<BookingScreen token={TOKEN} bookingId="booking-1" onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Farofa Premium')

    await userEvent.type(screen.getByPlaceholderText('Scan or type barcode…'), 'BARCODE-1{Enter}')

    // The confirm screen (BookLineSheet) is up -- nothing booked yet.
    expect(await screen.findByText('Received 0 of 36')).toBeInTheDocument()
    expect(api.scanBookingItem).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(api.scanBookingItem).toHaveBeenCalledWith(TOKEN, 'booking-1', expect.objectContaining({
      sku_id: 'sku-1',
    })))
  })

  it('defaults the confirm screen to the full remaining quantity for a line item match, ignoring any alias multiplier', async () => {
    vi.mocked(api.getBooking).mockResolvedValue(bookingDetail())
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue({ purchase_order: purchaseOrder(), line_items: [], documents: [] })
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([])
    vi.mocked(api.resolveBookingScanCode).mockResolvedValueOnce({ sku_id: 'sku-1', product_name: 'Farofa Premium', units_per_scan: 12 })

    render(<BookingScreen token={TOKEN} bookingId="booking-1" onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Farofa Premium')

    await userEvent.type(screen.getByPlaceholderText('Scan or type barcode…'), 'CASE-1{Enter}')

    expect(await screen.findByText('Received 0 of 36')).toBeInTheDocument()
    expect(screen.getByLabelText('Quantity to receive')).toHaveValue('36')
  })

  it('defaults the confirm screen quantity to the alias multiplier when the resolved SKU is not on this PO', async () => {
    vi.mocked(api.getBooking).mockResolvedValue(bookingDetail())
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue({ purchase_order: purchaseOrder(), line_items: [], documents: [] })
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([])
    vi.mocked(api.resolveBookingScanCode).mockResolvedValueOnce({ sku_id: 'sku-9', product_name: 'Widget Pro', units_per_scan: 12 })

    render(<BookingScreen token={TOKEN} bookingId="booking-1" onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Farofa Premium')

    await userEvent.type(screen.getByPlaceholderText('Scan or type barcode…'), 'CASE-1{Enter}')

    expect(await screen.findByText('Widget Pro')).toBeInTheDocument()
    expect(screen.getByLabelText('Quantity to receive')).toHaveValue('12')
  })

  it('falls back to a partial match and opens the confirm screen when exactly one SKU matches', async () => {
    vi.mocked(api.getBooking).mockResolvedValue(bookingDetail())
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue({ purchase_order: purchaseOrder(), line_items: [], documents: [] })
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([])
    vi.mocked(api.resolveBookingScanCode).mockRejectedValueOnce(new Error('sku not found'))
    vi.mocked(api.scanBookingItem).mockResolvedValueOnce({
      scan: { id: 'scan-1', tenant_id: 't1', booking_id: 'booking-1', sku_id: 'sku-9', action_type: 'receive', quantity: 1, write_off_quantity: 0, scanned_at: '', scanned_by: '' },
    })
    const p = product({ id: 'p1', name: 'Widget' })
    const s = sku({ id: 'sku-9', sku_code: 'WID-9' })
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [p], total: 1 })
    vi.mocked(pmsApi.getProduct).mockResolvedValue(productDetail(p, [s]))

    render(<BookingScreen token={TOKEN} bookingId="booking-1" onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Farofa Premium')

    await userEvent.type(screen.getByPlaceholderText('Scan or type barcode…'), 'WID-{Enter}')

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(api.scanBookingItem).toHaveBeenCalledWith(TOKEN, 'booking-1', expect.objectContaining({
      sku_id: 'sku-9',
    })))
  })

  it('shows a candidate list when the partial-match fallback finds more than one SKU, and opens the confirm screen for the tapped one', async () => {
    vi.mocked(api.getBooking).mockResolvedValue(bookingDetail())
    vi.mocked(procurementApi.getPurchaseOrder).mockResolvedValue({ purchase_order: purchaseOrder(), line_items: [], documents: [] })
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([])
    vi.mocked(api.resolveBookingScanCode).mockRejectedValueOnce(new Error('sku not found'))
    vi.mocked(api.scanBookingItem).mockResolvedValueOnce({
      scan: { id: 'scan-1', tenant_id: 't1', booking_id: 'booking-1', sku_id: 'sku-9', action_type: 'receive', quantity: 1, write_off_quantity: 0, scanned_at: '', scanned_by: '' },
    })
    const p1 = product({ id: 'p1', name: 'Widget' })
    const p2 = product({ id: 'p2', name: 'Widget Pro' })
    const s1 = sku({ id: 'sku-8', sku_code: 'WID-L' })
    const s2 = sku({ id: 'sku-9', sku_code: 'WID-PRO' })
    vi.mocked(pmsApi.listProducts).mockResolvedValue({ products: [p1, p2], total: 2 })
    vi.mocked(pmsApi.getProduct).mockImplementation(async (_token, id) =>
      id === 'p1' ? productDetail(p1, [s1]) : productDetail(p2, [s2]))
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])

    render(<BookingScreen token={TOKEN} bookingId="booking-1" onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Farofa Premium')

    await userEvent.type(screen.getByPlaceholderText('Scan or type barcode…'), 'WID{Enter}')

    expect(await screen.findByText('Widget Pro')).toBeInTheDocument()
    await userEvent.click(screen.getByText('Widget Pro'))

    expect(await screen.findByRole('button', { name: 'Confirm' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    await waitFor(() => expect(api.scanBookingItem).toHaveBeenCalledWith(TOKEN, 'booking-1', expect.objectContaining({
      sku_id: 'sku-9',
    })))
  })
})
