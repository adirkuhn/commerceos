// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import WooCommerceSyncSettingsPage from '../WooCommerceSyncSettingsPage'
import type { Connection } from '../types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: { user_id: 'u1', tenant_id: 'tenant-1', role: 'customer_owner', exp: 0, iat: 0 },
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

async function openWebhooksTab() {
  render(<WooCommerceSyncSettingsPage connection={connection} onClose={vi.fn()} />)
  await userEvent.click(await screen.findByRole('button', { name: 'Webhooks' }))
}

describe('WooCommerceSyncSettingsPage — Webhooks tab', () => {
  it('shows a Webhooks tab for an owner and no secret generated yet by default', async () => {
    await openWebhooksTab()

    expect(screen.getByText(/No secret generated yet/)).toBeInTheDocument()
    expect(screen.getByText(/webhooks\/v1\/tenant-1\/woocommerce/)).toBeInTheDocument()
  })

  it('generating a secret calls the endpoint and shows the plaintext value once', async () => {
    vi.mocked(api.setWebhookSecret).mockResolvedValue({ connection_id: 'conn-1', webhook_secret: 'top-secret-value' })
    await openWebhooksTab()

    await userEvent.click(screen.getByRole('button', { name: 'Generate secret' }))

    expect(await screen.findByText('top-secret-value')).toBeInTheDocument()
    expect(api.setWebhookSecret).toHaveBeenCalledWith('test-token', 'conn-1')
    expect(screen.getByRole('button', { name: 'Regenerate secret' })).toBeInTheDocument()
  })

  it('surfaces an error inline if generating fails, without swallowing it', async () => {
    vi.mocked(api.setWebhookSecret).mockRejectedValue(new Error('connection not found'))
    await openWebhooksTab()

    await userEvent.click(screen.getByRole('button', { name: 'Generate secret' }))

    expect(await screen.findByText('connection not found')).toBeInTheDocument()
  })
})
