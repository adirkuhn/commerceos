// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { AppProvider } from '../../context/AppContext'
import Integrations from '../Integrations'
import type { Connection, ImportJob, DeleteAllImportedProductsResult } from '../../integrations/types'
import type { Warehouse } from '../../inventory/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../integrations/api', () => ({
  listConnections: vi.fn(),
  testConnection: vi.fn(),
  disconnectConnection: vi.fn(),
  startWooConnect: vi.fn(),
  manualConnectWooCommerce: vi.fn(),
  importWooProducts: vi.fn(),
  deleteAllImportedProducts: vi.fn(),
  listSyncSchedules: vi.fn(),
  upsertSyncSchedules: vi.fn(),
  updateConnectionSettings: vi.fn(),
  listWooTaxes: vi.fn(),
  connectWhatsapp: vi.fn(),
  connectStripe: vi.fn(),
  connectRevolut: vi.fn(),
  connectEmail: vi.fn(),
  setWebhookSecret: vi.fn(),
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

// Real MyParcelSyncSettingsPage needs its own heavy set of mocks (shipping
// api, extra integrations api functions) covered exhaustively in its own
// test file -- here we only care whether Integrations.tsx passes the right
// props when opening it, so a lightweight stub that just echoes them back.
vi.mock('../../shipping/MyParcelSyncSettingsPage', () => ({
  default: ({ focusWebhooks }: { focusWebhooks?: boolean }) => (
    <div>MyParcel settings (focusWebhooks: {String(!!focusWebhooks)})</div>
  ),
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
import { subscribeNotificationsChanged } from '../../notifications/notificationsBus'

function wooConnection(overrides: Partial<Connection> = {}): Connection {
  return { id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '', ...overrides }
}

function bolConnection(overrides: Partial<Connection> = {}): Connection {
  return { id: 'conn-bol-1', provider: 'bol', store_url: '', status: 'connected', created_at: '', ...overrides }
}

function myParcelConnection(overrides: Partial<Connection> = {}): Connection {
  return { id: 'conn-mp-1', provider: 'myparcel', store_url: '', status: 'connected', created_at: '', ...overrides }
}

const sampleWarehouse: Warehouse = {
  id: 'wh-1', tenant_id: 'tenant-1', name: 'Main DC',
  address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
  created_at: '', updated_at: '',
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([sampleWarehouse])
  vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '',
    prices_include_tax: false, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
  })
  vi.mocked(integrationsApi.listSyncSchedules).mockResolvedValue([])
  vi.mocked(integrationsApi.listWooTaxes).mockResolvedValue([])
  vi.mocked(taxApi.listTaxClasses).mockResolvedValue([])
  vi.mocked(taxApi.listExternalTaxClassLinks).mockResolvedValue([])
})

describe('Configure connection', () => {
  it('shows a Configure button for WooCommerce connections', async () => {
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)

    expect(await screen.findByRole('button', { name: 'Configure' })).toBeInTheDocument()
  })

  it('clicking Configure opens the WooCommerce sync settings page', async () => {
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)

    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))

    expect(await screen.findByText('Sync schedule')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'https://mystore.com' })).toBeInTheDocument()
  })

  it('Back from the sync settings page returns to the connections grid', async () => {
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('button', { name: /Integrations/ }))

    expect(await screen.findByText('Connected integrations')).toBeInTheDocument()
  })

  it('shows a Configure button for Bol connections and opens the Bol sync settings page', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([bolConnection()])
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)

    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))

    expect(await screen.findByText('Sync schedule')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'Bol.com' })).toBeInTheDocument()
    expect(integrationsApi.listSyncSchedules).toHaveBeenCalledWith('test-token', 'conn-bol-1')
  })

  it('shows a Register webhook button for MyParcel connections that opens the settings page focused on Webhooks', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([myParcelConnection()])
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)

    await userEvent.click(await screen.findByRole('button', { name: 'Register webhook' }))

    expect(await screen.findByText('MyParcel settings (focusWebhooks: true)')).toBeInTheDocument()
  })

  it('MyParcel Configure opens the settings page without focusing Webhooks', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([myParcelConnection()])
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)

    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))

    expect(await screen.findByText('MyParcel settings (focusWebhooks: false)')).toBeInTheDocument()
  })
})

describe('Import products', () => {
  it('clicking Import opens a modal asking for warehouse and currency', async () => {
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))

    expect(screen.getByText('Import products')).toBeInTheDocument()
    expect(screen.getByLabelText('Default warehouse ID')).toBeInTheDocument()
    expect(screen.getByLabelText('Default currency')).toBeInTheDocument()
  })

  it('prefills the currency from the connection default', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: 'EUR' })])

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('prefills the currency even when stored in a different case or with stray whitespace', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: ' eur ' })])

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('falls back to the tenant-wide Localization currency when the connection has none of its own', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: '' })])
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '',
      prices_include_tax: false, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('prefers the connection-level currency over the tenant-wide default when both are set', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: 'GBP' })])
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '',
      prices_include_tax: false, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
    })

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))

    expect(screen.getByLabelText('Default currency')).toHaveValue('GBP')
  })

  it('submitting calls importWooProducts and shows a background-job confirmation, not a blocking result', async () => {
    const job: ImportJob = {
      id: 'job-1', connection_id: 'conn-1', status: 'pending', imported: 0, already_linked: 0, skipped_type: 0,
      errors: [], error_message: '', created_at: '2026-08-03T00:00:00Z', started_at: null, finished_at: null,
    }
    vi.mocked(integrationsApi.importWooProducts).mockResolvedValue(job)

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))
    await screen.findByText('Main DC')
    await userEvent.selectOptions(screen.getByLabelText('Default warehouse ID'), 'wh-1')
    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.click(screen.getByRole('button', { name: 'Start import' }))

    await waitFor(() => expect(integrationsApi.importWooProducts).toHaveBeenCalledWith('test-token', 'conn-1', 'wh-1', 'USD'))
    expect(await screen.findByText(/running in the background/)).toBeInTheDocument()
  })

  it('notifies the notification bell that a notification was created (job start)', async () => {
    const job: ImportJob = {
      id: 'job-1', connection_id: 'conn-1', status: 'pending', imported: 0, already_linked: 0, skipped_type: 0,
      errors: [], error_message: '', created_at: '2026-08-03T00:00:00Z', started_at: null, finished_at: null,
    }
    vi.mocked(integrationsApi.importWooProducts).mockResolvedValue(job)
    const listener = vi.fn()
    const unsubscribe = subscribeNotificationsChanged(listener)

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))
    await screen.findByText('Main DC')
    await userEvent.selectOptions(screen.getByLabelText('Default warehouse ID'), 'wh-1')
    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.click(screen.getByRole('button', { name: 'Start import' }))

    await waitFor(() => expect(listener).toHaveBeenCalled())
    unsubscribe()
  })

  it('shows an error if the connection already has an import running (409)', async () => {
    vi.mocked(integrationsApi.importWooProducts).mockRejectedValue(new Error('an import is already running for this connection'))

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))
    await screen.findByText('Main DC')
    await userEvent.selectOptions(screen.getByLabelText('Default warehouse ID'), 'wh-1')
    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.click(screen.getByRole('button', { name: 'Start import' }))

    expect(await screen.findByText('an import is already running for this connection')).toBeInTheDocument()
  })

  it('disables the submit button while the request is in flight (prevents double-clicking)', async () => {
    let resolveImport!: (job: ImportJob) => void
    vi.mocked(integrationsApi.importWooProducts).mockReturnValue(new Promise(resolve => { resolveImport = resolve }))

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))
    await screen.findByText('Main DC')
    await userEvent.selectOptions(screen.getByLabelText('Default warehouse ID'), 'wh-1')
    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.click(screen.getByRole('button', { name: 'Start import' }))

    expect(screen.getByRole('button', { name: 'Importing…' })).toBeDisabled()
    expect(integrationsApi.importWooProducts).toHaveBeenCalledTimes(1)

    resolveImport({
      id: 'job-1', connection_id: 'conn-1', status: 'pending', imported: 0, already_linked: 0, skipped_type: 0,
      errors: [], error_message: '', created_at: '', started_at: null, finished_at: null,
    })
    await screen.findByText(/running in the background/)
  })

  it('shows an error if the import fails', async () => {
    vi.mocked(integrationsApi.importWooProducts).mockRejectedValue(new Error('woo unreachable'))

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))
    await screen.findByText('Main DC')
    await userEvent.selectOptions(screen.getByLabelText('Default warehouse ID'), 'wh-1')
    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.click(screen.getByRole('button', { name: 'Start import' }))

    expect(await screen.findByText('woo unreachable')).toBeInTheDocument()
  })

  it('closing the modal resets it', async () => {
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))
    await screen.findByText('Main DC')
    await userEvent.selectOptions(screen.getByLabelText('Default warehouse ID'), 'wh-1')

    await userEvent.click(screen.getByRole('button', { name: 'Close' }))

    expect(screen.queryByText('Import products')).not.toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Import' }))
    expect(screen.getByLabelText('Default warehouse ID')).toHaveValue('')
  })
})

describe('Delete imported products', () => {
  it('clicking Delete imported shows a warning and requires typing DELETE', async () => {
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))

    expect(screen.getByText('Delete imported products')).toBeInTheDocument()
    expect(screen.getAllByText(/https:\/\/mystore\.com/).length).toBeGreaterThan(0)
    const confirmButton = screen.getByRole('button', { name: 'Delete' })
    expect(confirmButton).toBeDisabled()

    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'nope')
    expect(confirmButton).toBeDisabled()

    await userEvent.clear(screen.getByLabelText('Type DELETE to confirm'))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    expect(confirmButton).not.toBeDisabled()
  })

  it('confirming calls deleteAllImportedProducts and shows the result', async () => {
    const result: DeleteAllImportedProductsResult = { deleted_sales_orders: 0, deleted_products: 805, deleted_categories: 12, errors: [] }
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue(result)

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(integrationsApi.deleteAllImportedProducts).toHaveBeenCalledWith('test-token', 'conn-1'))
    expect(await screen.findByText(/805/)).toBeInTheDocument()
    expect(screen.getByText(/12/)).toBeInTheDocument()
  })

  it('shows the deleted sales orders count when any were removed', async () => {
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue({
      deleted_sales_orders: 3, deleted_products: 805, deleted_categories: 12, errors: [],
    })

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(await screen.findByText(/3/)).toBeInTheDocument()
    expect(screen.getByText(/sales order/)).toBeInTheDocument()
  })

  it('notifies other components (e.g. the Sidebar badge) that products changed', async () => {
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue({ deleted_sales_orders: 0, deleted_products: 805, deleted_categories: 12, errors: [] })
    const listener = vi.fn()
    const unsubscribe = subscribeProductsChanged(listener)

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(listener).toHaveBeenCalled())
    unsubscribe()
  })

  it('does not crash when the backend sends errors: null instead of []', async () => {
    const result = { deleted_products: 805, deleted_categories: 12, errors: null } as unknown as DeleteAllImportedProductsResult
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue(result)

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(await screen.findByText(/805/)).toBeInTheDocument()
  })

  it('shows an error if the delete fails', async () => {
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockRejectedValue(new Error('db down'))

    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(await screen.findByText('db down')).toBeInTheDocument()
  })

  it('Cancel closes the modal without calling the API', async () => {
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByText('Delete imported products')).not.toBeInTheDocument()
    expect(integrationsApi.deleteAllImportedProducts).not.toHaveBeenCalled()
  })
})

describe('Connect WhatsApp/Stripe/Revolut', () => {
  it('connecting WhatsApp shows the one-time webhook URL and refreshes connections', async () => {
    vi.mocked(integrationsApi.connectWhatsapp).mockResolvedValue({
      connection: { id: 'conn-wa-1', provider: 'whatsapp', store_url: 'https://evolution.example.com', status: 'connected', created_at: '' },
      webhook_url: 'https://api.example.com/api/v1/webhooks/v1/tenant-1/whatsapp/secret-token',
    })
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Browse Integrations' }))
    await userEvent.click(screen.getByText('WhatsApp'))

    await userEvent.type(screen.getByPlaceholderText('https://evolution.example.com'), 'https://evolution.example.com')
    await userEvent.type(screen.getByLabelText('Instance name'), 'tenant-1-instance')
    await userEvent.type(screen.getByLabelText('API key'), 'api-key-1')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText(/secret-token/)).toBeInTheDocument()
    expect(integrationsApi.connectWhatsapp).toHaveBeenCalledWith('test-token', {
      baseUrl: 'https://evolution.example.com', instanceName: 'tenant-1-instance', apiKey: 'api-key-1',
    })
    expect(integrationsApi.listConnections).toHaveBeenCalledTimes(2) // initial load + refresh after connect
  })

  it('connecting Stripe submits the secret key and shows the webhook setup step', async () => {
    vi.mocked(integrationsApi.connectStripe).mockResolvedValue({
      connection: { id: 'conn-stripe-1', provider: 'stripe', store_url: '', status: 'connected', created_at: '' },
      webhook_url: 'https://api.example.com/api/v1/webhooks/v1/tenant-1/stripe',
    })
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Browse Integrations' }))
    await userEvent.click(screen.getByText('Stripe'))

    await userEvent.type(screen.getByPlaceholderText('sk_live_...'), 'sk_test_secret')
    await userEvent.type(screen.getByPlaceholderText('pk_live_...'), 'pk_test_publishable')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    await waitFor(() => expect(integrationsApi.connectStripe).toHaveBeenCalledWith('test-token', 'sk_test_secret', 'pk_test_publishable'))
    expect(screen.queryByLabelText('Secret key')).not.toBeInTheDocument()
    expect(await screen.findByText(/tenant-1\/stripe/)).toBeInTheDocument()
  })

  it('saving the Stripe webhook secret calls setWebhookSecret and shows a confirmation', async () => {
    vi.mocked(integrationsApi.connectStripe).mockResolvedValue({
      connection: { id: 'conn-stripe-1', provider: 'stripe', store_url: '', status: 'connected', created_at: '' },
      webhook_url: 'https://api.example.com/api/v1/webhooks/v1/tenant-1/stripe',
    })
    vi.mocked(integrationsApi.setWebhookSecret).mockResolvedValue({ connection_id: 'conn-stripe-1', webhook_secret: 'whsec_from_dashboard' })
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Browse Integrations' }))
    await userEvent.click(screen.getByText('Stripe'))
    await userEvent.type(screen.getByPlaceholderText('sk_live_...'), 'sk_test_secret')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))
    await screen.findByLabelText('Stripe webhook signing secret')

    await userEvent.type(screen.getByLabelText('Stripe webhook signing secret'), 'whsec_from_dashboard')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.setWebhookSecret).toHaveBeenCalledWith('test-token', 'conn-stripe-1', 'whsec_from_dashboard'))
    expect(await screen.findByText('Webhook secret saved.')).toBeInTheDocument()
  })

  it('connecting Revolut submits the API key and shows the webhook setup step', async () => {
    vi.mocked(integrationsApi.connectRevolut).mockResolvedValue({
      connection: { id: 'conn-revolut-1', provider: 'revolut', store_url: '', status: 'connected', created_at: '' },
      webhook_url: 'https://api.example.com/api/v1/webhooks/v1/tenant-1/revolut',
    })
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Browse Integrations' }))
    await userEvent.click(screen.getByText('Revolut'))

    await userEvent.type(screen.getByLabelText(/merchant api secret key/i), 'revolut-secret')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    await waitFor(() => expect(integrationsApi.connectRevolut).toHaveBeenCalledWith('test-token', 'revolut-secret'))
    expect(screen.queryByLabelText(/merchant api secret key/i)).not.toBeInTheDocument()
    expect(await screen.findByText(/tenant-1\/revolut/)).toBeInTheDocument()
  })

  it('skipping the webhook setup step closes the modal without saving a secret', async () => {
    vi.mocked(integrationsApi.connectRevolut).mockResolvedValue({
      connection: { id: 'conn-revolut-1', provider: 'revolut', store_url: '', status: 'connected', created_at: '' },
      webhook_url: 'https://api.example.com/api/v1/webhooks/v1/tenant-1/revolut',
    })
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Browse Integrations' }))
    await userEvent.click(screen.getByText('Revolut'))
    await userEvent.type(screen.getByLabelText(/merchant api secret key/i), 'revolut-secret')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))
    await screen.findByRole('button', { name: 'Skip for now' })

    await userEvent.click(screen.getByRole('button', { name: 'Skip for now' }))

    expect(screen.queryByLabelText('Revolut webhook signing secret')).not.toBeInTheDocument()
    expect(integrationsApi.setWebhookSecret).not.toHaveBeenCalled()
  })

  it('shows an error if connecting Stripe fails', async () => {
    vi.mocked(integrationsApi.connectStripe).mockRejectedValue(new Error('invalid key'))
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Browse Integrations' }))
    await userEvent.click(screen.getByText('Stripe'))
    await userEvent.type(screen.getByPlaceholderText('sk_live_...'), 'sk_bad')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText('invalid key')).toBeInTheDocument()
  })

  it('connecting Email submits host/port/from and closes the modal (no webhook step)', async () => {
    vi.mocked(integrationsApi.connectEmail).mockResolvedValue(
      { id: 'conn-email-1', provider: 'email', store_url: 'smtp.mystore.com', status: 'connected', created_at: '' },
    )
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Browse Integrations' }))
    await userEvent.click(screen.getByText('Email'))

    await userEvent.type(screen.getByLabelText('SMTP host'), 'smtp.mystore.com')
    await userEvent.clear(screen.getByLabelText('Port'))
    await userEvent.type(screen.getByLabelText('Port'), '587')
    await userEvent.type(screen.getByLabelText('From address'), 'orders@mystore.com')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    await waitFor(() => expect(integrationsApi.connectEmail).toHaveBeenCalledWith('test-token', {
      host: 'smtp.mystore.com', port: '587', username: '', password: '', from: 'orders@mystore.com',
    }))
    expect(screen.queryByLabelText('SMTP host')).not.toBeInTheDocument()
    expect(integrationsApi.listConnections).toHaveBeenCalledTimes(2) // initial load + refresh after connect
  })

  it('shows an error if connecting Email fails', async () => {
    vi.mocked(integrationsApi.connectEmail).mockRejectedValue(new Error('connection refused'))
    render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Browse Integrations' }))
    await userEvent.click(screen.getByText('Email'))
    await userEvent.type(screen.getByLabelText('SMTP host'), 'smtp.mystore.com')
    await userEvent.type(screen.getByLabelText('From address'), 'orders@mystore.com')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText('connection refused')).toBeInTheDocument()
  })
})
