import { queryOptions } from '@tanstack/react-query'
import type { StorefrontAPI } from '../api/storefront'
import type { DocumentKind, StorefrontTheme } from '../core/document'
import { resolveRoute, type ResolvedRoute } from './route'

// Shared by SSR prefetching and client rendering, so the server's
// dehydrated cache is exactly what the client reads on hydration.

export const themeQuery = (api: StorefrontAPI) =>
  queryOptions({ queryKey: ['theme', api.previewing], queryFn: () => api.getTheme(), staleTime: 30_000, retry: false })

export const documentQuery = (api: StorefrontAPI, theme: StorefrontTheme, kind: DocumentKind, key: string) =>
  queryOptions({
    queryKey: ['document', theme.version_id, theme.preview, kind, key],
    queryFn: () => api.getDocument(theme, kind, key),
    // Published documents never change; a preview is re-read each visit.
    staleTime: theme.preview ? 0 : Infinity,
    retry: false,
  })

export function hasDocument(theme: StorefrontTheme, kind: DocumentKind, key: string): boolean {
  return theme.documents.some(d => d.kind === kind && d.key === key)
}

/** The document a path renders, falling back to not-found when the theme lacks it. */
export function documentFor(theme: StorefrontTheme, path: string): ResolvedRoute {
  const route = resolveRoute(path)
  if (route.notFound || route.builtin || hasDocument(theme, route.kind, route.key)) return route
  return { kind: 'template', key: 'not-found', notFound: true }
}
