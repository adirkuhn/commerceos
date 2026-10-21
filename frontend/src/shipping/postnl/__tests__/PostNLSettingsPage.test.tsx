// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PostNLSettingsPage } from '../PostNLSettingsPage'
import { CodedApiError } from '../../../api/codedApiError'
import type { ShippingConnectionSettings } from '../../types'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: { user_id: 'u1', tenant_id: 'tenant-1', role: 'customer_owner', exp: 0, iat: 0 },
    login: vi.fn(), logout: vi.fn(), isAuthenticated: true,
  }),
}))

vi.mock('../../api', () => ({
  getPostNLConnection: vi.fn(), updatePostNLConnection: vi.fn(),
  getShippingSettings: vi.fn(), updateShippingSettings: vi.fn(),
}))
import * as shippingApi from '../../api'

const account = { connection_id: 'pn', customer_code: 'DEVC', customer_number: '11223344', sandbox: true }
const saved: ShippingConnectionSettings = {
  connection_id: 'pn', default_carrier: 1, default_package_type: 2, default_label_format: 'pdf', default_label_size: 'A6',
  default_signature_required: false, default_only_recipient: false, insure_by_default: false,
  myparcel_account_id: null, myparcel_shop_id: null, auto_print_enabled: false, export_order_to_myparcel: false,
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(shippingApi.getPostNLConnection).mockResolvedValue(account)
  vi.mocked(shippingApi.updatePostNLConnection).mockImplementation(async (_t, id, input) => ({
    connection_id: id, customer_code: input.customer_code, customer_number: input.customer_number, sandbox: input.sandbox,
  }))
  vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: saved, carrier_overrides: [] })
  vi.mocked(shippingApi.updateShippingSettings).mockImplementation(async (_t, id, input) => ({ connection_id: id, ...input }))
})

async function renderPage(onClose = vi.fn()) {
  render(<PostNLSettingsPage connectionId="pn" onClose={onClose} />)
  await screen.findByDisplayValue('DEVC')
  await waitFor(() => expect(screen.getByLabelText('Default package type')).toHaveValue('2'))
}

const account$ = () => screen.getByRole('region', { name: 'Account' })
const defaults$ = () => screen.getByRole('region', { name: 'Label defaults' })

describe('PostNLSettingsPage account', () => {
  it('loads the account and shows the sandbox badge', async () => {
    await renderPage()
    expect(within(account$()).getByLabelText('Customer number')).toHaveValue('11223344')
    expect(within(account$()).getByLabelText('Sandbox (test environment)')).toBeChecked()
    expect(within(account$()).getByLabelText('New API key')).toHaveValue('')
    expect(screen.getByTestId('sandbox-badge')).toHaveTextContent('Sandbox (test environment)')
  })

  it('going live with an empty key keeps the stored key', async () => {
    await renderPage()
    await userEvent.click(within(account$()).getByLabelText('Sandbox (test environment)'))
    await userEvent.click(screen.getByRole('button', { name: 'Save account' }))
    expect(shippingApi.updatePostNLConnection).toHaveBeenCalledWith('test-token', 'pn', {
      api_key: '', customer_code: 'DEVC', customer_number: '11223344', sandbox: false,
    })
    expect(await screen.findByText('Account saved.')).toBeInTheDocument()
    expect(screen.queryByTestId('sandbox-badge')).not.toBeInTheDocument()
  })

  it('sends a new API key and clears the field after saving', async () => {
    await renderPage()
    await userEvent.type(within(account$()).getByLabelText('New API key'), 'secret-key')
    await userEvent.click(screen.getByRole('button', { name: 'Save account' }))
    expect(shippingApi.updatePostNLConnection).toHaveBeenCalledWith('test-token', 'pn', expect.objectContaining({ api_key: 'secret-key' }))
    await waitFor(() => expect(within(account$()).getByLabelText('New API key')).toHaveValue(''))
  })

  it('shows why saving the account failed', async () => {
    vi.mocked(shippingApi.updatePostNLConnection).mockRejectedValue(new CodedApiError('POSTNL_ACCOUNT_INVALID', 'x', undefined))
    await renderPage()
    await userEvent.click(screen.getByRole('button', { name: 'Save account' }))
    expect(await within(account$()).findByRole('alert')).toHaveTextContent('PostNL needs an API key')
  })

  it('shows why the account could not load', async () => {
    vi.mocked(shippingApi.getPostNLConnection).mockRejectedValue(new CodedApiError('NOT_POSTNL_CONNECTION', 'x', undefined))
    render(<PostNLSettingsPage connectionId="pn" onClose={vi.fn()} />)
    expect(await screen.findByText('This is not a PostNL connection.')).toBeInTheDocument()
  })

  it('Back closes the page', async () => {
    const onClose = vi.fn()
    await renderPage(onClose)
    await userEvent.click(screen.getByRole('button', { name: 'Back' }))
    expect(onClose).toHaveBeenCalled()
  })
})

describe('PostNLSettingsPage label defaults', () => {
  it('offers only the package types PostNL supports', async () => {
    await renderPage()
    const options = within(screen.getByLabelText('Default package type')).getAllByRole('option').map(o => o.getAttribute('value'))
    expect(options).toEqual(['1', '2', '6', '7'])
  })

  it('saves the full settings object with the shown fields changed', async () => {
    await renderPage()
    const d = within(defaults$())
    await userEvent.selectOptions(d.getByLabelText('Label format'), 'zpl')
    await userEvent.selectOptions(d.getByLabelText('Default package type'), '6')
    await userEvent.click(d.getByLabelText('Signature required'))
    await userEvent.click(d.getByLabelText('Only recipient'))
    await userEvent.click(d.getByLabelText('Insure shipments by default'))
    await userEvent.click(d.getByLabelText('Auto-print labels'))
    await userEvent.click(screen.getByRole('button', { name: 'Save label defaults' }))
    expect(shippingApi.updateShippingSettings).toHaveBeenCalledWith('test-token', 'pn', {
      default_carrier: 1, default_package_type: 6, default_label_format: 'zpl', default_label_size: 'A6',
      default_signature_required: true, default_only_recipient: true, insure_by_default: true,
      myparcel_account_id: null, myparcel_shop_id: null, auto_print_enabled: true, export_order_to_myparcel: false,
    })
    expect(await screen.findByText('Label defaults saved.')).toBeInTheDocument()
  })

  it('shows why saving the label defaults failed', async () => {
    vi.mocked(shippingApi.updateShippingSettings).mockRejectedValue(new Error('settings down'))
    await renderPage()
    await userEvent.click(screen.getByRole('button', { name: 'Save label defaults' }))
    expect(await within(defaults$()).findByRole('alert')).toHaveTextContent('settings down')
  })

  it('shows why the label defaults could not load', async () => {
    vi.mocked(shippingApi.getShippingSettings).mockRejectedValue(new Error('no settings'))
    render(<PostNLSettingsPage connectionId="pn" onClose={vi.fn()} />)
    expect(await screen.findByText('no settings')).toBeInTheDocument()
  })
})
