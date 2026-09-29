import i18next from 'i18next'
import { useEffect, useState, type DependencyList } from 'react'

export interface ResourceState<T> {
  data: T | null
  error: string
  loading: boolean
  setData: (data: T) => void
}

// Loads the resource a URL points at (a session, a record, a SKU). Screens
// reached by URL must fetch what they show from the id in that URL rather
// than receive it through in-memory state -- that state is exactly what a
// refresh throws away. A response for superseded deps (the user already
// navigated to another id) is dropped instead of overwriting newer data.
export function useResource<T>(load: () => Promise<T>, deps: DependencyList): ResourceState<T> {
  const [data, setData] = useState<T | null>(null)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    let current = true
    setLoading(true)
    setError('')
    setData(null)
    load()
      .then(result => { if (current) setData(result) })
      .catch(err => { if (current) setError(err instanceof Error ? err.message : i18next.t('ui:errors.load')) })
      .finally(() => { if (current) setLoading(false) })
    return () => { current = false }
  }, deps) // caller owns deps, like useEffect

  return { data, error, loading, setData }
}
