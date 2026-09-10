import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { APIResponse, ThemePage } from '../types'

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

beforeEach(() => { vi.restoreAllMocks() })

const samplePage: ThemePage = {
  sales_channel_id: 'chan-1',
  slug: 'home',
  canvas_width: 1440,
  blocks: [
    { id: 'b1', type: 'hero', layout: { desktop: { x: 0, y: 0, width: 1440, height: 400 }, mobile: null }, props: { heading: 'Hi' } },
  ],
  custom_css: '.hero { color: red; }',
  editor_mode: 'freeform',
}

describe('getThemePage', () => {
  it('returns the page', async () => {
    mockFetch(samplePage)
    const result = await api.getThemePage(TOKEN, 'chan-1', 'home')
    expect(result).toEqual(samplePage)
  })

  it('sends the bearer token against the right path', async () => {
    mockFetch(samplePage)
    await api.getThemePage(TOKEN, 'chan-1', 'home')
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/theme/sales-channels/chan-1/pages/home')
    expect((options?.headers as Record<string, string>).Authorization).toBe(`Bearer ${TOKEN}`)
  })

  it('throws with the server error message on failure', async () => {
    mockFetchError('NOT_FOUND', 'sales channel not found', 404)
    await expect(api.getThemePage(TOKEN, 'missing', 'home')).rejects.toThrow('sales channel not found')
  })
})

describe('saveThemePage', () => {
  it('PUTs the payload and returns the saved page', async () => {
    mockFetch(samplePage)
    const result = await api.saveThemePage(TOKEN, 'chan-1', 'home', {
      canvas_width: 1440, blocks: samplePage.blocks, custom_css: samplePage.custom_css, editor_mode: 'freeform',
    })
    expect(result).toEqual(samplePage)
    const [, options] = vi.mocked(fetch).mock.calls[0]
    expect(options?.method).toBe('PUT')
    expect(JSON.parse(options?.body as string)).toEqual({
      canvas_width: 1440, blocks: samplePage.blocks, custom_css: samplePage.custom_css, editor_mode: 'freeform',
    })
  })

  it('throws on a validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'invalid block type', 422)
    await expect(api.saveThemePage(TOKEN, 'chan-1', 'home', { canvas_width: 1440, blocks: [], custom_css: '', editor_mode: 'freeform' })).rejects.toThrow('invalid block type')
  })
})

describe('createEditSession', () => {
  it('POSTs to the edit-sessions endpoint and returns the session', async () => {
    mockFetch({ token: 'edit-token-1', expires_at: '2026-01-01T00:00:00Z' })
    const result = await api.createEditSession(TOKEN, 'chan-1')
    expect(result).toEqual({ token: 'edit-token-1', expires_at: '2026-01-01T00:00:00Z' })
    const [url, options] = vi.mocked(fetch).mock.calls[0]
    expect(url).toContain('/theme/sales-channels/chan-1/edit-sessions')
    expect(options?.method).toBe('POST')
    expect((options?.headers as Record<string, string>).Authorization).toBe(`Bearer ${TOKEN}`)
  })

  it('throws with the server error message on failure', async () => {
    mockFetchError('NOT_FOUND', 'sales channel not found', 404)
    await expect(api.createEditSession(TOKEN, 'missing')).rejects.toThrow('sales channel not found')
  })
})
