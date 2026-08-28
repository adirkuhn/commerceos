// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi } from 'vitest'
import Webhooks from '../Webhooks'

// Separate file (its own top-level vi.mock, hoisted per-file) so this
// customer_user role doesn't collide with Webhooks.test.tsx's
// customer_owner mock -- same reasoning as
// WooCommerceSyncSettingsPage.webhooksRbac.test.tsx.
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: { user_id: 'u1', tenant_id: 'tenant-1', role: 'customer_user', exp: 0, iat: 0 },
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
}))

describe('Webhooks page — RBAC', () => {
  it('shows a permission message instead of the tabs for a plain customer_user', async () => {
    render(<Webhooks />)

    expect(await screen.findByText(/don't have permission/)).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Fan-out' })).not.toBeInTheDocument()
  })
})
