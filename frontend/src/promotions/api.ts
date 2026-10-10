import type { Promotion, PromotionDetail, PromotionInput, CreatePromotionResult, APIResponse } from './types'
import { CodedApiError } from '../api/codedApiError'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/promotions`
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

export function listPromotions(token: string): Promise<Promotion[]> {
  return request<Promotion[]>(`${base()}`, { method: 'GET' }, token)
}

export function getPromotion(token: string, id: string): Promise<PromotionDetail> {
  return request<PromotionDetail>(`${base()}/${id}`, { method: 'GET' }, token)
}

export function createPromotion(token: string, input: PromotionInput): Promise<CreatePromotionResult> {
  return request<CreatePromotionResult>(`${base()}`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updatePromotion(token: string, id: string, input: PromotionInput): Promise<PromotionDetail> {
  return request<PromotionDetail>(`${base()}/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function deletePromotion(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/${id}`, { method: 'DELETE' }, token)
}
