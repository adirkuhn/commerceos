// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi } from 'vitest'
import { AppShell } from '../App'

vi.mock('../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: { role: 'customer_admin' }, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

// DateFormatProvider loads the tenant date format on mount; without this
// mock that is a real network request to the API base URL.
vi.mock('../settings/api', async importOriginal => ({
  ...await importOriginal<typeof import('../settings/api')>(),
  getTenantSettings: vi.fn().mockRejectedValue(new Error('no settings in tests')),
}))

// AppearanceProvider loads the shop's appearance on mount.
vi.mock('../settings/appearance/api', () => ({
  getAppearance: vi.fn().mockRejectedValue(new Error('no appearance in tests')),
  updateAppearance: vi.fn(),
}))

vi.mock('../scanner/api', () => ({
  generatePairingCode: vi.fn(),
  getPairingCodeStatus: vi.fn(),
}))

vi.mock('../notifications/hooks/useNotifications', () => ({
  useNotifications: () => ({ notifications: [], unreadCount: 0, loading: false, error: null, refresh: vi.fn(), markRead: vi.fn(), markAllRead: vi.fn() }),
}))

vi.mock('../pms/hooks/useProducts', () => ({
  useProducts: () => ({ products: [], total: 0, loading: false, error: null, refresh: vi.fn() }),
}))

vi.mock('../dashboard/TodayPage', () => ({
  default: () => <div>TODAY_PAGE</div>,
}))

vi.mock('../pages/SalesOrders', () => ({
  default: () => <div>SALES_ORDERS_PAGE</div>,
}))

// Regression test for: refreshing the browser used to always drop the user
// back on Dashboard because "current page" only lived in React state
// (AppContext's activeNav), never in the URL. Rendering AppShell directly at
// a given URL (MemoryRouter's initialEntries) simulates a hard refresh on
// that page -- the app must show that page, not fall back to Dashboard.
describe('AppShell routing', () => {
  it('shows the page matching the current URL on load (refresh-equivalent), not always Dashboard', () => {
    render(
      <MemoryRouter initialEntries={['/sales-orders']}>
        <AppShell />
      </MemoryRouter>,
    )

    expect(screen.getByText('SALES_ORDERS_PAGE')).toBeInTheDocument()
    expect(screen.queryByText('TODAY_PAGE')).not.toBeInTheDocument()
  })

  it('shows Today at the root path', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <AppShell />
      </MemoryRouter>,
    )

    expect(screen.getByText('TODAY_PAGE')).toBeInTheDocument()
  })

  it('updates the URL when a sidebar nav item is clicked', async () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <AppShell />
      </MemoryRouter>,
    )

    await userEvent.click(screen.getByRole('button', { name: 'Sales Orders' }))

    expect(await screen.findByText('SALES_ORDERS_PAGE')).toBeInTheDocument()
    expect(screen.queryByText('TODAY_PAGE')).not.toBeInTheDocument()
  })

  it('shows a not-found page instead of crashing on an unknown path', () => {
    render(
      <MemoryRouter initialEntries={['/this-page-does-not-exist']}>
        <AppShell />
      </MemoryRouter>,
    )

    expect(screen.getByText(/page not found/i)).toBeInTheDocument()
  })
})

describe('AppShell keyboard shortcuts', () => {
  it('jumps to a page with g then its letter', async () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <AppShell />
      </MemoryRouter>,
    )

    await userEvent.keyboard('go')

    expect(await screen.findByText('SALES_ORDERS_PAGE')).toBeInTheDocument()
  })
})
