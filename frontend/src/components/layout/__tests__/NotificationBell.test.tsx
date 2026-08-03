// @vitest-environment jsdom
import { render, screen, waitFor, act } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import NotificationBell from '../NotificationBell'
import type { Notification } from '../../../notifications/types'

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
    render(<NotificationBell />)
    await waitFor(() => expect(notificationsApi.listNotifications).toHaveBeenCalled())

    expect(screen.queryByText('1')).not.toBeInTheDocument()
  })

  it('shows an unread count badge', async () => {
    vi.mocked(notificationsApi.listNotifications).mockResolvedValue([notification(), notification({ id: 'notif-2', read: true })])

    render(<NotificationBell />)

    expect(await screen.findByText('1')).toBeInTheDocument()
  })

  it('opening the bell shows the notification list and refetches', async () => {
    vi.mocked(notificationsApi.listNotifications).mockResolvedValue([notification({ title: 'Import completed' })])

    render(<NotificationBell />)
    await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))

    expect(await screen.findByText('Import completed')).toBeInTheDocument()
    expect(notificationsApi.listNotifications).toHaveBeenCalledTimes(2)
  })

  it('shows an empty state when there are no notifications', async () => {
    render(<NotificationBell />)
    await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))

    expect(await screen.findByText('No notifications yet.')).toBeInTheDocument()
  })

  it('clicking an unread notification marks it read', async () => {
    vi.mocked(notificationsApi.listNotifications).mockResolvedValue([notification()])
    vi.mocked(notificationsApi.markNotificationRead).mockResolvedValue(null)

    render(<NotificationBell />)
    await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
    await userEvent.click(await screen.findByText('Import started'))

    await waitFor(() => expect(notificationsApi.markNotificationRead).toHaveBeenCalledWith('test-token', 'notif-1'))
  })

  it('Mark all read clears the unread badge', async () => {
    vi.mocked(notificationsApi.listNotifications).mockResolvedValue([notification()])
    vi.mocked(notificationsApi.markAllNotificationsRead).mockResolvedValue(null)

    render(<NotificationBell />)
    expect(await screen.findByText('1')).toBeInTheDocument()

    await userEvent.click(screen.getByRole('button', { name: 'Notifications' }))
    await userEvent.click(await screen.findByRole('button', { name: 'Mark all read' }))

    await waitFor(() => expect(notificationsApi.markAllNotificationsRead).toHaveBeenCalledWith('test-token'))
    await waitFor(() => expect(screen.queryByText('1')).not.toBeInTheDocument())
  })

  it('refetches when notified that notifications changed elsewhere (e.g. a new import job)', async () => {
    render(<NotificationBell />)
    await waitFor(() => expect(notificationsApi.listNotifications).toHaveBeenCalledTimes(1))

    act(() => { notifyNotificationsChanged() })

    await waitFor(() => expect(notificationsApi.listNotifications).toHaveBeenCalledTimes(2))
  })
})
