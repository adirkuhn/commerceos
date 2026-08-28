// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { ExistingShipmentConfirmModal } from '../ExistingShipmentConfirmModal'
import type { Shipment } from '../types'

function shipment(overrides: Partial<Shipment> = {}): Shipment {
  return {
    id: 'ship-1', sales_order_id: 'so-1', connection_id: 'conn-1', carrier: 1, barcode: '3SMYPA1',
    status: 1, status_main: 'registered', label_format: 'pdf',
    recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
    created_at: '2026-08-23T00:00:00Z',
    ...overrides,
  }
}

describe('ExistingShipmentConfirmModal', () => {
  it('shows the existing shipment barcode and calls onReprint when confirmed', async () => {
    const onReprint = vi.fn()
    const onCancel = vi.fn()
    const user = userEvent.setup()
    render(<ExistingShipmentConfirmModal shipment={shipment({ barcode: '3SMYPA9' })} onReprint={onReprint} onCancel={onCancel} />)

    expect(screen.getByText(/3SMYPA9/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /Download & print again/ }))
    expect(onReprint).toHaveBeenCalled()
    expect(onCancel).not.toHaveBeenCalled()
  })

  it('calls onCancel without calling onReprint', async () => {
    const onReprint = vi.fn()
    const onCancel = vi.fn()
    const user = userEvent.setup()
    render(<ExistingShipmentConfirmModal shipment={shipment()} onReprint={onReprint} onCancel={onCancel} />)

    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(onCancel).toHaveBeenCalled()
    expect(onReprint).not.toHaveBeenCalled()
  })
})
