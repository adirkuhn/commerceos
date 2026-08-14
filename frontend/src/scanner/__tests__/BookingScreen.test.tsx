// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { BookingScreen } from '../BookingScreen'
import type { BookingDetail } from '../types'
import type { PurchaseOrder } from '../../procurement/types'
import type { Warehouse } from '../../inventory/types'

vi.mock('../api', () => ({
  getBooking: vi.fn(),
  scanBookingItem: vi.fn(),
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

import * as api from '../api'
import * as procurementApi from '../../procurement/api'
import * as inventoryApi from '../../inventory/api'

const TOKEN = 'test-token'

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
