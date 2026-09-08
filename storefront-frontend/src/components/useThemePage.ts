import { useEffect, useState } from 'react'
import { getStorefrontTheme } from '../api/theme'
import type { PageSlug, ThemePage } from '../types'

export interface UseThemePageResult {
  page: ThemePage | null
  loading: boolean
  error: string | null
}

// Fetches one page's canvas document. Until the block-renderer lands
// (next increment), consumers only need loading/error/page-summary --
// see ThemePagePlaceholder.
export function useThemePage(slug: PageSlug): UseThemePageResult {
  const [page, setPage] = useState<ThemePage | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)
    getStorefrontTheme(slug)
      .then((result) => {
        if (!cancelled) setPage(result)
      })
      .catch((err: Error) => {
        if (!cancelled) setError(err.message)
      })
      .finally(() => {
        if (!cancelled) setLoading(false)
      })
    return () => {
      cancelled = true
    }
  }, [slug])

  return { page, loading, error }
}
