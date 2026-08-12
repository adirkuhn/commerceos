// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Topbar from '../Topbar'
import { AppProvider } from '../../../context/AppContext'

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../scanner/api', () => ({
  generatePairingCode: vi.fn(),
  getPairingCodeStatus: vi.fn(),
}))

vi.mock('../../../notifications/hooks/useNotifications', () => ({
  useNotifications: () => ({ notifications: [], unreadCount: 0, loading: false, error: null, refresh: vi.fn(), markRead: vi.fn(), markAllRead: vi.fn() }),
}))

import * as scannerApi from '../../../scanner/api'

beforeEach(() => { vi.clearAllMocks() })

function renderTopbar() {
  return render(<AppProvider><Topbar /></AppProvider>)
}

describe('Topbar Pair Scanner', () => {
  it('generates a pairing code and shows it -- reachable from every page, not just Procurement', async () => {
    vi.mocked(scannerApi.generatePairingCode).mockResolvedValue({
      id: 'pairing-1', tenant_id: 'tenant-1', code: '654321', token: 'opaque-abc', status: 'pending',
      expires_at: '', created_at: '',
    })

    renderTopbar()
    await userEvent.click(screen.getByTitle('Pair Scanner'))

    expect(await screen.findByRole('img', { name: 'Scan to pair' })).toBeInTheDocument()
    expect(screen.getByText('Waiting for scanner…')).toBeInTheDocument()
  })

  it('closes the modal', async () => {
    vi.mocked(scannerApi.generatePairingCode).mockResolvedValue({
      id: 'pairing-1', tenant_id: 'tenant-1', code: '654321', token: 'opaque-abc', status: 'pending',
      expires_at: '', created_at: '',
    })

    renderTopbar()
    await userEvent.click(screen.getByTitle('Pair Scanner'))
    await screen.findByRole('img', { name: 'Scan to pair' })
    await userEvent.click(screen.getByTitle('Close'))

    expect(screen.queryByRole('img', { name: 'Scan to pair' })).not.toBeInTheDocument()
  })
})
