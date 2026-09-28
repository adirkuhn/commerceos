import { describe, it, expect, vi, afterEach } from 'vitest'
import * as api from '../api'

afterEach(() => vi.unstubAllGlobals())

function stubFetch(body: unknown, status = 200) {
  const f = vi.fn(async () => new Response(JSON.stringify(body), { status }))
  vi.stubGlobal('fetch', f)
  return f
}

describe('theme api', () => {
  it('lists a channel\'s themes with paging', async () => {
    const f = stubFetch({ data: [{ id: 't1' }], total: 1, error: null })
    const res = await api.listThemes('tok', 'chan-1')
    expect(res).toEqual({ items: [{ id: 't1' }], total: 1 })
    const [url, init] = f.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/theme/themes?sales_channel_id=chan-1&page=1&per_page=50')
    expect((init.headers as Record<string, string>).Authorization).toBe('Bearer tok')
  })

  it('creates a theme', async () => {
    const f = stubFetch({ data: { id: 't1' }, error: null }, 201)
    await api.createTheme('tok', 'chan-1', 'Autumn')
    const [url, init] = f.mock.calls[0] as unknown as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/theme/themes')
    expect(JSON.parse(String(init.body))).toEqual({ sales_channel_id: 'chan-1', name: 'Autumn' })
  })

  it('mints an edit session', async () => {
    const f = stubFetch({ data: { token: 'raw', expires_at: 'x' }, error: null }, 201)
    await expect(api.createThemeSession('tok', 't1', 'edit')).resolves.toEqual({ token: 'raw', expires_at: 'x' })
    expect(JSON.parse(String((f.mock.calls[0] as unknown as [string, RequestInit])[1].body))).toEqual({ purpose: 'edit' })
  })

  it('surfaces the API error message', async () => {
    stubFetch({ data: null, error: { code: 'CONFLICT', message: 'the live theme cannot be deleted' } }, 409)
    await expect(api.deleteTheme('tok', 't1')).rejects.toThrow('the live theme cannot be deleted')
  })
})
