// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { AppProvider } from '../../context/AppContext'
import Integrations from '../../pages/Integrations'
import type { Connection } from '../types'
import { renderAt } from '../../test/renderAt'
import { act } from 'react'
import * as shippingApi from '../../shipping/api'

// Covers the parts of the Integrations page that Integrations.test.tsx
// doesn't: the WooCommerce connect forms, the per-card Test / Disconnect
// buttons, the empty state and the admin-only Webhooks shortcut.

let mockRole: string | null = null
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: mockRole ? { user_id: 'u1', tenant_id: 'tenant-1', role: mockRole, exp: 0, iat: 0 } : null,
    login: vi.fn(), logout: vi.fn(), isAuthenticated: true,
  }),
}))

vi.mock('../api', () => ({
  listConnections: vi.fn(),
  testConnection: vi.fn(),
  disconnectConnection: vi.fn(),
  startWooConnect: vi.fn(),
  manualConnectWooCommerce: vi.fn(),
  listWebhookFallbackEvents: vi.fn(),
}))

// The default-shipping-provider setting loads its own list; none here.
vi.mock('../../shipping/api', async importOriginal => ({
  ...await importOriginal<typeof import('../../shipping/api')>(),
  listShippingConnections: vi.fn().mockResolvedValue({ connections: [], default_connection_id: null, configured_connection_id: null }),
  getShippingAddresses: vi.fn().mockResolvedValue({ sender_address: null, return_address: null, company_address: null }),
}))
vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn(),
}))

import * as integrationsApi from '../api'
import * as inventoryApi from '../../inventory/api'
import * as tenantSettingsApi from '../../settings/api'

function wooConnection(overrides: Partial<Connection> = {}): Connection {
  return { id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '', ...overrides }
}

function renderPage() {
  return render(<MemoryRouter><AppProvider><Integrations /></AppProvider></MemoryRouter>)
}

async function openWooForm() {
  renderPage()
  await userEvent.click(await screen.findByRole('button', { name: 'Browse integrations' }))
  await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))
}

beforeEach(() => {
  vi.clearAllMocks()
  mockRole = null
  vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
  vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '',
    prices_include_tax: false, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
  })
  vi.mocked(integrationsApi.listWebhookFallbackEvents).mockResolvedValue({ events: [], total: 0 })
  vi.mocked(shippingApi.listShippingConnections).mockResolvedValue({ connections: [], default_connection_id: null, configured_connection_id: null })
})

describe('Connection card actions', () => {
  it('Test shows the success message on the card and reloads connections', async () => {
    vi.mocked(integrationsApi.testConnection).mockResolvedValue(undefined as never)
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Test' }))

    expect(await screen.findByText('Connection OK')).toBeInTheDocument()
    expect(integrationsApi.testConnection).toHaveBeenCalledWith('test-token', 'conn-1')
    await waitFor(() => expect(integrationsApi.listConnections).toHaveBeenCalledTimes(2))
  })

  it('Test surfaces the failure message on the card', async () => {
    vi.mocked(integrationsApi.testConnection).mockRejectedValue(new Error('401 from woo'))
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Test' }))

    expect(await screen.findByText('401 from woo')).toBeInTheDocument()
  })

  it('shows Testing… and disables the button while the test runs', async () => {
    let resolveTest: () => void = () => {}
    vi.mocked(integrationsApi.testConnection).mockReturnValue(new Promise<never>(resolve => { resolveTest = resolve as () => void }))
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Test' }))

    expect(screen.getByRole('button', { name: 'Testing…' })).toBeDisabled()
    resolveTest()
    expect(await screen.findByText('Connection OK')).toBeInTheDocument()
  })

  it('renders the WooCommerce card buttons in order', async () => {
    renderPage()
    await screen.findByText('Connected integrations')
    const labels = screen.getAllByRole('button').map(b => b.textContent)
    expect(labels).toEqual(['Browse integrations', 'Configure', 'Test', 'Import', 'Delete imported', 'Disconnect'])
  })

  it('renders the WhatsApp card buttons in order', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([
      wooConnection({ id: 'wa', provider: 'whatsapp' }),
    ])
    renderPage()
    await screen.findByText('Connected integrations')
    const labels = screen.getAllByRole('button').map(b => b.textContent)
    expect(labels).toEqual(['Browse integrations', 'Test mode', 'AI parsing', 'Edit credentials', 'Webhook', 'Disconnect'])
  })

  it('Disconnect removes the connection', async () => {
    vi.mocked(integrationsApi.disconnectConnection).mockResolvedValue(undefined as never)
    renderPage()
    await userEvent.click(await screen.findByRole('button', { name: 'Disconnect' }))

    expect(integrationsApi.disconnectConnection).toHaveBeenCalledWith('test-token', 'conn-1')
    expect(await screen.findByText('No integrations connected')).toBeInTheDocument()
  })
})

describe('Empty state and admin shortcut', () => {
  it('shows the empty state with a Browse button when nothing is connected', async () => {
    vi.mocked(integrationsApi.listConnections).mockResolvedValue([])
    renderPage()

    expect(await screen.findByText('No integrations connected')).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Browse integrations' }))
    expect(screen.getByRole('heading', { name: 'Browse integrations' })).toBeInTheDocument()
  })

  it('hides the Webhooks shortcut for a non-admin', async () => {
    renderPage()
    await screen.findByText('Connected integrations')
    expect(screen.queryByText('Webhooks')).not.toBeInTheDocument()
  })

  it('shows the Webhooks shortcut for an owner', async () => {
    mockRole = 'customer_owner'
    renderPage()
    expect(await screen.findByText('Fan-out rules and every active provider subscription, in one place')).toBeInTheDocument()
  })
})

// The default shipping provider and shipping addresses moved to
// Settings -> Shipping (they pushed the connections off a page that
// didn't scroll); Integrations only links there.
describe('Shipping settings', () => {
  it('are not on the Integrations page, which links to Settings -> Shipping', async () => {
    mockRole = 'customer_owner'
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue({
      connections: [{ id: 'mp', provider: 'myparcel', status: 'active', created_at: '2026-01-01T00:00:00Z', is_default: true, missing: [] }],
      default_connection_id: 'mp', configured_connection_id: null,
    })
    renderPage()
    expect(await screen.findByRole('link', { name: 'Open shipping settings' })).toHaveAttribute('href', '/settings/shipping')
    expect(screen.queryByLabelText('Default shipping provider')).not.toBeInTheDocument()
    expect(screen.queryByRole('region', { name: 'Shipping addresses' })).not.toBeInTheDocument()
  })
})

describe('Connect WooCommerce', () => {
  const originalLocation = window.location
  afterEach(() => {
    Object.defineProperty(window, 'location', { configurable: true, value: originalLocation })
  })

  it('automatic connect redirects to the authorize URL', async () => {
    const fakeLocation = { href: 'http://localhost/' }
    Object.defineProperty(window, 'location', { configurable: true, value: fakeLocation })
    vi.mocked(integrationsApi.startWooConnect).mockResolvedValue({ authorize_url: 'https://mystore.com/wc-auth' })
    await openWooForm()

    await userEvent.type(screen.getByPlaceholderText('https://mystore.com'), 'https://mystore.com')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    await waitFor(() => expect(fakeLocation.href).toBe('https://mystore.com/wc-auth'))
    expect(integrationsApi.startWooConnect).toHaveBeenCalledWith('test-token', 'https://mystore.com')
  })

  it('automatic connect surfaces an error', async () => {
    vi.mocked(integrationsApi.startWooConnect).mockRejectedValue(new Error('store unreachable'))
    await openWooForm()

    await userEvent.type(screen.getByPlaceholderText('https://mystore.com'), 'https://mystore.com')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText('store unreachable')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Connect' })).not.toBeDisabled()
  })

  it('manual setup submits the keys, closes the catalog and reloads connections', async () => {
    vi.mocked(integrationsApi.manualConnectWooCommerce).mockResolvedValue(wooConnection())
    await openWooForm()

    await userEvent.click(screen.getByRole('button', { name: 'Manual setup' }))
    expect(screen.getByRole('button', { name: 'Manual setup' })).toHaveAttribute('aria-pressed', 'true')
    await userEvent.type(screen.getByPlaceholderText('https://mystore.com'), 'https://mystore.com')
    await userEvent.type(screen.getByPlaceholderText('ck_...'), 'ck_1')
    await userEvent.type(screen.getByPlaceholderText('cs_...'), 'cs_1')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    // Closing a connect form steps back to the catalog it was opened from.
    await waitFor(() => expect(screen.queryByPlaceholderText('ck_...')).not.toBeInTheDocument())
    expect(screen.getByRole('heading', { name: 'Browse integrations' })).toBeInTheDocument()
    expect(integrationsApi.manualConnectWooCommerce).toHaveBeenCalledWith('test-token', {
      storeUrl: 'https://mystore.com', consumerKey: 'ck_1', consumerSecret: 'cs_1',
    })
    expect(integrationsApi.listConnections).toHaveBeenCalledTimes(2)
  })

  it('manual setup surfaces an error', async () => {
    vi.mocked(integrationsApi.manualConnectWooCommerce).mockRejectedValue(new Error('bad keys'))
    await openWooForm()

    await userEvent.click(screen.getByRole('button', { name: 'Manual setup' }))
    await userEvent.type(screen.getByPlaceholderText('https://mystore.com'), 'https://mystore.com')
    await userEvent.type(screen.getByPlaceholderText('ck_...'), 'ck_1')
    await userEvent.type(screen.getByPlaceholderText('cs_...'), 'cs_1')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText('bad keys')).toBeInTheDocument()
  })

  it('closing the catalog resets typed values', async () => {
    await openWooForm()
    await userEvent.type(screen.getByPlaceholderText('https://mystore.com'), 'https://typed.com')
    await userEvent.click(screen.getByRole('button', { name: 'Close' })) // form -> catalog
    await userEvent.click(screen.getByRole('button', { name: 'Close' })) // catalog -> list
    await waitFor(() => expect(screen.queryByRole('heading', { name: 'Browse integrations' })).not.toBeInTheDocument())
    await userEvent.click(await screen.findByRole('button', { name: 'Browse integrations' }))
    await userEvent.click(await screen.findByRole('button', { name: /WooCommerce/ }))

    expect(screen.getByPlaceholderText('https://mystore.com')).toHaveValue('')
  })

  it('values typed in a connect form survive going Back to the catalog and returning', async () => {
    const { router } = renderAt(<AppProvider><Integrations /></AppProvider>, '/integrations', '/integrations/connect')
    await userEvent.click(await screen.findByRole('button', { name: /Revolut/ }))
    await userEvent.type(screen.getByLabelText('Merchant API secret key'), 'rk_1')

    await act(() => router.navigate(-1))
    expect(router.state.location.pathname).toBe('/integrations/connect')
    await userEvent.click(await screen.findByRole('button', { name: /Revolut/ }))

    expect(screen.getByLabelText('Merchant API secret key')).toHaveValue('rk_1')
  })
})
