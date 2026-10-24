// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { MemoryRouter } from 'react-router-dom'

vi.mock('../../search/api', () => ({ globalSearch: vi.fn(async () => ({
  products: [], suppliers: [], sales_orders: [], customers: [{ id: 'c-2', first_name: 'Bruno', last_name: 'Lima', email: 'b@x.nl' }],
})) }))
vi.mock('../../customers/api', () => ({ getCustomer: vi.fn(async () => ({ id: 'c-2', email: 'b@x.nl', billing: { first_name: 'Bruno', city: 'Delft' } })) }))
import { getCustomer } from '../../customers/api'
import { OrderCustomerSection } from '../OrderCustomerSection'
import type { SalesOrder } from '../types'

const order = (over: Partial<SalesOrder> = {}) => ({
  id: 'so-1', customer_id: 'c-1', quote_id: 'q-1', billing: { first_name: 'Ana', last_name: 'Souza' }, ...over,
}) as SalesOrder

beforeEach(() => vi.clearAllMocks())

function renderSection(o: SalesOrder) {
  const onUpdate = vi.fn(async () => {})
  render(<MemoryRouter><OrderCustomerSection token="tok" order={o} onUpdate={onUpdate} /></MemoryRouter>)
  return { onUpdate }
}

async function pickBruno() {
  fireEvent.click(screen.getByRole('button', { name: 'Edit customer' }))
  fireEvent.change(screen.getByRole('textbox', { name: 'Customer' }), { target: { value: 'bru' } })
  fireEvent.click(await screen.findByRole('button', { name: /Bruno Lima/ }))
}

describe('OrderCustomerSection', () => {
  it('links the customer and the quote the order came from, with no remove button', () => {
    renderSection(order())
    expect(screen.getByRole('link', { name: 'Ana Souza' })).toHaveAttribute('href', '/customers/c-1')
    expect(screen.getByRole('link', { name: /from quote/i })).toHaveAttribute('href', '/quotes/q-1')
    expect(screen.queryByRole('button', { name: /remove customer/i })).not.toBeInTheDocument()
  })

  it('changes nothing until Save, then saves the customer with its addresses', async () => {
    const { onUpdate } = renderSection(order())
    await pickBruno()

    expect(screen.getByText('Bruno Lima')).toBeInTheDocument()
    expect(onUpdate).not.toHaveBeenCalled()
    expect(getCustomer).not.toHaveBeenCalled()

    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(onUpdate).toHaveBeenCalledWith({ customer_id: 'c-2', billing: { first_name: 'Bruno', city: 'Delft' } }))
    await waitFor(() => expect(screen.queryByRole('button', { name: 'Save' })).not.toBeInTheDocument())
  })

  it('Cancel drops the pick without saving', async () => {
    const { onUpdate } = renderSection(order())
    await pickBruno()

    fireEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(onUpdate).not.toHaveBeenCalled()
    expect(screen.getByRole('link', { name: 'Ana Souza' })).toBeInTheDocument()
  })

  it('Save is off until another customer is picked', () => {
    renderSection(order())
    fireEvent.click(screen.getByRole('button', { name: 'Edit customer' }))
    expect(screen.getByRole('button', { name: 'Save' })).toBeDisabled()
  })

  it('a failed save keeps the edit open and changes nothing', async () => {
    const { onUpdate } = renderSection(order())
    onUpdate.mockRejectedValueOnce(new Error('boom'))
    await pickBruno()
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('boom')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })
})
