// @vitest-environment jsdom
import { render, screen, fireEvent } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import MyParcelSyncSettingsPage from '../MyParcelSyncSettingsPage'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: { user_id: 'u1', tenant_id: 'tenant-1', role: 'customer_owner', exp: 0, iat: 0 },
    login: vi.fn(), logout: vi.fn(), isAuthenticated: true,
  }),
}))

vi.mock('../api', () => ({
  connectMyParcel: vi.fn(),
  getShippingSettings: vi.fn(),
  updateShippingSettings: vi.fn(),
  upsertCarrierOverride: vi.fn(),
}))

vi.mock('../../integrations/api', () => ({
  setWebhookSecret: vi.fn(),
  listProviderWebhooks: vi.fn(),
  createProviderWebhook: vi.fn(),
  deleteProviderWebhook: vi.fn(),
}))

import * as api from '../api'
import * as integrationsApi from '../../integrations/api'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.listProviderWebhooks).mockResolvedValue([])
})

describe('MyParcelSyncSettingsPage -- connect step', () => {
  it('submits the api key and calls onConnected on success', async () => {
    vi.mocked(api.connectMyParcel).mockResolvedValue({ connection_id: 'conn-mp-1', provider: 'myparcel', status: 'connected' })
    vi.mocked(api.getShippingSettings).mockResolvedValue({ settings: null, carrier_overrides: [] })
    const onConnected = vi.fn()
    render(<MyParcelSyncSettingsPage onClose={vi.fn()} onConnected={onConnected} />)

    await userEvent.type(screen.getByLabelText('API key'), 'api-key-1')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(api.connectMyParcel).toHaveBeenCalledWith('test-token', 'api-key-1')
    expect(onConnected).toHaveBeenCalled()
  })

  it('shows an error and does not call onConnected when the API call fails', async () => {
    vi.mocked(api.connectMyParcel).mockRejectedValue(new Error('invalid api key'))
    const onConnected = vi.fn()
    render(<MyParcelSyncSettingsPage onClose={vi.fn()} onConnected={onConnected} />)

    await userEvent.type(screen.getByLabelText('API key'), 'bad-key')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText('invalid api key')).toBeInTheDocument()
    expect(onConnected).not.toHaveBeenCalled()
  })

  it('calls onClose when the back button is clicked', async () => {
    const onClose = vi.fn()
    render(<MyParcelSyncSettingsPage onClose={onClose} onConnected={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Back' }))

    expect(onClose).toHaveBeenCalled()
  })
})

describe('MyParcelSyncSettingsPage -- already-connected settings', () => {
  it('loads and displays existing settings, and saves edits', async () => {
    vi.mocked(api.getShippingSettings).mockResolvedValue({
      settings: {
        connection_id: 'conn-mp-1', default_carrier: 1, default_package_type: 1,
        default_label_format: 'pdf', default_signature_required: false, default_only_recipient: false,
        insure_by_default: false, myparcel_account_id: null, myparcel_shop_id: null, auto_print_enabled: false,
      },
      carrier_overrides: [],
    })
    vi.mocked(api.updateShippingSettings).mockResolvedValue({
      connection_id: 'conn-mp-1', default_carrier: 4, default_package_type: 1,
      default_label_format: 'pdf', default_signature_required: false, default_only_recipient: false,
      insure_by_default: false, myparcel_account_id: null, myparcel_shop_id: null, auto_print_enabled: false,
    })
    render(<MyParcelSyncSettingsPage connectionId="conn-mp-1" onClose={vi.fn()} />)

    expect(await screen.findByLabelText('Default carrier')).toHaveValue('1')

    await userEvent.selectOptions(screen.getByLabelText('Default carrier'), '4')
    await userEvent.click(screen.getByRole('button', { name: 'Save defaults' }))

    expect(api.updateShippingSettings).toHaveBeenCalledWith('test-token', 'conn-mp-1', expect.objectContaining({ default_carrier: 4 }))
    expect(await screen.findByText('Saved')).toBeInTheDocument()
  })

  it('saves a per-carrier override independently of the global form', async () => {
    vi.mocked(api.getShippingSettings).mockResolvedValue({ settings: null, carrier_overrides: [] })
    vi.mocked(api.upsertCarrierOverride).mockResolvedValue({
      connection_id: 'conn-mp-1', carrier: 1, package_type: null, signature_required: true, only_recipient: null,
      insure_by_default: null,
    })
    render(<MyParcelSyncSettingsPage connectionId="conn-mp-1" onClose={vi.fn()} />)

    await screen.findByText('Per-carrier overrides')
    await userEvent.click(screen.getByLabelText('PostNL signature required override'))

    expect(api.upsertCarrierOverride).toHaveBeenCalledWith('test-token', 'conn-mp-1', 1, expect.objectContaining({ signature_required: true }))
  })

  it('saves insure-by-default, MyParcel account/shop id, and auto-print together', async () => {
    vi.mocked(api.getShippingSettings).mockResolvedValue({
      settings: {
        connection_id: 'conn-mp-1', default_carrier: 1, default_package_type: 1,
        default_label_format: 'pdf', default_signature_required: false, default_only_recipient: false,
        insure_by_default: false, myparcel_account_id: null, myparcel_shop_id: null, auto_print_enabled: false,
      },
      carrier_overrides: [],
    })
    vi.mocked(api.updateShippingSettings).mockResolvedValue({
      connection_id: 'conn-mp-1', default_carrier: 1, default_package_type: 1,
      default_label_format: 'pdf', default_signature_required: true, default_only_recipient: true,
      insure_by_default: true, myparcel_account_id: 111, myparcel_shop_id: 222, auto_print_enabled: true,
    })
    render(<MyParcelSyncSettingsPage connectionId="conn-mp-1" onClose={vi.fn()} />)
    await screen.findByLabelText('Default carrier')

    await userEvent.click(screen.getByLabelText('Insure shipments by default'))
    fireEvent.change(screen.getByLabelText('Account ID'), { target: { value: '111' } })
    fireEvent.change(screen.getByLabelText('Shop ID'), { target: { value: '222' } })
    await userEvent.click(screen.getByLabelText('Auto-print labels'))
    await userEvent.click(screen.getByRole('button', { name: 'Save defaults' }))

    expect(api.updateShippingSettings).toHaveBeenCalledWith('test-token', 'conn-mp-1', expect.objectContaining({
      insure_by_default: true, myparcel_account_id: 111, myparcel_shop_id: 222, auto_print_enabled: true,
    }))
  })

  it('saves a per-carrier insure-by-default override', async () => {
    vi.mocked(api.getShippingSettings).mockResolvedValue({ settings: null, carrier_overrides: [] })
    vi.mocked(api.upsertCarrierOverride).mockResolvedValue({
      connection_id: 'conn-mp-1', carrier: 1, package_type: null, signature_required: null, only_recipient: null,
      insure_by_default: true,
    })
    render(<MyParcelSyncSettingsPage connectionId="conn-mp-1" onClose={vi.fn()} />)

    await screen.findByText('Per-carrier overrides')
    await userEvent.click(screen.getByLabelText('PostNL insure by default override'))

    expect(api.upsertCarrierOverride).toHaveBeenCalledWith('test-token', 'conn-mp-1', 1, expect.objectContaining({ insure_by_default: true }))
  })
})

describe('MyParcelSyncSettingsPage -- Webhooks section', () => {
  it('shows the delivery URL, no secret by default, and the provider webhooks panel', async () => {
    vi.mocked(api.getShippingSettings).mockResolvedValue({ settings: null, carrier_overrides: [] })
    render(<MyParcelSyncSettingsPage connectionId="conn-mp-1" onClose={vi.fn()} />)

    await screen.findByText('Per-carrier overrides')

    expect(screen.getByText(/webhooks\/v1\/tenant-1\/myparcel/)).toBeInTheDocument()
    expect(screen.getByText(/No secret generated yet/)).toBeInTheDocument()
    expect(screen.getByText('Provider webhook subscriptions')).toBeInTheDocument()
  })

  it('generating a secret calls setWebhookSecret and shows the plaintext value once', async () => {
    vi.mocked(api.getShippingSettings).mockResolvedValue({ settings: null, carrier_overrides: [] })
    vi.mocked(integrationsApi.setWebhookSecret).mockResolvedValue({ connection_id: 'conn-mp-1', webhook_secret: 'top-secret-value' })
    render(<MyParcelSyncSettingsPage connectionId="conn-mp-1" onClose={vi.fn()} />)
    await screen.findByText('Per-carrier overrides')

    await userEvent.click(screen.getByRole('button', { name: 'Generate secret' }))

    expect(await screen.findByText('top-secret-value')).toBeInTheDocument()
    expect(integrationsApi.setWebhookSecret).toHaveBeenCalledWith('test-token', 'conn-mp-1')
  })

  it('Create registers a new provider webhook subscription', async () => {
    vi.mocked(api.getShippingSettings).mockResolvedValue({ settings: null, carrier_overrides: [] })
    vi.mocked(integrationsApi.createProviderWebhook).mockResolvedValue({
      id: '1', provider: 'myparcel', topic: 'shipment_status_change', delivery_url: 'https://commerceos.example.com/hook?token=abc', status: 'active',
    })
    render(<MyParcelSyncSettingsPage connectionId="conn-mp-1" onClose={vi.fn()} />)
    await screen.findByText('Per-carrier overrides')

    await userEvent.type(screen.getByLabelText('Topic'), 'shipment_status_change')
    await userEvent.click(screen.getByRole('button', { name: 'Create' }))

    expect(integrationsApi.createProviderWebhook).toHaveBeenCalledWith('test-token', 'conn-mp-1', 'shipment_status_change')
    expect(await screen.findByText('shipment_status_change')).toBeInTheDocument()
  })

  it('focusWebhooks scrolls the Webhooks section into view once settings finish loading', async () => {
    vi.mocked(api.getShippingSettings).mockResolvedValue({ settings: null, carrier_overrides: [] })
    // jsdom doesn't implement scrollIntoView at all (not even a stub) --
    // assign one directly rather than vi.spyOn, which requires the
    // property to already exist.
    const scrollIntoView = vi.fn()
    HTMLDivElement.prototype.scrollIntoView = scrollIntoView

    render(<MyParcelSyncSettingsPage connectionId="conn-mp-1" focusWebhooks onClose={vi.fn()} />)
    await screen.findByText('Per-carrier overrides')

    expect(scrollIntoView).toHaveBeenCalledWith({ block: 'start' })
  })
})
