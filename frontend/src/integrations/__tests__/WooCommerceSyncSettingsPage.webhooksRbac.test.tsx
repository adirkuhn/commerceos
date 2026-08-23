// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import WooCommerceSyncSettingsPage from '../WooCommerceSyncSettingsPage'
import type { Connection } from '../types'

// Separate file (its own top-level vi.mock, hoisted per-file) so this
// customer_user role doesn't collide with the customer_owner mock in
// WooCommerceSyncSettingsPage.webhooks.test.tsx.
vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: { user_id: 'u1', tenant_id: 'tenant-1', role: 'customer_user', exp: 0, iat: 0 },
    login: vi.fn(), logout: vi.fn(), isAuthenticated: true,
  }),
}))

vi.mock('../api', () => ({
  listSyncSchedules: vi.fn(),
  upsertSyncSchedules: vi.fn(),
  updateConnectionSettings: vi.fn(),
  setWebhookSecret: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
}))

vi.mock('../../settings/api', () => ({
  getTenantSettings: vi.fn(),
}))

import * as api from '../api'
import * as inventoryApi from '../../inventory/api'
import * as tenantSettingsApi from '../../settings/api'

const connection: Connection = { id: 'conn-1', provider: 'woocommerce', store_url: 'https://mystore.com', status: 'connected', created_at: '' }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listSyncSchedules).mockResolvedValue([])
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
  vi.mocked(tenantSettingsApi.getTenantSettings).mockResolvedValue({
    tenant_id: 'tenant-1', default_currency: '', timezone: '', date_format: '', language: '',
    prices_include_tax: false, default_tax_country: '', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0, updated_at: '',
  })
})

describe('WooCommerceSyncSettingsPage — Webhooks tab RBAC', () => {
  it('is hidden for a plain customer_user', async () => {
    render(<WooCommerceSyncSettingsPage connection={connection} onClose={vi.fn()} />)

    expect(await screen.findByRole('button', { name: 'Sync' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Webhooks' })).not.toBeInTheDocument()
  })
})
