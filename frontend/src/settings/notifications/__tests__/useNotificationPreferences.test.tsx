// @vitest-environment jsdom
import { act, renderHook, waitFor } from '@testing-library/react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { useNotificationPreferences } from '../useNotificationPreferences'
import { NotificationPreferencesError } from '../api'
import type { NotificationPreferences } from '../types'

vi.mock('../api', async importOriginal => {
  const actual = await importOriginal<typeof import('../api')>()
  return { ...actual, getNotificationPreferences: vi.fn(), updateNotificationPreferences: vi.fn() }
})
import * as api from '../api'

const prefs: NotificationPreferences = { categories: [{ category: 'imports', in_app: true, email: false }], email_recipients: [], updated_at: null }

beforeEach(() => { vi.clearAllMocks() })

describe('useNotificationPreferences', () => {
  it('loads, then save replaces with the server copy', async () => {
    vi.mocked(api.getNotificationPreferences).mockResolvedValue(prefs)
    vi.mocked(api.updateNotificationPreferences).mockResolvedValue({ ...prefs, email_recipients: ['a@b.nl'], updated_at: 'x' })
    const { result } = renderHook(() => useNotificationPreferences('tok'))
    await waitFor(() => expect(result.current.loading).toBe(false))
    expect(result.current.preferences).toEqual(prefs)

    await act(() => result.current.save({ categories: prefs.categories, email_recipients: ['a@b.nl'] }))

    expect(result.current.preferences?.email_recipients).toEqual(['a@b.nl'])
  })

  it('exposes load errors and rethrows save errors', async () => {
    vi.mocked(api.getNotificationPreferences).mockRejectedValue(new NotificationPreferencesError('INTERNAL', 'boom'))
    const { result } = renderHook(() => useNotificationPreferences('tok'))
    await waitFor(() => expect(result.current.error).toBe('Something went wrong on our side. Please try again.'))

    vi.mocked(api.updateNotificationPreferences).mockRejectedValue(new NotificationPreferencesError('INVALID_RECIPIENT', 'bad'))
    await expect(result.current.save({ categories: [], email_recipients: ['x'] })).rejects.toBeInstanceOf(NotificationPreferencesError)
  })
})
