// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
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

import * as integrationsApi from '../../integrations/api'
import * as inventoryApi from '../../inventory/api'
import * as tenantSettingsApi from '../../settings/api'
import { subscribeProductsChanged } from '../../pms/productsBus'

function wooConnection(overrides: Partial<Connection> = {}): Connection {
  return { id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '', ...overrides }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listSyncSchedules).mockResolvedValue([])
  vi.mocked(integrationsApi.getAIConnection).mockResolvedValue(null)
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
  vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '', updated_at: '',
  })
})

async function openLocalizationSection() {
  await userEvent.click(await screen.findByText('Localization'))
}

async function openIntegrationsSection() {
  await userEvent.click(await screen.findByText('Integrations'))
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
  it('shows Products and Stock levels as configurable, Orders and Categories as disabled', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])

    render(<Settings />)
    await openIntegrationsSection()
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    expect(screen.getByRole('switch', { name: 'Products import' })).not.toBeDisabled()
    expect(screen.getByRole('switch', { name: 'Stock levels export' })).not.toBeDisabled()
    expect(screen.getByRole('switch', { name: 'Orders import' })).toBeDisabled()
    expect(screen.getByRole('switch', { name: 'Categories export' })).toBeDisabled()
    expect(screen.getAllByText('Coming soon — not supported yet.')).toHaveLength(2)
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
    await screen.findByText('Main DC')
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
    const result: DeleteAllImportedProductsResult = { deleted_products: 805, deleted_categories: 12, errors: [] }
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
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue({ deleted_products: 805, deleted_categories: 12, errors: [] })
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
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '', updated_at: '',
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

    await waitFor(() => expect(integrationsApi.updateConnectionSettings).toHaveBeenCalledWith('test-token', 'conn-1', 'USD'))
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
    }))
    expect(await screen.findByText('Saved.')).toBeInTheDocument()
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
