import { useState, useEffect, useCallback } from 'react'
import type { PurchaseOrderDetail, UpdatePurchaseOrderInput, AddLineItemInput, UpdateLineItemInput } from '../types'
import * as api from '../api'

interface UsePurchaseOrderDetailState {
  detail: PurchaseOrderDetail | null
  loading: boolean
  error: string | null
}

interface UsePurchaseOrderDetailActions {
  refresh: () => void
  updateHeader: (input: UpdatePurchaseOrderInput) => Promise<void>
  addLineItem: (input: AddLineItemInput) => Promise<void>
  updateLineItem: (id: string, input: UpdateLineItemInput) => Promise<void>
  removeLineItem: (id: string) => Promise<void>
}

// purchaseOrderId is null while creating a new PO (no line items to manage
// until the header itself has been saved and has a real id).
export function usePurchaseOrderDetail(token: string, purchaseOrderId: string | null): UsePurchaseOrderDetailState & UsePurchaseOrderDetailActions {
  const [detail, setDetail] = useState<PurchaseOrderDetail | null>(null)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !purchaseOrderId) {
      setDetail(null)
      return
    }
    setLoading(true)
    setError(null)
    try {
      const data = await api.getPurchaseOrder(token, purchaseOrderId)
      setDetail(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, purchaseOrderId])

  useEffect(() => { load() }, [load])

  const updateHeader = useCallback(async (input: UpdatePurchaseOrderInput) => {
    if (!purchaseOrderId) return
    await api.updatePurchaseOrder(token, purchaseOrderId, input)
    await load()
  }, [token, purchaseOrderId, load])

  const addLineItem = useCallback(async (input: AddLineItemInput) => {
    if (!purchaseOrderId) return
    await api.addLineItem(token, purchaseOrderId, input)
    await load()
  }, [token, purchaseOrderId, load])

  const updateLineItem = useCallback(async (id: string, input: UpdateLineItemInput) => {
    await api.updateLineItem(token, id, input)
    await load()
  }, [token, load])

  const removeLineItem = useCallback(async (id: string) => {
    await api.removeLineItem(token, id)
    await load()
  }, [token, load])

  return { detail, loading, error, refresh: load, updateHeader, addLineItem, updateLineItem, removeLineItem }
}
