// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Suppliers from '../Suppliers'
import { MemoryRouter } from 'react-router-dom'
import type { Supplier } from '../../suppliers/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../suppliers/api', () => ({
  listSuppliers: vi.fn(),
  createSupplier: vi.fn(),
  updateSupplier: vi.fn(),
  deleteSupplier: vi.fn(),
}))

import * as suppliersApi from '../../suppliers/api'

function supplier(overrides: Partial<Supplier> = {}): Supplier {
  return {
    id: 'supplier-1', tenant_id: 'tenant-1', name: 'Acme Foods',
    email: '', phone: '', country: '', lead_time_days: null, status: 'active', notes: '',
    created_at: '', updated_at: '',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([])
})

describe('Suppliers list', () => {
  it('shows an empty state when there are no suppliers', async () => {
    render(<MemoryRouter><Suppliers /></MemoryRouter>)

    expect(await screen.findByText('No suppliers yet')).toBeInTheDocument()
  })

  it('lists suppliers', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])

    render(<MemoryRouter><Suppliers /></MemoryRouter>)

    expect(await screen.findByText('Acme Foods')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier(), supplier({ id: 'supplier-2', name: 'Beta Supply' })])

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await screen.findByText('Acme Foods')

    await userEvent.type(screen.getByPlaceholderText('Search by name or country...'), 'Beta')

    expect(screen.queryByText('Acme Foods')).not.toBeInTheDocument()
    expect(screen.getByText('Beta Supply')).toBeInTheDocument()
  })

  it('deleting a row does not also navigate to the edit page', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])
    vi.stubGlobal('confirm', vi.fn(() => true))

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Delete'))

    await waitFor(() => expect(suppliersApi.deleteSupplier).toHaveBeenCalledWith('test-token', 'supplier-1'))
    expect(screen.queryByText('Add Supplier')).toBeInTheDocument()
  })

  it('does not delete when the confirm dialog is dismissed', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])
    vi.stubGlobal('confirm', vi.fn(() => false))

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await userEvent.click(await screen.findByTitle('Delete'))

    expect(suppliersApi.deleteSupplier).not.toHaveBeenCalled()
    expect(screen.getByText('Acme Foods')).toBeInTheDocument()
  })

  it('shows country, lead time, and status in the row', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier({ country: 'US', lead_time_days: 14, status: 'inactive' })])

    render(<MemoryRouter><Suppliers /></MemoryRouter>)

    expect(await screen.findByText('US')).toBeInTheDocument()
    expect(screen.getByText('14d')).toBeInTheDocument()
    expect(screen.getByText('inactive')).toBeInTheDocument()
  })

  it('paginates when there are more than 20 suppliers', async () => {
    const many = Array.from({ length: 25 }, (_, i) => supplier({ id: `supplier-${i}`, name: `Supplier ${String(i).padStart(2, '0')}` }))
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue(many)

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await screen.findByText('Supplier 00')

    expect(screen.queryByText('Supplier 20')).not.toBeInTheDocument()
    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))

    expect(await screen.findByText('Supplier 20')).toBeInTheDocument()
    expect(screen.queryByText('Supplier 00')).not.toBeInTheDocument()
  })
})

describe('Supplier edit page', () => {
  it('opens the edit page when clicking anywhere on a row, not just the Edit button', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(screen.getByDisplayValue('Acme Foods')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })

  it('opening Add Supplier shows a blank create form', async () => {
    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Supplier' }))

    expect(screen.getByText('New Supplier', { selector: 'h1' })).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Create Supplier' })).toBeInTheDocument()
  })

  it('creates a new supplier and returns to the list', async () => {
    vi.mocked(suppliersApi.createSupplier).mockResolvedValue(supplier({ id: 'supplier-2', name: 'Beta Supply' }))

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Supplier' }))
    await userEvent.type(screen.getByLabelText('Name'), 'Beta Supply')
    await userEvent.click(screen.getByRole('button', { name: 'Create Supplier' }))

    await waitFor(() => expect(suppliersApi.createSupplier).toHaveBeenCalledWith('test-token', expect.objectContaining({ name: 'Beta Supply' })))
    expect(await screen.findByText('Beta Supply')).toBeInTheDocument()
  })

  it('submits all fields including lead time and status', async () => {
    vi.mocked(suppliersApi.createSupplier).mockResolvedValue(supplier({ id: 'supplier-2', name: 'Beta Supply' }))

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Supplier' }))
    await userEvent.type(screen.getByLabelText('Name'), 'Beta Supply')
    await userEvent.type(screen.getByLabelText('Email'), 'hi@beta.example')
    await userEvent.type(screen.getByLabelText('Phone'), '+1 555-0100')
    await userEvent.type(screen.getByLabelText('Country'), 'US')
    await userEvent.type(screen.getByLabelText('Lead time (days)'), '14')
    await userEvent.selectOptions(screen.getByLabelText('Status'), 'inactive')
    await userEvent.type(screen.getByLabelText('Notes'), 'net-30')
    await userEvent.click(screen.getByRole('button', { name: 'Create Supplier' }))

    await waitFor(() => expect(suppliersApi.createSupplier).toHaveBeenCalledWith('test-token', {
      name: 'Beta Supply', email: 'hi@beta.example', phone: '+1 555-0100', country: 'US',
      lead_time_days: 14, status: 'inactive', notes: 'net-30',
    }))
  })

  it('shows an error if creating fails and stays on the form', async () => {
    vi.mocked(suppliersApi.createSupplier).mockRejectedValue(new Error('name already taken'))

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Add Supplier' }))
    await userEvent.type(screen.getByLabelText('Name'), 'Beta Supply')
    await userEvent.click(screen.getByRole('button', { name: 'Create Supplier' }))

    expect(await screen.findByText('name already taken')).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toBeInTheDocument()
  })

  it('renames a supplier and returns to the list', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])
    vi.mocked(suppliersApi.updateSupplier).mockResolvedValue(supplier({ name: 'Acme Foods Ltd' }))

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Acme Foods'))

    const input = screen.getByLabelText('Name')
    await userEvent.clear(input)
    await userEvent.type(input, 'Acme Foods Ltd')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(suppliersApi.updateSupplier).toHaveBeenCalledWith(
      'test-token', 'supplier-1', expect.objectContaining({ name: 'Acme Foods Ltd' }),
    ))
    expect(await screen.findByText('Acme Foods Ltd')).toBeInTheDocument()
  })

  it('prefills fields when editing an existing supplier', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier({
      email: 'hi@acme.example', phone: '+1 555-0100', country: 'US', lead_time_days: 14, status: 'inactive', notes: 'net-30',
    })])

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Acme Foods'))

    expect(screen.getByLabelText('Email')).toHaveValue('hi@acme.example')
    expect(screen.getByLabelText('Phone')).toHaveValue('+1 555-0100')
    expect(screen.getByLabelText('Country')).toHaveValue('US')
    expect(screen.getByLabelText('Lead time (days)')).toHaveValue(14)
    expect(screen.getByLabelText('Status')).toHaveValue('inactive')
    expect(screen.getByLabelText('Notes')).toHaveValue('net-30')
  })

  it('Discard returns to the list without saving', async () => {
    vi.mocked(suppliersApi.listSuppliers).mockResolvedValue([supplier()])

    render(<MemoryRouter><Suppliers /></MemoryRouter>)
    await userEvent.click(await screen.findByText('Acme Foods'))
    await userEvent.type(screen.getByLabelText('Name'), ' extra')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(suppliersApi.updateSupplier).not.toHaveBeenCalled()
    expect(await screen.findByText('Acme Foods')).toBeInTheDocument()
  })
})
