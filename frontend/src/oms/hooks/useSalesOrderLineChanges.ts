import i18next from 'i18next'
import { useState, useEffect, useCallback } from 'react'
import type { SalesOrderLineChange } from '../types'
import * as api from '../api'
import { apiErrorText } from '../../api/apiErrorText'

// What picking and packing replaced, added or refunded on the order.
export function useSalesOrderLineChanges(token: string, salesOrderId: string | null) {
  const [changes, setChanges] = useState<SalesOrderLineChange[]>([])
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !salesOrderId) {
      setChanges([])
      return
    }
    setError(null)
    try {
      setChanges(await api.listSalesOrderLineChanges(token, salesOrderId))
    } catch (e) {
      setError(apiErrorText(e) ?? i18next.t('orders:errors.unknown'))
    }
  }, [token, salesOrderId])

  useEffect(() => { load() }, [load])

  return { changes, error }
}
