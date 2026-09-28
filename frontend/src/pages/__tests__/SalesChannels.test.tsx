// @vitest-environment jsdom
import { screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { act } from 'react'
import SalesChannels from '../SalesChannels'
import { renderAt } from '../../test/renderAt'

function renderPage(url = '/sales-channels') {
  return renderAt(<SalesChannels />, '/sales-channels', url)
}
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
  updateConnectionSettings: vi.fn(),
  setSalesChannelPaymentMethod: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listCategories: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
}))

vi.mock('../../fx/api', () => ({
  listExchangeRates: vi.fn(),
  upsertExchangeRate: vi.fn(),
  deleteExchangeRate: vi.fn(),
}))

import * as integrationsApi from '../../integrations/api'
import * as pmsApi from '../../pms/api'
import * as inventoryApi from '../../inventory/api'
import * as fxApi from '../../fx/api'

const wooConnection: Connection = {
  id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '', default_currency: 'USD',
}

const bolConnection: Connection = {
  id: 'conn-bol-1', provider: 'bol', store_url: '', status: 'connected', created_at: '',
}

const stripeConnection: Connection = {
  id: 'conn-stripe-1', provider: 'stripe', store_url: '', status: 'connected', created_at: '',
}

const revolutConnection: Connection = {
  id: 'conn-revolut-1', provider: 'revolut', store_url: '', status: 'error', created_at: '',
}

const whatsappConnection: Connection = {
  id: 'conn-wa-1', provider: 'whatsapp', store_url: '', status: 'connected', created_at: '',
}

function warehouse(overrides: Partial<{ id: string; name: string }> = {}) {
  return {
    id: 'wh-1', tenant_id: 'tenant-1', name: 'Main DC',
    address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
    created_at: '', updated_at: '', ...overrides,
  }
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
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])
  vi.mocked(fxApi.listExchangeRates).mockResolvedValue([])
})

describe('Sales channels list', () => {
  it('shows an empty state when there are no channels', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([])
    renderPage()
    expect(await screen.findByText('No sales channels yet')).toBeInTheDocument()
  })

  it('lists channels with their connection and enrollment mode', async () => {
    renderPage()
    expect(await screen.findByText('My Woo Store')).toBeInTheDocument()
    expect(screen.getByText('Woocommerce — https://mystore.com')).toBeInTheDocument()
    expect(screen.getByText('All products')).toBeInTheDocument()
  })
})

describe('Create sales channel', () => {
  it('creates a channel with the selected connection and enrollment mode', async () => {
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-2', name: 'New Channel' }))
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.type(await screen.findByLabelText('Name'), 'New Channel')
    await userEvent.selectOptions(screen.getByLabelText('Connection'), 'conn-1')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.createSalesChannel).toHaveBeenCalledWith('test-token', {
        name: 'New Channel', type: 'woocommerce', connectionId: 'conn-1', enrollmentMode: 'all',
        enrollmentCategoryId: null, enrollmentTag: null, priceAdjustmentBasisPoints: null, quantityBreaks: [],
      })
    })
  })

  it('filters the connection list to the selected type, and submits that type', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection, bolConnection])
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-bol', name: 'BOL', type: 'bol', connection_id: 'conn-bol-1' }))
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(await screen.findByLabelText('Type'), 'bol')
    const connectionSelect = screen.getByLabelText('Connection')
    expect(within(connectionSelect).queryByText(/mystore.com/)).not.toBeInTheDocument()
    await userEvent.type(screen.getByLabelText('Name'), 'BOL')
    await userEvent.selectOptions(connectionSelect, 'conn-bol-1')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.createSalesChannel).toHaveBeenCalledWith('test-token', expect.objectContaining({
        type: 'bol', connectionId: 'conn-bol-1',
      }))
    })
  })

  it('picking Internal hides connection/default-location fields and creates with no connection', async () => {
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null }))
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(await screen.findByLabelText('Type'), 'internal')
    expect(screen.queryByLabelText('Connection')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Default location')).not.toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Name'), 'B2C Store')
    await userEvent.selectOptions(screen.getByLabelText('Fulfillment warehouse'), 'wh-1')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.createSalesChannel).toHaveBeenCalledWith('test-token', {
        name: 'B2C Store', type: 'internal', connectionId: null, enrollmentMode: 'all',
        enrollmentCategoryId: null, enrollmentTag: null, priceAdjustmentBasisPoints: null, quantityBreaks: [],
        defaultWarehouseId: 'wh-1', storefrontMode: 'headless',
      })
    })
  })

  it('shows a currency picker only for Internal channels, offering the full currency list regardless of configured exchange rates', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    expect(screen.queryByLabelText('Currency')).not.toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Type'), 'internal')
    const currencySelect = await screen.findByLabelText('Currency')
    expect(within(currencySelect).getByText('USD — US Dollar')).toBeInTheDocument()
    expect(within(currencySelect).getByText('GBP — British Pound')).toBeInTheDocument()
    expect(within(currencySelect).getByText('EUR — Euro')).toBeInTheDocument()
    expect(within(currencySelect).getByText('CHF — Swiss Franc')).toBeInTheDocument()
  })

  it('warns when the selected currency has no configured exchange rate yet', async () => {
    vi.mocked(fxApi.listExchangeRates).mockResolvedValue([{ currency_code: 'USD', rate_basis_points: 10876, updated_at: '' }])
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(await screen.findByLabelText('Type'), 'internal')
    await userEvent.selectOptions(await screen.findByLabelText('Currency'), 'GBP')

    expect(screen.getByText(/No exchange rate configured for GBP yet/)).toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Currency'), 'USD')
    expect(screen.queryByText(/No exchange rate configured/)).not.toBeInTheDocument()
  })

  it('creates an Internal channel with the selected non-default currency', async () => {
    vi.mocked(fxApi.listExchangeRates).mockResolvedValue([{ currency_code: 'USD', rate_basis_points: 10876, updated_at: '' }])
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-usd', name: 'US Store', type: 'internal', connection_id: null, currency_code: 'USD' }))
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(await screen.findByLabelText('Type'), 'internal')
    await userEvent.type(screen.getByLabelText('Name'), 'US Store')
    await userEvent.selectOptions(await screen.findByLabelText('Currency'), 'USD')
    await userEvent.selectOptions(screen.getByLabelText('Fulfillment warehouse'), 'wh-1')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.createSalesChannel).toHaveBeenCalledWith('test-token', expect.objectContaining({
        currencyCode: 'USD',
      }))
    })
  })

  it('shows the tag input only when enrollment mode is tag, and sends it', async () => {
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-3', name: 'Tagged Channel', enrollment_mode: 'tag', enrollment_tag: 'summer' }))
    renderPage()
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
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.type(await screen.findByLabelText('Name'), 'New Channel')
    await userEvent.selectOptions(screen.getByLabelText('Connection'), 'conn-1')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    expect(await screen.findByText('connection is required')).toBeInTheDocument()
  })
})

describe('Edit sales channel', () => {
  it('prefills the form and saves changes', async () => {
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ name: 'Renamed' }))
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    expect(await screen.findByLabelText('Name')).toHaveValue('My Woo Store')

    await userEvent.clear(screen.getByLabelText('Name'))
    await userEvent.type(screen.getByLabelText('Name'), 'Renamed')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => {
      expect(integrationsApi.updateSalesChannel).toHaveBeenCalledWith('test-token', 'chan-1', {
        name: 'Renamed', connectionId: 'conn-1', enrollmentMode: 'all',
        enrollmentCategoryId: null, enrollmentTag: null, priceAdjustmentBasisPoints: null, quantityBreaks: [], active: true,
      })
    })
  })
})

describe('Delete sales channel', () => {
  it('asks for confirmation and calls delete', async () => {
    vi.mocked(integrationsApi.deleteSalesChannel).mockResolvedValue(null)
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))
    const dialog = screen.getByText('Delete sales channel').closest('div')!
    await userEvent.click(within(dialog).getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(integrationsApi.deleteSalesChannel).toHaveBeenCalledWith('test-token', 'chan-1'))
  })
})

describe('Manage products', () => {
  it('lists products with their computed enrollment status', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Products' }))

    expect(await screen.findByText('T-Shirt')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Enrolled — remove' })).toBeInTheDocument()
  })

  it('toggling a product calls the override endpoint', async () => {
    vi.mocked(integrationsApi.setSalesChannelProductOverride).mockResolvedValue(null)
    renderPage()
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
    renderPage()
    await screen.findByText('My Woo Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))
    await screen.findByText('T-Shirt')

    await userEvent.click(screen.getByRole('button', { name: 'Add all' }))

    await waitFor(() => {
      expect(integrationsApi.updateSalesChannel).toHaveBeenCalledWith('test-token', 'chan-1', {
        name: 'My Woo Store', connectionId: 'conn-1', enrollmentMode: 'all',
        enrollmentCategoryId: null, enrollmentTag: null, priceAdjustmentBasisPoints: null, quantityBreaks: [], active: true,
      })
    })
  })

  it('"Add by category" applies the category enrollment rule for the selected category', async () => {
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ enrollment_mode: 'category', enrollment_category_id: 'cat-1' }))
    renderPage()
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
    renderPage()
    await screen.findByText('My Woo Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))
    await screen.findByText('T-Shirt')

    await userEvent.click(screen.getByRole('button', { name: /Sales channels/ }))

    expect(await screen.findByRole('button', { name: 'Add sales channel' })).toBeInTheDocument()
  })
})

describe('Sales channel default location', () => {
  it('shows Default location once a connection is picked, and saves it alongside the channel', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse({ id: 'wh-1', name: 'Main DC' })])
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-2', name: 'New Channel' }))
    vi.mocked(integrationsApi.updateConnectionSettings).mockResolvedValue({ ...wooConnection, default_warehouse_id: 'wh-1' })

    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    expect(screen.queryByLabelText('Default location')).not.toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Name'), 'New Channel')
    await userEvent.selectOptions(screen.getByLabelText('Connection'), 'conn-1')
    await userEvent.selectOptions(screen.getByLabelText('Default location'), 'wh-1')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.updateConnectionSettings).toHaveBeenCalledWith('test-token', 'conn-1', 'USD', 'wh-1')
    })
  })

  it('prefills Default location from the connection when editing', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([{ ...wooConnection, default_warehouse_id: 'wh-1' }])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse({ id: 'wh-1', name: 'Main DC' })])

    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(await screen.findByLabelText('Default location')).toHaveValue('wh-1')
  })

  it('keeps the connection default location unchanged when saving other channel edits', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([{ ...wooConnection, default_warehouse_id: 'wh-1' }])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse({ id: 'wh-1', name: 'Main DC' })])
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ name: 'Renamed' }))
    vi.mocked(integrationsApi.updateConnectionSettings).mockResolvedValue({ ...wooConnection, default_warehouse_id: 'wh-1' })

    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    await userEvent.clear(await screen.findByLabelText('Name'))
    await userEvent.type(screen.getByLabelText('Name'), 'Renamed')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => {
      expect(integrationsApi.updateConnectionSettings).toHaveBeenCalledWith('test-token', 'conn-1', 'USD', 'wh-1')
    })
  })
})

describe('Sales channel fulfillment warehouse', () => {
  it('does not show a fulfillment warehouse field for a WooCommerce channel', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(screen.queryByLabelText('Fulfillment warehouse')).not.toBeInTheDocument()
  })

  it('shows a required fulfillment warehouse field for an Internal channel', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(await screen.findByLabelText('Type'), 'internal')

    expect(screen.getByLabelText('Fulfillment warehouse')).toBeRequired()
  })

  it('prefills the fulfillment warehouse from the channel itself, not the connection, when editing a WhatsApp channel', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse({ id: 'wh-1', name: 'Main DC' }), warehouse({ id: 'wh-2', name: 'Overflow' })])
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([{ ...whatsappConnection, default_warehouse_id: 'wh-1' }])
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([
      sampleChannel({ id: 'chan-wa', name: 'WA Channel', type: 'whatsapp', connection_id: 'conn-wa-1', default_warehouse_id: 'wh-2' }),
    ])
    renderPage()
    await screen.findByText('WA Channel')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(await screen.findByLabelText('Fulfillment warehouse')).toHaveValue('wh-2')
  })

  it('saves a changed fulfillment warehouse on an Internal channel edit', async () => {
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse({ id: 'wh-1', name: 'Main DC' }), warehouse({ id: 'wh-2', name: 'Overflow' })])
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([
      sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null, default_warehouse_id: 'wh-1' }),
    ])
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-int', type: 'internal', default_warehouse_id: 'wh-2' }))
    renderPage()
    await screen.findByText('B2C Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    await userEvent.selectOptions(await screen.findByLabelText('Fulfillment warehouse'), 'wh-2')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => {
      expect(integrationsApi.updateSalesChannel).toHaveBeenCalledWith('test-token', 'chan-int', expect.objectContaining({
        defaultWarehouseId: 'wh-2',
      }))
    })
  })
})

describe('Headless API docs', () => {
  it('shows an "API docs" button only for Internal channels', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    expect(screen.queryByRole('button', { name: 'API docs' })).not.toBeInTheDocument()
  })

  it('opens a reference showing the base URL, bearer auth using the channel id, and the endpoint list', async () => {
    vi.stubEnv('VITE_API_BASE_URL', 'https://api.example.com/api/v1')
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null })])
    renderPage()
    await screen.findByText('B2C Store')

    await userEvent.click(screen.getByRole('button', { name: 'API docs' }))

    expect(await screen.findByText('https://api.example.com/api/v1/storefront/v1')).toBeInTheDocument()
    expect(screen.getByText('Authorization: Bearer chan-int')).toBeInTheDocument()
    expect(screen.getByText('/products')).toBeInTheDocument()
    expect(screen.getByText('/carts/{token}/checkout')).toBeInTheDocument()
    expect(screen.queryByText(/theme/)).not.toBeInTheDocument()
  })

  it('closes the API docs modal', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null })])
    renderPage()
    await screen.findByText('B2C Store')
    await userEvent.click(screen.getByRole('button', { name: 'API docs' }))

    await userEvent.click(await screen.findByLabelText('Close'))

    expect(screen.queryByText('B2C Store — headless API')).not.toBeInTheDocument()
  })
})

describe('Pricing (Products tab)', () => {
  it('shows the Pricing section only for Internal channels', async () => {
    renderPage()
    await screen.findByText('My Woo Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))

    expect(screen.queryByText('Pricing')).not.toBeInTheDocument()
  })

  it('shows Pricing for an Internal channel, disabling Save until something changes', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null })])
    renderPage()
    await screen.findByText('B2C Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))

    expect(await screen.findByText('Pricing')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Save pricing' })).toBeDisabled()
  })

  it('prefills the existing discount and quantity tiers', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({
      id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null,
      price_adjustment_basis_points: 1500, quantity_breaks: [{ min_quantity: 10, discount_basis_points: 2000 }],
    })])
    renderPage()
    await screen.findByText('B2C Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))

    expect(await screen.findByLabelText('Discount off list price (%)')).toHaveValue(15)
    expect(screen.getByLabelText('Quantity tier 1 minimum quantity')).toHaveValue(10)
    expect(screen.getByLabelText('Quantity tier 1 discount percent')).toHaveValue(20)
  })

  it('saves a changed discount percent, preserving the channel\'s other fields', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null, enrollment_mode: 'manual' })])
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', price_adjustment_basis_points: 2000 }))
    renderPage()
    await screen.findByText('B2C Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))

    await userEvent.type(await screen.findByLabelText('Discount off list price (%)'), '20')
    await userEvent.click(screen.getByRole('button', { name: 'Save pricing' }))

    await waitFor(() => {
      expect(integrationsApi.updateSalesChannel).toHaveBeenCalledWith('test-token', 'chan-int', {
        name: 'B2C Store', connectionId: null, enrollmentMode: 'manual', enrollmentCategoryId: null, enrollmentTag: null,
        priceAdjustmentBasisPoints: 2000, quantityBreaks: [], active: true,
      })
    })
  })

  it('rejects a discount over 100%', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null })])
    renderPage()
    await screen.findByText('B2C Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))

    await userEvent.type(await screen.findByLabelText('Discount off list price (%)'), '150')
    await userEvent.click(screen.getByRole('button', { name: 'Save pricing' }))

    expect(await screen.findByText('Discount must be between 0 and 100%.')).toBeInTheDocument()
    expect(integrationsApi.updateSalesChannel).not.toHaveBeenCalled()
  })

  it('adds and saves a quantity tier', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null })])
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({
      id: 'chan-int', name: 'B2C Store', type: 'internal', quantity_breaks: [{ min_quantity: 10, discount_basis_points: 1500 }],
    }))
    renderPage()
    await screen.findByText('B2C Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))

    await userEvent.click(await screen.findByRole('button', { name: '+ Add tier' }))
    await userEvent.type(await screen.findByLabelText('Quantity tier 1 minimum quantity'), '10')
    await userEvent.type(screen.getByLabelText('Quantity tier 1 discount percent'), '15')
    await userEvent.click(screen.getByRole('button', { name: 'Save pricing' }))

    await waitFor(() => {
      expect(integrationsApi.updateSalesChannel).toHaveBeenCalledWith('test-token', 'chan-int', expect.objectContaining({
        quantityBreaks: [{ min_quantity: 10, discount_basis_points: 1500 }],
      }))
    })
  })

  it('removes a quantity tier row', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({
      id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null,
      quantity_breaks: [{ min_quantity: 10, discount_basis_points: 1500 }],
    })])
    renderPage()
    await screen.findByText('B2C Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))
    expect(await screen.findByLabelText('Quantity tier 1 minimum quantity')).toHaveValue(10)

    await userEvent.click(screen.getByRole('button', { name: 'Remove quantity tier 1' }))

    expect(screen.queryByLabelText('Quantity tier 1 minimum quantity')).not.toBeInTheDocument()
  })

  it('rejects a quantity tier with no minimum quantity', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null })])
    renderPage()
    await screen.findByText('B2C Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))

    await userEvent.click(await screen.findByRole('button', { name: '+ Add tier' }))
    await userEvent.type(await screen.findByLabelText('Quantity tier 1 discount percent'), '15')
    await userEvent.click(screen.getByRole('button', { name: 'Save pricing' }))

    expect(await screen.findByText('Each quantity tier needs a minimum quantity of at least 1.')).toBeInTheDocument()
    expect(integrationsApi.updateSalesChannel).not.toHaveBeenCalled()
  })
})

describe('WhatsApp payment method', () => {
  function whatsappChannel(overrides: Partial<SalesChannel> = {}): SalesChannel {
    return sampleChannel({ id: 'chan-wa', name: 'WA Channel', type: 'whatsapp', connection_id: 'conn-wa-1', default_warehouse_id: 'wh-1', ...overrides })
  }

  it('does not show a payment method field for a non-WhatsApp/Internal channel', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(screen.queryByLabelText('Payment method')).not.toBeInTheDocument()
  })

  it("does not wipe quantity tiers when applying a bulk enrollment action", async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({
      id: 'chan-1', name: 'My Woo Store', quantity_breaks: [{ min_quantity: 10, discount_basis_points: 1500 }],
    })])
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ enrollment_mode: 'all' }))
    renderPage()
    await screen.findByText('My Woo Store')
    await userEvent.click(screen.getByRole('button', { name: 'Products' }))
    await screen.findByText('T-Shirt')

    await userEvent.click(screen.getByRole('button', { name: 'Add all' }))

    await waitFor(() => {
      expect(integrationsApi.updateSalesChannel).toHaveBeenCalledWith('test-token', 'chan-1', expect.objectContaining({
        quantityBreaks: [{ min_quantity: 10, discount_basis_points: 1500 }],
      }))
    })
  })

  it('shows the discount badge on the channel card when set', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2B Store', type: 'internal', connection_id: null, price_adjustment_basis_points: 2000 })])
    renderPage()

    expect(await screen.findByText('20% off')).toBeInTheDocument()
  })

  it('shows a payment method field for an Internal channel too', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null })])
    renderPage()
    await screen.findByText('B2C Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(await screen.findByLabelText('Payment method')).toBeInTheDocument()
  })

  it('lists only connected Stripe/Revolut connections as payment method options', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection, whatsappConnection, stripeConnection, revolutConnection])
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([whatsappChannel()])
    renderPage()
    await screen.findByText('WA Channel')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    const select = await screen.findByLabelText('Payment method')
    expect(within(select).getByText('Stripe')).toBeInTheDocument()
    expect(within(select).queryByText('Revolut')).not.toBeInTheDocument()
  })

  it('prefills the current payment connection when editing', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection, whatsappConnection, stripeConnection])
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([whatsappChannel({ payment_connection_id: 'conn-stripe-1' })])
    renderPage()
    await screen.findByText('WA Channel')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(await screen.findByLabelText('Payment method')).toHaveValue('conn-stripe-1')
  })

  it('saves the selected payment connection alongside other channel edits', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection, whatsappConnection, stripeConnection])
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([whatsappChannel()])
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(whatsappChannel())
    vi.mocked(integrationsApi.setSalesChannelPaymentMethod).mockResolvedValue(whatsappChannel({ payment_connection_id: 'conn-stripe-1' }))
    renderPage()
    await screen.findByText('WA Channel')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    await userEvent.selectOptions(await screen.findByLabelText('Payment method'), 'conn-stripe-1')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => {
      expect(integrationsApi.setSalesChannelPaymentMethod).toHaveBeenCalledWith('test-token', 'chan-wa', 'conn-stripe-1')
    })
  })

  it('disabling payment method sends null', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection, whatsappConnection, stripeConnection])
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([whatsappChannel({ payment_connection_id: 'conn-stripe-1' })])
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(whatsappChannel({ payment_connection_id: 'conn-stripe-1' }))
    vi.mocked(integrationsApi.setSalesChannelPaymentMethod).mockResolvedValue(whatsappChannel())
    renderPage()
    await screen.findByText('WA Channel')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    await userEvent.selectOptions(await screen.findByLabelText('Payment method'), '')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => {
      expect(integrationsApi.setSalesChannelPaymentMethod).toHaveBeenCalledWith('test-token', 'chan-wa', null)
    })
  })
})

describe('Sales channels URLs (refresh + Back)', () => {
  it('Products pushes /sales-channels/:id and a refresh there reopens the manager', async () => {
    const { router, unmount } = renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Products' }))
    expect(router.state.location.pathname).toBe('/sales-channels/chan-1')
    unmount()

    renderPage('/sales-channels/chan-1')
    expect(await screen.findByText('T-Shirt')).toBeInTheDocument()
  })

  it('/sales-channels/:id/edit opens the prefilled edit form on load', async () => {
    renderPage('/sales-channels/chan-1/edit')
    await waitFor(() => expect(screen.getByLabelText('Name')).toHaveValue('My Woo Store'))
  })

  it('/sales-channels/new opens the create form on load', async () => {
    renderPage('/sales-channels/new')
    expect(await screen.findByText('Add sales channel', { selector: 'h2, h3' })).toBeInTheDocument()
  })

  it('/sales-channels/:id/api opens the API docs modal on load', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2C Store', type: 'internal', connection_id: null })])
    renderPage('/sales-channels/chan-int/api')
    expect(await screen.findByText('B2C Store — headless API')).toBeInTheDocument()
  })

  it('browser Back closes the edit form', async () => {
    const { router } = renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Edit' }))
    expect(router.state.location.pathname).toBe('/sales-channels/chan-1/edit')

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/sales-channels')
    expect(screen.queryByLabelText('Name')).not.toBeInTheDocument()
  })

  it('an unknown channel id shows not-found', async () => {
    renderPage('/sales-channels/nope')
    expect(await screen.findByText('Sales channel not found')).toBeInTheDocument()
  })

  it('manager page and search live in the query string', async () => {
    renderPage('/sales-channels/chan-1?page=2&q=shirt')
    await screen.findByText('T-Shirt')
    expect(integrationsApi.listSalesChannelProducts).toHaveBeenCalledWith('test-token', 'chan-1', expect.objectContaining({ page: 2, search: 'shirt' }))
  })
})

describe('Storefront mode', () => {
  it('does not offer a storefront mode for a WooCommerce channel', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))

    expect(screen.queryByLabelText('Storefront')).not.toBeInTheDocument()
  })

  it('defaults a new Internal channel to headless and can create it with a hosted storefront', async () => {
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-int', name: 'Shop', type: 'internal', connection_id: null, storefront_mode: 'theme' }))
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(await screen.findByLabelText('Type'), 'internal')
    expect(screen.getByLabelText('Storefront')).toHaveValue('headless')

    await userEvent.type(screen.getByLabelText('Name'), 'Shop')
    await userEvent.selectOptions(screen.getByLabelText('Fulfillment warehouse'), 'wh-1')
    await userEvent.selectOptions(screen.getByLabelText('Storefront'), 'theme')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.createSalesChannel).toHaveBeenCalledWith('test-token', expect.objectContaining({
        type: 'internal', storefrontMode: 'theme',
      }))
    })
  })

  it('prefills and saves the storefront mode when editing an Internal channel', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([
      sampleChannel({ id: 'chan-int', name: 'Shop', type: 'internal', connection_id: null, default_warehouse_id: 'wh-1', storefront_mode: 'theme' }),
    ])
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-int', type: 'internal', storefront_mode: 'headless' }))
    renderPage()
    await screen.findByText('Shop')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    expect(await screen.findByLabelText('Storefront')).toHaveValue('theme')
    await userEvent.selectOptions(screen.getByLabelText('Storefront'), 'headless')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => {
      expect(integrationsApi.updateSalesChannel).toHaveBeenCalledWith('test-token', 'chan-int', expect.objectContaining({
        storefrontMode: 'headless',
      }))
    })
  })
})
