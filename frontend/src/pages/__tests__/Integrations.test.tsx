// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
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
import { subscribeNotificationsChanged } from '../../notifications/notificationsBus'

function wooConnection(overrides: Partial<Connection> = {}): Connection {
  return { id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '', ...overrides }
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
    prices_include_tax: false, default_tax_country: '', updated_at: '',
  })
  vi.mocked(integrationsApi.listSyncSchedules).mockResolvedValue([])
  vi.mocked(integrationsApi.listWooTaxes).mockResolvedValue([])
  vi.mocked(taxApi.listTaxClasses).mockResolvedValue([])
  vi.mocked(taxApi.listExternalTaxClassLinks).mockResolvedValue([])
})

describe('Configure connection', () => {
  it('shows a Configure button for WooCommerce connections', async () => {
    render(<Integrations />)

    expect(await screen.findByRole('button', { name: 'Configure' })).toBeInTheDocument()
  })

  it('clicking Configure opens the WooCommerce sync settings page', async () => {
    render(<Integrations />)

    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))

    expect(await screen.findByText('Sync schedule')).toBeInTheDocument()
    expect(screen.getByRole('heading', { name: 'https://mystore.com' })).toBeInTheDocument()
  })

  it('Back from the sync settings page returns to the connections grid', async () => {
    render(<Integrations />)
    await userEvent.click(await screen.findByRole('button', { name: 'Configure' }))
    await screen.findByText('Sync schedule')

    await userEvent.click(screen.getByRole('button', { name: /Integrations/ }))

    expect(await screen.findByText('Connected integrations')).toBeInTheDocument()
  })
})

describe('Import products', () => {
  it('clicking Import opens a modal asking for warehouse and currency', async () => {
    render(<Integrations />)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))

    expect(screen.getByText('Import products')).toBeInTheDocument()
    expect(screen.getByLabelText('Default warehouse ID')).toBeInTheDocument()
    expect(screen.getByLabelText('Default currency')).toBeInTheDocument()
  })

  it('prefills the currency from the connection default', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: 'EUR' })])

    render(<Integrations />)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('prefills the currency even when stored in a different case or with stray whitespace', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: ' eur ' })])

    render(<Integrations />)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('falls back to the tenant-wide Localization currency when the connection has none of its own', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: '' })])
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '',
      prices_include_tax: false, default_tax_country: '', updated_at: '',
    })

    render(<Integrations />)
    await waitFor(() => expect(tenantSettingsApi.getTenantSettings).toHaveBeenCalled())
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))

    expect(screen.getByLabelText('Default currency')).toHaveValue('EUR')
  })

  it('prefers the connection-level currency over the tenant-wide default when both are set', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection({ default_currency: 'GBP' })])
    vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
      tenant_id: 'tenant-1', default_currency: 'EUR', timezone: '', date_format: '', language: '',
      prices_include_tax: false, default_tax_country: '', updated_at: '',
    })

    render(<Integrations />)
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

    render(<Integrations />)
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

    render(<Integrations />)
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

    render(<Integrations />)
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

    render(<Integrations />)
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

    render(<Integrations />)
    await userEvent.click(await screen.findByRole('button', { name: 'Import' }))
    await screen.findByText('Main DC')
    await userEvent.selectOptions(screen.getByLabelText('Default warehouse ID'), 'wh-1')
    await userEvent.selectOptions(screen.getByLabelText('Default currency'), 'USD')
    await userEvent.click(screen.getByRole('button', { name: 'Start import' }))

    expect(await screen.findByText('woo unreachable')).toBeInTheDocument()
  })

  it('closing the modal resets it', async () => {
    render(<Integrations />)
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
    render(<Integrations />)
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
    const result: DeleteAllImportedProductsResult = { deleted_products: 805, deleted_categories: 12, errors: [] }
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue(result)

    render(<Integrations />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(integrationsApi.deleteAllImportedProducts).toHaveBeenCalledWith('test-token', 'conn-1'))
    expect(await screen.findByText(/805/)).toBeInTheDocument()
    expect(screen.getByText(/12/)).toBeInTheDocument()
  })

  it('notifies other components (e.g. the Sidebar badge) that products changed', async () => {
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue({ deleted_products: 805, deleted_categories: 12, errors: [] })
    const listener = vi.fn()
    const unsubscribe = subscribeProductsChanged(listener)

    render(<Integrations />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    await waitFor(() => expect(listener).toHaveBeenCalled())
    unsubscribe()
  })

  it('does not crash when the backend sends errors: null instead of []', async () => {
    const result = { deleted_products: 805, deleted_categories: 12, errors: null } as unknown as DeleteAllImportedProductsResult
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockResolvedValue(result)

    render(<Integrations />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(await screen.findByText(/805/)).toBeInTheDocument()
  })

  it('shows an error if the delete fails', async () => {
    vi.mocked(integrationsApi.deleteAllImportedProducts).mockRejectedValue(new Error('db down'))

    render(<Integrations />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')
    await userEvent.click(screen.getByRole('button', { name: 'Delete' }))

    expect(await screen.findByText('db down')).toBeInTheDocument()
  })

  it('Cancel closes the modal without calling the API', async () => {
    render(<Integrations />)
    await userEvent.click(await screen.findByRole('button', { name: 'Delete imported' }))
    await userEvent.type(screen.getByLabelText('Type DELETE to confirm'), 'DELETE')

    await userEvent.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByText('Delete imported products')).not.toBeInTheDocument()
    expect(integrationsApi.deleteAllImportedProducts).not.toHaveBeenCalled()
  })
})
