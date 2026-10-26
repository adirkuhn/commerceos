// @vitest-environment jsdom
import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent, waitFor } from '@testing-library/react'
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

  it('a company cannot be saved without its company name', () => {
    render(<CustomerEditPage customer={null} onCreate={vi.fn()} onUpdate={vi.fn()} onClose={vi.fn()} />)
    fireEvent.click(screen.getByRole('radio', { name: 'Company' }))
    fireEvent.change(screen.getByLabelText('Email (optional)'), { target: { value: 'a@x.nl' } })
    expect(screen.getByRole('button', { name: 'Create customer' })).toBeDisabled()
  })
})
