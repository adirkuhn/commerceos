// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('../../../customers/api', () => ({ createCustomer: vi.fn() }))
import { createCustomer } from '../../../customers/api'
import { NewCustomerDialog } from '../NewCustomerDialog'

beforeEach(() => vi.clearAllMocks())

function fillAddress() {
  fireEvent.change(screen.getByLabelText('Street and number'), { target: { value: 'Bosweg 19' } })
  fireEvent.change(screen.getByLabelText('Postcode'), { target: { value: '2131LX' } })
  fireEvent.change(screen.getByLabelText('City'), { target: { value: 'Hoofddorp' } })
}

describe('NewCustomerDialog', () => {
  it('creates a company with its contact person, no person name or email needed', async () => {
    vi.mocked(createCustomer).mockResolvedValue({ id: 'c-9' } as never)
    const onCreated = vi.fn()
    render(<NewCustomerDialog token="tok" onClose={vi.fn()} onCreated={onCreated} />)

    expect(screen.getByRole('radio', { name: 'Private' })).toHaveAttribute('aria-checked', 'true')
    fireEvent.click(screen.getByRole('radio', { name: 'Company' }))
    fireEvent.change(screen.getByLabelText('Company name'), { target: { value: 'Claus Hoofddorp B.V.' } })
    fireEvent.change(screen.getByLabelText('KVK number'), { target: { value: '34073293' } })
    fireEvent.change(screen.getByLabelText('Contact first name'), { target: { value: 'Kylian' } })
    fireEvent.change(screen.getByLabelText('Contact last name'), { target: { value: 'Klockenbrink' } })
    fillAddress()
    fireEvent.click(screen.getByRole('button', { name: 'Create customer' }))

    await waitFor(() => expect(onCreated).toHaveBeenCalledWith({ id: 'c-9' }))
    const address = { first_name: 'Kylian', last_name: 'Klockenbrink', company: 'Claus Hoofddorp B.V.', address_1: 'Bosweg 19', postcode: '2131LX', city: 'Hoofddorp', country: 'NL', email: '', phone: '' }
    expect(createCustomer).toHaveBeenCalledWith('tok', {
      customer_type: 'company', company_name: 'Claus Hoofddorp B.V.', kvk_number: '34073293', vat_number: '', first_name: '', last_name: '', email: '',
      contact_persons: [{ first_name: 'Kylian', last_name: 'Klockenbrink' }], billing: address, shipping: address,
    })
  })

  it('creates a private customer by name, the default', async () => {
    vi.mocked(createCustomer).mockResolvedValue({ id: 'c-8' } as never)
    render(<NewCustomerDialog token="tok" onClose={vi.fn()} onCreated={vi.fn()} />)

    fireEvent.change(screen.getByLabelText('First name'), { target: { value: 'Bruno' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create customer' }))

    await waitFor(() => expect(createCustomer).toHaveBeenCalled())
    expect(vi.mocked(createCustomer).mock.calls[0][1]).toMatchObject({ customer_type: 'private', first_name: 'Bruno', billing: { first_name: 'Bruno', company: '' } })
  })

  it('a company needs its company name before it can be created', () => {
    render(<NewCustomerDialog token="tok" onClose={vi.fn()} onCreated={vi.fn()} />)
    fireEvent.click(screen.getByRole('radio', { name: 'Company' }))
    const create = screen.getByRole('button', { name: 'Create customer' })
    fireEvent.change(screen.getByLabelText('Phone'), { target: { value: '06 1234 5678' } })
    expect(create).toBeDisabled()
    fireEvent.change(screen.getByLabelText('Company name'), { target: { value: 'Sabor' } })
    expect(create).toBeEnabled()
  })

  it('shows a failed create', async () => {
    vi.mocked(createCustomer).mockRejectedValue(new Error('boom'))
    render(<NewCustomerDialog token="tok" onClose={vi.fn()} onCreated={vi.fn()} />)
    fireEvent.change(screen.getByLabelText('First name'), { target: { value: 'Bruno' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create customer' }))
    expect(await screen.findByText('boom')).toBeInTheDocument()
  })
})
