import type { ExchangeRate, UpsertExchangeRateInput, APIResponse } from './types'
import { CodedApiError } from '../api/codedApiError'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/fx`
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
    throw new CodedApiError(body.error?.code ?? 'HTTP_ERROR', body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

export function listExchangeRates(token: string): Promise<ExchangeRate[]> {
  return request<ExchangeRate[]>(`${base()}/rates`, { method: 'GET' }, token)
}

export function upsertExchangeRate(token: string, input: UpsertExchangeRateInput): Promise<ExchangeRate> {
  return request<ExchangeRate>(`${base()}/rates`, { method: 'PUT', body: JSON.stringify(input) }, token)
}

export function deleteExchangeRate(token: string, currencyCode: string): Promise<null> {
  return request<null>(`${base()}/rates/${currencyCode}`, { method: 'DELETE' }, token)
}
