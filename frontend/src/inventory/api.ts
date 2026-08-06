import type {
  Warehouse, WarehouseInput, APIResponse,
  Batch, CreateBatchInput, UpdateBatchInput,
  StockMovement, StockLevel,
  RecordStockMovementInput, RecordRecountInput, TransferStockInput, TransferStockResult, MarkDamagedOrLostInput,
  ExpiringBatch, ReceiveStockInput, ReceiveStockResult,
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

export function receiveStock(token: string, skuId: string, input: ReceiveStockInput): Promise<ReceiveStockResult> {
  return request<ReceiveStockResult>(`${base()}/skus/${skuId}/receive`, { method: 'POST', body: JSON.stringify(input) }, token)
}

// ── Reports ───────────────────────────────────────────────────────────────────

export function listExpiringBatches(token: string, withinDays = 30): Promise<ExpiringBatch[]> {
  return request<ExpiringBatch[]>(`${base()}/batches/expiring-soon?within_days=${withinDays}`, { method: 'GET' }, token)
}
