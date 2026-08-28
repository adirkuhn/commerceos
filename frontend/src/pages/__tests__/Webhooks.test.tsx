// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Webhooks from '../Webhooks'
import type { WebhookForwardRule, WebhookForwardDelivery, ProviderWebhook } from '../../integrations/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: { user_id: 'u1', tenant_id: 'tenant-1', role: 'customer_owner', exp: 0, iat: 0 },
    login: vi.fn(), logout: vi.fn(), isAuthenticated: true,
  }),
}))

vi.mock('../../integrations/api', () => ({
  listWebhookForwardRules: vi.fn(),
  createWebhookForwardRule: vi.fn(),
  deleteWebhookForwardRule: vi.fn(),
  listWebhookForwardDeliveries: vi.fn(),
  listAllProviderWebhooks: vi.fn(),
  deleteProviderWebhook: vi.fn(),
}))

import * as integrationsApi from '../../integrations/api'

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

function providerWebhook(overrides: Partial<ProviderWebhook> = {}): ProviderWebhook {
  return {
    id: '1', provider: 'woocommerce', topic: 'order.updated', delivery_url: 'https://commerceos.example.com/hook',
    status: 'active', connection_id: 'conn-1', store_label: 'https://mystore.com', ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listWebhookForwardRules).mockResolvedValue([])
  vi.mocked(integrationsApi.listWebhookForwardDeliveries).mockResolvedValue([])
  vi.mocked(integrationsApi.listAllProviderWebhooks).mockResolvedValue([])
})

describe('Webhooks page — Fan-out tab', () => {
  it('always shows the add-rule form, even with no rules yet', async () => {
    render(<Webhooks />)

    expect(await screen.findByText('Webhook forwarding')).toBeInTheDocument()
    expect(screen.getByLabelText('Provider')).toBeInTheDocument()
    expect(screen.getByText('No forwarding rules configured yet.')).toBeInTheDocument()
  })

  it('renders existing rules', async () => {
    vi.mocked(integrationsApi.listWebhookForwardRules).mockResolvedValue([forwardRule()])

    render(<Webhooks />)

    expect(await screen.findByText('https://shop.example.com/hook')).toBeInTheDocument()
  })

  it('Add rule submits provider/topic/target_url and prepends the new rule', async () => {
    vi.mocked(integrationsApi.createWebhookForwardRule).mockResolvedValue(
      forwardRule({ id: 'rule-2', provider: 'myparcel', topic: '', target_url: 'https://example.com/mp-hook' }),
    )
    render(<Webhooks />)
    await screen.findByText('Webhook forwarding')

    await userEvent.type(screen.getByLabelText('Provider'), 'myparcel')
    await userEvent.type(screen.getByLabelText('Target URL'), 'https://example.com/mp-hook')
    await userEvent.click(screen.getByRole('button', { name: 'Add rule' }))

    expect(integrationsApi.createWebhookForwardRule).toHaveBeenCalledWith('test-token', {
      provider: 'myparcel', topic: undefined, target_url: 'https://example.com/mp-hook',
    })
    expect(await screen.findByText('https://example.com/mp-hook')).toBeInTheDocument()
  })

  it('Remove calls the delete endpoint and removes the row', async () => {
    vi.mocked(integrationsApi.listWebhookForwardRules).mockResolvedValue([forwardRule()])
    vi.mocked(integrationsApi.deleteWebhookForwardRule).mockResolvedValue(null)

    render(<Webhooks />)
    await screen.findByText('https://shop.example.com/hook')

    await userEvent.click(screen.getByRole('button', { name: 'Remove' }))

    expect(integrationsApi.deleteWebhookForwardRule).toHaveBeenCalledWith('test-token', 'rule-1')
    expect(screen.queryByText('https://shop.example.com/hook')).not.toBeInTheDocument()
  })

  it('shows the recent-deliveries log when there are any', async () => {
    vi.mocked(integrationsApi.listWebhookForwardDeliveries).mockResolvedValue([forwardDelivery()])

    render(<Webhooks />)

    expect(await screen.findByText('Recent deliveries')).toBeInTheDocument()
  })
})

describe('Webhooks page — Subscriptions tab', () => {
  it('lists provider subscriptions aggregated across every connection', async () => {
    vi.mocked(integrationsApi.listAllProviderWebhooks).mockResolvedValue([providerWebhook()])

    render(<Webhooks />)
    await userEvent.click(await screen.findByRole('button', { name: 'Subscriptions' }))

    expect(await screen.findByText('https://commerceos.example.com/hook')).toBeInTheDocument()
    expect(screen.getByText(/mystore\.com/)).toBeInTheDocument()
  })

  it('does not fetch subscriptions until the Subscriptions tab is opened', async () => {
    render(<Webhooks />)
    await screen.findByText('Webhook forwarding')

    expect(integrationsApi.listAllProviderWebhooks).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Subscriptions' }))

    expect(await screen.findByText('Active provider subscriptions')).toBeInTheDocument()
    expect(integrationsApi.listAllProviderWebhooks).toHaveBeenCalled()
  })

  it('Remove calls the delete endpoint with the row\'s own connection and removes it', async () => {
    vi.mocked(integrationsApi.listAllProviderWebhooks).mockResolvedValue([providerWebhook()])
    vi.mocked(integrationsApi.deleteProviderWebhook).mockResolvedValue(null)

    render(<Webhooks />)
    await userEvent.click(await screen.findByRole('button', { name: 'Subscriptions' }))
    await screen.findByText('https://commerceos.example.com/hook')

    await userEvent.click(screen.getByRole('button', { name: 'Remove' }))

    expect(integrationsApi.deleteProviderWebhook).toHaveBeenCalledWith('test-token', 'conn-1', '1')
    expect(screen.queryByText('https://commerceos.example.com/hook')).not.toBeInTheDocument()
  })

  it('shows an empty state when nothing is registered anywhere', async () => {
    render(<Webhooks />)
    await userEvent.click(await screen.findByRole('button', { name: 'Subscriptions' }))

    expect(await screen.findByText('Nothing registered yet at any connection.')).toBeInTheDocument()
  })
})
