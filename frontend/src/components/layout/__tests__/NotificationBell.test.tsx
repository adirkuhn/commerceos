// @vitest-environment jsdom
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MemoryRouter } from 'react-router-dom'
import NotificationBell from '../NotificationBell'
import type { Notification } from '../../../notifications/types'

// NotificationBell navigates via react-router's useNavigate on a linked
// notification click -- needs a Router in the tree even for tests that
// never click a linked one.
function renderBell() {
  return render(<NotificationBell />, { wrapper: MemoryRouter })
}

vi.mock('../../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../../notifications/api', () => ({
  listNotifications: vi.fn(),
  markNotificationRead: vi.fn(),
  markAllNotificationsRead: vi.fn(),
}))

import * as notificationsApi from '../../../notifications/api'
import { notifyNotificationsChanged } from '../../../notifications/notificationsBus'

function notification(overrides: Partial<Notification> = {}): Notification {
  return {
    id: 'notif-1', type: 'import.started', title: 'Import started', message: 'Importing…',
    read: false, created_at: '2026-08-03T00:00:00Z', ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(notificationsApi.listNotifications).mockResolvedValue([])
})

describe('NotificationBell', () => {
  it('shows no badge when there are no unread notifications', async () => {
    renderBell()
    await waitFor(() => expect(notificationsApi.listNotifications).toHaveBeenCalled())

    expect(screen.queryByText('1')).not.toBeInTheDocument()
  })

  it('shows an unread count badge', async () => {
    vi.mocked(notificationsApi.listNotifications).mockResolvedValue([notification(), notification({ id: 'notif-2', read: true })])

    renderBell()

    expect(await screen.findByText('1')).toBeInTheDocument()
  })

  it('opening the bell shows the notification list and refetches', async () => {
    vi.mocked(notificationsApi.listNotifications).mockResolvedValue([notification({ title: 'Import completed' })])

    renderBell()
    await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))

    expect(await screen.findByText('Import completed')).toBeInTheDocument()
    expect(notificationsApi.listNotifications).toHaveBeenCalledTimes(2)
  })

  it('shows an empty state when there are no notifications', async () => {
    renderBell()
    await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))

    expect(await screen.findByText('No notifications yet.')).toBeInTheDocument()
  })

  it('clicking an unread notification marks it read', async () => {
    vi.mocked(notificationsApi.listNotifications).mockResolvedValue([notification()])
    vi.mocked(notificationsApi.markNotificationRead).mockResolvedValue(null)

    renderBell()
    await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
    await userEvent.click(await screen.findByText('Import started'))

    await waitFor(() => expect(notificationsApi.markNotificationRead).toHaveBeenCalledWith('test-token', 'notif-1'))
  })

  it('Mark all read clears the unread badge', async () => {
    vi.mocked(notificationsApi.listNotifications).mockResolvedValue([notification()])
    vi.mocked(notificationsApi.markAllNotificationsRead).mockResolvedValue(null)

    renderBell()
    expect(await screen.findByText('1')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Mark all read' }))

    await waitFor(() => expect(notificationsApi.markAllNotificationsRead).toHaveBeenCalledWith('test-token'))
    await waitFor(() => expect(screen.queryByText('1')).not.toBeInTheDocument())
  })

  it('clicking a linked notification marks it read and navigates to its link', async () => {
    vi.mocked(notificationsApi.listNotifications).mockResolvedValue([
      notification({ type: 'bol.offer.not_for_sale', title: 'Bol: "Widget" is not for sale', link: '/products/prod-1' }),
    ])
    vi.mocked(notificationsApi.markNotificationRead).mockResolvedValue(null)

    render(<NotificationBell />, { wrapper: MemoryRouter })
    await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
    await userEvent.click(await screen.findByText('Bol: "Widget" is not for sale'))

    await waitFor(() => expect(notificationsApi.markNotificationRead).toHaveBeenCalledWith('test-token', 'notif-1'))
    // The click closes the panel and navigates -- the panel (and its "Mark
    // all read" control, only rendered while open) disappearing is the
    // observable proxy for "navigation happened" without reaching into
    // react-router's internals.
    await waitFor(() => expect(screen.queryByText('No notifications yet.')).not.toBeInTheDocument())
  })

  it('clicking an unlinked notification only marks it read, no navigation', async () => {
    vi.mocked(notificationsApi.listNotifications).mockResolvedValue([notification()])
    vi.mocked(notificationsApi.markNotificationRead).mockResolvedValue(null)

    renderBell()
    await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
    await userEvent.click(await screen.findByText('Import started'))

    await waitFor(() => expect(notificationsApi.markNotificationRead).toHaveBeenCalledWith('test-token', 'notif-1'))
    // Panel stays open -- no link means no navigate() call, which is what
    // closes it.
    expect(screen.getByText('Import started')).toBeInTheDocument()
  })

  it('refetches when notified that notifications changed elsewhere (e.g. a new import job)', async () => {
    renderBell()
    await waitFor(() => expect(notificationsApi.listNotifications).toHaveBeenCalledTimes(1))

    act(() => { notifyNotificationsChanged() })

    await waitFor(() => expect(notificationsApi.listNotifications).toHaveBeenCalledTimes(2))
  })
})
