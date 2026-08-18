import type {
  Warehouse, WarehouseInput, APIResponse,
  Batch, CreateBatchInput, UpdateBatchInput,
  StockMovement, StockLevel,
  RecordStockMovementInput, RecordRecountInput, TransferStockInput, TransferStockResult, MarkDamagedOrLostInput,
  ExpiringBatch, ReceiveStockInput, ReceiveStockResult, SplitCaseIntoUnitsInput, SplitCaseIntoUnitsResult,
  PortionFromBulkInput, PortionFromBulkResult, ExpiredBatchReview, ExtendBatchExpiryInput,
  DiscardBatchInput, DiscardRecord, DiscardRecordReport, ListDiscardsParams, ListDiscardsResult,
} from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/inventory`
}

async function request<T>(
  path: string,
  options: RequestInit,
  token: string,
): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...options.headers,
    },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

export function listWarehouses(token: string): Promise<Warehouse[]> {
  return request<Warehouse[]>(`${base()}/warehouses`, { method: 'GET' }, token)
}

export function createWarehouse(token: string, input: WarehouseInput): Promise<Warehouse> {
  return request<Warehouse>(
    `${base()}/warehouses`,
    { method: 'POST', body: JSON.stringify(input) },
    token,
  )
}

export function updateWarehouse(token: string, id: string, input: WarehouseInput): Promise<Warehouse> {
  return request<Warehouse>(
    `${base()}/warehouses/${id}`,
    { method: 'PATCH', body: JSON.stringify(input) },
    token,
  )
}

export function deleteWarehouse(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/warehouses/${id}`, { method: 'DELETE' }, token)
}

// ── Batches ───────────────────────────────────────────────────────────────────

export function listBatches(token: string, skuId: string): Promise<Batch[]> {
  return request<Batch[]>(`${base()}/skus/${skuId}/batches`, { method: 'GET' }, token)
}

export function createBatch(token: string, skuId: string, input: CreateBatchInput): Promise<Batch> {
  return request<Batch>(`${base()}/skus/${skuId}/batches`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updateBatch(token: string, id: string, input: UpdateBatchInput): Promise<Batch> {
  return request<Batch>(`${base()}/batches/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function deleteBatch(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/batches/${id}`, { method: 'DELETE' }, token)
}

// ── Stock level / movements ──────────────────────────────────────────────────

export function getSKUStockLevel(token: string, skuId: string): Promise<StockLevel> {
  return request<StockLevel>(`${base()}/skus/${skuId}/stock-level`, { method: 'GET' }, token)
}

export function recordStockMovement(token: string, skuId: string, input: RecordStockMovementInput): Promise<StockMovement> {
  return request<StockMovement>(`${base()}/skus/${skuId}/movements`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function listStockMovements(token: string, skuId: string): Promise<StockMovement[]> {
  return request<StockMovement[]>(`${base()}/skus/${skuId}/movements`, { method: 'GET' }, token)
}

export function listBatchMovements(token: string, batchId: string): Promise<StockMovement[]> {
  return request<StockMovement[]>(`${base()}/batches/${batchId}/movements`, { method: 'GET' }, token)
}

// data is null when the count matched and nothing changed.
export function recordRecount(token: string, skuId: string, input: RecordRecountInput): Promise<StockMovement | null> {
  return request<StockMovement | null>(`${base()}/skus/${skuId}/recount`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function transferStock(token: string, skuId: string, input: TransferStockInput): Promise<TransferStockResult> {
  return request<TransferStockResult>(`${base()}/skus/${skuId}/transfer`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function markDamagedOrLost(token: string, skuId: string, input: MarkDamagedOrLostInput): Promise<StockMovement> {
  return request<StockMovement>(`${base()}/skus/${skuId}/mark-damaged-lost`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function splitCaseIntoUnits(token: string, skuId: string, input: SplitCaseIntoUnitsInput): Promise<SplitCaseIntoUnitsResult> {
  return request<SplitCaseIntoUnitsResult>(`${base()}/skus/${skuId}/split-case`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function portionFromBulk(token: string, skuId: string, input: PortionFromBulkInput): Promise<PortionFromBulkResult> {
  return request<PortionFromBulkResult>(`${base()}/skus/${skuId}/portion-from-bulk`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function receiveStock(token: string, skuId: string, input: ReceiveStockInput): Promise<ReceiveStockResult> {
  return request<ReceiveStockResult>(`${base()}/skus/${skuId}/receive`, { method: 'POST', body: JSON.stringify(input) }, token)
}

// ── Reports ───────────────────────────────────────────────────────────────────

export function listExpiringBatches(token: string, withinDays = 30): Promise<ExpiringBatch[]> {
  return request<ExpiringBatch[]>(`${base()}/batches/expiring-soon?within_days=${withinDays}`, { method: 'GET' }, token)
}

// ── Expired-batch review queue ───────────────────────────────────────────────

export function listPendingExpiredBatches(token: string): Promise<ExpiredBatchReview[]> {
  return request<ExpiredBatchReview[]>(`${base()}/batches/expired-review`, { method: 'GET' }, token)
}

export function writeOffExpiredBatch(token: string, batchId: string): Promise<null> {
  return request<null>(`${base()}/batches/${batchId}/write-off`, { method: 'POST' }, token)
}

export function extendBatchExpiry(token: string, batchId: string, input: ExtendBatchExpiryInput): Promise<null> {
  return request<null>(`${base()}/batches/${batchId}/extend-expiry`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function dismissExpiredBatch(token: string, batchId: string): Promise<null> {
  return request<null>(`${base()}/batches/${batchId}/dismiss`, { method: 'POST' }, token)
}

// ── Discard (expiry write-off with an optional photo) ───────────────────────

// Deliberately doesn't go through request(): that helper always sets
// Content-Type: application/json, which would break multipart/form-data --
// the browser must set that header itself (it includes the boundary). Same
// reasoning as pms/api.ts's uploadProductMedia.
export async function discardBatch(token: string, skuId: string, input: DiscardBatchInput): Promise<DiscardRecord> {
  const form = new FormData()
  form.append('warehouse_id', input.warehouse_id)
  if (input.batch_id) form.append('batch_id', input.batch_id)
  form.append('quantity', String(input.quantity))
  if (input.reason) form.append('reason', input.reason)
  if (input.photo) form.append('photo', input.photo)

  const res = await fetch(`${base()}/skus/${skuId}/discard`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
    body: form,
  })
  const body: APIResponse<DiscardRecord> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as DiscardRecord
}

export async function listDiscards(token: string, params: ListDiscardsParams = {}): Promise<ListDiscardsResult> {
  const qs = new URLSearchParams()
  if (params.warehouseId) qs.set('warehouse_id', params.warehouseId)
  if (params.skuId) qs.set('sku_id', params.skuId)
  if (params.type) qs.set('type', params.type)
  if (params.from) qs.set('from', params.from)
  if (params.to) qs.set('to', params.to)
  if (params.limit) qs.set('limit', String(params.limit))
  if (params.offset) qs.set('offset', String(params.offset))
  const query = qs.toString()

  const res = await fetch(`${base()}/discards${query ? `?${query}` : ''}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<DiscardRecordReport[]> & { total?: number } = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return { discards: body.data ?? [], total: body.total ?? 0 }
}
