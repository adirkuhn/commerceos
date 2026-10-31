import type { TaxClass, CreateTaxClassInput, UpdateTaxClassInput, TaxRate, UpsertTaxRateInput, ExternalTaxClassLink, UpsertExternalTaxClassLinkInput, APIResponse } from './types'
import { CodedApiError } from '../api/codedApiError'
import { appEnv } from '../config/appEnv'

function base(): string {
  return `${appEnv().VITE_API_BASE_URL}/tax`
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

export function listTaxClasses(token: string): Promise<TaxClass[]> {
  return request<TaxClass[]>(`${base()}/classes`, { method: 'GET' }, token)
}

export function createTaxClass(token: string, input: CreateTaxClassInput): Promise<TaxClass> {
  return request<TaxClass>(`${base()}/classes`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updateTaxClass(token: string, id: string, input: UpdateTaxClassInput): Promise<TaxClass> {
  return request<TaxClass>(`${base()}/classes/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function deleteTaxClass(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/classes/${id}`, { method: 'DELETE' }, token)
}

export function listTaxRates(token: string): Promise<TaxRate[]> {
  return request<TaxRate[]>(`${base()}/rates`, { method: 'GET' }, token)
}

export function upsertTaxRate(token: string, input: UpsertTaxRateInput): Promise<TaxRate> {
  return request<TaxRate>(`${base()}/rates`, { method: 'PUT', body: JSON.stringify(input) }, token)
}

export function deleteTaxRate(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/rates/${id}`, { method: 'DELETE' }, token)
}

export function listExternalTaxClassLinks(token: string): Promise<ExternalTaxClassLink[]> {
  return request<ExternalTaxClassLink[]>(`${base()}/external-links`, { method: 'GET' }, token)
}

export function upsertExternalTaxClassLink(token: string, input: UpsertExternalTaxClassLinkInput): Promise<ExternalTaxClassLink> {
  return request<ExternalTaxClassLink>(`${base()}/external-links`, { method: 'PUT', body: JSON.stringify(input) }, token)
}

export function deleteExternalTaxClassLink(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/external-links/${id}`, { method: 'DELETE' }, token)
}
