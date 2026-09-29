import i18next from 'i18next'
import { useEffect, useState } from 'react'
import type { PickList } from '../types'
import * as api from '../api'

interface UsePickListState {
  pickList: PickList | null
  loading: boolean
  error: string | null
}

// Loads the read-only pick list for the given orders. Keyed on the ids'
// content, not the array's identity, so a caller passing a freshly built
// array each render doesn't refetch.
export function usePickList(token: string, salesOrderIds: string[]): UsePickListState {
  const [pickList, setPickList] = useState<PickList | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const key = salesOrderIds.join(',')

  useEffect(() => {
    const ids = key ? key.split(',') : []
    if (!token || ids.length === 0) {
      setPickList(null)
      return
    }
    let cancelled = false
    setLoading(true)
    setError(null)
    api.getPickList(token, ids)
      .then(data => { if (!cancelled) setPickList(data) })
      .catch(e => { if (!cancelled) setError(e instanceof Error ? e.message : i18next.t('orders:errors.loadPickList')) })
      .finally(() => { if (!cancelled) setLoading(false) })
    return () => { cancelled = true }
  }, [token, key])

  return { pickList, loading, error }
}
