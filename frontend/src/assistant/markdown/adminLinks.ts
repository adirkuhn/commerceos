import { PAGE_ROUTES } from '../../routes'

// isAdminPath: a link the assistant may render as a link -- a path of one of
// the admin's own pages ("/sales-orders/<id>"). Read from PAGE_ROUTES at call
// time (routes.ts imports this module's page indirectly), so there's no
// second list to keep in sync. Anything else -- another site, javascript:,
// data:, "//host" -- isn't a link.
export function isAdminPath(path: string): boolean {
  if (!path.startsWith('/') || path.startsWith('//') || path.includes('\\')) return false
  const first = path.slice(1).split(/[/?#]/)[0]
  if (!first) return false
  return PAGE_ROUTES.some(r => r.path !== '/' && r.path.slice(1).split('/')[0] === first)
}
