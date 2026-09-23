// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import BolSyncSettingsPage from '../BolSyncSettingsPage'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: { user_id: 'u1', tenant_id: 'tenant-1', role: 'customer_owner', exp: 0, iat: 0 },
    login: vi.fn(), logout: vi.fn(), isAuthenticated: true,
  }),
}))

vi.mock('../api', () => ({
  connectBol: vi.fn(),
  listSyncSchedules: vi.fn(),
  upsertSyncSchedules: vi.fn(),
  updateConnectionSettings: vi.fn(),
  listBolEconomicOperators: vi.fn(),
}))

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
}))

import * as api from '../api'
import * as inventoryApi from '../../inventory/api'

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listSyncSchedules).mockResolvedValue([])
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([])
})

describe('BolSyncSettingsPage', () => {
  it('submits client_id/client_secret/economic_operator_id and calls onConnected on success', async () => {
    vi.mocked(api.connectBol).mockResolvedValue({
      id: 'conn-bol-1', provider: 'bol', store_url: '', status: 'connected', created_at: '',
    })
    const onConnected = vi.fn()
    render(<BolSyncSettingsPage onClose={vi.fn()} onConnected={onConnected} />)

    await userEvent.type(screen.getByLabelText('Client ID'), 'client-1')
    await userEvent.type(screen.getByLabelText('Client secret'), 'secret-1')
    await userEvent.type(screen.getByLabelText('Economic operator ID'), 'op-1')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(api.connectBol).toHaveBeenCalledWith('test-token', {
      clientId: 'client-1', clientSecret: 'secret-1', economicOperatorId: 'op-1',
    })
    expect(onConnected).toHaveBeenCalled()
  })

  it('shows an error and does not call onConnected when the API call fails', async () => {
    vi.mocked(api.connectBol).mockRejectedValue(new Error('invalid credentials'))
    const onConnected = vi.fn()
    render(<BolSyncSettingsPage onClose={vi.fn()} onConnected={onConnected} />)

    await userEvent.type(screen.getByLabelText('Client ID'), 'client-1')
    await userEvent.type(screen.getByLabelText('Client secret'), 'secret-1')
    await userEvent.type(screen.getByLabelText('Economic operator ID'), 'op-1')
    await userEvent.click(screen.getByRole('button', { name: 'Connect' }))

    expect(await screen.findByText('invalid credentials')).toBeInTheDocument()
    expect(onConnected).not.toHaveBeenCalled()
  })

  it('calls onClose when the back button is clicked', async () => {
    const onClose = vi.fn()
    render(<BolSyncSettingsPage onClose={onClose} onConnected={vi.fn()} />)

    await userEvent.click(screen.getByRole('button', { name: 'Integrations' }))

    expect(onClose).toHaveBeenCalled()
  })

  it('shows the sync schedule grid instead of the connect form when connectionId is set', async () => {
    render(<BolSyncSettingsPage connectionId="conn-bol-1" onClose={vi.fn()} onConnected={vi.fn()} />)

    expect(await screen.findByText('Sync schedule')).toBeInTheDocument()
    expect(screen.queryByLabelText('Client ID')).not.toBeInTheDocument()
    expect(screen.getByLabelText('Orders import')).toBeInTheDocument()
    expect(screen.getByLabelText('Products export')).toBeInTheDocument()
    expect(screen.getByLabelText('Stock levels export')).toBeInTheDocument()
    expect(api.listSyncSchedules).toHaveBeenCalledWith('test-token', 'conn-bol-1')
  })

  it('disables the non-runnable direction toggle for each row', async () => {
    render(<BolSyncSettingsPage connectionId="conn-bol-1" onClose={vi.fn()} onConnected={vi.fn()} />)

    await screen.findByText('Sync schedule')
    expect(screen.getByLabelText('Orders export')).toBeDisabled()
    expect(screen.getByLabelText('Products import')).toBeDisabled()
    expect(screen.getByLabelText('Stock levels import')).toBeDisabled()
  })

  it('saving sends all three rows with only the runnable direction enableable', async () => {
    vi.mocked(api.upsertSyncSchedules).mockResolvedValue([])
    render(<BolSyncSettingsPage connectionId="conn-bol-1" onClose={vi.fn()} onConnected={vi.fn()} />)

    await screen.findByText('Sync schedule')
    await userEvent.click(screen.getByLabelText('Orders import'))
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(api.upsertSyncSchedules).toHaveBeenCalled())
    const payload = vi.mocked(api.upsertSyncSchedules).mock.calls[0][2]
    expect(payload).toHaveLength(6)
    const ordersImport = payload.find(s => s.entity_type === 'bol_orders' && s.direction === 'import')
    expect(ordersImport?.enabled).toBe(true)
    const ordersExport = payload.find(s => s.entity_type === 'bol_orders' && s.direction === 'export')
    expect(ordersExport?.enabled).toBe(false)
  })

  it('prefills the configuration section from initialConnection and saves edits', async () => {
    vi.mocked(api.updateConnectionSettings).mockResolvedValue({
      id: 'conn-bol-1', provider: 'bol', store_url: '', status: 'connected', created_at: '',
      bol_economic_operator_id: '3fa85f64-5717-4562-b3fc-2c963f66afa6', bol_delivery_code: '1-2d',
    })
    render(
      <BolSyncSettingsPage
        connectionId="conn-bol-1"
        initialConnection={{
          id: 'conn-bol-1', provider: 'bol', store_url: '', status: 'connected', created_at: '',
          bol_economic_operator_id: 'old-op-id', bol_delivery_code: '2-3d',
        }}
        onClose={vi.fn()} onConnected={vi.fn()}
      />,
    )

    await screen.findByText('Sync schedule')
    expect(screen.getByLabelText('Economic operator ID')).toHaveValue('old-op-id')
    expect(screen.getByLabelText('Delivery promise')).toHaveValue('2-3d')

    await userEvent.clear(screen.getByLabelText('Economic operator ID'))
    await userEvent.type(screen.getByLabelText('Economic operator ID'), '3fa85f64-5717-4562-b3fc-2c963f66afa6')
    await userEvent.selectOptions(screen.getByLabelText('Delivery promise'), '1-2d')
    await userEvent.click(screen.getByRole('button', { name: 'Save configuration' }))

    await waitFor(() => expect(api.updateConnectionSettings).toHaveBeenCalledWith(
      'test-token', 'conn-bol-1', '', null, '3fa85f64-5717-4562-b3fc-2c963f66afa6', '1-2d',
    ))
    expect(await screen.findByText('Saved')).toBeInTheDocument()
  })

  it('fetching from Bol shows a select populated from the result, even for a single operator', async () => {
    vi.mocked(api.listBolEconomicOperators).mockResolvedValue([
      { id: '3fa85f64-5717-4562-b3fc-2c963f66afa6', name: 'Cantinho Brasileiro', status: 'VALID' },
    ])
    render(<BolSyncSettingsPage connectionId="conn-bol-1" onClose={vi.fn()} onConnected={vi.fn()} />)

    await screen.findByText('Sync schedule')
    await userEvent.click(screen.getByRole('button', { name: 'Fetch from Bol' }))

    await waitFor(() => expect(api.listBolEconomicOperators).toHaveBeenCalledWith('test-token', 'conn-bol-1'))
    const select = await screen.findByLabelText('Select from Bol')
    expect(screen.getByText(/Cantinho Brasileiro/)).toBeInTheDocument()
    await userEvent.selectOptions(select, '3fa85f64-5717-4562-b3fc-2c963f66afa6')
    expect(screen.getByLabelText('Economic operator ID')).toHaveValue('3fa85f64-5717-4562-b3fc-2c963f66afa6')
  })

  it('fetching from Bol lets picking among multiple operators via the select', async () => {
    vi.mocked(api.listBolEconomicOperators).mockResolvedValue([
      { id: 'op-1', name: 'Operator One', status: 'VALID' },
      { id: 'op-2', name: 'Operator Two', status: 'PENDING' },
    ])
    render(<BolSyncSettingsPage connectionId="conn-bol-1" onClose={vi.fn()} onConnected={vi.fn()} />)

    await screen.findByText('Sync schedule')
    await userEvent.click(screen.getByRole('button', { name: 'Fetch from Bol' }))

    const select = await screen.findByLabelText('Select from Bol')
    expect(screen.getByText(/Operator One/)).toBeInTheDocument()
    expect(screen.getByText(/Operator Two/)).toBeInTheDocument()
    await userEvent.selectOptions(select, 'op-2')
    expect(screen.getByLabelText('Economic operator ID')).toHaveValue('op-2')
  })

  it('fetching from Bol shows an error and leaves manual entry available on failure', async () => {
    vi.mocked(api.listBolEconomicOperators).mockRejectedValue(new Error('missing scope economic-operators.read'))
    render(<BolSyncSettingsPage connectionId="conn-bol-1" onClose={vi.fn()} onConnected={vi.fn()} />)

    await screen.findByText('Sync schedule')
    await userEvent.click(screen.getByRole('button', { name: 'Fetch from Bol' }))

    expect(await screen.findByText('missing scope economic-operators.read')).toBeInTheDocument()
    expect(screen.getByLabelText('Economic operator ID')).not.toBeDisabled()
  })

  it('prefills an already-saved schedule', async () => {
    vi.mocked(api.listSyncSchedules).mockResolvedValue([
      {
        entity_type: 'bol_products', direction: 'export', enabled: true, interval_minutes: 30,
        last_run_status: 'success', last_run_error: '', last_run_at: '2026-01-01T00:00:00Z',
      },
    ])
    render(<BolSyncSettingsPage connectionId="conn-bol-1" onClose={vi.fn()} onConnected={vi.fn()} />)

    expect(await screen.findByLabelText('Products export')).toBeChecked()
    expect(screen.getByText(/Last run:/)).toBeInTheDocument()
  })
})
