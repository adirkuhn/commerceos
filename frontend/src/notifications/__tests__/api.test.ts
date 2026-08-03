import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { Notification, APIResponse } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 422) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  }))
}

const sampleNotification: Notification = {
  id: 'notif-1', type: 'import.started', title: 'Import started', message: 'Importing…',
  read: false, created_at: '2026-08-03T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listNotifications', () => {
  it('returns notifications', async () => {
    mockFetch([sampleNotification])
    const result = await api.listNotifications(TOKEN)
    expect(result).toEqual([sampleNotification])
  })

  it('calls the correct endpoint', async () => {
    mockFetch([])
    await api.listNotifications(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/notifications',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listNotifications(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('markNotificationRead', () => {
  it('calls the correct endpoint with POST', async () => {
    mockFetch(null)
    await api.markNotificationRead(TOKEN, 'notif-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/notifications/notif-1/read',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})

describe('markAllNotificationsRead', () => {
  it('calls the correct endpoint with POST', async () => {
    mockFetch(null)
    await api.markAllNotificationsRead(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/notifications/read-all',
      expect.objectContaining({ method: 'POST' }),
    )
  })
})
