// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
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
}))

import * as api from '../api'

beforeEach(() => {
  vi.clearAllMocks()
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

    await userEvent.click(screen.getByRole('button', { name: 'Back' }))

    expect(onClose).toHaveBeenCalled()
  })
})
