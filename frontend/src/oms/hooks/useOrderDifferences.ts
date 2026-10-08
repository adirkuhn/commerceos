import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import { listOrderDifferences, dismissOrderDifference } from '../orderDifferences'
import type { OrderDifference } from '../orderDifferences'

export const ORDER_DIFFERENCES_PAGE_SIZE = 20

// useOrderDifferences loads one page of open order differences (Stock
// Issues). dismiss() reloads the page afterwards.
export function useOrderDifferences(token: string, page: number) {
  const [differences, setDifferences] = useState<OrderDifference[]>([])
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const result = await listOrderDifferences(token, page, ORDER_DIFFERENCES_PAGE_SIZE)
      setDifferences(result.differences)
      setTotal(result.total)
    } catch (e) {
      setError(e instanceof Error ? e.message : i18next.t('stockIssues:differences.loadFailed'))
    } finally {
      setLoading(false)
    }
  }, [token, page])

  useEffect(() => { load() }, [load])

  const dismiss = useCallback(async (id: string) => {
    await dismissOrderDifference(token, id)
    await load()
  }, [token, load])

  return { differences, total, loading, error, refresh: load, dismiss }
}
