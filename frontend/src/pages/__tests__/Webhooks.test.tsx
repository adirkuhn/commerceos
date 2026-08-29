// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import { AppProvider } from '../../context/AppContext'
import Webhooks from '../Webhooks'
import type { WebhookForwardRule, WebhookForwardDelivery, ProviderWebhook, WebhookEvent } from '../../integrations/types'

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
  listAvailableWebhookTopics: vi.fn(),
  listWebhookEvents: vi.fn(),
  retryWebhookEventDispatch: vi.fn(),
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
    id: '1', provider: 'woocommerce', topic: 'order.updated', delivery_url: 'https://commerceos.example.com/webhooks/v1/tenant-1/woocommerce',
    status: 'active', connection_id: 'conn-1', store_label: 'https://mystore.com', ...overrides,
  }
}

function webhookEvent(overrides: Partial<WebhookEvent> = {}): WebhookEvent {
  return {
    id: 'evt-1', provider: 'myparcel', topic: 'shipment_status_change', event_id: 'e-1',
    payload: { status: 'shipped' }, received_at: '2026-08-23T10:00:00Z', processed_at: '2026-08-23T10:00:01Z',
    status: 'received', dispatch_error: '',
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listWebhookForwardRules).mockResolvedValue([])
  vi.mocked(integrationsApi.listWebhookForwardDeliveries).mockResolvedValue([])
  vi.mocked(integrationsApi.listAllProviderWebhooks).mockResolvedValue([])
  vi.mocked(integrationsApi.listAvailableWebhookTopics).mockResolvedValue({ all_events: {}, active_webhooks: {} })
  vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([])
})

describe('Webhooks page — Fan-out tab', () => {
  it('always shows the add-rule form, even with no rules yet', async () => {
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)

    expect(await screen.findByText('Webhook forwarding')).toBeInTheDocument()
    expect(screen.getByLabelText('Provider')).toBeInTheDocument()
    expect(screen.getByText('No forwarding rules configured yet.')).toBeInTheDocument()
  })

  it('renders existing rules', async () => {
    vi.mocked(integrationsApi.listWebhookForwardRules).mockResolvedValue([forwardRule()])

    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)

    expect(await screen.findByText('https://shop.example.com/hook')).toBeInTheDocument()
  })

  it('Add rule submits provider/topic/target_url and prepends the new rule', async () => {
    vi.mocked(integrationsApi.createWebhookForwardRule).mockResolvedValue(
      forwardRule({ id: 'rule-2', provider: 'myparcel', topic: '', target_url: 'https://example.com/mp-hook' }),
    )
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await screen.findByText('Webhook forwarding')

    await userEvent.type(screen.getByLabelText('Provider'), 'myparcel')
    await userEvent.type(screen.getByLabelText('Target URL'), 'https://example.com/mp-hook')
    await userEvent.click(screen.getByRole('button', { name: 'Add rule' }))

    expect(integrationsApi.createWebhookForwardRule).toHaveBeenCalledWith('test-token', {
      provider: 'myparcel', topic: undefined, target_url: 'https://example.com/mp-hook',
    })
    expect(await screen.findByText('https://example.com/mp-hook')).toBeInTheDocument()
  })

  it('Topic dropdown groups all_events by provider -- who is responsible for each topic stays visible, generic for any future provider', async () => {
    // all_events already carries the backend's seeded baseline catalog
    // (backend/domain/integrations' WooWebhookTopic/MyParcelHookType
    // constants -- see ListAvailableWebhookTopicsUseCase) -- the frontend
    // must not flatten this across providers, since which service owns a
    // topic is exactly the information an admin needs when picking one.
    vi.mocked(integrationsApi.listAvailableWebhookTopics).mockResolvedValue({
      all_events: {
        woocommerce: ['order.updated', 'product.updated'],
        customprovider: ['product.updated', 'custom.thing.happened'],
      },
      active_webhooks: {},
    })
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await screen.findByText('Webhook forwarding')

    expect(screen.getByRole('group', { name: 'WooCommerce — all events' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'Customprovider — all events' })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'custom.thing.happened' })).toBeInTheDocument()
    // Same topic name offered once per provider that actually has it, not deduped away
    expect(screen.getAllByRole('option', { name: 'product.updated' })).toHaveLength(2)
  })

  it('Topic dropdown separates one provider\'s active subscriptions from its all-events group', async () => {
    vi.mocked(integrationsApi.listAvailableWebhookTopics).mockResolvedValue({
      all_events: { myparcel: ['shipment_status_change', 'shop_updated'] },
      active_webhooks: { myparcel: ['shipment_status_change'] },
    })
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await screen.findByText('Webhook forwarding')

    expect(screen.getByRole('group', { name: 'MyParcel — active' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'MyParcel — all events' })).toBeInTheDocument()
    expect(screen.getAllByRole('option', { name: 'shipment_status_change' })).toHaveLength(2)
    expect(screen.getAllByRole('option', { name: 'shop_updated' })).toHaveLength(1)
  })

  it('Topic dropdown supports multiple providers with active subscriptions at once, each in its own group', async () => {
    vi.mocked(integrationsApi.listAvailableWebhookTopics).mockResolvedValue({
      all_events: { myparcel: ['shipment_status_change'], woocommerce: ['order.updated'] },
      active_webhooks: { myparcel: ['shipment_status_change'], woocommerce: ['order.updated'] },
    })
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await screen.findByText('Webhook forwarding')

    expect(screen.getByRole('group', { name: 'MyParcel — active' })).toBeInTheDocument()
    expect(screen.getByRole('group', { name: 'WooCommerce — active' })).toBeInTheDocument()
  })

  it('Remove calls the delete endpoint and removes the row', async () => {
    vi.mocked(integrationsApi.listWebhookForwardRules).mockResolvedValue([forwardRule()])
    vi.mocked(integrationsApi.deleteWebhookForwardRule).mockResolvedValue(null)

    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await screen.findByText('https://shop.example.com/hook')

    await userEvent.click(screen.getByRole('button', { name: 'Remove' }))

    expect(integrationsApi.deleteWebhookForwardRule).toHaveBeenCalledWith('test-token', 'rule-1')
    expect(screen.queryByText('https://shop.example.com/hook')).not.toBeInTheDocument()
  })

  it('shows the recent-deliveries log when there are any', async () => {
    vi.mocked(integrationsApi.listWebhookForwardDeliveries).mockResolvedValue([forwardDelivery()])

    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)

    expect(await screen.findByText('Recent deliveries')).toBeInTheDocument()
  })
})

describe('Webhooks page — Subscriptions tab', () => {
  it('lists provider subscriptions aggregated across every connection', async () => {
    vi.mocked(integrationsApi.listAllProviderWebhooks).mockResolvedValue([providerWebhook()])

    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Subscriptions' }))

    expect(await screen.findByText('https://commerceos.example.com/webhooks/v1/tenant-1/woocommerce')).toBeInTheDocument()
    expect(screen.getByText(/mystore\.com/)).toBeInTheDocument()
  })

  it('does not fetch subscriptions until the Subscriptions tab is opened', async () => {
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await screen.findByText('Webhook forwarding')

    expect(integrationsApi.listAllProviderWebhooks).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Subscriptions' }))

    expect(await screen.findByText('Active provider subscriptions')).toBeInTheDocument()
    expect(integrationsApi.listAllProviderWebhooks).toHaveBeenCalled()
  })

  it('Remove calls the delete endpoint with the row\'s own connection and removes it', async () => {
    vi.mocked(integrationsApi.listAllProviderWebhooks).mockResolvedValue([providerWebhook()])
    vi.mocked(integrationsApi.deleteProviderWebhook).mockResolvedValue(null)

    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Subscriptions' }))
    await screen.findByText('https://commerceos.example.com/webhooks/v1/tenant-1/woocommerce')

    await userEvent.click(screen.getByRole('button', { name: 'Remove' }))

    expect(integrationsApi.deleteProviderWebhook).toHaveBeenCalledWith('test-token', 'conn-1', '1')
    expect(screen.queryByText('https://commerceos.example.com/webhooks/v1/tenant-1/woocommerce')).not.toBeInTheDocument()
  })

  it('shows an empty state when nothing is registered anywhere', async () => {
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Subscriptions' }))

    expect(await screen.findByText('Nothing registered yet at any connection.')).toBeInTheDocument()
  })

  it('always shows a "Register a webhook" action, and navigating it does not throw', async () => {
    vi.mocked(integrationsApi.listAllProviderWebhooks).mockResolvedValue([providerWebhook()])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Subscriptions' }))
    await screen.findByText('Active provider subscriptions')

    await userEvent.click(screen.getByRole('button', { name: 'Register a webhook' }))
  })

  it('warns when none of the registered subscriptions point to CommerceOS itself', async () => {
    vi.mocked(integrationsApi.listAllProviderWebhooks).mockResolvedValue([
      providerWebhook({ delivery_url: 'https://erp.example.com/hook' }),
    ])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Subscriptions' }))

    expect(await screen.findByText(/None of these are registered with CommerceOS/)).toBeInTheDocument()
  })

  it('does not warn when at least one subscription points to CommerceOS', async () => {
    vi.mocked(integrationsApi.listAllProviderWebhooks).mockResolvedValue([providerWebhook()])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Subscriptions' }))
    await screen.findByText('Active provider subscriptions')

    expect(screen.queryByText(/None of these are registered with CommerceOS/)).not.toBeInTheDocument()
  })
})

describe('Webhooks page — Messages tab', () => {
  it('does not fetch messages until the Messages tab is opened', async () => {
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await screen.findByText('Webhook forwarding')

    expect(integrationsApi.listWebhookEvents).not.toHaveBeenCalled()

    await userEvent.click(screen.getByRole('button', { name: 'Messages' }))

    expect(await screen.findByText('Received messages')).toBeInTheDocument()
    expect(integrationsApi.listWebhookEvents).toHaveBeenCalledWith('test-token', { provider: undefined })
  })

  it('lists received messages with sender and topic', async () => {
    vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([webhookEvent()])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))

    expect(await screen.findByText('MyParcel')).toBeInTheDocument()
    expect(screen.getByText('shipment_status_change')).toBeInTheDocument()
  })

  it('shows an empty state when nothing has been received yet', async () => {
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))

    expect(await screen.findByText('No messages received yet.')).toBeInTheDocument()
  })

  it('changing the sender filter refetches with the selected provider', async () => {
    vi.mocked(integrationsApi.listAvailableWebhookTopics).mockResolvedValue({
      all_events: { myparcel: ['shipment_status_change'], woocommerce: ['order.updated'] },
      active_webhooks: {},
    })
    vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([webhookEvent()])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))
    await screen.findByText('Received messages')

    await userEvent.selectOptions(screen.getByLabelText('Sender'), 'myparcel')

    expect(integrationsApi.listWebhookEvents).toHaveBeenCalledWith('test-token', { provider: 'myparcel' })
  })

  it('payload is truncated by default and expands on click', async () => {
    vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([
      webhookEvent({ payload: { status: 'shipped', tracking_id: 'a-very-long-tracking-identifier-that-is-quite-long' } }),
    ])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))

    const preview = await screen.findByTitle('View full payload')
    expect(preview.textContent?.length).toBeLessThan(70)

    await userEvent.click(preview)

    expect(screen.getByText('Collapse')).toBeInTheDocument()
    expect(screen.getByText(/a-very-long-tracking-identifier-that-is-quite-long/)).toBeInTheDocument()
  })

  it('shows a Received badge for a message with no dispatch handler', async () => {
    vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([webhookEvent({ status: 'received' })])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))

    expect(await screen.findByText('Received')).toBeInTheDocument()
  })

  it('shows a Processed badge when a dispatch handler ran successfully', async () => {
    vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([webhookEvent({ status: 'processed' })])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))

    expect(await screen.findByText('Processed')).toBeInTheDocument()
  })

  it('shows a Failed badge with the dispatch error in a tooltip', async () => {
    vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([
      webhookEvent({ status: 'failed', dispatch_error: 'myparcel unreachable' }),
    ])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))

    const badge = await screen.findByText('Failed')
    expect(badge).toHaveAttribute('title', 'myparcel unreachable')
  })

  it('Needs attention view shows only Failed/still-pending-dispatchable messages, with a count', async () => {
    vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([
      webhookEvent({ id: 'evt-failed', status: 'failed' }),
      webhookEvent({ id: 'evt-received-dispatchable', status: 'received' }),
      webhookEvent({ id: 'evt-processed', status: 'processed' }),
      webhookEvent({ id: 'evt-other-topic', provider: 'woocommerce', topic: 'order.updated', status: 'received' }),
    ])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))
    await screen.findByText('Received messages')

    expect(screen.getByRole('button', { name: 'Needs attention (2)' })).toBeInTheDocument()
    await userEvent.click(screen.getByRole('button', { name: 'Needs attention (2)' }))

    expect(await screen.findAllByRole('button', { name: 'Retry' })).toHaveLength(2)
    expect(screen.queryByText('Nothing needs attention.')).not.toBeInTheDocument()
  })

  it('Retry calls the API for that one message and updates its badge in place', async () => {
    vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([webhookEvent({ id: 'evt-1', status: 'failed', dispatch_error: 'myparcel unreachable' })])
    vi.mocked(integrationsApi.retryWebhookEventDispatch).mockResolvedValue(webhookEvent({ id: 'evt-1', status: 'processed', dispatch_error: '' }))
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))
    await screen.findByText('Failed')

    await userEvent.click(screen.getByRole('button', { name: 'Retry' }))

    expect(integrationsApi.retryWebhookEventDispatch).toHaveBeenCalledWith('test-token', 'evt-1')
    expect(await screen.findByText('Processed')).toBeInTheDocument()
    expect(screen.queryByText('Failed')).not.toBeInTheDocument()
  })

  it('Retry all retries every currently-filtered message sequentially', async () => {
    vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([
      webhookEvent({ id: 'evt-1', status: 'failed' }),
      webhookEvent({ id: 'evt-2', status: 'failed' }),
    ])
    vi.mocked(integrationsApi.retryWebhookEventDispatch).mockImplementation(async (_token, eventId) =>
      webhookEvent({ id: eventId, status: 'processed', dispatch_error: '' }),
    )
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Needs attention (2)' }))

    await userEvent.click(await screen.findByRole('button', { name: 'Retry all (2)' }))

    await screen.findByText('Nothing needs attention.')
    expect(integrationsApi.retryWebhookEventDispatch).toHaveBeenCalledWith('test-token', 'evt-1')
    expect(integrationsApi.retryWebhookEventDispatch).toHaveBeenCalledWith('test-token', 'evt-2')
  })

  it('a message that does not need attention has no Retry button', async () => {
    vi.mocked(integrationsApi.listWebhookEvents).mockResolvedValue([webhookEvent({ status: 'processed' })])
    render(<MemoryRouter><AppProvider><Webhooks /></AppProvider></MemoryRouter>)
    await userEvent.click(await screen.findByRole('button', { name: 'Messages' }))
    await screen.findByText('Processed')

    expect(screen.queryByRole('button', { name: 'Retry' })).not.toBeInTheDocument()
  })
})
