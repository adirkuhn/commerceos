import { extname, join, normalize, sep } from 'node:path'

export interface StaticFile {
  path: string
  contentType: string
  cacheControl: string
}

const TYPES: Record<string, string> = {
  '.js': 'text/javascript; charset=utf-8', '.css': 'text/css; charset=utf-8', '.html': 'text/html; charset=utf-8',
  '.svg': 'image/svg+xml', '.png': 'image/png', '.jpg': 'image/jpeg', '.webp': 'image/webp', '.woff2': 'font/woff2',
  '.json': 'application/json', '.map': 'application/json',
}

/** The built file a request path maps to, or undefined when it isn't a static asset. */
export function staticFileFor(clientRoot: string, pathname: string): StaticFile | undefined {
  if (pathname === '/store/editor' || pathname === '/store/editor/') {
    return { path: join(clientRoot, 'editor.html'), contentType: TYPES['.html'], cacheControl: 'no-cache' }
  }
  if (!pathname.startsWith('/store/assets/')) return undefined
  let rel: string
  try {
    rel = decodeURIComponent(pathname.slice('/store/'.length))
  } catch {
    return undefined
  }
  const full = normalize(join(clientRoot, rel))
  if (!full.startsWith(join(clientRoot, 'assets') + sep)) return undefined
  const type = TYPES[extname(full)]
  if (!type) return undefined
  // Vite fingerprints everything under assets/, so it can be cached forever.
  return { path: full, contentType: type, cacheControl: 'public, max-age=31536000, immutable' }
}
