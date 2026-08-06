// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import SalesChannels from '../SalesChannels'
import type { Connection, SalesChannel, SalesChannelProductEnrollment } from '../../integrations/types'
import type { Category } from '../../pms/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../integrations/api', () => ({
  listConnections: vi.fn(),
  listSalesChannels: vi.fn(),
  createSalesChannel: vi.fn(),
  updateSalesChannel: vi.fn(),
  deleteSalesChannel: vi.fn(),
  listSalesChannelProducts: vi.fn(),
  setSalesChannelProductOverride: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listCategories: vi.fn(),
}))

import * as integrationsApi from '../../integrations/api'
import * as pmsApi from '../../pms/api'

const wooConnection: Connection = {
  id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '',
}

const sampleCategory: Category = {
  id: 'cat-1', tenant_id: 'tenant-1', name: 'Apparel', slug: 'apparel', position: 0, created_at: '', updated_at: '',
}

function sampleChannel(overrides: Partial<SalesChannel> = {}): SalesChannel {
  return {
    id: 'chan-1', name: 'My Woo Store', type: 'woocommerce', connection_id: 'conn-1',
    enrollment_mode: 'all', active: true, created_at: '', updated_at: '', ...overrides,
  }
}

function enrollment(overrides: Partial<SalesChannelProductEnrollment> = {}): SalesChannelProductEnrollment {
  return {
    product: { id: 'p1', name: 'T-Shirt', slug: 't-shirt' },
    enrolled: true,
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection])
  vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel()])
  vi.mocked(integrationsApi.listSalesChannelProducts).mockResolvedValue({ items: [enrollment()], total: 1 })
  vi.mocked(pmsApi.listCategories).mockResolvedValue([sampleCategory])
})

describe('Sales channels list', () => {
  it('shows an empty state when there are no channels', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([])
    render(<SalesChannels />)
    expect(await screen.findByText('No sales channels yet')).toBeInTheDocument()
  })

  it('lists channels with their connection and enrollment mode', async () => {
    render(<SalesChannels />)
    expect(await screen.findByText('My Woo Store')).toBeInTheDocument()
    expect(screen.getByText('Woocommerce — https://mystore.com')).toBeInTheDocument()
    expect(screen.getByText('All products')).toBeInTheDocument()
  })
})

describe('Create sales channel', () => {
  it('creates a channel with the selected connection and enrollment mode', async () => {
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-2', name: 'New Channel' }))
    render(<SalesChannels />)
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.type(screen.getByLabelText('Name'), 'New Channel')
    await userEvent.selectOptions(screen.getByLabelText('Connection'), 'conn-1')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.createSalesChannel).toHaveBeenCalledWith('test-token', {
        name: 'New Channel', type: 'woocommerce', connectionId: 'conn-1', enrollmentMode: 'all',
        enrollmentCategoryId: null, enrollmentTag: null,
      })
    })
  })

  it('shows the tag input only when enrollment mode is tag, and sends it', async () => {
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-3', name: 'Tagged Channel', enrollment_mode: 'tag', enrollment_tag: 'summer' }))
    render(<SalesChannels />)
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    expect(screen.queryByLabelText('Tag')).not.toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Default enrollment'), 'tag')
    await userEvent.type(screen.getByLabelText('Name'), 'Tagged Channel')
    await userEvent.selectOptions(screen.getByLabelText('Connection'), 'conn-1')
    await userEvent.type(screen.getByLabelText('Tag'), 'summer')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.createSalesChannel).toHaveBeenCalledWith('test-token', expect.objectContaining({
        enrollmentMode: 'tag', enrollmentTag: 'summer', enrollmentCategoryId: null,
      }))
    })
  })

  it('shows an error if creating fails', async () => {
    vi.mocked(integrationsApi.createSalesChannel).mockRejectedValue(new Error('connection is required'))
    render(<SalesChannels />)
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.type(screen.getByLabelText('Name'), 'New Channel')
    await userEvent.selectOptions(screen.getByLabelText('Connection'), 'conn-1')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    expect(await screen.findByText('connection is required')).toBeInTheDocument()
  })
})

describe('Edit sales channel', () => {
  it('prefills the form and saves changes', async () => {
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ name: 'Renamed' }))
    render(<SalesChannels />)
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    expect(screen.getByLabelText('Name')).toHaveValue('My Woo Store')

    await userEvent.clear(screen.getByLabelText('Name'))
    await userEvent.type(screen.getByLabelText('Name'), 'Renamed')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => {
      expect(integrationsApi.updateSalesChannel).toHaveBeenCalledWith('test-token', 'chan-1', {
        name: 'Renamed', connectionId: 'conn-1', enrollmentMode: 'all',
        enrollmentCategoryId: null, enrollmentTag: null, active: true,
      })
    })
  })
})

describe('Delete sales channel', () => {
  it('asks for confirmation and calls delete', async () => {
    vi.mocked(integrationsApi.deleteSalesChannel).mockResolvedValue(null)
    render(<SalesChannels />)
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = screen.getByText('Delete sales channel').closest('div')!
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(integrationsApi.deleteSalesChannel).toHaveBeenCalledWith('test-token', 'chan-1'))
  })
})

describe('Manage products', () => {
  it('lists products with their computed enrollment status', async () => {
    render(<SalesChannels />)
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Products' }))

    expect(await screen.findByText('T-Shirt')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Enrolled — remove' })).toBeInTheDocument()
  })

  it('toggling a product calls the override endpoint', async () => {
    vi.mocked(integrationsApi.setSalesChannelProductOverride).mockResolvedValue(null)
    render(<SalesChannels />)
    await screen.findByText('My Woo Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))
    await screen.findByText('T-Shirt')

    await userEvent.click(screen.getByRole('button', { name: 'Enrolled — remove' }))

    await waitFor(() => {
      expect(integrationsApi.setSalesChannelProductOverride).toHaveBeenCalledWith('test-token', 'chan-1', 'p1', false)
    })
  })

  it('"Add all" applies the all-products enrollment rule', async () => {
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ enrollment_mode: 'all' }))
    render(<SalesChannels />)
    await screen.findByText('My Woo Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))
    await screen.findByText('T-Shirt')

    await userEvent.click(screen.getByRole('button', { name: 'Add all' }))

    await waitFor(() => {
      expect(integrationsApi.updateSalesChannel).toHaveBeenCalledWith('test-token', 'chan-1', {
        name: 'My Woo Store', connectionId: 'conn-1', enrollmentMode: 'all',
        enrollmentCategoryId: null, enrollmentTag: null, active: true,
      })
    })
  })

  it('"Add by category" applies the category enrollment rule for the selected category', async () => {
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ enrollment_mode: 'category', enrollment_category_id: 'cat-1' }))
    render(<SalesChannels />)
    await screen.findByText('My Woo Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))
    await screen.findByText('T-Shirt')

    await userEvent.selectOptions(screen.getByLabelText('Bulk add by category'), 'cat-1')
    await userEvent.click(screen.getAllByRole('button', { name: 'Apply' })[0])

    await waitFor(() => {
      expect(integrationsApi.updateSalesChannel).toHaveBeenCalledWith('test-token', 'chan-1', expect.objectContaining({
        enrollmentMode: 'category', enrollmentCategoryId: 'cat-1', enrollmentTag: null,
      }))
    })
  })

  it('Back returns to the channel list', async () => {
    render(<SalesChannels />)
    await screen.findByText('My Woo Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))
    await screen.findByText('T-Shirt')

    await userEvent.click(screen.getByRole('button', { name: /Sales channels/ }))

    expect(await screen.findByRole('button', { name: 'Add sales channel' })).toBeInTheDocument()
  })
})
