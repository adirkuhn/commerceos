// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ShipmentTrackingSection } from '../ShipmentTrackingSection'
import type { Shipment } from '../../shipping/types'

vi.mock('../../shipping/api', () => ({
  listShipmentTrackingEvents: vi.fn(),
  refreshShipmentTracking: vi.fn(),
  cancelShipment: vi.fn(),
}))
import * as shippingApi from '../../shipping/api'

function shipment(overrides: Partial<Shipment> = {}): Shipment {
  return {
    id: 'ship-1', sales_order_id: 'so-1', connection_id: 'pn', carrier: 1, barcode: '3SABC',
    status: 1, status_main: 'registered', label_format: 'pdf',
    recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
    created_at: '2026-10-01T00:00:00Z',
    ...overrides,
  }
}

const KEPT = /don’t hand over the old label/i

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.listShipmentTrackingEvents).mockResolvedValue([])
})

describe('ShipmentTrackingSection cancel outcome', () => {
  it('shows a stored "label kept" outcome on a cancelled shipment', async () => {
    render(<ShipmentTrackingSection token="t" onCancelled={vi.fn()} shipment={shipment({
      cancelled_at: '2026-10-02T00:00:00Z', cancel_note: 'English note', cancel_deleted_on_carrier: false, cancel_label_kept: true,
    })} />)
    expect(await screen.findByText(KEPT)).toBeInTheDocument()
    expect(screen.queryByText('English note')).not.toBeInTheDocument()
  })

  it('still shows the stored deleted outcome and the English note fallback', async () => {
    const { rerender } = render(<ShipmentTrackingSection token="t" onCancelled={vi.fn()} shipment={shipment({
      cancelled_at: '2026-10-02T00:00:00Z', cancel_note: 'English note', cancel_deleted_on_carrier: true,
    })} />)
    expect(await screen.findByText('Shipment deleted on MyParcel.')).toBeInTheDocument()
    rerender(<ShipmentTrackingSection token="t" onCancelled={vi.fn()} shipment={shipment({
      cancelled_at: '2026-10-02T00:00:00Z', cancel_note: 'English note',
    })} />)
    expect(await screen.findByText('English note')).toBeInTheDocument()
  })

  it('explains a kept label right after cancelling', async () => {
    vi.mocked(shippingApi.cancelShipment).mockResolvedValue({ deleted_on_myparcel: false, label_kept: true, message: 'English' })
    const onCancelled = vi.fn()
    const user = userEvent.setup()
    render(<ShipmentTrackingSection token="t" onCancelled={onCancelled} shipment={shipment()} />)

    await user.click(screen.getByRole('button', { name: 'Delete' }))
    await user.click(await screen.findByRole('button', { name: 'Cancel shipment' }))

    expect(await screen.findByText(KEPT)).toBeInTheDocument()
    expect(shippingApi.cancelShipment).toHaveBeenCalledWith('t', 'ship-1')
    expect(onCancelled).toHaveBeenCalled()
  })
})
