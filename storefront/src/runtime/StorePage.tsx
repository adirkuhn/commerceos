import { useSuspenseQuery } from '@tanstack/react-query'
import { useStorefrontAPI } from './ApiContext'
import { documentQuery, hasDocument, themeQuery } from './queries'
import { DocumentRenderer } from './DocumentRenderer'
import { resolveRoute, type ResolvedRoute } from './route'
import type { StorefrontTheme } from '../core/document'

/** The document a path renders, falling back to not-found when the theme lacks it. */
export function documentFor(theme: StorefrontTheme, path: string): ResolvedRoute {
  const route = resolveRoute(path)
  if (route.notFound || hasDocument(theme, route.kind, route.key)) return route
  return { kind: 'template', key: 'not-found', notFound: true }
}

function Region({ theme, kind, docKey }: { theme: StorefrontTheme; kind: 'template' | 'page' | 'section_group'; docKey: string }) {
  const api = useStorefrontAPI()
  const { data } = useSuspenseQuery(documentQuery(api, theme, kind, docKey))
  return <DocumentRenderer root={data.root} />
}

export function StorePage({ path }: { path: string }) {
  const api = useStorefrontAPI()
  const { data: theme } = useSuspenseQuery(themeQuery(api))
  const route = documentFor(theme, path)
  const has = (key: string) => hasDocument(theme, 'section_group', key)
  return (
    <>
      {has('header') && <header className="cos-header"><Region theme={theme} kind="section_group" docKey="header" /></header>}
      <main className="cos-main"><Region theme={theme} kind={route.kind} docKey={route.key} /></main>
      {has('footer') && <footer className="cos-footer"><Region theme={theme} kind="section_group" docKey="footer" /></footer>}
    </>
  )
}

export function StoreUnavailable() {
  return (
    <main className="cos-unavailable">
      <h1>This store is not available</h1>
      <p>Please check the address or come back later.</p>
    </main>
  )
}
