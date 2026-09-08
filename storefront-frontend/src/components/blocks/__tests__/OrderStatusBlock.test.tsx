// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import { OrderStatusBlock } from '../OrderStatusBlock'
import type { StorefrontOrderStatus } from '../../../types'

vi.mock('../../../api/orders', () => ({ lookupOrder: vi.fn() }))

import * as ordersApi from '../../../api/orders'

beforeEach(() => vi.clearAllMocks())

const status: StorefrontOrderStatus = { order_number: 'WEB-1', status: 'processing', fulfilled: false, total_minor_units: 1000, currency_code: 'EUR' }

describe('OrderStatusBlock (order-confirmation)', () => {
  it('shows the order number without an email in router state', async () => {
    render(
      <MemoryRouter initialEntries={['/order-confirmation/WEB-1']}>
        <Routes>
          <Route path="/order-confirmation/:orderNumber" element={<OrderStatusBlock pageSlug="order-confirmation" />} />
        </Routes>
      </MemoryRouter>,
    )

    expect(await screen.findByText(/Order number: WEB-1/)).toBeInTheDocument()
    expect(ordersApi.lookupOrder).not.toHaveBeenCalled()
  })

  it('looks up and shows the order status when an email is carried in router state', async () => {
    vi.mocked(ordersApi.lookupOrder).mockResolvedValue(status)

    render(
      <MemoryRouter initialEntries={[{ pathname: '/order-confirmation/WEB-1', state: { email: 'shopper@example.com' } }]}>
        <Routes>
          <Route path="/order-confirmation/:orderNumber" element={<OrderStatusBlock pageSlug="order-confirmation" />} />
        </Routes>
      </MemoryRouter>,
    )

    expect(await screen.findByText(/WEB-1 — processing/)).toBeInTheDocument()
    expect(ordersApi.lookupOrder).toHaveBeenCalledWith('WEB-1', 'shopper@example.com')
  })
})

describe('OrderStatusBlock (order-lookup)', () => {
  it('looks up an order by number and email from the form', async () => {
    const user = userEvent.setup()
    vi.mocked(ordersApi.lookupOrder).mockResolvedValue(status)

    render(<MemoryRouter><OrderStatusBlock pageSlug="order-lookup" /></MemoryRouter>)

    await user.type(screen.getByLabelText('Order number'), 'WEB-1')
    await user.type(screen.getByLabelText('Email'), 'shopper@example.com')
    await user.click(screen.getByRole('button', { name: 'Track order' }))

    expect(await screen.findByText('Status: processing')).toBeInTheDocument()
    expect(ordersApi.lookupOrder).toHaveBeenCalledWith('WEB-1', 'shopper@example.com')
  })

  it('shows an error when the order cannot be found', async () => {
    const user = userEvent.setup()
    vi.mocked(ordersApi.lookupOrder).mockRejectedValue(new Error('order not found'))

    render(<MemoryRouter><OrderStatusBlock pageSlug="order-lookup" /></MemoryRouter>)

    await user.type(screen.getByLabelText('Order number'), 'WEB-2')
    await user.type(screen.getByLabelText('Email'), 'nope@example.com')
    await user.click(screen.getByRole('button', { name: 'Track order' }))

    expect(await screen.findByText('order not found')).toBeInTheDocument()
  })
})
