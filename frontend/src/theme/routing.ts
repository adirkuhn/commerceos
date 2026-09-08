import type { PageSlug } from './types'
import { PAGE_SLUGS } from './types'

export interface ThemeEditorParams {
  channelId: string
  slug: PageSlug
}

const PATH_PATTERN = /^\/sales-channels\/([^/]+)\/theme\/([^/]+)$/

// Deliberately parsed from the URL rather than read via useParams -- same
// reasoning as routes.ts's idFromPath: reflects the current location on
// every render (no remount needed to pick up a slug switch) and lets the
// page render directly in tests without a matching <Route>.
export function themeEditorParamsFromPath(pathname: string): ThemeEditorParams | null {
  const match = pathname.match(PATH_PATTERN)
  if (!match) return null
  const [, channelId, slugRaw] = match
  const slug = decodeURIComponent(slugRaw) as PageSlug
  if (!PAGE_SLUGS.includes(slug)) return null
  return { channelId: decodeURIComponent(channelId), slug }
}

export function themeEditorPath(channelId: string, slug: PageSlug): string {
  return `/sales-channels/${encodeURIComponent(channelId)}/theme/${slug}`
}
