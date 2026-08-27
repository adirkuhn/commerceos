// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Integrations from '../Integrations'
import type { Connection, WebhookForwardRule, WebhookForwardDelivery } from '../../integrations/types'
import type { Warehouse } from '../../inventory/types'

// Separate file, own top-level mocks (customer_owner role), so it doesn't
// collide with Integrations.test.tsx's fixed `user: null` mock -- same
// reasoning as IntegrationsWebhookFallback.test.tsx.
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
  listWebhookForwardRules: vi.fn(),
  createWebhookForwardRule: vi.fn(),
  deleteWebhookForwardRule: vi.fn(),
  listWebhookForwardDeliveries: vi.fn(),
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

function forwardRule(overrides: Partial<WebhookForwardRule> = {}): WebhookForwardRule {
  return { id: 'rule-1', provider: 'woocommerce', topic: 'order.updated', target_url: 'https://shop.example.com/hook', active: true, created_at: '2026-08-23T10:00:00Z', ...overrides }
}

function forwardDelivery(overrides: Partial<WebhookForwardDelivery> = {}): WebhookForwardDelivery {
  return {
    id: 'd-1', rule_id: 'rule-1', provider: 'woocommerce', topic: 'order.updated', target_url: 'https://shop.example.com/hook',
    status: 'delivered', attempts: 1, last_error: '', next_attempt_at: '2026-08-23T10:00:00Z', delivered_at: '2026-08-23T10:00:01Z',
    created_at: '2026-08-23T10:00:00Z', ...overrides,
  }
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
  vi.mocked(integrationsApi.listWebhookForwardRules).mockResolvedValue([])
  vi.mocked(integrationsApi.listWebhookForwardDeliveries).mockResolvedValue([])
})

describe('Integrations — webhook forwarding panel', () => {
  it('always shows the add-rule form, even with no rules yet', async () => {
    render(<Integrations />)

    expect(await screen.findByText('Webhook forwarding')).toBeInTheDocument()
    expect(screen.getByLabelText('Provider')).toBeInTheDocument()
    expect(screen.getByLabelText('Target URL')).toBeInTheDocument()
    expect(screen.getByText('No forwarding rules configured yet.')).toBeInTheDocument()
  })

  it('renders existing rules', async () => {
    vi.mocked(integrationsApi.listWebhookForwardRules).mockResolvedValue([forwardRule()])

    render(<Integrations />)

    expect(await screen.findByText('https://shop.example.com/hook')).toBeInTheDocument()
    expect(screen.getByText('order.updated')).toBeInTheDocument()
  })

  it('a blank topic renders as "any"', async () => {
    vi.mocked(integrationsApi.listWebhookForwardRules).mockResolvedValue([forwardRule({ topic: '' })])

    render(<Integrations />)

    expect(await screen.findByText('any')).toBeInTheDocument()
  })

  it('Add rule submits provider/topic/target_url and prepends the new rule', async () => {
    vi.mocked(integrationsApi.createWebhookForwardRule).mockResolvedValue(forwardRule({ id: 'rule-2', provider: 'myparcel', topic: '', target_url: 'https://example.com/mp-hook' }))

    render(<Integrations />)
    await screen.findByText('Webhook forwarding')

    await userEvent.type(screen.getByLabelText('Provider'), 'myparcel')
    await userEvent.type(screen.getByLabelText('Target URL'), 'https://example.com/mp-hook')
    await userEvent.click(screen.getByRole('button', { name: 'Add rule' }))

    expect(integrationsApi.createWebhookForwardRule).toHaveBeenCalledWith('test-token', {
      provider: 'myparcel', topic: undefined, target_url: 'https://example.com/mp-hook',
    })
    expect(await screen.findByText('https://example.com/mp-hook')).toBeInTheDocument()
  })

  it('surfaces a create failure inline instead of swallowing it', async () => {
    vi.mocked(integrationsApi.createWebhookForwardRule).mockRejectedValue(new Error('target_url must be an absolute http(s) URL'))

    render(<Integrations />)
    await screen.findByText('Webhook forwarding')

    await userEvent.type(screen.getByLabelText('Provider'), 'woocommerce')
    await userEvent.type(screen.getByLabelText('Target URL'), 'not-a-url')
    await userEvent.click(screen.getByRole('button', { name: 'Add rule' }))

    expect(await screen.findByText('target_url must be an absolute http(s) URL')).toBeInTheDocument()
  })

  it('Remove calls the delete endpoint and removes the row on success', async () => {
    vi.mocked(integrationsApi.listWebhookForwardRules).mockResolvedValue([forwardRule()])
    vi.mocked(integrationsApi.deleteWebhookForwardRule).mockResolvedValue(null)

    render(<Integrations />)
    await screen.findByText('https://shop.example.com/hook')

    await userEvent.click(screen.getByRole('button', { name: 'Remove' }))

    expect(integrationsApi.deleteWebhookForwardRule).toHaveBeenCalledWith('test-token', 'rule-1')
    expect(screen.queryByText('https://shop.example.com/hook')).not.toBeInTheDocument()
  })

  it('shows the recent-deliveries log when there are any', async () => {
    vi.mocked(integrationsApi.listWebhookForwardDeliveries).mockResolvedValue([forwardDelivery()])

    render(<Integrations />)

    expect(await screen.findByText('Recent deliveries')).toBeInTheDocument()
    expect(screen.getByText('woocommerce / order.updated')).toBeInTheDocument()
    expect(screen.getByText('delivered')).toBeInTheDocument()
  })

  it('hides the recent-deliveries log when there are none', async () => {
    render(<Integrations />)

    await screen.findByText('Webhook forwarding')
    expect(screen.queryByText('Recent deliveries')).not.toBeInTheDocument()
  })
})
