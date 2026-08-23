// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Integrations from '../Integrations'
import type { Connection, WebhookFallbackEvent } from '../../integrations/types'
import type { Warehouse } from '../../inventory/types'

// Separate file, own top-level mocks (customer_owner role), so it doesn't
// collide with Integrations.test.tsx's fixed `user: null` mock.
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: { user_id: 'u1', tenant_id: 'tenant-1', role: 'customer_owner', exp: 0, iat: 0 },
    login: vi.fn(), logout: vi.fn(), isAuthenticated: true,
  }),
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
  listWebhookFallbackEvents: vi.fn(),
  retryWebhookFallbackEvent: vi.fn(),
  abandonWebhookFallbackEvent: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn(),
}))

import * as integrationsApi from '../../integrations/api'
import * as inventoryApi from '../../inventory/api'
import * as tenantSettingsApi from '../../settings/api'

function wooConnection(overrides: Partial<Connection> = {}): Connection {
  return { id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '', ...overrides }
}

const sampleWarehouse: Warehouse = {
  id: 'wh-1', tenant_id: 'tenant-1', name: 'Main DC',
  address1: '', address2: '', city: '', province: '', zip: '', country: '', phone: '', active: true,
  created_at: '', updated_at: '',
}

function fallbackEvent(overrides: Partial<WebhookFallbackEvent> = {}): WebhookFallbackEvent {
  return { id: 'fb-1', provider: 'woocommerce', event_id: 'hash-abc', status: 'pending_republish', received_at: '2026-08-23T10:00:00Z', ...overrides }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listConnections).mockResolvedValue([wooConnection()])
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([sampleWarehouse])
  vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '',
    prices_include_tax: false, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
  })
  vi.mocked(integrationsApi.listWebhookFallbackEvents).mockResolvedValue([])
})

describe('Integrations — undelivered webhooks table', () => {
  it('renders rows for pending fallback events', async () => {
    vi.mocked(integrationsApi.listWebhookFallbackEvents).mockResolvedValue([fallbackEvent()])

    render(<Integrations />)

    expect(await screen.findByText('hash-abc')).toBeInTheDocument()
    expect(screen.getByText('woocommerce', { selector: 'td' })).toBeInTheDocument()
    expect(integrationsApi.listWebhookFallbackEvents).toHaveBeenCalledWith('test-token', 'pending_republish')
  })

  it('shows nothing pending when the list is empty', async () => {
    render(<Integrations />)

    await screen.findByRole('heading', { name: 'Connected integrations' })
    expect(screen.queryByText('Undelivered webhooks')).not.toBeInTheDocument()
  })

  it('Retry calls the retry endpoint and removes the row on success', async () => {
    vi.mocked(integrationsApi.listWebhookFallbackEvents).mockResolvedValue([fallbackEvent()])
    vi.mocked(integrationsApi.retryWebhookFallbackEvent).mockResolvedValue(null)

    render(<Integrations />)
    await screen.findByText('hash-abc')

    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))

    expect(integrationsApi.retryWebhookFallbackEvent).toHaveBeenCalledWith('test-token', 'fb-1')
    expect(screen.queryByText('hash-abc')).not.toBeInTheDocument()
  })

  it('Abandon calls the abandon endpoint and removes the row on success', async () => {
    vi.mocked(integrationsApi.listWebhookFallbackEvents).mockResolvedValue([fallbackEvent()])
    vi.mocked(integrationsApi.abandonWebhookFallbackEvent).mockResolvedValue(null)

    render(<Integrations />)
    await screen.findByText('hash-abc')

    await userEvent.click(screen.getByRole('button', { name: 'Abandon' }))

    expect(integrationsApi.abandonWebhookFallbackEvent).toHaveBeenCalledWith('test-token', 'fb-1')
    expect(screen.queryByText('hash-abc')).not.toBeInTheDocument()
  })

  it('surfaces a retry failure inline instead of swallowing it', async () => {
    vi.mocked(integrationsApi.listWebhookFallbackEvents).mockResolvedValue([fallbackEvent()])
    vi.mocked(integrationsApi.retryWebhookFallbackEvent).mockRejectedValue(new Error('publish failed: nats unreachable'))

    render(<Integrations />)
    await screen.findByText('hash-abc')

    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))

    expect(await screen.findByText('publish failed: nats unreachable')).toBeInTheDocument()
    // Row stays -- retry failed, nothing to remove.
    expect(screen.getByText('hash-abc')).toBeInTheDocument()
  })
})
