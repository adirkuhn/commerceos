import { useState, useEffect, useCallback } from 'react'
import type { ReceivingRecord } from '../types'
import * as api from '../api'

interface UseReceivingRecordsState {
  records: ReceivingRecord[]
  loading: boolean
  error: string | null
}

interface UseReceivingRecordsActions {
  refresh: () => void
  upload: (file: File) => Promise<void>
}

// purchaseOrderId is null while creating a new PO -- there's nothing to
// upload an invoice against until the header itself has been saved.
export function useReceivingRecords(token: string, purchaseOrderId: string | null): UseReceivingRecordsState & UseReceivingRecordsActions {
  const [records, setRecords] = useState<ReceivingRecord[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !purchaseOrderId) {
      setRecords([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      const data = await api.listReceivingRecords(token, purchaseOrderId)
      setRecords(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, purchaseOrderId])

  useEffect(() => { load() }, [load])

  const upload = useCallback(async (file: File) => {
    if (!purchaseOrderId) return
    await api.uploadInvoice(token, purchaseOrderId, file)
    await load()
  }, [token, purchaseOrderId, load])

  return { records, loading, error, refresh: load, upload }
}
