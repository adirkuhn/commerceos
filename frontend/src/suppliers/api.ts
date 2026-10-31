import type { Supplier, SupplierInput, APIResponse } from './types'
import { CodedApiError } from '../api/codedApiError'
import { appEnv } from '../config/appEnv'

function base(): string {
  return `${appEnv().VITE_API_BASE_URL}/suppliers`
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
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as T
}

export function listSuppliers(token: string): Promise<Supplier[]> {
  return request<Supplier[]>(`${base()}`, { method: 'GET' }, token)
}

export function createSupplier(token: string, input: SupplierInput): Promise<Supplier> {
  return request<Supplier>(`${base()}`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updateSupplier(token: string, id: string, input: SupplierInput): Promise<Supplier> {
  return request<Supplier>(`${base()}/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function deleteSupplier(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/${id}`, { method: 'DELETE' }, token)
}
