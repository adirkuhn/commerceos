// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
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
  return render(<MemoryRouter><AppProvider><Topbar /></AppProvider></MemoryRouter>)
}

describe('Topbar Pair Scanner', () => {
  it('generates a pairing code and shows it -- reachable from every page, not just Procurement', async () => {
    vi.mocked(scannerApi.generatePairingCode).mockResolvedValue({
      id: 'pairing-1', tenant_id: 'tenant-1', code: '654321', token: 'opaque-abc', status: 'pending',
      expires_at: '', created_at: '',
    })

    renderTopbar()
    await userEvent.click(screen.getByTitle('Pair scanner'))

    expect(await screen.findByRole('img', { name: 'Scan to pair' })).toBeInTheDocument()
    expect(screen.getByText('Waiting for scanner…')).toBeInTheDocument()
  })

  it('closes the modal', async () => {
    vi.mocked(scannerApi.generatePairingCode).mockResolvedValue({
      id: 'pairing-1', tenant_id: 'tenant-1', code: '654321', token: 'opaque-abc', status: 'pending',
      expires_at: '', created_at: '',
    })

    renderTopbar()
    await userEvent.click(screen.getByTitle('Pair scanner'))
    await screen.findByRole('img', { name: 'Scan to pair' })
    await userEvent.click(screen.getByTitle('Close'))

    expect(screen.queryByRole('img', { name: 'Scan to pair' })).not.toBeInTheDocument()
  })
})

describe('Topbar appearance control', () => {
  it('is labelled with the current appearance and cycles System → Light → Dark', async () => {
    localStorage.clear()
    renderTopbar()

    await userEvent.click(screen.getByRole('button', { name: 'Appearance: System' }))
    expect(screen.getByRole('button', { name: 'Appearance: Light' })).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Appearance: Light' }))
    expect(screen.getByRole('button', { name: 'Appearance: Dark' })).toBeInTheDocument()
    expect(document.documentElement.classList.contains('dark')).toBe(true)
  })

  it('labels the Pair Scanner button for screen readers', () => {
    renderTopbar()
    expect(screen.getByRole('button', { name: 'Pair scanner' })).toBeInTheDocument()
  })
})

describe('Topbar title', () => {
  it('names the current page the way the sidebar does', () => {
    render(<MemoryRouter initialEntries={['/reports/sales-trends']}><AppProvider><Topbar /></AppProvider></MemoryRouter>)
    expect(screen.getByRole('heading', { level: 1, name: 'Reports: Sales Trends' })).toBeInTheDocument()
  })

  it('shows no made-up system status', () => {
    renderTopbar()
    expect(screen.queryByText('All systems live')).not.toBeInTheDocument()
  })
})

describe('Topbar amounts toggle', () => {
  it('starts with amounts hidden and reveals them all on click', async () => {
    const { AmountVisibilityProvider, Money } = await import('../../../format/amountVisibility')
    render(
      <MemoryRouter><AppProvider><AmountVisibilityProvider>
        <Topbar /><Money minor={1250} currency="EUR" />
      </AmountVisibilityProvider></AppProvider></MemoryRouter>,
    )
    const toggle = screen.getByRole('button', { name: 'Show amounts' })
    expect(toggle).toHaveAttribute('aria-pressed', 'false')
    expect(screen.getByRole('button', { name: 'Show amount' })).toBeInTheDocument()

    await userEvent.click(toggle)

    expect(screen.getByRole('button', { name: 'Hide amounts' })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.queryByRole('button', { name: 'Show amount' })).toBeNull()
  })
})
