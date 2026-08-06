import { useState, useEffect, useCallback } from 'react'
import type { PurchaseOrder, CreatePurchaseOrderInput } from '../types'
import * as api from '../api'

interface UsePurchaseOrdersState {
  purchaseOrders: PurchaseOrder[]
  loading: boolean
  error: string | null
}

interface UsePurchaseOrdersActions {
  refresh: () => void
  create: (input: CreatePurchaseOrderInput) => Promise<PurchaseOrder>
  remove: (id: string) => Promise<void>
}

export function usePurchaseOrders(token: string): UsePurchaseOrdersState & UsePurchaseOrdersActions {
  const [purchaseOrders, setPurchaseOrders] = useState<PurchaseOrder[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token) return
    setLoading(true)
    setError(null)
    try {
      const data = await api.listPurchaseOrders(token)
      setPurchaseOrders(data)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token])

  useEffect(() => { load() }, [load])

  const create = useCallback(async (input: CreatePurchaseOrderInput): Promise<PurchaseOrder> => {
    const po = await api.createPurchaseOrder(token, input)
    setPurchaseOrders(prev => [...prev, po])
    return po
  }, [token])

  const remove = useCallback(async (id: string): Promise<void> => {
    await api.deletePurchaseOrder(token, id)
    setPurchaseOrders(prev => prev.filter(p => p.id !== id))
  }, [token])

  return { purchaseOrders, loading, error, refresh: load, create, remove }
}
