// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ExistingShipmentConfirmModal } from '../ExistingShipmentConfirmModal'
import type { Shipment } from '../types'

vi.mock('../api', () => ({
  cancelShipment: vi.fn(),
}))

import * as shippingApi from '../api'

function shipment(overrides: Partial<Shipment> = {}): Shipment {
  return {
    id: 'ship-1', sales_order_id: 'so-1', connection_id: 'conn-1', carrier: 1, barcode: '3SMYPA1',
    status: 1, status_main: 'registered', label_format: 'pdf',
    recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
    created_at: '2026-08-23T00:00:00Z',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('ExistingShipmentConfirmModal', () => {
  it('shows the existing shipment barcode and calls onReprint when confirmed', async () => {
    const onReprint = vi.fn()
    const onCancel = vi.fn()
    const onCreateNew = vi.fn()
    const user = userEvent.setup()
    render(<ExistingShipmentConfirmModal token="tok" shipment={shipment({ barcode: '3SMYPA9' })} onReprint={onReprint} onCancel={onCancel} onCreateNew={onCreateNew} />)

    expect(screen.getByText(/3SMYPA9/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /Download & print again/ }))
    expect(onReprint).toHaveBeenCalled()
    expect(onCancel).not.toHaveBeenCalled()
  })

  it('calls onCancel without calling onReprint', async () => {
    const onReprint = vi.fn()
    const onCancel = vi.fn()
    const onCreateNew = vi.fn()
    const user = userEvent.setup()
    render(<ExistingShipmentConfirmModal token="tok" shipment={shipment()} onReprint={onReprint} onCancel={onCancel} onCreateNew={onCreateNew} />)

    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onCancel).toHaveBeenCalled()
    expect(onReprint).not.toHaveBeenCalled()
  })

  it('shows MyParcel\'s outcome and lets staff create a new shipment even when the delete was rejected', async () => {
    vi.mocked(shippingApi.cancelShipment).mockResolvedValue({
      deleted_on_myparcel: false,
      message: "MyParcel could not delete this shipment (422); marked as cancelled locally.",
    })
    const onCreateNew = vi.fn()
    const user = userEvent.setup()
    render(<ExistingShipmentConfirmModal token="tok" shipment={shipment()} onReprint={vi.fn()} onCancel={vi.fn()} onCreateNew={onCreateNew} />)

    await user.click(screen.getByRole('button', { name: /Wrong carrier\? Delete & create new/ }))

    await waitFor(() => expect(shippingApi.cancelShipment).toHaveBeenCalledWith('tok', 'ship-1'))
    expect(await screen.findByText(/could not delete this shipment/)).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Create new shipment' }))
    expect(onCreateNew).toHaveBeenCalled()
  })

  it('shows an error and stays on the original view when the cancel call itself fails', async () => {
    vi.mocked(shippingApi.cancelShipment).mockRejectedValue(new Error('network error'))
    const user = userEvent.setup()
    render(<ExistingShipmentConfirmModal token="tok" shipment={shipment()} onReprint={vi.fn()} onCancel={vi.fn()} onCreateNew={vi.fn()} />)

    await user.click(screen.getByRole('button', { name: /Wrong carrier\? Delete & create new/ }))

    expect(await screen.findByText('network error')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Download & print again/ })).toBeInTheDocument()
  })
})
