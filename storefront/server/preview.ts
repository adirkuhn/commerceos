// A preview link (/store/{key}/...?preview=<token>) is turned into a
// cookie scoped to that store, then the token is dropped from the URL so
// it doesn't linger in history, analytics or Referer headers.

const COOKIE = 'cos_preview'
const MAX_AGE = 24 * 60 * 60 // matches the preview session TTL

export interface PreviewRedirect {
  location: string
  cookie: string
}

export function previewRedirect(rawURL: string): PreviewRedirect | undefined {
  const url = new URL(rawURL, 'http://x')
  const token = url.searchParams.get('preview')
  const store = /^\/store\/[^/]+/.exec(url.pathname)?.[0]
  if (token === null || !store) return undefined
  url.searchParams.delete('preview')
  const clear = token === 'off' || token === ''
  const value = clear ? '' : encodeURIComponent(token)
  return {
    location: url.pathname + url.search,
    cookie: `${COOKIE}=${value}; Path=${store}; HttpOnly; SameSite=Lax; Max-Age=${clear ? 0 : MAX_AGE}`,
  }
}

export function previewTokenFrom(cookieHeader: string | undefined): string | undefined {
  for (const part of cookieHeader?.split(';') ?? []) {
    const [name, ...rest] = part.trim().split('=')
    if (name === COOKIE) {
      const value = decodeURIComponent(rest.join('='))
      return value || undefined
    }
  }
  return undefined
}
