// @vitest-environment jsdom
import { screen, waitFor, within } from '@testing-library/react'
import { act } from 'react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Settings from '../Settings'
import { renderAt } from '../../test/renderAt'
import type { Connection, SyncSchedule, DeleteAllImportedProductsResult } from '../../integrations/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../integrations/api', () => ({
  listConnections: vi.fn(),
  listSalesChannels: vi.fn(),
  listSyncSchedules: vi.fn(),
  upsertSyncSchedules: vi.fn(),
  deleteAllImportedProducts: vi.fn(),
  updateConnectionSettings: vi.fn(),
  getAIConnection: vi.fn(),
  connectAIProvider: vi.fn(),
  disconnectAIProvider: vi.fn(),
  listWooTaxes: vi.fn(),
  listWooCategories: vi.fn(),
  listExternalCategoryLinks: vi.fn(),
  upsertExternalCategoryLink: vi.fn(),
  deleteExternalCategoryLink: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listCategories: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
  createWarehouse: vi.fn(),
  updateWarehouse: vi.fn(),
  deleteWarehouse: vi.fn(),
}))

vi.mock('../../settings/notifications/api', async importOriginal => {
  const actual = await importOriginal<typeof import('../../settings/notifications/api')>()
  return {
    ...actual,
    getNotificationPreferences: vi.fn().mockResolvedValue({ categories: [], email_recipients: [], updated_at: null }),
    updateNotificationPreferences: vi.fn(),
  }
})

vi.mock('../../settings/company/api', async importOriginal => {
  const actual = await importOriginal<typeof import('../../settings/company/api')>()
  return {
    ...actual,
    getCompanyProfile: vi.fn().mockResolvedValue({
      legal_name: '', trading_name: '', vat_number: '', chamber_of_commerce: '', email: '', phone: '', iban: '',
      address: { street: '', house_number: '', postal_code: '', city: '', country: '' }, updated_at: '',
    }),
    updateCompanyProfile: vi.fn(),
  }
})

vi.mock('../../fx/api', async importOriginal => ({
  ...await importOriginal<typeof import('../../fx/api')>(),
  listExchangeRates: vi.fn().mockResolvedValue([]),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn(),
  updateTenantSettings: vi.fn(),
  listApiKeys: vi.fn(),
  createApiKey: vi.fn(),
  revokeApiKey: vi.fn(),
}))

vi.mock('../../tax/api', () => ({
  listTaxClasses: vi.fn(),
  createTaxClass: vi.fn(),
  updateTaxClass: vi.fn(),
  deleteTaxClass: vi.fn(),
  listTaxRates: vi.fn(),
  upsertTaxRate: vi.fn(),
  deleteTaxRate: vi.fn(),
  listExternalTaxClassLinks: vi.fn(),
  upsertExternalTaxClassLink: vi.fn(),
  deleteExternalTaxClassLink: vi.fn(),
}))

import * as integrationsApi from '../../integrations/api'
import * as inventoryApi from '../../inventory/api'
import * as tenantSettingsApi from '../../settings/api'
import * as taxApi from '../../tax/api'
import * as pmsApi from '../../pms/api'
import { subscribeProductsChanged } from '../../pms/productsBus'
import { acceptConfirm } from '../../test/confirm'

function wooConnection(overrides: Partial<Connection> = {}): Connection {
  return { id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '', ...overrides }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listSyncSchedules).mockResolvedValue([])
  vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([])
  vi.mocked(integrationsApi.getAIConnection).mockResolvedValue(null)
  vi.mocked(integrationsApi.listWooTaxes).mockResolvedValue([])
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
  vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '',
    prices_include_tax: false, default_tax_country: '',
    inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
  })
  vi.mocked(tenantSettingsApi.listApiKeys).mockResolvedValue([])
  vi.mocked(taxApi.listTaxClasses).mockResolvedValue([])
  vi.mocked(taxApi.listTaxRates).mockResolvedValue([])
  vi.mocked(taxApi.listExternalTaxClassLinks).mockResolvedValue([])
  vi.mocked(integrationsApi.listWooCategories).mockResolvedValue([])
  vi.mocked(integrationsApi.listExternalCategoryLinks).mockResolvedValue([])
  vi.mocked(pmsApi.listCategories).mockResolvedValue([])
})

async function openLocalizationSection() {
  await userEvent.click(await screen.findByText('Localization'))
}

async function openTaxSection() {
  await userEvent.click(await screen.findByText('Tax'))
}

async function openPricingSection() {
  await userEvent.click(await screen.findByText('Pricing'))
}

async function openIntegrationsSection() {
  await userEvent.click(await screen.findByText('Integrations'))
}

async function openInventorySection() {
  await userEvent.click(await screen.findByText('Inventory rules'))
}

describe('Settings > Integrations overview', () => {
  it('shows an empty state when there are no connections', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()

    expect(await screen.findByText('No integrations connected yet.')).toBeInTheDocument()
  })

  it('lists connections with status and a Configure button for WooCommerce', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()

    expect(await screen.findByText('https://mystore.com')).toBeInTheDocument()
    expect(screen.getByText('connected')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Configure' })).toBeInTheDocument()
  })

  it('clicking Configure opens the WooCommerce sync settings page', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))

    expect(await screen.findByText('Sync schedule')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'https://mystore.com' })).toBeInTheDocument()
  })
})

describe('Settings > Integrations > API keys', () => {
  beforeEach(() => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    vi.mocked(tenantSettingsApi.listApiKeys).mockResolvedValue([])
  })

  it('shows no keys yet by default', async () => {
    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()

    expect(await screen.findByText(/No keys yet/)).toBeInTheDocument()
  })

  it('lists existing keys with their status', async () => {
    vi.mocked(tenantSettingsApi.listApiKeys).mockResolvedValue([
      { id: 'key-1', name: 'Main storefront', description: 'Prod Woo site', created_at: '2026-01-01T00:00:00Z', revoked_at: null, sales_channel_id: null },
    ])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()

    expect(await screen.findByText('Main storefront')).toBeInTheDocument()
    expect(screen.getByText('Prod Woo site')).toBeInTheDocument()
    expect(screen.getByText('active')).toBeInTheDocument()
  })

  it('creating a key calls the endpoint and shows the plaintext value once', async () => {
    vi.mocked(tenantSettingsApi.createApiKey).mockResolvedValue({
      id: 'key-1', name: 'Main storefront', description: '', created_at: '2026-01-01T00:00:00Z', revoked_at: null, sales_channel_id: null, key: 'top-secret-key',
    })

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await screen.findByText(/No keys yet/)
    await userEvent.type(screen.getByLabelText('Name'), 'Main storefront')
    await userEvent.click(screen.getByRole('button', { name: 'Create key' }))

    expect(await screen.findByText('top-secret-key')).toBeInTheDocument()
    expect(tenantSettingsApi.createApiKey).toHaveBeenCalledWith('test-token', 'Main storefront', '', null)
    expect(await screen.findByText('Main storefront')).toBeInTheDocument()
  })

  it('surfaces an error inline if creating fails', async () => {
    vi.mocked(tenantSettingsApi.createApiKey).mockRejectedValue(new Error('server error'))

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await screen.findByText(/No keys yet/)
    await userEvent.type(screen.getByLabelText('Name'), 'Main storefront')
    await userEvent.click(screen.getByRole('button', { name: 'Create key' }))

    expect(await screen.findByText('server error')).toBeInTheDocument()
  })

  it('revoking a key calls the endpoint and marks it revoked', async () => {
    vi.mocked(tenantSettingsApi.listApiKeys).mockResolvedValue([
      { id: 'key-1', name: 'Main storefront', description: '', created_at: '2026-01-01T00:00:00Z', revoked_at: null, sales_channel_id: null },
    ])
    vi.mocked(tenantSettingsApi.revokeApiKey).mockResolvedValue(null)

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Revoke' }))

    expect(tenantSettingsApi.revokeApiKey).toHaveBeenCalledWith('test-token', 'key-1')
    expect(await screen.findByText('revoked')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Revoke' })).not.toBeInTheDocument()
  })

  it('does not show a sales channel picker when the tenant has no internal channels', async () => {
    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await screen.findByText(/No keys yet/)

    expect(screen.queryByLabelText('Sales channel')).not.toBeInTheDocument()
  })

  it('offers only internal channels in the sales channel picker, and sends the chosen one', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([
      { id: 'chan-woo', name: 'Woo Store', type: 'woocommerce', connection_id: 'conn-1', enrollment_mode: 'all', active: true, created_at: '', updated_at: '' },
      { id: 'chan-b2c', name: 'B2C Store', type: 'internal', connection_id: null, enrollment_mode: 'all', active: true, created_at: '', updated_at: '' },
      { id: 'chan-b2b', name: 'B2B Store', type: 'internal', connection_id: null, enrollment_mode: 'all', active: true, created_at: '', updated_at: '' },
    ])
    vi.mocked(tenantSettingsApi.createApiKey).mockResolvedValue({
      id: 'key-1', name: 'B2C key', description: '', created_at: '2026-01-01T00:00:00Z', revoked_at: null, sales_channel_id: 'chan-b2c', key: 'top-secret-key',
    })

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await screen.findByText(/No keys yet/)

    const select = screen.getByLabelText('Sales channel')
    expect(within(select).queryByText('Woo Store')).not.toBeInTheDocument()
    expect(within(select).getByText('B2C Store')).toBeInTheDocument()
    expect(within(select).getByText('B2B Store')).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('Name'), 'B2C key')
    await userEvent.selectOptions(select, 'chan-b2c')
    await userEvent.click(screen.getByRole('button', { name: 'Create key' }))

    await waitFor(() => {
      expect(tenantSettingsApi.createApiKey).toHaveBeenCalledWith('test-token', 'B2C key', '', 'chan-b2c')
    })
  })

  it('shows which sales channel each key is scoped to, or "All channels" for a tenant-wide key', async () => {
    vi.mocked(integrationsApi.listSalesChannels).mockResolvedValue([
      { id: 'chan-b2c', name: 'B2C Store', type: 'internal', connection_id: null, enrollment_mode: 'all', active: true, created_at: '', updated_at: '' },
    ])
    vi.mocked(tenantSettingsApi.listApiKeys).mockResolvedValue([
      { id: 'key-1', name: 'B2C key', description: '', created_at: '2026-01-01T00:00:00Z', revoked_at: null, sales_channel_id: 'chan-b2c' },
      { id: 'key-2', name: 'Woo plugin', description: '', created_at: '2026-01-01T00:00:00Z', revoked_at: null, sales_channel_id: null },
    ])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()

    await screen.findByText('B2C key')
    const table = screen.getByRole('table')
    expect(within(table).getByText('B2C Store')).toBeInTheDocument()
    expect(within(table).getByText('All channels')).toBeInTheDocument()
  })
})

describe('Settings > Integrations > WooCommerce > Taxes tab', () => {
  function taxClass(overrides: Partial<import('../../tax/types').TaxClass> = {}): import('../../tax/types').TaxClass {
    return { id: 'class-1', name: 'Standard', is_default: true, created_at: '', updated_at: '', ...overrides }
  }

  function externalLink(overrides: Partial<import('../../tax/types').ExternalTaxClassLink> = {}): import('../../tax/types').ExternalTaxClassLink {
    return { id: 'link-1', provider: 'woocommerce', external_slug: 'reduced-rate', tax_class_id: 'class-1', created_at: '', updated_at: '', ...overrides }
  }

  async function openWooTaxesTab() {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Taxes' }))
  }

  it('lists WooCommerce tax links with slug and the linked class name', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(taxApi.listExternalTaxClassLinks).mockResolvedValue([externalLink()])

    await openWooTaxesTab()

    expect(await screen.findByText('reduced-rate')).toBeInTheDocument()
    expect(screen.getByLabelText('Linked tax class')).toHaveValue('class-1')
  })

  it('does not show links from other providers', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(taxApi.listExternalTaxClassLinks).mockResolvedValue([externalLink({ id: 'link-2', provider: 'shopify', external_slug: 'shopify-standard' })])

    await openWooTaxesTab()

    expect(await screen.findByText('No WooCommerce taxes yet — run an import to pull them in, or add one below.')).toBeInTheDocument()
    expect(screen.queryByText('shopify-standard')).not.toBeInTheDocument()
  })

  it("repointing a link's class calls upsertExternalTaxClassLink", async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass(), taxClass({ id: 'class-2', name: 'Zero Rate', is_default: false })])
    vi.mocked(taxApi.listExternalTaxClassLinks).mockResolvedValue([externalLink()])
    vi.mocked(taxApi.upsertExternalTaxClassLink).mockResolvedValue(externalLink({ tax_class_id: 'class-2' }))

    await openWooTaxesTab()
    await screen.findByText('reduced-rate')

    await userEvent.selectOptions(screen.getByLabelText('Linked tax class'), 'class-2')

    await waitFor(() => expect(taxApi.upsertExternalTaxClassLink).toHaveBeenCalledWith('test-token', {
      provider: 'woocommerce', external_slug: 'reduced-rate', tax_class_id: 'class-2',
    }))
  })

  it('populates the Woo tax class select from listWooTaxes', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(integrationsApi.listWooTaxes).mockResolvedValue([
      { slug: '', label: 'Standard' },
      { slug: 'zero-rate', label: 'Zero Rate' },
    ])

    await openWooTaxesTab()

    const select = await screen.findByLabelText('Woo tax class')
    await waitFor(() => expect(select).not.toBeDisabled())
    expect(within(select).getByRole('option', { name: 'Standard' })).toBeInTheDocument()
    expect(within(select).getByRole('option', { name: 'Zero Rate' })).toBeInTheDocument()
  })

  it('adding a link calls upsertExternalTaxClassLink with provider woocommerce', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(integrationsApi.listWooTaxes).mockResolvedValue([{ slug: 'zero-rate', label: 'Zero Rate' }])
    vi.mocked(taxApi.upsertExternalTaxClassLink).mockResolvedValue(externalLink({ id: 'link-2', external_slug: 'zero-rate' }))

    await openWooTaxesTab()
    const select = await screen.findByLabelText('Woo tax class')
    await waitFor(() => expect(select).not.toBeDisabled())

    await userEvent.selectOptions(select, 'zero-rate')
    await userEvent.selectOptions(screen.getByLabelText('Tax class'), 'class-1')
    await userEvent.click(screen.getByRole('button', { name: 'Add link' }))

    await waitFor(() => expect(taxApi.upsertExternalTaxClassLink).toHaveBeenCalledWith('test-token', {
      provider: 'woocommerce', external_slug: 'zero-rate', tax_class_id: 'class-1',
    }))
  })

  it('removing a link calls deleteExternalTaxClassLink', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(taxApi.listExternalTaxClassLinks).mockResolvedValue([externalLink()])

    await openWooTaxesTab()
    await screen.findByText('reduced-rate')

    await userEvent.click(screen.getByRole('button', { name: 'Remove link' }))

    await waitFor(() => expect(taxApi.deleteExternalTaxClassLink).toHaveBeenCalledWith('test-token', 'link-1'))
  })
})

describe('Settings > Integrations > WooCommerce > Categories tab', () => {
  function category(overrides: Partial<import('../../pms/types').Category> = {}): import('../../pms/types').Category {
    return { id: 'cat-1', tenant_id: 'tenant-1', name: 'Beverages', slug: 'beverages', position: 0, created_at: '', updated_at: '', ...overrides }
  }

  function wooCategory(overrides: Partial<import('../../integrations/types').WooCategory> = {}): import('../../integrations/types').WooCategory {
    return { external_id: '20', name: 'Beverages', slug: 'beverages', ...overrides }
  }

  function externalCategoryLink(overrides: Partial<import('../../integrations/types').ExternalCategoryLink> = {}): import('../../integrations/types').ExternalCategoryLink {
    return { id: 'link-1', connection_id: 'conn-1', provider: 'woocommerce', external_id: '20', category_id: 'cat-1', created_at: '', updated_at: '', ...overrides }
  }

  async function openWooCategoriesTab() {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Categories' }))
  }

  it('lists category links with category name and linked Woo category name', async () => {
    vi.mocked(pmsApi.listCategories).mockResolvedValue([category()])
    vi.mocked(integrationsApi.listExternalCategoryLinks).mockResolvedValue([externalCategoryLink()])

    await openWooCategoriesTab()

    expect(await screen.findByText('Beverages', { selector: 'span' })).toBeInTheDocument()
    expect(screen.getByLabelText('Linked Woo category')).toHaveValue('20')
  })

  it('repointing a link calls upsertExternalCategoryLink', async () => {
    vi.mocked(pmsApi.listCategories).mockResolvedValue([category()])
    vi.mocked(integrationsApi.listExternalCategoryLinks).mockResolvedValue([externalCategoryLink()])
    vi.mocked(integrationsApi.listWooCategories).mockResolvedValue([wooCategory(), wooCategory({ external_id: '30', name: 'Snacks', slug: 'snacks' })])
    vi.mocked(integrationsApi.upsertExternalCategoryLink).mockResolvedValue(externalCategoryLink({ external_id: '30' }))

    await openWooCategoriesTab()
    await screen.findByText('Beverages', { selector: 'span' })

    await userEvent.selectOptions(screen.getByLabelText('Linked Woo category'), '30')

    await waitFor(() => expect(integrationsApi.upsertExternalCategoryLink).toHaveBeenCalledWith('test-token', 'conn-1', {
      external_id: '30', category_id: 'cat-1',
    }))
  })

  it('adding a link calls upsertExternalCategoryLink', async () => {
    vi.mocked(pmsApi.listCategories).mockResolvedValue([category()])
    vi.mocked(integrationsApi.listWooCategories).mockResolvedValue([wooCategory()])
    vi.mocked(integrationsApi.upsertExternalCategoryLink).mockResolvedValue(externalCategoryLink())

    await openWooCategoriesTab()
    const wooSelect = await screen.findByLabelText('Woo category')
    await waitFor(() => expect(wooSelect).not.toBeDisabled())

    await userEvent.selectOptions(screen.getByLabelText('Category'), 'cat-1')
    await userEvent.selectOptions(wooSelect, '20')
    await userEvent.click(screen.getByRole('button', { name: 'Add link' }))

    await waitFor(() => expect(integrationsApi.upsertExternalCategoryLink).toHaveBeenCalledWith('test-token', 'conn-1', {
      external_id: '20', category_id: 'cat-1',
    }))
  })

  it('removing a link calls deleteExternalCategoryLink', async () => {
    vi.mocked(pmsApi.listCategories).mockResolvedValue([category()])
    vi.mocked(integrationsApi.listExternalCategoryLinks).mockResolvedValue([externalCategoryLink()])

    await openWooCategoriesTab()
    await screen.findByText('Beverages', { selector: 'span' })

    await userEvent.click(screen.getByRole('button', { name: 'Remove link' }))

    await waitFor(() => expect(integrationsApi.deleteExternalCategoryLink).toHaveBeenCalledWith('test-token', 'conn-1', 'link-1'))
  })
})

describe('Settings > Integrations > AI', () => {
  it('shows not configured and a Connect button when no AI provider is set up', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()

    expect(await screen.findByText(/Not configured/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Connect' })).toBeInTheDocument()
  })

  it('shows the configured provider and status', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    vi.mocked(integrationsApi.getAIConnection).mockResolvedValue({
      id: 'ai-conn-1', provider: 'anthropic', status: 'connected', created_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()

    expect(await screen.findByText('Anthropic (Claude)')).toBeInTheDocument()
    expect(screen.getByText('connected')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Configure' })).toBeInTheDocument()
  })

  it('connects a hosted provider with an API key', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    vi.mocked(integrationsApi.connectAIProvider).mockResolvedValue({
      id: 'ai-conn-1', provider: 'openai', status: 'connected', created_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Connect' }))
    await userEvent.type(await screen.findByLabelText('API key'), 'sk-test-key')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.connectAIProvider).toHaveBeenCalledWith('test-token', {
      provider: 'openai', apiKey: 'sk-test-key', baseUrl: '', model: '', fallbackModels: '',
    }))
  })

  it('shows fallback models only for Gemini, and sends it on save', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    vi.mocked(integrationsApi.connectAIProvider).mockResolvedValue({
      id: 'ai-conn-1', provider: 'gemini', status: 'connected', created_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Connect' }))

    expect(screen.queryByLabelText(/Fallback models/)).not.toBeInTheDocument()

    await userEvent.selectOptions(screen.getByLabelText('Provider'), 'gemini')
    expect(screen.getByLabelText(/Fallback models/)).toBeInTheDocument()

    await userEvent.type(screen.getByLabelText('API key'), 'gm-test-key')
    await userEvent.type(screen.getByLabelText(/Fallback models/), 'gemini-3.5-flash-lite,gemini-3.1-flash-lite')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.connectAIProvider).toHaveBeenCalledWith('test-token', {
      provider: 'gemini', apiKey: 'gm-test-key', baseUrl: '', model: '', fallbackModels: 'gemini-3.5-flash-lite,gemini-3.1-flash-lite',
    }))
  })

  it('prefills fallback models from the existing connection', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    vi.mocked(integrationsApi.getAIConnection).mockResolvedValue({
      id: 'ai-conn-1', provider: 'gemini', status: 'connected', created_at: '',
      model: 'gemini-3.6-flash', fallback_models: 'gemini-3.5-flash-lite',
    })

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))

    expect(await screen.findByLabelText(/Fallback models/)).toHaveValue('gemini-3.5-flash-lite')
  })

  it('requires a server URL instead of an API key for Ollama', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Connect' }))
    await userEvent.selectOptions(await screen.findByLabelText('Provider'), 'ollama')

    expect(screen.queryByLabelText('API key')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Server URL')).toBeInTheDocument()
  })

  it('shows an error if connecting fails', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    vi.mocked(integrationsApi.connectAIProvider).mockRejectedValue(new Error('an API key is required for this provider'))

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Connect' }))
    await userEvent.type(await screen.findByLabelText('API key'), 'sk-test')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('an API key is required for this provider')).toBeInTheDocument()
  })

  it('disconnects after confirming', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    vi.mocked(integrationsApi.getAIConnection).mockResolvedValue({
      id: 'ai-conn-1', provider: 'gemini', status: 'connected', created_at: '',
    })
    vi.mocked(integrationsApi.disconnectAIProvider).mockResolvedValue(null)

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Disconnect' }))

    await acceptConfirm()
    await waitFor(() => expect(integrationsApi.disconnectAIProvider).toHaveBeenCalledWith('test-token'))
  })
})

describe('WooCommerce sync settings page', () => {
  it('shows Products, Stock levels, and Orders import as configurable, Categories and Orders export as disabled', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByRole('switch', { name: 'Products import' })).not.toBeDisabled()
    expect(screen.getByRole('switch', { name: 'Stock levels export' })).not.toBeDisabled()
    expect(screen.getByRole('switch', { name: 'Orders import' })).not.toBeDisabled()
    expect(screen.getByRole('switch', { name: 'Orders export' })).toBeDisabled()
    expect(screen.getByRole('switch', { name: 'Categories export' })).toBeDisabled()
    expect(screen.getAllByText('Coming soon — not supported yet.')).toHaveLength(1)
  })

  it('prefills fields from existing schedules', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    const existing: SyncSchedule = {
      entity_type: 'stock', direction: 'export', enabled: true, interval_minutes: 360,
      default_warehouse_id: null, default_currency: null, last_run_at: null, last_run_status: '', last_run_error: '',
    }
    vi.mocked(integrationsApi.listSyncSchedules).mockResolvedValue([existing])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await waitFor(() => expect(screen.getByRole('switch', { name: 'Stock levels export' })).toBeChecked())
    expect(screen.getByRole('combobox', { name: 'Stock levels export interval' })).toHaveValue('360')
  })

  it('offers intervals shorter than the old 15-minute minimum', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    const combobox = screen.getByRole('combobox', { name: 'Products import interval' })
    await userEvent.selectOptions(combobox, '1')
    expect(combobox).toHaveValue('1')
  })

  it('Save sends all rows (including disabled ones) via the bulk PUT', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.upsertSyncSchedules).mockResolvedValue([])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([{
      id: 'wh-1', tenant_id: 'tenant-1', name: 'Main DC',
      address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
      created_at: '', updated_at: '',
    }])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('switch', { name: 'Products import' }))
    await screen.findAllByText('Main DC')
    await userEvent.selectOptions(screen.getByLabelText('Default warehouse ID'), 'wh-1')
    await userEvent.selectOptions(screen.getByLabelText('Default currency (e.g. USD)'), 'USD')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertSyncSchedules).toHaveBeenCalledWith(
      'test-token', 'conn-1', expect.arrayContaining([
        expect.objectContaining({ entity_type: 'products', direction: 'import', enabled: true, default_warehouse_id: 'wh-1', default_currency: 'USD' }),
        expect.objectContaining({ entity_type: 'orders', enabled: false }),
        expect.objectContaining({ entity_type: 'categories', enabled: false }),
      ]),
    ))
  })

  it('enabling Orders import requires only a default warehouse, no currency', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.upsertSyncSchedules).mockResolvedValue([])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([{
      id: 'wh-1', tenant_id: 'tenant-1', name: 'Main DC',
      address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
      created_at: '', updated_at: '',
    }])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('switch', { name: 'Orders import' }))
    await screen.findAllByText('Main DC')
    await userEvent.selectOptions(screen.getByLabelText('Default warehouse ID'), 'wh-1')
    expect(screen.queryByLabelText('Default currency (e.g. USD)')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertSyncSchedules).toHaveBeenCalledWith(
      'test-token', 'conn-1', expect.arrayContaining([
        expect.objectContaining({ entity_type: 'orders', direction: 'import', enabled: true, default_warehouse_id: 'wh-1' }),
      ]),
    ))
  })

  it('shows an error if saving fails', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.upsertSyncSchedules).mockRejectedValue(new Error('server exploded'))

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('server exploded')).toBeInTheDocument()
  })

  it('Back returns to the Integrations overview', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('button', { name: 'Integrations' }))

    expect(await screen.findByRole('button', { name: 'Configure' })).toBeInTheDocument()
  })

  it('shows a Danger zone with a Delete imported products button', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByText('Danger zone')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete imported products' })).toBeInTheDocument()
  })

  it('requires typing DELETE before the confirm button is enabled', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported products' }))

    const confirmButton = await screen.findByRole('button', { name: 'Delete' })
    expect(confirmButton).toBeDisabled()

    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    expect(confirmButton).not.toBeDisabled()
  })

  it('confirming deletes and shows the result', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    const result: DeleteAllImportedProductsResult = { deleted_sales_orders: 0, deleted_products: 805, deleted_categories: 12, errors: [] }
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue(result)

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported products' }))
    await userEvent.type(await screen.findByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(integrationsApi.deleteAllImportedProducts).toHaveBeenCalledWith('test-token', 'conn-1'))
    expect(await screen.findByText(/805/)).toBeInTheDocument()
  })

  it('notifies other components (e.g. the Sidebar badge) that products changed', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue({ deleted_sales_orders: 0, deleted_products: 805, deleted_categories: 12, errors: [] })
    const listener = vi.fn()
    const unsubscribe = subscribeProductsChanged(listener)

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported products' }))
    await userEvent.type(await screen.findByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(listener).toHaveBeenCalled())
    unsubscribe()
  })

  it('does not crash when the backend sends errors: null instead of []', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    const result = { deleted_products: 805, deleted_categories: 12, errors: null } as unknown as DeleteAllImportedProductsResult
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue(result)

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported products' }))
    await userEvent.type(await screen.findByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(await screen.findByText(/805/)).toBeInTheDocument()
  })
})

describe('Store defaults currency', () => {
  it('shows a currency field prefilled from the connection', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: 'EUR' })])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('prefills the currency even when stored in a different case or with stray whitespace', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: ' eur ' })])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('falls back to the tenant-wide Localization currency when the connection has none of its own', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: '' })])
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('Save currency calls updateConnectionSettings and shows confirmation', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.updateConnectionSettings).mockResolvedValue(wooConnection({ default_currency: 'USD' }))

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.click(screen.getByRole('button', { name: 'Save currency' }))

    await waitFor(() => expect(integrationsApi.updateConnectionSettings).toHaveBeenCalledWith('test-token', 'conn-1', 'USD', null))
    expect(await screen.findByText('Saved.')).toBeInTheDocument()
  })

  it('shows an error if saving the currency fails', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.updateConnectionSettings).mockRejectedValue(new Error('currency save failed'))

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.click(screen.getByRole('button', { name: 'Save currency' }))

    expect(await screen.findByText('currency save failed')).toBeInTheDocument()
  })

  it('prefills the products/import row currency from the connection default', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: 'GBP' })])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('switch', { name: 'Products import' }))
    expect(await screen.findByLabelText('Default currency (e.g. USD)')).toHaveValue('GBP')
  })
})

describe('Store defaults location', () => {
  function warehouse(overrides: Partial<{ id: string; name: string }> = {}) {
    return {
      id: 'wh-1', tenant_id: 'tenant-1', name: 'Main DC',
      address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
      created_at: '', updated_at: '', ...overrides,
    }
  }

  it("auto-selects the tenant's only warehouse as the default location", async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse()])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await waitFor(() => expect(screen.getByLabelText('Default location')).toHaveValue('wh-1'))
  })

  it('leaves the default location unselected when there is more than one warehouse', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([
      warehouse({ id: 'wh-1', name: 'Main DC' }),
      warehouse({ id: 'wh-2', name: 'Overflow DC' }),
    ])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByLabelText('Default location')).toHaveValue('')
  })

  it('prefills the default location from the connection even when multiple warehouses exist', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_warehouse_id: 'wh-2' })])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([
      warehouse({ id: 'wh-1', name: 'Main DC' }),
      warehouse({ id: 'wh-2', name: 'Overflow DC' }),
    ])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByLabelText('Default location')).toHaveValue('wh-2')
  })

  it('Save currency also persists the chosen default location', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([
      warehouse({ id: 'wh-1', name: 'Main DC' }),
      warehouse({ id: 'wh-2', name: 'Overflow DC' }),
    ])
    vi.mocked(integrationsApi.updateConnectionSettings).mockResolvedValue(wooConnection({ default_warehouse_id: 'wh-2' }))

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.selectOptions(screen.getByLabelText('Default location'), 'wh-2')
    await userEvent.click(screen.getByRole('button', { name: 'Save currency' }))

    await waitFor(() => expect(integrationsApi.updateConnectionSettings).toHaveBeenCalledWith('test-token', 'conn-1', 'USD', 'wh-2'))
  })

  it("seeds a blank sync-schedule row's default warehouse from the connection default", async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_warehouse_id: 'wh-1' })])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse({ id: 'wh-1', name: 'Main DC' })])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('switch', { name: 'Products import' }))
    expect(await screen.findByLabelText('Default warehouse ID')).toHaveValue('wh-1')
  })

  it("keeps a row's own saved default warehouse instead of the connection default", async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_warehouse_id: 'wh-1' })])
    vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([
      warehouse({ id: 'wh-1', name: 'Main DC' }),
      warehouse({ id: 'wh-2', name: 'Overflow DC' }),
    ])
    vi.mocked(integrationsApi.listSyncSchedules).mockResolvedValue([{
      entity_type: 'products', direction: 'import', enabled: true, interval_minutes: 60,
      default_warehouse_id: 'wh-2', default_currency: 'USD', last_run_at: null, last_run_status: '', last_run_error: '',
    }])

    renderAt(<Settings />, '/settings')
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByLabelText('Default warehouse ID')).toHaveValue('wh-2')
  })
})

describe('Settings > Localization', () => {
  it('defaults to the first option in each dropdown when nothing is saved yet', async () => {
    renderAt(<Settings />, '/settings')
    await openLocalizationSection()

    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())
    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
    expect(screen.getByLabelText('Timezone')).toHaveValue('Europe/Amsterdam')
  })

  it('prefills fields from saved settings', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'USD', timezone: 'America/New_York', date_format: 'MM/DD/YYYY', language: 'fr',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0,
      updated_at: '2026-07-26T00:00:00Z',
    })

    renderAt(<Settings />, '/settings')
    await openLocalizationSection()

    await waitFor(() => expect(screen.getByLabelText('Default currency')).toHaveValue('USD'))
    expect(screen.getByLabelText('Timezone')).toHaveValue('America/New_York')
    expect(screen.getByLabelText('Date format')).toHaveValue('MM/DD/YYYY')
    expect(screen.getByLabelText('Language')).toHaveValue('fr')
  })

  it('Save Changes calls updateTenantSettings with all four fields and shows confirmation', async () => {
    vi.mocked(tenantSettingsApi.updateTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'USD', timezone: 'Europe/London', date_format: 'YYYY-MM-DD', language: 'nl',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0,
      updated_at: '2026-07-26T00:00:00Z',
    })

    renderAt(<Settings />, '/settings')
    await openLocalizationSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.selectOptions(screen.getByLabelText('Timezone'), 'Europe/London')
    await userEvent.selectOptions(screen.getByLabelText('Date format'), 'YYYY-MM-DD')
    await userEvent.selectOptions(screen.getByLabelText('Language'), 'nl')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => expect(tenantSettingsApi.updateTenantSettings).toHaveBeenCalledWith('test-token', {
      default_currency: 'USD', timezone: 'Europe/London', date_format: 'YYYY-MM-DD', language: 'nl',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0,
    }))
    expect(await screen.findByText('Saved.')).toBeInTheDocument()
  })

  it('Save Changes preserves already-saved tax settings instead of wiping them', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'NL',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '2026-07-26T00:00:00Z',
    })
    renderAt(<Settings />, '/settings')
    await openLocalizationSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => expect(tenantSettingsApi.updateTenantSettings).toHaveBeenCalledWith('test-token', expect.objectContaining({
      prices_include_tax: true, default_tax_country: 'NL',
    })))
  })

  it('shows an error if saving fails', async () => {
    vi.mocked(tenantSettingsApi.updateTenantSettings).mockRejectedValue(new Error('save failed'))

    renderAt(<Settings />, '/settings')
    await openLocalizationSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByText('save failed')).toBeInTheDocument()
  })
})

describe('Settings > Inventory Rules > Sync & expiry rules', () => {
  it('defaults to off / 0 days when nothing is saved yet', async () => {
    renderAt(<Settings />, '/settings')
    await openInventorySection()

    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())
    expect(screen.getByRole('switch', { name: 'Enable inventory sync rules' })).not.toBeChecked()
    expect(screen.getByLabelText('Auto-expire batches after (days)')).toHaveValue(0)
  })

  it('prefills from saved settings', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: true, batch_expiry_threshold_days: 30, default_target_markup_basis_points: 0, updated_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openInventorySection()

    await waitFor(() => expect(screen.getByRole('switch', { name: 'Enable inventory sync rules' })).toBeChecked())
    expect(screen.getByLabelText('Auto-expire batches after (days)')).toHaveValue(30)
  })

  it('Save calls updateTenantSettings with the new values and preserves other tenant settings fields', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'NL',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openInventorySection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.click(screen.getByRole('switch', { name: 'Enable inventory sync rules' }))
    await userEvent.clear(await screen.findByLabelText('Auto-expire batches after (days)'))
    await userEvent.type(screen.getByLabelText('Auto-expire batches after (days)'), '45')
    await userEvent.click(screen.getByRole('button', { name: 'Save inventory rules' }))

    await waitFor(() => expect(tenantSettingsApi.updateTenantSettings).toHaveBeenCalledWith('test-token', expect.objectContaining({
      default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'NL',
      inventory_sync_rules_enabled: true, batch_expiry_threshold_days: 45, default_target_markup_basis_points: 0,
    })))
  })

  it('shows an error if saving fails', async () => {
    vi.mocked(tenantSettingsApi.updateTenantSettings).mockRejectedValue(new Error('inventory rules save failed'))

    renderAt(<Settings />, '/settings')
    await openInventorySection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())
    await userEvent.click(screen.getByRole('button', { name: 'Save inventory rules' }))

    expect(await screen.findByText('inventory rules save failed')).toBeInTheDocument()
  })
})

describe('Settings > Tax', () => {
  function taxClass(overrides: Partial<import('../../tax/types').TaxClass> = {}): import('../../tax/types').TaxClass {
    return { id: 'class-1', name: 'Standard', is_default: true, created_at: '', updated_at: '', ...overrides }
  }
  function taxRate(overrides: Partial<import('../../tax/types').TaxRate> = {}): import('../../tax/types').TaxRate {
    return { id: 'rate-1', tax_class_id: 'class-1', country: 'NL', rate_basis_points: 2100, created_at: '', updated_at: '', ...overrides }
  }

  it('shows store tax defaults prefilled from tenant settings', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '',
      prices_include_tax: true, default_tax_country: 'NL',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openTaxSection()

    await waitFor(() => expect(screen.getByRole('switch', { name: 'Prices include tax' })).toBeChecked())
    expect(screen.getByLabelText('Default tax country')).toHaveValue('NL')
  })

  it('saving store tax defaults preserves the other tenant settings fields', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })
    vi.mocked(tenantSettingsApi.updateTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'US',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openTaxSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.click(screen.getByRole('switch', { name: 'Prices include tax' }))
    await userEvent.clear(await screen.findByLabelText('Default tax country'))
    await userEvent.type(screen.getByLabelText('Default tax country'), 'us')
    await userEvent.click(screen.getByRole('button', { name: 'Save tax defaults' }))

    await waitFor(() => expect(tenantSettingsApi.updateTenantSettings).toHaveBeenCalledWith('test-token', expect.objectContaining({
      default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'US',
    })))
  })

  it('lists tax classes with a default badge', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])

    renderAt(<Settings />, '/settings')
    await openTaxSection()

    expect((await screen.findAllByText('Standard')).length).toBeGreaterThan(0)
    expect(screen.getByText('Default')).toBeInTheDocument()
  })

  it('Add tax class creates it via the API and shows it in the list', async () => {
    vi.mocked(taxApi.createTaxClass).mockResolvedValue(taxClass({ id: 'class-2', name: 'Reduced rate', is_default: false }))

    renderAt(<Settings />, '/settings')
    await openTaxSection()
    await screen.findByText('Tax classes')

    await userEvent.click(screen.getByRole('button', { name: 'Add tax class' }))
    await userEvent.type(await screen.findByLabelText('Name'), 'Reduced rate')
    await userEvent.click(screen.getByRole('button', { name: 'Add class' }))

    await waitFor(() => expect(taxApi.createTaxClass).toHaveBeenCalledWith('test-token', { name: 'Reduced rate', is_default: false }))
    expect((await screen.findAllByText('Reduced rate')).length).toBeGreaterThan(0)
  })

  it('Edit tax class prefills the form and saves changes', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(taxApi.updateTaxClass).mockResolvedValue(taxClass({ name: 'Standard rate' }))

    renderAt(<Settings />, '/settings')
    await openTaxSection()
    await screen.findAllByText('Standard')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    expect(await screen.findByLabelText('Name')).toHaveValue('Standard')

    await userEvent.clear(screen.getByLabelText('Name'))
    await userEvent.type(screen.getByLabelText('Name'), 'Standard rate')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => expect(taxApi.updateTaxClass).toHaveBeenCalledWith('test-token', 'class-1', { name: 'Standard rate', is_default: true }))
  })

  it('Delete tax class calls the API after confirming', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass({ is_default: false })])

    renderAt(<Settings />, '/settings')
    await openTaxSection()
    await screen.findAllByText('Standard')

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await acceptConfirm()
    await waitFor(() => expect(taxApi.deleteTaxClass).toHaveBeenCalledWith('test-token', 'class-1'))
  })

  it('lists tax rates under their class with the computed percent', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(taxApi.listTaxRates).mockResolvedValue([taxRate()])

    renderAt(<Settings />, '/settings')
    await openTaxSection()

    expect(await screen.findByText('NL')).toBeInTheDocument()
    expect(screen.getByText('21.00%')).toBeInTheDocument()
  })

  it('Add tax rate converts the entered percent to basis points', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(taxApi.upsertTaxRate).mockResolvedValue(taxRate({ id: 'rate-2', country: 'US', rate_basis_points: 850 }))

    renderAt(<Settings />, '/settings')
    await openTaxSection()
    await screen.findAllByText('Standard')

    await userEvent.type(screen.getByLabelText('Country'), 'us')
    await userEvent.type(screen.getByLabelText('Rate %'), '8.5')
    await userEvent.click(screen.getByRole('button', { name: 'Add rate' }))

    await waitFor(() => expect(taxApi.upsertTaxRate).toHaveBeenCalledWith('test-token', {
      tax_class_id: 'class-1', country: 'US', rate_basis_points: 850,
    }))
  })

  it('Delete tax rate calls the API', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(taxApi.listTaxRates).mockResolvedValue([taxRate()])

    renderAt(<Settings />, '/settings')
    await openTaxSection()
    await screen.findByText('NL')

    await userEvent.click(screen.getByRole('button', { name: 'Remove rate' }))

    await waitFor(() => expect(taxApi.deleteTaxRate).toHaveBeenCalledWith('test-token', 'rate-1'))
  })
})

describe('Settings > Pricing', () => {
  it('has its own nav item, separate from Tax', async () => {
    renderAt(<Settings />, '/settings')
    await openPricingSection()

    expect(await screen.findByLabelText('Default target markup (%)')).toBeInTheDocument()

    await openTaxSection()
    expect(screen.queryByLabelText('Default target markup (%)')).not.toBeInTheDocument()
  })

  it('shows the default target markup prefilled from tenant settings', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '',
      prices_include_tax: true, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 3000, updated_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openPricingSection()

    await waitFor(() => expect(screen.getByLabelText('Default target markup (%)')).toHaveValue(30))
  })

  it('saves the default target markup, preserving other tenant settings fields', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'NL',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })
    vi.mocked(tenantSettingsApi.updateTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'NL',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 2500, updated_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openPricingSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.type(screen.getByLabelText('Default target markup (%)'), '25')
    await userEvent.click(screen.getByRole('button', { name: 'Save pricing settings' }))

    await waitFor(() => expect(tenantSettingsApi.updateTenantSettings).toHaveBeenCalledWith('test-token', expect.objectContaining({
      default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'NL',
      default_target_markup_basis_points: 2500,
    })))
  })

  it('rejects a negative target markup client-side instead of round-tripping to the 422 the backend would return', async () => {
    renderAt(<Settings />, '/settings')
    await openPricingSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.type(screen.getByLabelText('Default target markup (%)'), '-5')
    await userEvent.click(screen.getByRole('button', { name: 'Save pricing settings' }))

    expect(await screen.findByText(/cannot be negative/i)).toBeInTheDocument()
    expect(tenantSettingsApi.updateTenantSettings).not.toHaveBeenCalled()
  })

  it('accepts a 100% (or higher) target markup, unlike margin which was capped under 100%', async () => {
    vi.mocked(tenantSettingsApi.updateTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 10000, updated_at: '',
    })

    renderAt(<Settings />, '/settings')
    await openPricingSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.type(screen.getByLabelText('Default target markup (%)'), '100')
    await userEvent.click(screen.getByRole('button', { name: 'Save pricing settings' }))

    await waitFor(() => expect(tenantSettingsApi.updateTenantSettings).toHaveBeenCalledWith('test-token', expect.objectContaining({
      default_target_markup_basis_points: 10000,
    })))
  })

  it('does not show the "Configuration coming soon" placeholder on the Pricing tab', async () => {
    renderAt(<Settings />, '/settings')
    await openPricingSection()

    expect(await screen.findByLabelText('Default target markup (%)')).toBeInTheDocument()
    expect(screen.queryByText('Configuration coming soon')).not.toBeInTheDocument()
  })

  it('does not show the "Configuration coming soon" placeholder on the Tax tab', async () => {
    renderAt(<Settings />, '/settings')
    await openTaxSection()

    expect(screen.queryByText('Configuration coming soon')).not.toBeInTheDocument()
  })
})

describe('Settings URLs (refresh + Back)', () => {
  function taxClass(overrides: Partial<import('../../tax/types').TaxClass> = {}): import('../../tax/types').TaxClass {
    return { id: 'class-1', name: 'Standard', is_default: true, created_at: '', updated_at: '', ...overrides }
  }

  it('each section has its own URL, and Back returns to the previous one', async () => {
    const { router } = renderAt(<Settings />, '/settings')
    await userEvent.click(await screen.findByText('Tax'))
    expect(router.state.location.pathname).toBe('/settings/tax')
    expect(await screen.findByText('Tax classes')).toBeInTheDocument()

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/settings')
  })

  it('/settings/:section opens that section on load', async () => {
    renderAt(<Settings />, '/settings', '/settings/tax')
    expect(await screen.findByText('Tax classes')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Tax' })).toHaveAttribute('aria-current', 'page')
  })

  it('no section shows a "coming soon" placeholder', async () => {
    for (const section of ['fulfillment', 'notifications', 'security', 'automation']) {
      const { unmount } = renderAt(<Settings />, '/settings', `/settings/${section}`)
      await screen.findAllByRole('heading')
      expect(screen.queryByText('Configuration coming soon'), section).not.toBeInTheDocument()
      unmount()
    }
  })

  it('an unknown section falls back to Company', async () => {
    renderAt(<Settings />, '/settings', '/settings/nope')
    expect(await screen.findByRole('heading', { name: 'Company' })).toBeInTheDocument()
  })

  it('Configure pushes /settings/integrations/:connectionId, which also works on load', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    const { router, unmount } = renderAt(<Settings />, '/settings', '/settings/integrations')
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    expect(router.state.location.pathname).toBe(`/settings/integrations/${wooConnection().id}`)
    unmount()

    renderAt(<Settings />, '/settings', `/settings/integrations/${wooConnection().id}`)
    expect(await screen.findByText('Sync schedule')).toBeInTheDocument()
  })

  it('the connection settings tab lives in ?tab=', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    const { router } = renderAt(<Settings />, '/settings', `/settings/integrations/${wooConnection().id}?tab=categories`)
    await screen.findByRole('heading', { name: 'https://mystore.com' })
    expect(screen.queryByText('Sync schedule')).not.toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Taxes' }))
    expect(router.state.location.search).toBe('?tab=taxes')
  })

  it('an unknown connection id shows not-found', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    renderAt(<Settings />, '/settings', '/settings/integrations/nope')
    expect(await screen.findByText('Connection not found')).toBeInTheDocument()
  })

  it('tax class editor URLs: /settings/tax/classes/new and /:id', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    const { router, unmount } = renderAt(<Settings />, '/settings', '/settings/tax')
    await screen.findAllByText('Standard')
    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    expect(router.state.location.pathname).toBe('/settings/tax/classes/class-1')
    unmount()

    renderAt(<Settings />, '/settings', '/settings/tax/classes/class-1')
    await waitFor(() => expect(screen.getByLabelText('Name')).toHaveValue('Standard'))
  })

  it('/settings/tax/classes/new opens the add form on load', async () => {
    renderAt(<Settings />, '/settings', '/settings/tax/classes/new')
    expect(await screen.findByRole('button', { name: 'Add class' })).toBeInTheDocument()
  })
})
