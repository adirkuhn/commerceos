import { beforeEach, describe, expect, it, vi } from 'vitest'
import { getAppearance, updateAppearance } from '../api'
import { CodedApiError } from '../../codedRequest'
import { DEFAULT_APPEARANCE, type Appearance } from '../../../theme/appearance'

const TOKEN = 'test-token'
const BASE = 'https://api.commerceos.dev.localhost/api/v1/settings/appearance'
const saved: Appearance = { ...DEFAULT_APPEARANCE, font: 'inter', updated_at: '2026-10-12T10:00:00Z' }

function respond(status: number, body: unknown) {
  const fetchMock = vi.fn().mockResolvedValue({ ok: status < 300, status, json: () => Promise.resolve(body) })
  vi.stubGlobal('fetch', fetchMock)
  return fetchMock
}

beforeEach(() => { vi.restoreAllMocks() })

describe('appearance api', () => {
  it('GETs the appearance with the bearer token', async () => {
    const fetchMock = respond(200, { data: saved, error: null })

    expect(await getAppearance(TOKEN)).toEqual(saved)
    expect(fetchMock).toHaveBeenCalledWith(BASE, expect.objectContaining({
      method: 'GET', headers: expect.objectContaining({ Authorization: `Bearer ${TOKEN}` }),
    }))
  })

  it('PUTs the whole appearance', async () => {
    const fetchMock = respond(200, { data: saved, error: null })
    const input = { ...DEFAULT_APPEARANCE, font: 'inter' as const }

    await updateAppearance(TOKEN, input)

    expect(fetchMock).toHaveBeenCalledWith(BASE, expect.objectContaining({ method: 'PUT', body: JSON.stringify(input) }))
  })

  it('keeps the error code of a rejected appearance', async () => {
    respond(422, { data: null, error: { code: 'APPEARANCE_ACCENT_LOW_CONTRAST', message: 'too light' } })

    const err = await updateAppearance(TOKEN, { ...DEFAULT_APPEARANCE, accent_color: '#FFFF00' }).catch(e => e)

    expect(err).toBeInstanceOf(CodedApiError)
    expect((err as CodedApiError).code).toBe('APPEARANCE_ACCENT_LOW_CONTRAST')
  })
})
