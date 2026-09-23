// @vitest-environment jsdom
import { screen, waitFor } from '@testing-library/react'
import { act } from 'react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Warehouses from '../Warehouses'
import { renderAt } from '../../test/renderAt'
import type { Warehouse } from '../../inventory/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
  createWarehouse: vi.fn(),
  updateWarehouse: vi.fn(),
  deleteWarehouse: vi.fn(),
}))

import * as inventoryApi from '../../inventory/api'
import { acceptConfirm, dismissConfirm } from '../../test/confirm'

function warehouse(overrides: Partial<Warehouse> = {}): Warehouse {
  return {
    id: 'wh-1', tenant_id: 'tenant-1', name: 'Main DC',
    address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
    created_at: '', updated_at: '',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
})

describe('Warehouses list', () => {
  it('shows an empty state when there are no warehouses', async () => {
    renderAt(<Warehouses />, '/warehouses')

    expect(await screen.findByText('No warehouses yet')).toBeInTheDocument()
  })

  it('lists warehouses', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])

    renderAt(<Warehouses />, '/warehouses')

    expect(await screen.findByText('Main DC')).toBeInTheDocument()
  })

  it('filters by search', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse(), warehouse({ id: 'wh-2', name: 'East Coast' })])

    renderAt(<Warehouses />, '/warehouses')
    await screen.findByText('Main DC')

    await userEvent.type(screen.getByPlaceholderText('Search by name...'), 'East')

    expect(screen.queryByText('Main DC')).not.toBeInTheDocument()
    expect(screen.getByText('East Coast')).toBeInTheDocument()
  })

  it('deleting a row does not also navigate to the edit page', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])

    renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByTitle('Delete'))

    await acceptConfirm()
    await waitFor(() => expect(inventoryApi.deleteWarehouse).toHaveBeenCalledWith('test-token', 'wh-1'))
    expect(screen.queryByText('New Warehouse')).toBeInTheDocument()
  })

  it('does not delete when the confirm dialog is dismissed', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])

    renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByTitle('Delete'))

    await dismissConfirm()
    expect(inventoryApi.deleteWarehouse).not.toHaveBeenCalled()
    expect(screen.getByText('Main DC')).toBeInTheDocument()
  })

  it('shows city/country and an inactive badge in the row', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse({ city: 'Springfield', country: 'US', active: false })])

    renderAt(<Warehouses />, '/warehouses')

    expect(await screen.findByText('Springfield, US')).toBeInTheDocument()
    expect(screen.getByText('inactive')).toBeInTheDocument()
  })

  it('paginates when there are more than 20 warehouses', async () => {
    const many = Array.from({ length: 25 }, (_, i) => warehouse({ id: `wh-${i}`, name: `Warehouse ${String(i).padStart(2, '0')}` }))
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue(many)

    renderAt(<Warehouses />, '/warehouses')
    await screen.findByText('Warehouse 00')

    expect(screen.queryByText('Warehouse 20')).not.toBeInTheDocument()
    expect(screen.getByText('Page 1 of 2')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Next page' }))

    expect(await screen.findByText('Warehouse 20')).toBeInTheDocument()
    expect(screen.queryByText('Warehouse 00')).not.toBeInTheDocument()
  })
})

describe('Warehouse edit page', () => {
  it('opens the edit page when clicking anywhere on a row, not just the Edit button', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])

    renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByText('Main DC'))

    expect(await screen.findByDisplayValue('Main DC')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save' })).toBeInTheDocument()
  })

  it('opening New Warehouse shows a blank create form', async () => {
    renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByRole('button', { name: 'New Warehouse' }))

    expect(await screen.findByText('New Warehouse', { selector: 'h1' })).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toHaveValue('')
    expect(screen.getByRole('button', { name: 'Create Warehouse' })).toBeInTheDocument()
  })

  it('creates a new warehouse and returns to the list', async () => {
    vi.mocked(inventoryApi.createWarehouse).mockResolvedValue(warehouse({ id: 'wh-2', name: 'East Coast' }))

    renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByRole('button', { name: 'New Warehouse' }))
    await userEvent.type(await screen.findByLabelText('Name'), 'East Coast')
    await userEvent.click(screen.getByRole('button', { name: 'Create Warehouse' }))

    await waitFor(() => expect(inventoryApi.createWarehouse).toHaveBeenCalledWith('test-token', expect.objectContaining({ name: 'East Coast' })))
    expect(await screen.findByText('East Coast')).toBeInTheDocument()
  })

  it('submits all address fields and the active toggle', async () => {
    vi.mocked(inventoryApi.createWarehouse).mockResolvedValue(warehouse({ id: 'wh-2', name: 'East Coast' }))

    renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByRole('button', { name: 'New Warehouse' }))
    await userEvent.type(await screen.findByLabelText('Name'), 'East Coast')
    await userEvent.type(screen.getByLabelText('Address line 1'), '1 Main St')
    await userEvent.type(screen.getByLabelText('Address line 2'), 'Suite 100')
    await userEvent.type(screen.getByLabelText('City'), 'Springfield')
    await userEvent.type(screen.getByLabelText('Province / State'), 'IL')
    await userEvent.type(screen.getByLabelText('ZIP / Postal code'), '62701')
    await userEvent.type(screen.getByLabelText('Country'), 'US')
    await userEvent.type(screen.getByLabelText('Phone'), '+1 555-0100')
    await userEvent.click(screen.getByRole('switch', { name: 'Active' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Create Warehouse' }))

    await waitFor(() => expect(inventoryApi.createWarehouse).toHaveBeenCalledWith('test-token', {
      name: 'East Coast', address1: '1 Main St', address2: 'Suite 100', city: 'Springfield',
      province: 'IL', zip: '62701', country: 'US', phone: '+1 555-0100', active: false,
    }))
  })

  it('shows an error if creating fails and stays on the form', async () => {
    vi.mocked(inventoryApi.createWarehouse).mockRejectedValue(new Error('name already taken'))

    renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByRole('button', { name: 'New Warehouse' }))
    await userEvent.type(await screen.findByLabelText('Name'), 'East Coast')
    await userEvent.click(screen.getByRole('button', { name: 'Create Warehouse' }))

    expect(await screen.findByText('name already taken')).toBeInTheDocument()
    expect(screen.getByLabelText('Name')).toBeInTheDocument()
  })

  it('renames a warehouse and returns to the list', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    vi.mocked(inventoryApi.updateWarehouse).mockResolvedValue(warehouse({ name: 'Main Distribution Center' }))

    const { router } = renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByText('Main DC'))

    const input = await screen.findByLabelText('Name')
    await userEvent.clear(input)
    await userEvent.type(input, 'Main Distribution Center')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(inventoryApi.updateWarehouse).toHaveBeenCalledWith(
      'test-token', 'wh-1', expect.objectContaining({ name: 'Main Distribution Center' }),
    ))
    // The editor's heading also shows the new name -- wait until we're
    // actually back on the list URL before asserting on it.
    // (router state flips before React commits, so wait on the list's own UI)
    await screen.findByRole('button', { name: /New Warehouse/ })
    expect(router.state.location.pathname).toBe('/warehouses')
    expect(await screen.findByText('Main Distribution Center')).toBeInTheDocument()
  })

  it('prefills address fields and active state when editing an existing warehouse', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse({
      address1: '1 Main St', city: 'Springfield', province: 'IL', zip: '62701', country: 'US', phone: '+1 555-0100', active: false,
    })])

    renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByText('Main DC'))

    expect(await screen.findByLabelText('Address line 1')).toHaveValue('1 Main St')
    expect(screen.getByLabelText('City')).toHaveValue('Springfield')
    expect(screen.getByLabelText('Province / State')).toHaveValue('IL')
    expect(screen.getByLabelText('ZIP / Postal code')).toHaveValue('62701')
    expect(screen.getByLabelText('Country')).toHaveValue('US')
    expect(screen.getByLabelText('Phone')).toHaveValue('+1 555-0100')
    expect(screen.getByRole('switch', { name: 'Active' })).not.toBeChecked()
  })

  it('Discard returns to the list without saving', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])

    renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByText('Main DC'))
    await userEvent.type(await screen.findByLabelText('Name'), ' extra')
    await userEvent.click(screen.getByRole('button', { name: 'Discard' }))

    expect(inventoryApi.updateWarehouse).not.toHaveBeenCalled()
    expect(await screen.findByText('Main DC')).toBeInTheDocument()
  })
})

describe('Warehouses URLs (refresh + Back)', () => {
  it('New Warehouse pushes /warehouses/new and Back returns to the list', async () => {
    const { router } = renderAt(<Warehouses />, '/warehouses')
    await userEvent.click(await screen.findByRole('button', { name: /New Warehouse/ }))
    expect(router.state.location.pathname).toBe('/warehouses/new')

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/warehouses')
    expect(await screen.findByPlaceholderText('Search by name...')).toBeInTheDocument()
  })

  it('/warehouses/:id opens that warehouse for editing on load', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    renderAt(<Warehouses />, '/warehouses', '/warehouses/wh-1')
    expect(await screen.findByDisplayValue('Main DC')).toBeInTheDocument()
  })

  it('/warehouses/:id for an unknown id shows not-found', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
    renderAt(<Warehouses />, '/warehouses', '/warehouses/nope')
    expect(await screen.findByText('Warehouse not found')).toBeInTheDocument()
  })
})
