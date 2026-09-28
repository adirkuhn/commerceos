import { describe, it, expect, vi } from 'vitest'
import { cachingFetch, freshFor } from '../httpCache.ts'

const json = (body: unknown, cacheControl?: string, status = 200) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json', ...(cacheControl ? { 'Cache-Control': cacheControl } : {}) } })

const get = (key = 'k1', extra: Record<string, string> = {}) => ({ headers: { Authorization: `Bearer ${key}`, ...extra } })

function setup(respond: () => Response, opts: { maxEntries?: number; maxBytes?: number } = {}) {
  let now = 1_000_000
  const inner = vi.fn<typeof fetch>(async () => respond())
  const f = cachingFetch(inner, { maxEntries: opts.maxEntries ?? 100, maxBytes: opts.maxBytes ?? 1_000_000, now: () => now })
  return { f, inner, advance: (ms: number) => { now += ms } }
}

describe('freshFor', () => {
  it('reads public max-age and refuses anything private or uncacheable', () => {
    expect(freshFor('public, max-age=31536000, immutable')).toBe(31_536_000_000)
    expect(freshFor('public, max-age=30')).toBe(30_000)
    expect(freshFor('no-store')).toBe(0)
    expect(freshFor('private, max-age=60')).toBe(0)
    expect(freshFor('public, max-age=60, no-cache')).toBe(0)
    expect(freshFor(null)).toBe(0)
  })
})

describe('cachingFetch', () => {
  it('serves a fresh response from memory, and refetches once it expires', async () => {
    const { f, inner, advance } = setup(() => json({ data: 1 }, 'public, max-age=30'))
    expect(await (await f('http://api/theme', get())).json()).toEqual({ data: 1 })
    expect(await (await f('http://api/theme', get())).json()).toEqual({ data: 1 })
    expect(inner).toHaveBeenCalledTimes(1)
    advance(30_001)
    await f('http://api/theme', get())
    expect(inner).toHaveBeenCalledTimes(2)
  })

  it('keeps each store apart and never caches previews', async () => {
    const { f, inner } = setup(() => json({}, 'public, max-age=30'))
    await f('http://api/theme', get('k1'))
    await f('http://api/theme', get('k2'))
    await f('http://api/theme', get('k1', { 'X-Theme-Preview': 'p' }))
    await f('http://api/theme', get('k1', { 'X-Theme-Preview': 'p' }))
    expect(inner).toHaveBeenCalledTimes(4)
  })

  it('only caches successful GETs the backend marked cacheable', async () => {
    const uncached = setup(() => json({}))
    await uncached.f('http://api/store', get())
    await uncached.f('http://api/store', get())
    expect(uncached.inner).toHaveBeenCalledTimes(2)

    const failed = setup(() => json({ error: 'x' }, 'public, max-age=30', 500))
    await failed.f('http://api/theme', get())
    await failed.f('http://api/theme', get())
    expect(failed.inner).toHaveBeenCalledTimes(2)

    const post = setup(() => json({}, 'public, max-age=30'))
    await post.f('http://api/carts', { method: 'POST', ...get() })
    await post.f('http://api/carts', { method: 'POST', ...get() })
    expect(post.inner).toHaveBeenCalledTimes(2)
  })

  it('evicts the least recently used entries past its limits', async () => {
    const { f, inner } = setup(() => json({ pad: 'x'.repeat(10) }, 'public, max-age=300'), { maxEntries: 2 })
    await f('http://api/a', get())
    await f('http://api/b', get())
    await f('http://api/a', get()) // a is now most recent
    await f('http://api/c', get()) // evicts b
    expect(inner).toHaveBeenCalledTimes(3)
    await f('http://api/a', get())
    expect(inner).toHaveBeenCalledTimes(3)
    await f('http://api/b', get())
    expect(inner).toHaveBeenCalledTimes(4)
  })

  it('does not keep a response bigger than the byte budget', async () => {
    const { f, inner } = setup(() => json({ pad: 'x'.repeat(100) }, 'public, max-age=300'), { maxBytes: 50 })
    await f('http://api/big', get())
    await f('http://api/big', get())
    expect(inner).toHaveBeenCalledTimes(2)
  })
})
