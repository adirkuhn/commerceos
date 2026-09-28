// In-memory HTTP cache for the SSR server's calls to the backend. It keeps
// only what the backend itself marked shareable (public max-age): published
// theme documents are immutable for a year, the live theme index for 30s.
// Previews, shopper-specific and uncacheable responses always go through.

export interface CacheOptions {
  maxEntries: number
  maxBytes: number
  now?: () => number
}

interface Entry {
  body: string
  status: number
  headers: [string, string][]
  expires: number
  bytes: number
}

/** How long (ms) a response may be reused, from its Cache-Control. */
export function freshFor(cacheControl: string | null): number {
  if (!cacheControl) return 0
  const directives = cacheControl.toLowerCase().split(',').map(d => d.trim())
  if (directives.some(d => d === 'no-store' || d === 'no-cache' || d === 'private')) return 0
  const maxAge = directives.find(d => d.startsWith('max-age='))
  const seconds = maxAge ? Number(maxAge.slice('max-age='.length)) : 0
  return Number.isFinite(seconds) && seconds > 0 ? seconds * 1000 : 0
}

const DROPPED_HEADERS = new Set(['content-encoding', 'content-length', 'transfer-encoding'])

function cacheKey(url: string, headers: Headers): string | undefined {
  if (headers.has('X-Theme-Preview')) return undefined
  // The store key decides which tenant/channel the answer is about.
  return `${headers.get('Authorization') ?? ''}\n${url}`
}

export function cachingFetch(inner: typeof fetch, opts: CacheOptions): typeof fetch {
  const now = opts.now ?? Date.now
  // Map keeps insertion order: re-inserting on hit makes it an LRU.
  const entries = new Map<string, Entry>()
  let bytes = 0

  const remove = (key: string) => {
    const e = entries.get(key)
    if (!e) return
    bytes -= e.bytes
    entries.delete(key)
  }

  const store = (key: string, entry: Entry) => {
    if (entry.bytes > opts.maxBytes) return
    remove(key)
    entries.set(key, entry)
    bytes += entry.bytes
    for (const oldest of entries.keys()) {
      if (entries.size <= opts.maxEntries && bytes <= opts.maxBytes) break
      remove(oldest)
    }
  }

  return async (input, init) => {
    const method = (init?.method ?? 'GET').toUpperCase()
    const key = method === 'GET' ? cacheKey(String(input), new Headers(init?.headers)) : undefined
    if (!key) return inner(input, init)

    const hit = entries.get(key)
    if (hit && hit.expires > now()) {
      entries.delete(key)
      entries.set(key, hit)
      return new Response(hit.body, { status: hit.status, headers: hit.headers })
    }
    remove(key)

    const res = await inner(input, init)
    const ttl = res.ok ? freshFor(res.headers.get('Cache-Control')) : 0
    if (!ttl) return res
    const body = await res.text()
    // The body is stored decoded, so length/encoding no longer apply.
    const headers = [...res.headers.entries()].filter(([name]) => !DROPPED_HEADERS.has(name.toLowerCase()))
    store(key, { body, status: res.status, headers, expires: now() + ttl, bytes: body.length })
    return new Response(body, { status: res.status, headers })
  }
}
