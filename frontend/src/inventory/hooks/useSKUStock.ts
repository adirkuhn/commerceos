import { useState, useEffect, useCallback } from 'react'
import type {
  Batch, StockLevel, StockMovement,
  CreateBatchInput, UpdateBatchInput,
  RecordStockMovementInput, RecordRecountInput, TransferStockInput, MarkDamagedOrLostInput,
  SplitCaseIntoUnitsInput, SplitCaseIntoUnitsResult,
  PortionFromBulkInput, PortionFromBulkResult,
} from '../types'
import * as api from '../api'
import { useInvalidateStock, useOnStockInvalidated } from '../stockInvalidation'

interface UseSKUStockState {
  level: StockLevel | null
  batches: Batch[]
  loading: boolean
  error: string | null
}

interface UseSKUStockActions {
  refresh: () => void
  addBatch: (input: CreateBatchInput) => Promise<Batch>
  editBatch: (id: string, input: UpdateBatchInput) => Promise<Batch>
  removeBatch: (id: string) => Promise<void>
  adjust: (input: RecordStockMovementInput) => Promise<StockMovement>
  recount: (input: RecordRecountInput) => Promise<StockMovement | null>
  transfer: (input: TransferStockInput) => Promise<void>
  splitCase: (input: SplitCaseIntoUnitsInput) => Promise<SplitCaseIntoUnitsResult>
  portionFromBulk: (input: PortionFromBulkInput) => Promise<PortionFromBulkResult>
  markDamagedOrLost: (input: MarkDamagedOrLostInput) => Promise<StockMovement>
}

// SKU-scoped: stock level + its batches, plus every mutation that changes
// either one. Every mutation reloads both afterwards (rather than patching
// state locally) since a single action here can affect several batches at
// once (e.g. transfer) and the derived stock level always needs recomputing
// server-side. A mutation that also moves another SKU's stock (split,
// portion) invalidates that SKU too, so any other view showing it reloads
// (see stockInvalidation); this hook likewise reloads when another view
// invalidates its own SKU.
export function useSKUStock(token: string, skuId: string | null): UseSKUStockState & UseSKUStockActions {
  const [level, setLevel] = useState<StockLevel | null>(null)
  const [batches, setBatches] = useState<Batch[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)

  const load = useCallback(async () => {
    if (!token || !skuId) {
      setLevel(null)
      setBatches([])
      return
    }
    setLoading(true)
    setError(null)
    try {
      const [levelResult, batchesResult] = await Promise.all([
        api.getSKUStockLevel(token, skuId),
        api.listBatches(token, skuId),
      ])
      setLevel(levelResult)
      setBatches(batchesResult)
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Unknown error')
    } finally {
      setLoading(false)
    }
  }, [token, skuId])

  useEffect(() => { load() }, [load])
  useOnStockInvalidated(skuId ? [skuId] : [], load)
  const invalidateStock = useInvalidateStock()

  const addBatch = useCallback(async (input: CreateBatchInput): Promise<Batch> => {
    if (!skuId) throw new Error('no SKU selected')
    const batch = await api.createBatch(token, skuId, input)
    await load()
    return batch
  }, [token, skuId, load])

  const editBatch = useCallback(async (id: string, input: UpdateBatchInput): Promise<Batch> => {
    const batch = await api.updateBatch(token, id, input)
    await load()
    return batch
  }, [token, load])

  const removeBatch = useCallback(async (id: string): Promise<void> => {
    await api.deleteBatch(token, id)
    await load()
  }, [token, load])

  const adjust = useCallback(async (input: RecordStockMovementInput): Promise<StockMovement> => {
    if (!skuId) throw new Error('no SKU selected')
    const movement = await api.recordStockMovement(token, skuId, input)
    await load()
    return movement
  }, [token, skuId, load])

  const recount = useCallback(async (input: RecordRecountInput): Promise<StockMovement | null> => {
    if (!skuId) throw new Error('no SKU selected')
    const movement = await api.recordRecount(token, skuId, input)
    await load()
    return movement
  }, [token, skuId, load])

  const transfer = useCallback(async (input: TransferStockInput): Promise<void> => {
    if (!skuId) throw new Error('no SKU selected')
    await api.transferStock(token, skuId, input)
    await load()
  }, [token, skuId, load])

  const markDamagedOrLost = useCallback(async (input: MarkDamagedOrLostInput): Promise<StockMovement> => {
    if (!skuId) throw new Error('no SKU selected')
    const movement = await api.markDamagedOrLost(token, skuId, input)
    await load()
    return movement
  }, [token, skuId, load])

  const splitCase = useCallback(async (input: SplitCaseIntoUnitsInput): Promise<SplitCaseIntoUnitsResult> => {
    if (!skuId) throw new Error('no SKU selected')
    const result = await api.splitCaseIntoUnits(token, skuId, input)
    await load()
    invalidateStock([result.unit_in.sku_id])
    return result
  }, [token, skuId, load, invalidateStock])

  const portionFromBulk = useCallback(async (input: PortionFromBulkInput): Promise<PortionFromBulkResult> => {
    if (!skuId) throw new Error('no SKU selected')
    const result = await api.portionFromBulk(token, skuId, input)
    await load()
    invalidateStock([result.bulk_out.sku_id])
    return result
  }, [token, skuId, load, invalidateStock])

  return { level, batches, loading, error, refresh: load, addBatch, editBatch, removeBatch, adjust, recount, transfer, splitCase, portionFromBulk, markDamagedOrLost }
}
