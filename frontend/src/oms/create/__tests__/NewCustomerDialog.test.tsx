// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('../../../customers/api', () => ({ createCustomer: vi.fn() }))
import { createCustomer } from '../../../customers/api'
import { NewCustomerDialog } from '../NewCustomerDialog'

beforeEach(() => vi.clearAllMocks())

describe('NewCustomerDialog', () => {
  it('creates a customer without an email, its address on billing and shipping', async () => {
    vi.mocked(createCustomer).mockResolvedValue({ id: 'c-9' } as never)
    const onCreated = vi.fn()
    render(<NewCustomerDialog token="tok" onClose={vi.fn()} onCreated={onCreated} />)

    fireEvent.change(screen.getByLabelText('First name'), { target: { value: 'Bruno' } })
    fireEvent.change(screen.getByLabelText('Company'), { target: { value: 'Padaria Lima' } })
    fireEvent.change(screen.getByLabelText('Street and number'), { target: { value: 'Dorpsstraat 5' } })
    fireEvent.change(screen.getByLabelText('Postcode'), { target: { value: '1234 AB' } })
    fireEvent.change(screen.getByLabelText('City'), { target: { value: 'Delft' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create customer' }))

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith({ id: 'c-9' }))
    const address = { first_name: 'Bruno', last_name: '', company: 'Padaria Lima', address_1: 'Dorpsstraat 5', postcode: '1234 AB', city: 'Delft', country: 'NL', email: '', phone: '' }
    expect(createCustomer).toHaveBeenCalledWith('tok', { first_name: 'Bruno', last_name: '', email: '', billing: address, shipping: address })
  })

  it('needs a name, company, email or phone before it can create', () => {
    render(<NewCustomerDialog token="tok" onClose={vi.fn()} onCreated={vi.fn()} />)
    expect(screen.getByRole('button', { name: 'Create customer' })).toBeDisabled()
    fireEvent.change(screen.getByLabelText('Phone'), { target: { value: '06 1234 5678' } })
    expect(screen.getByRole('button', { name: 'Create customer' })).toBeEnabled()
  })

  it('shows a failed create', async () => {
    vi.mocked(createCustomer).mockRejectedValue(new Error('boom'))
    render(<NewCustomerDialog token="tok" onClose={vi.fn()} onCreated={vi.fn()} />)
    fireEvent.change(screen.getByLabelText('First name'), { target: { value: 'Bruno' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create customer' }))
    expect(await screen.findByText('boom')).toBeInTheDocument()
  })
})
