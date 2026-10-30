// @vitest-environment jsdom
import { screen } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ShippingSettingsSection } from '../ShippingSettingsSection'
import { renderAt } from '../../test/renderAt'
import * as shippingApi from '../../shipping/api'

let mockRole = 'customer_owner'
vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 't', user: { role: mockRole } }) }))
vi.mock('../../shipping/api', async importOriginal => ({
  ...await importOriginal<typeof import('../../shipping/api')>(),
  listShippingConnections: vi.fn(),
  getShippingAddresses: vi.fn().mockResolvedValue({ sender_address: null, return_address: null, company_address: null }),
}))

const withShipping = () => vi.mocked(shippingApi.listShippingConnections).mockResolvedValue({
  connections: [{ id: 'mp', provider: 'myparcel', status: 'active', created_at: '2026-01-01T00:00:00Z', is_default: true, missing: [] }],
  default_connection_id: 'mp', configured_connection_id: null,
})

beforeEach(() => {
  vi.clearAllMocks()
  mockRole = 'customer_owner'
  vi.mocked(shippingApi.listShippingConnections).mockResolvedValue({ connections: [], default_connection_id: null, configured_connection_id: null })
})

describe('ShippingSettingsSection', () => {
  it('holds the default shipping provider and the shipping addresses', async () => {
    withShipping()
    renderAt(<ShippingSettingsSection />, '/settings/shipping')
    expect(await screen.findByLabelText('Default shipping provider')).toBeEnabled()
    expect(await screen.findByRole('region', { name: 'Shipping addresses' })).toBeInTheDocument()
  })

  it('is read-only for a non-admin', async () => {
    mockRole = 'customer_user'
    withShipping()
    renderAt(<ShippingSettingsSection />, '/settings/shipping')
    expect(await screen.findByLabelText('Default shipping provider')).toBeDisabled()
  })

  it('without a shipping connection, points to Integrations', async () => {
    renderAt(<ShippingSettingsSection />, '/settings/shipping')
    expect(await screen.findByRole('link', { name: 'Connect a shipping provider' })).toHaveAttribute('href', '/integrations/connect')
    expect(screen.queryByLabelText('Default shipping provider')).not.toBeInTheDocument()
  })
})
