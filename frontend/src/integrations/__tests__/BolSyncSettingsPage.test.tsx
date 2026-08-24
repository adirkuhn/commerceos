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
    await userEvent.type(screen.getByLabelText('Client Secret'), 'secret-1')
    await userEvent.type(screen.getByLabelText('Economic Operator ID'), 'op-1')
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
    await userEvent.type(screen.getByLabelText('Client Secret'), 'secret-1')
    await userEvent.type(screen.getByLabelText('Economic Operator ID'), 'op-1')
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
