import { describe, it, expect, vi, beforeEach } from 'vitest'
import { getThemeDraft, saveThemeDraft } from '../themeDraft'
import type { APIResponse, ThemePage } from '../../types'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 401) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({ ok: false, status, json: () => Promise.resolve(body) }))
}

beforeEach(() => { vi.restoreAllMocks() })

const draftPage: ThemePage = {
  slug: 'home', canvas_width: 1440, blocks: [], custom_css: '',
  editor_mode: 'puck', puck_data: { content: [], root: {} },
}

describe('getThemeDraft', () => {
  it('sends the edit-session token (not the public storefront key) against the draft path', async () => {
    mockFetch(draftPage)
    await getThemeDraft('edit-token-1', 'home')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/storefront/v1/theme/draft/pages/home')
    expect((options?.headers as Record<string, string>).Authorization).toBe('Bearer edit-token-1')
  })

  it('returns the full draft page, including puck_data', async () => {
    mockFetch(draftPage)
    const result = await getThemeDraft('edit-token-1', 'home')
    expect(result).toEqual(draftPage)
  })

  it('throws with the server error message on an invalid/expired token', async () => {
    mockFetchError('UNAUTHORIZED', 'invalid edit session')
    await expect(getThemeDraft('wrong-token', 'home')).rejects.toThrow('invalid edit session')
  })
})

describe('saveThemeDraft', () => {
  it('PUTs the payload with the edit-session token', async () => {
    mockFetch(draftPage)
    await saveThemeDraft('edit-token-1', 'home', {
      canvas_width: 1440, blocks: [], custom_css: '', editor_mode: 'puck', puck_data: { content: [], root: {} },
    })
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/storefront/v1/theme/draft/pages/home')
    expect(options?.method).toBe('PUT')
    expect((options?.headers as Record<string, string>).Authorization).toBe('Bearer edit-token-1')
    expect(JSON.parse(options?.body as string)).toEqual({
      canvas_width: 1440, blocks: [], custom_css: '', editor_mode: 'puck', puck_data: { content: [], root: {} },
    })
  })
})
