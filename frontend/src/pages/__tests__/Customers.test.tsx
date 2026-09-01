// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Customers from '../Customers'
import { MemoryRouter } from 'react-router-dom'
import type { Customer } from '../../customers/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../customers/api', () => ({
  listCustomers: vi.fn(),
  createCustomer: vi.fn(),
  updateCustomer: vi.fn(),
  deleteCustomer: vi.fn(),
}))

import * as customersApi from '../../customers/api'

function customer(overrides: Partial<Customer> = {}): Customer {
  return {
    id: 'customer-1', tenant_id: 'tenant-1', email: 'jane@example.com',
    first_name: 'Jane', last_name: 'Doe', is_paying_customer: false,
    date_created: '', date_modified: '',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(customersApi.listCustomers).mockResolvedValue([])
})

describe('Customers list', () => {
  it('shows an empty state when there are no customers', async () => {
    render(<MemoryRouter><Customers /></MemoryRouter>)

    expect(await screen.findByText('No customers yet')).toBeInTheDocument()
  })

  it('lists customers', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])

    render(<MemoryRouter><Customers /></MemoryRouter>)

    expect(await screen.findByText('Jane Doe')).toBeInTheDocument()
    expect(screen.getByText('jane@example.com')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([
      customer(),
      customer({ id: 'customer-2', first_name: 'Bob', last_name: 'Smith', email: 'bob@example.com' }),
    ])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await screen.findByText('Jane Doe')

    await userEvent.type(screen.getByPlaceholderText('Search by name or email...'), 'Bob')

    expect(screen.queryByText('Jane Doe')).not.toBeInTheDocument()
    expect(screen.getByText('Bob Smith')).toBeInTheDocument()
  })

  it('deleting a row does not also navigate to the edit page', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Delete'))

    await waitFor(() => expect(customersApi.deleteCustomer).toHaveBeenCalledWith('test-token', 'customer-1'))
    expect(screen.queryByText('Add Customer')).toBeInTheDocument()
  })

  it('does not delete when the confirm dialog is dismissed', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.stubGlobal('confirm', vi.fn(() => false))

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Delete'))

    expect(customersApi.deleteCustomer).not.toHaveBeenCalled()
    expect(screen.getByText('Jane Doe')).toBeInTheDocument()
  })

  it('shows paying customer status in the row', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer({ is_paying_customer: true })])

    render(<MemoryRouter><Customers /></MemoryRouter>)

    expect(await screen.findByText('Paying')).toBeInTheDocument()
  })

  it('paginates when there are more than 20 customers', async () => {
    const many = Array.from({ length: 25 }, (_, i) =>
      customer({ id: `customer-${i}`, first_name: `Customer${String(i).padStart(2, '0')}`, email: `c${i}@example.com` }))
    vi.mocked(customersApi.listCustomers).mockResolvedValue(many)

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await screen.findByText('Customer00 Doe')

    expect(screen.queryByText('Customer20 Doe')).not.toBeInTheDocument()
    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))

    expect(await screen.findByText('Customer20 Doe')).toBeInTheDocument()
    expect(screen.queryByText('Customer00 Doe')).not.toBeInTheDocument()
  })
})

describe('Customer edit page', () => {
  it('opens the edit page when clicking anywhere on a row, not just the Edit button', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    expect(screen.getByLabelText('Email')).toHaveValue('jane@example.com')
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })

  it('opening Add Customer shows a blank create form', async () => {
    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Customer' }))

    expect(screen.getByText('New Customer', { selector: 'h1' })).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Create Customer' })).toBeInTheDocument()
  })

  it('creates a new customer and returns to the list', async () => {
    vi.mocked(customersApi.createCustomer).mockResolvedValue(
      customer({ id: 'customer-2', first_name: 'Bob', last_name: 'Smith', email: 'bob@example.com' }),
    )

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Customer' }))
    await userEvent.type(screen.getByLabelText('First name'), 'Bob')
    await userEvent.type(screen.getByLabelText('Last name'), 'Smith')
    await userEvent.type(screen.getByLabelText('Email'), 'bob@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Create Customer' }))

    await waitFor(() => expect(customersApi.createCustomer).toHaveBeenCalledWith('test-token', expect.objectContaining({
      email: 'bob@example.com', first_name: 'Bob', last_name: 'Smith',
    })))
    expect(await screen.findByText('Bob Smith')).toBeInTheDocument()
  })

  it('submits billing and shipping address fields', async () => {
    vi.mocked(customersApi.createCustomer).mockResolvedValue(
      customer({ id: 'customer-2', first_name: 'Bob', last_name: 'Smith', email: 'bob@example.com' }),
    )

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Customer' }))
    await userEvent.type(screen.getByLabelText('First name'), 'Bob')
    await userEvent.type(screen.getByLabelText('Last name'), 'Smith')
    await userEvent.type(screen.getByLabelText('Email'), 'bob@example.com')
    await userEvent.type(screen.getByLabelText('Billing city'), 'Rotterdam')
    await userEvent.click(screen.getByRole('button', { name: 'Create Customer' }))

    await waitFor(() => expect(customersApi.createCustomer).toHaveBeenCalledWith('test-token', expect.objectContaining({
      billing: expect.objectContaining({ city: 'Rotterdam' }),
    })))
  })

  it('shows an error if creating fails and stays on the form', async () => {
    vi.mocked(customersApi.createCustomer).mockRejectedValue(new Error('email already taken'))

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Customer' }))
    await userEvent.type(screen.getByLabelText('Email'), 'bob@example.com')
    await userEvent.click(screen.getByRole('button', { name: 'Create Customer' }))

    expect(await screen.findByText('email already taken')).toBeInTheDocument()
    expect(screen.getByLabelText('Email')).toBeInTheDocument()
  })

  it('renames a customer and returns to the list', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])
    vi.mocked(customersApi.updateCustomer).mockResolvedValue(customer({ last_name: 'Doelist' }))

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    const input = screen.getByLabelText('Last name')
    await userEvent.clear(input)
    await userEvent.type(input, 'Doelist')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(customersApi.updateCustomer).toHaveBeenCalledWith(
      'test-token', 'customer-1', expect.objectContaining({ last_name: 'Doelist' }),
    ))
    expect(await screen.findByText('Jane Doelist')).toBeInTheDocument()
  })

  it('prefills fields when editing an existing customer', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer({
      billing: { city: 'Rotterdam', country: 'NL', phone: '+31 6 1234 5678' },
    })])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))

    expect(screen.getByLabelText('Email')).toHaveValue('jane@example.com')
    expect(screen.getByLabelText('Billing city')).toHaveValue('Rotterdam')
    expect(screen.getByLabelText('Billing country')).toHaveValue('NL')
    expect(screen.getByLabelText('Billing phone')).toHaveValue('+31 6 1234 5678')
  })

  it('Discard returns to the list without saving', async () => {
    vi.mocked(customersApi.listCustomers).mockResolvedValue([customer()])

    render(<MemoryRouter><Customers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Jane Doe'))
    await userEvent.type(screen.getByLabelText('Last name'), ' extra')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(customersApi.updateCustomer).not.toHaveBeenCalled()
    expect(await screen.findByText('Jane Doe')).toBeInTheDocument()
  })
})
