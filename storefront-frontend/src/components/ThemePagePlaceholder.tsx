import { useThemePage } from './useThemePage'
import type { PageSlug } from '../types'

interface Props {
  slug: PageSlug
  title: string
}

// Stands in for the real canvas block-renderer (a later increment, once
// the admin editor's block library is stable) while still proving the
// page-document round trip end to end: every route already fetches its
// own ThemePage from the live backend, not static placeholder text.
export function ThemePagePlaceholder({ slug, title }: Props) {
  const { page, loading, error } = useThemePage(slug)

  return (
    <section className="mx-auto max-w-3xl px-6 py-16">
      <h1 className="text-2xl font-semibold text-brand-700">{title}</h1>
      {loading && <p className="mt-4 text-brand-500">Loading page…</p>}
      {error && <p className="mt-4 text-red-600">Could not load this page: {error}</p>}
      {page && !loading && !error && (
        <p className="mt-4 text-brand-500">
          {page.blocks.length === 0
            ? 'This page has no content yet.'
            : `${page.blocks.length} block${page.blocks.length === 1 ? '' : 's'} configured (canvas width ${page.canvas_width}px).`}
        </p>
      )}
    </section>
  )
}
