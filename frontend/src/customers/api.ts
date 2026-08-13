import type { Customer, CustomerInput, APIResponse } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/customers`
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

export function listCustomers(token: string): Promise<Customer[]> {
  return request<Customer[]>(`${base()}`, { method: 'GET' }, token)
}

export function createCustomer(token: string, input: CustomerInput): Promise<Customer> {
  return request<Customer>(`${base()}`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updateCustomer(token: string, id: string, input: CustomerInput): Promise<Customer> {
  return request<Customer>(`${base()}/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function deleteCustomer(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/${id}`, { method: 'DELETE' }, token)
}
