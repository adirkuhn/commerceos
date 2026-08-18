// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Settings from '../Settings'
import type { Connection, SyncSchedule, DeleteAllImportedProductsResult } from '../../integrations/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../integrations/api', () => ({
  listConnections: vi.fn(),
  listSyncSchedules: vi.fn(),
  upsertSyncSchedules: vi.fn(),
  deleteAllImportedProducts: vi.fn(),
  updateConnectionSettings: vi.fn(),
  getAIConnection: vi.fn(),
  connectAIProvider: vi.fn(),
  disconnectAIProvider: vi.fn(),
  listWooTaxes: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
  createWarehouse: vi.fn(),
  updateWarehouse: vi.fn(),
  deleteWarehouse: vi.fn(),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn(),
  updateTenantSettings: vi.fn(),
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
import { subscribeProductsChanged } from '../../pms/productsBus'

function wooConnection(overrides: Partial<Connection> = {}): Connection {
  return { id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '', ...overrides }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listSyncSchedules).mockResolvedValue([])
  vi.mocked(integrationsApi.getAIConnection).mockResolvedValue(null)
  vi.mocked(integrationsApi.listWooTaxes).mockResolvedValue([])
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
  vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '',
    prices_include_tax: false, default_tax_country: '',
    inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, updated_at: '',
  })
  vi.mocked(taxApi.listTaxClasses).mockResolvedValue([])
  vi.mocked(taxApi.listTaxRates).mockResolvedValue([])
  vi.mocked(taxApi.listExternalTaxClassLinks).mockResolvedValue([])
})

async function openLocalizationSection() {
  await userEvent.click(await screen.findByText('Localization'))
}

async function openTaxSection() {
  await userEvent.click(await screen.findByText('Tax'))
}

async function openIntegrationsSection() {
  await userEvent.click(await screen.findByText('Integrations'))
}

async function openInventorySection() {
  await userEvent.click(await screen.findByText('Inventory Rules'))
}

describe('Settings > Integrations overview', () => {
  it('shows an empty state when there are no connections', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])

    render(<Settings />)
    await openIntegrationsSection()

    expect(await screen.findByText('No integrations connected yet.')).toBeInTheDocument()
  })

  it('lists connections with status and a Configure button for WooCommerce', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    render(<Settings />)
    await openIntegrationsSection()

    expect(await screen.findByText('https://mystore.com')).toBeInTheDocument()
    expect(screen.getByText('connected')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Configure' })).toBeInTheDocument()
  })

  it('clicking Configure opens the WooCommerce sync settings page', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))

    expect(await screen.findByText('Sync schedule')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'https://mystore.com' })).toBeInTheDocument()
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
    render(<Settings />)
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

describe('Settings > Integrations > AI', () => {
  it('shows not configured and a Connect button when no AI provider is set up', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])

    render(<Settings />)
    await openIntegrationsSection()

    expect(await screen.findByText(/Not configured/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Connect' })).toBeInTheDocument()
  })

  it('shows the configured provider and status', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    vi.mocked(integrationsApi.getAIConnection).mockResolvedValue({
      id: 'ai-conn-1', provider: 'anthropic', status: 'connected', created_at: '',
    })

    render(<Settings />)
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

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Connect' }))
    await userEvent.type(screen.getByLabelText('API Key'), 'sk-test-key')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.connectAIProvider).toHaveBeenCalledWith('test-token', {
      provider: 'openai', apiKey: 'sk-test-key', baseUrl: '', model: '',
    }))
  })

  it('requires a server URL instead of an API key for Ollama', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Connect' }))
    await userEvent.selectOptions(screen.getByLabelText('Provider'), 'ollama')

    expect(screen.queryByLabelText('API Key')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Server URL')).toBeInTheDocument()
  })

  it('shows an error if connecting fails', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    vi.mocked(integrationsApi.connectAIProvider).mockRejectedValue(new Error('an API key is required for this provider'))

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Connect' }))
    await userEvent.type(screen.getByLabelText('API Key'), 'sk-test')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('an API key is required for this provider')).toBeInTheDocument()
  })

  it('disconnects after confirming', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    vi.mocked(integrationsApi.getAIConnection).mockResolvedValue({
      id: 'ai-conn-1', provider: 'gemini', status: 'connected', created_at: '',
    })
    vi.mocked(integrationsApi.disconnectAIProvider).mockResolvedValue(null)
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Disconnect' }))

    await waitFor(() => expect(integrationsApi.disconnectAIProvider).toHaveBeenCalledWith('test-token'))
  })
})

describe('WooCommerce sync settings page', () => {
  it('shows Products, Stock levels, and Orders import as configurable, Categories and Orders export as disabled', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    render(<Settings />)
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

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await waitFor(() => expect(screen.getByRole('switch', { name: 'Stock levels export' })).toBeChecked())
    expect(screen.getByRole('combobox', { name: 'Stock levels export interval' })).toHaveValue('360')
  })

  it('offers intervals shorter than the old 15-minute minimum', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    render(<Settings />)
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

    render(<Settings />)
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

    render(<Settings />)
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

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    expect(await screen.findByText('server exploded')).toBeInTheDocument()
  })

  it('Back returns to the Integrations overview', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('button', { name: 'Integrations' }))

    expect(await screen.findByRole('button', { name: 'Configure' })).toBeInTheDocument()
  })

  it('shows a Danger zone with a Delete imported products button', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByText('Danger zone')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Delete imported products' })).toBeInTheDocument()
  })

  it('requires typing DELETE before the confirm button is enabled', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported products' }))

    const confirmButton = screen.getByRole('button', { name: 'Delete' })
    expect(confirmButton).toBeDisabled()

    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    expect(confirmButton).not.toBeDisabled()
  })

  it('confirming deletes and shows the result', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    const result: DeleteAllImportedProductsResult = { deleted_sales_orders: 0, deleted_products: 805, deleted_categories: 12, errors: [] }
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue(result)

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported products' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(integrationsApi.deleteAllImportedProducts).toHaveBeenCalledWith('test-token', 'conn-1'))
    expect(await screen.findByText(/805/)).toBeInTheDocument()
  })

  it('notifies other components (e.g. the Sidebar badge) that products changed', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue({ deleted_sales_orders: 0, deleted_products: 805, deleted_categories: 12, errors: [] })
    const listener = vi.fn()
    const unsubscribe = subscribeProductsChanged(listener)

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported products' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(listener).toHaveBeenCalled())
    unsubscribe()
  })

  it('does not crash when the backend sends errors: null instead of []', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    const result = { deleted_products: 805, deleted_categories: 12, errors: null } as unknown as DeleteAllImportedProductsResult
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue(result)

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported products' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(await screen.findByText(/805/)).toBeInTheDocument()
  })
})

describe('Store defaults currency', () => {
  it('shows a currency field prefilled from the connection', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: 'EUR' })])

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('prefills the currency even when stored in a different case or with stray whitespace', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: ' eur ' })])

    render(<Settings />)
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
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, updated_at: '',
    })

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('Save currency calls updateConnectionSettings and shows confirmation', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
    vi.mocked(integrationsApi.updateConnectionSettings).mockResolvedValue(wooConnection({ default_currency: 'USD' }))

    render(<Settings />)
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

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.click(screen.getByRole('button', { name: 'Save currency' }))

    expect(await screen.findByText('currency save failed')).toBeInTheDocument()
  })

  it('prefills the products/import row currency from the connection default', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: 'GBP' })])

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('switch', { name: 'Products import' }))
    expect(screen.getByLabelText('Default currency (e.g. USD)')).toHaveValue('GBP')
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

    render(<Settings />)
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

    render(<Settings />)
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

    render(<Settings />)
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

    render(<Settings />)
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

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('switch', { name: 'Products import' }))
    expect(screen.getByLabelText('Default warehouse ID')).toHaveValue('wh-1')
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

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByLabelText('Default warehouse ID')).toHaveValue('wh-2')
  })
})

describe('Settings > Localization', () => {
  it('defaults to the first option in each dropdown when nothing is saved yet', async () => {
    render(<Settings />)
    await openLocalizationSection()

    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())
    expect(screen.getByLabelText('Default Currency')).toHaveValue('EUR')
    expect(screen.getByLabelText('Timezone')).toHaveValue('Europe/Amsterdam')
  })

  it('prefills fields from saved settings', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'USD', timezone: 'America/New_York', date_format: 'MM/DD/YYYY', language: 'fr',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0,
      updated_at: '2026-07-26T00:00:00Z',
    })

    render(<Settings />)
    await openLocalizationSection()

    await waitFor(() => expect(screen.getByLabelText('Default Currency')).toHaveValue('USD'))
    expect(screen.getByLabelText('Timezone')).toHaveValue('America/New_York')
    expect(screen.getByLabelText('Date Format')).toHaveValue('MM/DD/YYYY')
    expect(screen.getByLabelText('Language')).toHaveValue('fr')
  })

  it('Save Changes calls updateTenantSettings with all four fields and shows confirmation', async () => {
    vi.mocked(tenantSettingsApi.updateTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'USD', timezone: 'Europe/London', date_format: 'YYYY-MM-DD', language: 'nl',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0,
      updated_at: '2026-07-26T00:00:00Z',
    })

    render(<Settings />)
    await openLocalizationSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.selectOptions(screen.getByLabelText('Default Currency'), 'USD')
    await userEvent.selectOptions(screen.getByLabelText('Timezone'), 'Europe/London')
    await userEvent.selectOptions(screen.getByLabelText('Date Format'), 'YYYY-MM-DD')
    await userEvent.selectOptions(screen.getByLabelText('Language'), 'nl')
    await userEvent.click(screen.getByRole('button', { name: 'Save Changes' }))

    await waitFor(() => expect(tenantSettingsApi.updateTenantSettings).toHaveBeenCalledWith('test-token', {
      default_currency: 'USD', timezone: 'Europe/London', date_format: 'YYYY-MM-DD', language: 'nl',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0,
    }))
    expect(await screen.findByText('Saved.')).toBeInTheDocument()
  })

  it('Save Changes preserves already-saved tax settings instead of wiping them', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'NL',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, updated_at: '2026-07-26T00:00:00Z',
    })
    render(<Settings />)
    await openLocalizationSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.click(screen.getByRole('button', { name: 'Save Changes' }))

    await waitFor(() => expect(tenantSettingsApi.updateTenantSettings).toHaveBeenCalledWith('test-token', expect.objectContaining({
      prices_include_tax: true, default_tax_country: 'NL',
    })))
  })

  it('shows an error if saving fails', async () => {
    vi.mocked(tenantSettingsApi.updateTenantSettings).mockRejectedValue(new Error('save failed'))

    render(<Settings />)
    await openLocalizationSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())
    await userEvent.click(screen.getByRole('button', { name: 'Save Changes' }))

    expect(await screen.findByText('save failed')).toBeInTheDocument()
  })
})

describe('Settings > Inventory Rules > Sync & expiry rules', () => {
  it('defaults to off / 0 days when nothing is saved yet', async () => {
    render(<Settings />)
    await openInventorySection()

    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())
    expect(screen.getByRole('switch', { name: 'Enable inventory sync rules' })).not.toBeChecked()
    expect(screen.getByLabelText('Auto-expire batches after (days)')).toHaveValue(0)
  })

  it('prefills from saved settings', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: true, batch_expiry_threshold_days: 30, updated_at: '',
    })

    render(<Settings />)
    await openInventorySection()

    await waitFor(() => expect(screen.getByRole('switch', { name: 'Enable inventory sync rules' })).toBeChecked())
    expect(screen.getByLabelText('Auto-expire batches after (days)')).toHaveValue(30)
  })

  it('Save calls updateTenantSettings with the new values and preserves other tenant settings fields', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'NL',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, updated_at: '',
    })

    render(<Settings />)
    await openInventorySection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.click(screen.getByRole('switch', { name: 'Enable inventory sync rules' }))
    await userEvent.clear(screen.getByLabelText('Auto-expire batches after (days)'))
    await userEvent.type(screen.getByLabelText('Auto-expire batches after (days)'), '45')
    await userEvent.click(screen.getByRole('button', { name: 'Save inventory rules' }))

    await waitFor(() => expect(tenantSettingsApi.updateTenantSettings).toHaveBeenCalledWith('test-token', expect.objectContaining({
      default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'NL',
      inventory_sync_rules_enabled: true, batch_expiry_threshold_days: 45,
    })))
  })

  it('shows an error if saving fails', async () => {
    vi.mocked(tenantSettingsApi.updateTenantSettings).mockRejectedValue(new Error('inventory rules save failed'))

    render(<Settings />)
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
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, updated_at: '',
    })

    render(<Settings />)
    await openTaxSection()

    await waitFor(() => expect(screen.getByRole('switch', { name: 'Prices include tax' })).toBeChecked())
    expect(screen.getByLabelText('Default tax country')).toHaveValue('NL')
  })

  it('saving store tax defaults preserves the other tenant settings fields', async () => {
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: false, default_tax_country: '',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, updated_at: '',
    })
    vi.mocked(tenantSettingsApi.updateTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'US',
      inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, updated_at: '',
    })

    render(<Settings />)
    await openTaxSection()
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())

    await userEvent.click(screen.getByRole('switch', { name: 'Prices include tax' }))
    await userEvent.clear(screen.getByLabelText('Default tax country'))
    await userEvent.type(screen.getByLabelText('Default tax country'), 'us')
    await userEvent.click(screen.getByRole('button', { name: 'Save tax defaults' }))

    await waitFor(() => expect(tenantSettingsApi.updateTenantSettings).toHaveBeenCalledWith('test-token', expect.objectContaining({
      default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
      prices_include_tax: true, default_tax_country: 'US',
    })))
  })

  it('lists tax classes with a default badge', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])

    render(<Settings />)
    await openTaxSection()

    expect((await screen.findAllByText('Standard')).length).toBeGreaterThan(0)
    expect(screen.getByText('Default')).toBeInTheDocument()
  })

  it('Add tax class creates it via the API and shows it in the list', async () => {
    vi.mocked(taxApi.createTaxClass).mockResolvedValue(taxClass({ id: 'class-2', name: 'Reduced rate', is_default: false }))

    render(<Settings />)
    await openTaxSection()
    await screen.findByText('Tax classes')

    await userEvent.click(screen.getByRole('button', { name: 'Add tax class' }))
    await userEvent.type(screen.getByLabelText('Name'), 'Reduced rate')
    await userEvent.click(screen.getByRole('button', { name: 'Add class' }))

    await waitFor(() => expect(taxApi.createTaxClass).toHaveBeenCalledWith('test-token', { name: 'Reduced rate', is_default: false }))
    expect((await screen.findAllByText('Reduced rate')).length).toBeGreaterThan(0)
  })

  it('Edit tax class prefills the form and saves changes', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(taxApi.updateTaxClass).mockResolvedValue(taxClass({ name: 'Standard rate' }))

    render(<Settings />)
    await openTaxSection()
    await screen.findAllByText('Standard')

    await userEvent.click(screen.getByRole('button', { name: 'Edit' }))
    expect(screen.getByLabelText('Name')).toHaveValue('Standard')

    await userEvent.clear(screen.getByLabelText('Name'))
    await userEvent.type(screen.getByLabelText('Name'), 'Standard rate')
    await userEvent.click(screen.getByRole('button', { name: 'Save changes' }))

    await waitFor(() => expect(taxApi.updateTaxClass).toHaveBeenCalledWith('test-token', 'class-1', { name: 'Standard rate', is_default: true }))
  })

  it('Delete tax class calls the API after confirming', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass({ is_default: false })])
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    render(<Settings />)
    await openTaxSection()
    await screen.findAllByText('Standard')

    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(taxApi.deleteTaxClass).toHaveBeenCalledWith('test-token', 'class-1'))
  })

  it('lists tax rates under their class with the computed percent', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(taxApi.listTaxRates).mockResolvedValue([taxRate()])

    render(<Settings />)
    await openTaxSection()

    expect(await screen.findByText('NL')).toBeInTheDocument()
    expect(screen.getByText('21.00%')).toBeInTheDocument()
  })

  it('Add tax rate converts the entered percent to basis points', async () => {
    vi.mocked(taxApi.listTaxClasses).mockResolvedValue([taxClass()])
    vi.mocked(taxApi.upsertTaxRate).mockResolvedValue(taxRate({ id: 'rate-2', country: 'US', rate_basis_points: 850 }))

    render(<Settings />)
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

    render(<Settings />)
    await openTaxSection()
    await screen.findByText('NL')

    await userEvent.click(screen.getByRole('button', { name: 'Remove rate' }))

    await waitFor(() => expect(taxApi.deleteTaxRate).toHaveBeenCalledWith('test-token', 'rate-1'))
  })
})
