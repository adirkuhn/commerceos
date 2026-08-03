import type { Warehouse, WarehouseInput, APIResponse } from './types'

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
