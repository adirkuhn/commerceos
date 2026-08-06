import { useState, useEffect, useCallback } from 'react'
import type { ReceivingParsingJob, ReceivingRecord } from '../types'
import * as api from '../api'

interface UseReceivingRecordsState {
  records: ReceivingRecord[]
  loading: boolean
  error: string | null
}

interface UseReceivingRecordsActions {
  refresh: () => void
  upload: (file: File) => Promise<ReceivingParsingJob>
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
    if (!purchaseOrderId) throw new Error('Cannot upload an invoice before the purchase order has been saved')
    return api.uploadInvoice(token, purchaseOrderId, file)
  }, [token, purchaseOrderId])

  return { records, loading, error, refresh: load, upload }
}
