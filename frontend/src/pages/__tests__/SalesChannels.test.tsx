// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import SalesChannels from '../SalesChannels'

function renderPage() {
  return render(<MemoryRouter><SalesChannels /></MemoryRouter>)
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
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
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
    await userEvent.type(screen.getByLabelText('Name'), 'New Channel')
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
    await userEvent.selectOptions(screen.getByLabelText('Type'), 'bol')
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
    await userEvent.selectOptions(screen.getByLabelText('Type'), 'internal')
    expect(screen.queryByLabelText('Connection')).not.toBeInTheDocument()
    expect(screen.queryByLabelText('Default location')).not.toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Name'), 'B2C Store')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.createSalesChannel).toHaveBeenCalledWith('test-token', {
        name: 'B2C Store', type: 'internal', connectionId: null, enrollmentMode: 'all',
        enrollmentCategoryId: null, enrollmentTag: null, priceAdjustmentBasisPoints: null, quantityBreaks: [],
      })
    })
  })

  it('shows a currency picker only for Internal channels, listing configured exchange rates', async () => {
    vi.mocked(fxApi.listExchangeRates).mockResolvedValue([
      { currency_code: 'USD', rate_basis_points: 10876, updated_at: '' },
      { currency_code: 'GBP', rate_basis_points: 8700, updated_at: '' },
    ])
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    expect(screen.queryByLabelText('Currency')).not.toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Type'), 'internal')
    const currencySelect = await screen.findByLabelText('Currency')
    expect(within(currencySelect).getByText('USD')).toBeInTheDocument()
    expect(within(currencySelect).getByText('GBP')).toBeInTheDocument()
  })

  it('creates an Internal channel with the selected non-default currency', async () => {
    vi.mocked(fxApi.listExchangeRates).mockResolvedValue([{ currency_code: 'USD', rate_basis_points: 10876, updated_at: '' }])
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-usd', name: 'US Store', type: 'internal', connection_id: null, currency_code: 'USD' }))
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(screen.getByLabelText('Type'), 'internal')
    await userEvent.type(screen.getByLabelText('Name'), 'US Store')
    await userEvent.selectOptions(await screen.findByLabelText('Currency'), 'USD')
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
    await userEvent.type(screen.getByLabelText('Name'), 'New Channel')
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
    expect(screen.getByLabelText('Name')).toHaveValue('My Woo Store')

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

    expect(screen.getByLabelText('Default location')).toHaveValue('wh-1')
  })

  it('keeps the connection default location unchanged when saving other channel edits', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([{ ...wooConnection, default_warehouse_id: 'wh-1' }])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse({ id: 'wh-1', name: 'Main DC' })])
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(sampleChannel({ name: 'Renamed' }))
    vi.mocked(integrationsApi.updateConnectionSettings).mockResolvedValue({ ...wooConnection, default_warehouse_id: 'wh-1' })

    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    await userEvent.clear(screen.getByLabelText('Name'))
    await userEvent.type(screen.getByLabelText('Name'), 'Renamed')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => {
      expect(integrationsApi.updateConnectionSettings).toHaveBeenCalledWith('test-token', 'conn-1', 'USD', 'wh-1')
    })
  })
})

describe('WhatsApp payment method', () => {
  function whatsappChannel(overrides: Partial<SalesChannel> = {}): SalesChannel {
    return sampleChannel({ id: 'chan-wa', name: 'WA Channel', type: 'whatsapp', connection_id: 'conn-wa-1', ...overrides })
  }

  it('does not show a payment method field for a non-WhatsApp/Internal channel', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(screen.queryByLabelText('Payment method')).not.toBeInTheDocument()
  })

  it('shows the discount field only for Internal channels', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    expect(screen.queryByLabelText('Discount off list price (%)')).not.toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Type'), 'internal')
    expect(screen.getByLabelText('Discount off list price (%)')).toBeInTheDocument()
  })

  it('converts a percent into basis points and sends it on create', async () => {
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({ id: 'chan-int', name: 'B2B Store', type: 'internal', connection_id: null, price_adjustment_basis_points: 2000 }))
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(screen.getByLabelText('Type'), 'internal')
    await userEvent.type(screen.getByLabelText('Name'), 'B2B Store')
    await userEvent.type(screen.getByLabelText('Discount off list price (%)'), '20')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.createSalesChannel).toHaveBeenCalledWith('test-token', expect.objectContaining({
        priceAdjustmentBasisPoints: 2000,
      }))
    })
  })

  it('prefills the discount field from basis points when editing', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({ id: 'chan-int', name: 'B2B Store', type: 'internal', connection_id: null, price_adjustment_basis_points: 1500 })])
    renderPage()
    await screen.findByText('B2B Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(screen.getByLabelText('Discount off list price (%)')).toHaveValue(15)
  })

  it('rejects a discount over 100%', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(screen.getByLabelText('Type'), 'internal')
    await userEvent.type(screen.getByLabelText('Name'), 'B2B Store')
    await userEvent.type(screen.getByLabelText('Discount off list price (%)'), '150')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    expect(await screen.findByText('Discount must be between 0 and 100%.')).toBeInTheDocument()
    expect(integrationsApi.createSalesChannel).not.toHaveBeenCalled()
  })

  it('adds a quantity tier and saves it', async () => {
    vi.mocked(integrationsApi.createSalesChannel).mockResolvedValue(sampleChannel({
      id: 'chan-int', name: 'B2B Store', type: 'internal', connection_id: null,
      quantity_breaks: [{ min_quantity: 10, discount_basis_points: 1500 }],
    }))
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(screen.getByLabelText('Type'), 'internal')
    await userEvent.type(screen.getByLabelText('Name'), 'B2B Store')
    await userEvent.click(screen.getByRole('button', { name: '+ Add tier' }))
    await userEvent.type(screen.getByLabelText('Quantity tier 1 minimum quantity'), '10')
    await userEvent.type(screen.getByLabelText('Quantity tier 1 discount percent'), '15')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    await waitFor(() => {
      expect(integrationsApi.createSalesChannel).toHaveBeenCalledWith('test-token', expect.objectContaining({
        quantityBreaks: [{ min_quantity: 10, discount_basis_points: 1500 }],
      }))
    })
  })

  it('prefills quantity tiers back when editing', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({
      id: 'chan-int', name: 'B2B Store', type: 'internal', connection_id: null,
      quantity_breaks: [{ min_quantity: 10, discount_basis_points: 1500 }],
    })])
    renderPage()
    await screen.findByText('B2B Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(screen.getByLabelText('Quantity tier 1 minimum quantity')).toHaveValue(10)
    expect(screen.getByLabelText('Quantity tier 1 discount percent')).toHaveValue(15)
  })

  it('removes a quantity tier row', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([sampleChannel({
      id: 'chan-int', name: 'B2B Store', type: 'internal', connection_id: null,
      quantity_breaks: [{ min_quantity: 10, discount_basis_points: 1500 }],
    })])
    renderPage()
    await screen.findByText('B2B Store')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    expect(screen.getByLabelText('Quantity tier 1 minimum quantity')).toHaveValue(10)

    await userEvent.click(screen.getByRole('button', { name: 'Remove quantity tier 1' }))

    expect(screen.queryByLabelText('Quantity tier 1 minimum quantity')).not.toBeInTheDocument()
  })

  it('rejects a quantity tier with no minimum quantity', async () => {
    renderPage()
    await screen.findByText('My Woo Store')

    await userEvent.click(screen.getByRole('button', { name: 'Add sales channel' }))
    await userEvent.selectOptions(screen.getByLabelText('Type'), 'internal')
    await userEvent.type(screen.getByLabelText('Name'), 'B2B Store')
    await userEvent.click(screen.getByRole('button', { name: '+ Add tier' }))
    await userEvent.type(screen.getByLabelText('Quantity tier 1 discount percent'), '15')
    await userEvent.click(screen.getByRole('button', { name: 'Add channel' }))

    expect(await screen.findByText('Each quantity tier needs a minimum quantity of at least 1.')).toBeInTheDocument()
    expect(integrationsApi.createSalesChannel).not.toHaveBeenCalled()
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

    expect(screen.getByLabelText('Payment method')).toBeInTheDocument()
  })

  it('lists only connected Stripe/Revolut connections as payment method options', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection, whatsappConnection, stripeConnection, revolutConnection])
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([whatsappChannel()])
    renderPage()
    await screen.findByText('WA Channel')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    const select = screen.getByLabelText('Payment method')
    expect(within(select).getByText('Stripe')).toBeInTheDocument()
    expect(within(select).queryByText('Revolut')).not.toBeInTheDocument()
  })

  it('prefills the current payment connection when editing', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection, whatsappConnection, stripeConnection])
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([whatsappChannel({ payment_connection_id: 'conn-stripe-1' })])
    renderPage()
    await screen.findByText('WA Channel')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))

    expect(screen.getByLabelText('Payment method')).toHaveValue('conn-stripe-1')
  })

  it('saves the selected payment connection alongside other channel edits', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection, whatsappConnection, stripeConnection])
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([whatsappChannel()])
    vi.mocked(integrationsApi.updateSalesChannel).mockResolvedValue(whatsappChannel())
    vi.mocked(integrationsApi.setSalesChannelPaymentMethod).mockResolvedValue(whatsappChannel({ payment_connection_id: 'conn-stripe-1' }))
    renderPage()
    await screen.findByText('WA Channel')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    await userEvent.selectOptions(screen.getByLabelText('Payment method'), 'conn-stripe-1')
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
    await userEvent.selectOptions(screen.getByLabelText('Payment method'), '')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => {
      expect(integrationsApi.setSalesChannelPaymentMethod).toHaveBeenCalledWith('test-token', 'chan-wa', null)
    })
  })
})
