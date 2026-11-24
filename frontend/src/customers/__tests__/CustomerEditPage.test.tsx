// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({
  listShippingAddresses: vi.fn().mockResolvedValue([{
    id: 'sa-1', customer_id: 'c-1', label: 'Warehouse', first_name: '', last_name: '', company: '', address_1: 'Dock 2', address_2: '',
    city: 'Hoofddorp', state: '', postcode: '', country: 'NL', phone: '', is_default: true,
  }]),
  addShippingAddress: vi.fn(), updateShippingAddress: vi.fn(), removeShippingAddress: vi.fn(), setDefaultShippingAddress: vi.fn(),
}))
import { CustomerEditPage } from '../CustomerEditPage'
import type { Customer } from '../types'

const company: Customer = {
  id: 'c-1', tenant_id: 't', email: 'info@claus.nl', first_name: '', last_name: '', is_paying_customer: false, date_created: '', date_modified: '',
  customer_type: 'company', company_name: 'Claus B.V.', display_name: 'Claus B.V.', client_number: '3286', kvk_number: '34073293', vat_number: '',
  billing: { company: 'Claus B.V.', city: 'Hoofddorp' },
}

describe('CustomerEditPage', () => {
  it('creates a private customer by default, email optional', async () => {
    const onCreate = vi.fn().mockResolvedValue(company)
    render(<CustomerEditPage customer={null} onCreate={onCreate} onUpdate={vi.fn()} onClose={vi.fn()} />)

    expect(screen.getByRole('radio', { name: 'Private' })).toHaveAttribute('aria-checked', 'true')
    fireEvent.change(screen.getByLabelText('First name'), { target: { value: 'Ana' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create customer' }))

    await waitFor(() => expect(onCreate).toHaveBeenCalled())
    expect(onCreate.mock.calls[0][0]).toMatchObject({ customer_type: 'private', first_name: 'Ana', email: '' })
  })

  it('creates a company with its contact person and puts the company on the billing address', async () => {
    const onCreate = vi.fn().mockResolvedValue(company)
    render(<CustomerEditPage customer={null} onCreate={onCreate} onUpdate={vi.fn()} onClose={vi.fn()} />)

    fireEvent.click(screen.getByRole('radio', { name: 'Company' }))
    fireEvent.change(screen.getByLabelText('Company name'), { target: { value: 'Claus B.V.' } })
    fireEvent.change(screen.getByLabelText('Contact first name'), { target: { value: 'Kylian' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create customer' }))

    await waitFor(() => expect(onCreate).toHaveBeenCalled())
    expect(onCreate.mock.calls[0][0]).toMatchObject({
      customer_type: 'company', company_name: 'Claus B.V.', contact_persons: [{ first_name: 'Kylian', last_name: '' }], billing: { company: 'Claus B.V.' },
    })
  })

  it('edits a company: its fields filled in, no contact person block, client number kept', async () => {
    const onUpdate = vi.fn().mockResolvedValue(company)
    render(<CustomerEditPage customer={company} onCreate={vi.fn()} onUpdate={onUpdate} onClose={vi.fn()} />)

    expect(screen.getByRole('heading', { name: 'Claus B.V.' })).toBeInTheDocument()
    expect(screen.getByLabelText('KVK number')).toHaveValue('34073293')
    expect(screen.getByLabelText('Client number')).toHaveValue('3286')
    expect(screen.queryByRole('group', { name: 'Contact person' })).not.toBeInTheDocument()
    fireEvent.change(screen.getByLabelText('VAT ID'), { target: { value: 'NL1B01' } })
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(onUpdate).toHaveBeenCalled())
    expect(onUpdate.mock.calls[0]).toMatchObject(['c-1', { customer_type: 'company', vat_number: 'NL1B01', client_number: '3286' }])
    expect(onUpdate.mock.calls[0][1].contact_persons).toBeUndefined()
  })

  it('has no paying-customer switch: it comes from the orders', async () => {
    const onUpdate = vi.fn().mockResolvedValue(company)
    render(<CustomerEditPage customer={{ ...company, is_paying_customer: true }} onCreate={vi.fn()} onUpdate={onUpdate} onClose={vi.fn()} />)

    expect(screen.queryByRole('checkbox', { name: 'Paying customer' })).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(onUpdate).toHaveBeenCalled())
    expect(onUpdate.mock.calls[0][1]).not.toHaveProperty('is_paying_customer')
  })

  it('a company cannot be saved without its company name', () => {
    render(<CustomerEditPage customer={null} onCreate={vi.fn()} onUpdate={vi.fn()} onClose={vi.fn()} />)
    fireEvent.click(screen.getByRole('radio', { name: 'Company' }))
    fireEvent.change(screen.getByLabelText('Email (optional)'), { target: { value: 'a@x.nl' } })
    expect(screen.getByRole('button', { name: 'Create customer' })).toBeDisabled()
  })

  it('a new customer ships to the address typed in, or to billing', async () => {
    const onCreate = vi.fn().mockResolvedValue(company)
    render(<CustomerEditPage customer={null} onCreate={onCreate} onUpdate={vi.fn()} onClose={vi.fn()} />)
    fireEvent.change(screen.getByLabelText('First name'), { target: { value: 'Ana' } })

    const same = screen.getByRole('checkbox', { name: 'Same as billing' })
    expect(same).not.toBeChecked()
    fireEvent.change(screen.getByLabelText('Shipping address line 1'), { target: { value: 'Dock 2' } })
    fireEvent.click(screen.getByRole('button', { name: 'Create customer' }))
    await waitFor(() => expect(onCreate).toHaveBeenCalledTimes(1))
    expect(onCreate.mock.calls[0][0]).toMatchObject({ ship_to_billing: false, shipping: { address_1: 'Dock 2' } })

    fireEvent.click(same)
    expect(screen.queryByLabelText('Shipping address line 1')).not.toBeInTheDocument()
    expect(screen.getByText('Goods go to the billing address.')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Create customer' }))
    await waitFor(() => expect(onCreate).toHaveBeenCalledTimes(2))
    expect(onCreate.mock.calls[1][0]).toMatchObject({ ship_to_billing: true })
    expect(onCreate.mock.calls[1][0]).not.toHaveProperty('shipping')
  })

  it('a saved customer manages their shipping addresses in the book and never sends shipping', async () => {
    const onUpdate = vi.fn().mockResolvedValue(company)
    render(<CustomerEditPage customer={{ ...company, shipping: { address_1: 'Dock 2' } }} onCreate={vi.fn()} onUpdate={onUpdate} onClose={vi.fn()} />)

    expect(await screen.findByText('Warehouse')).toBeInTheDocument()
    expect(screen.queryByLabelText('Shipping address line 1')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(onUpdate).toHaveBeenCalled())
    expect(onUpdate.mock.calls[0][1]).toMatchObject({ ship_to_billing: false })
    expect(onUpdate.mock.calls[0][1]).not.toHaveProperty('shipping')
  })

  it('a saved customer shipping to billing hides the book', async () => {
    const onUpdate = vi.fn().mockResolvedValue(company)
    render(<CustomerEditPage customer={{ ...company, ship_to_billing: true }} onCreate={vi.fn()} onUpdate={onUpdate} onClose={vi.fn()} />)

    expect(screen.getByRole('checkbox', { name: 'Same as billing' })).toBeChecked()
    expect(screen.queryByText('Shipping addresses')).not.toBeInTheDocument()
    fireEvent.click(screen.getByRole('checkbox', { name: 'Same as billing' }))
    expect(await screen.findByText('Warehouse')).toBeInTheDocument()
    fireEvent.click(screen.getByRole('button', { name: 'Save' }))
    await waitFor(() => expect(onUpdate).toHaveBeenCalled())
    expect(onUpdate.mock.calls[0][1]).toMatchObject({ ship_to_billing: false })
  })
})
