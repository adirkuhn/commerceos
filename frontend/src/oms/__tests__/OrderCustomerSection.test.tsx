// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

vi.mock('../../search/api', () => ({ globalSearch: vi.fn(async () => ({
  products: [], suppliers: [], sales_orders: [], customers: [{ id: 'c-2', first_name: 'Bruno', last_name: 'Lima', email: 'b@x.nl' }],
})) }))
vi.mock('../../customers/api', () => ({ getCustomer: vi.fn(async () => ({ id: 'c-2', email: 'b@x.nl', billing: { first_name: 'Bruno', city: 'Delft' } })) }))
import { OrderCustomerSection } from '../OrderCustomerSection'
import type { SalesOrder } from '../types'

const order = (over: Partial<SalesOrder> = {}) => ({
  id: 'so-1', customer_id: null, quote_id: 'q-1', billing: { first_name: 'Ana', last_name: 'Souza' }, ...over,
}) as SalesOrder

beforeEach(() => vi.clearAllMocks())

function renderSection(o: SalesOrder) {
  const onUpdate = vi.fn(async () => {})
  render(<MemoryRouter><OrderCustomerSection token="tok" order={o} onUpdate={onUpdate} /></MemoryRouter>)
  return { onUpdate }
}

describe('OrderCustomerSection', () => {
  it('links the quote the order came from', () => {
    renderSection(order())
    expect(screen.getByRole('link', { name: /from quote/i })).toHaveAttribute('href', '/quotes/q-1')
  })

  it('picks a customer and copies its addresses onto the order', async () => {
    const { onUpdate } = renderSection(order())
    fireEvent.click(screen.getByRole('button', { name: 'Choose customer' }))
    fireEvent.change(screen.getByRole('textbox', { name: 'Customer' }), { target: { value: 'bru' } })
    fireEvent.click(await screen.findByRole('button', { name: /Bruno Lima/ }))

    await waitFor(() => expect(onUpdate).toHaveBeenCalledWith({ customer_id: 'c-2', billing: { first_name: 'Bruno', city: 'Delft' } }))
  })

  it('removes the customer', async () => {
    const { onUpdate } = renderSection(order({ customer_id: 'c-1' }))
    expect(screen.getByRole('link', { name: 'Ana Souza' })).toHaveAttribute('href', '/customers/c-1')

    fireEvent.click(screen.getByRole('button', { name: 'Remove customer' }))

    await waitFor(() => expect(onUpdate).toHaveBeenCalledWith({ customer_id: '' }))
  })

  it('shows a failed change', async () => {
    const { onUpdate } = renderSection(order({ customer_id: 'c-1' }))
    onUpdate.mockRejectedValueOnce(new Error('boom'))
    fireEvent.click(screen.getByRole('button', { name: 'Remove customer' }))
    expect(await screen.findByText('boom')).toBeInTheDocument()
  })
})
