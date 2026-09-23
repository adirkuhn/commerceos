import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getNotificationPreferences, updateNotificationPreferences, NotificationPreferencesError } from '../api'

const URL_ = 'https://api.commerceos.dev.localhost/api/v1/settings/notification-preferences'

function respond(status: number, body: unknown) {
  const f = vi.fn().mockResolvedValue({ ok: status < 300, status, json: () => Promise.resolve(body) })
  vi.stubGlobal('fetch', f)
  return f
}

beforeEach(() => { vi.restoreAllMocks() })

describe('notification preferences api', () => {
  it('GETs and PUTs the preferences', async () => {
    const f = respond(200, { data: { categories: [], email_recipients: [], updated_at: null }, error: null })
    await getNotificationPreferences('tok')
    expect(f).toHaveBeenCalledWith(URL_, expect.objectContaining({ method: 'GET' }))

    const input = { categories: [], email_recipients: ['a@b.nl'] }
    await updateNotificationPreferences('tok', input)
    expect(f).toHaveBeenLastCalledWith(URL_, expect.objectContaining({ method: 'PUT', body: JSON.stringify(input) }))
  })

  it('keeps the error code and marks recipient errors', async () => {
    respond(422, { data: null, error: { code: 'INVALID_RECIPIENT', message: 'bad' } })
    const err = await updateNotificationPreferences('tok', { categories: [], email_recipients: ['x'] }).catch(e => e)
    expect(err).toBeInstanceOf(NotificationPreferencesError)
    expect(err.code).toBe('INVALID_RECIPIENT')
    expect(err.isRecipientError).toBe(true)
  })
})
