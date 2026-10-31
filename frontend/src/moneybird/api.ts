import type {
  APIResponse, MoneybirdDocument, MoneybirdDocumentState, MoneybirdSettings, MoneybirdSetup, MoneybirdSourceType,
  UpdateMoneybirdSettingsInput,
} from './types'
import { CodedApiError } from '../api/codedApiError'
import { appEnv } from '../config/appEnv'

function base(): string {
  return `${appEnv().VITE_API_BASE_URL}/integrations/moneybird`
}

async function request<T>(path: string, options: RequestInit, token: string): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as T
}

const SOURCE_PATH: Record<MoneybirdSourceType, string> = { quote: 'quotes', sales_order: 'sales-orders' }

export function connectMoneybird(token: string, apiToken: string): Promise<{ connection_id: string }> {
  return request(`${base()}/connect`, { method: 'POST', body: JSON.stringify({ api_token: apiToken }) }, token)
}

export function getMoneybirdSetup(token: string): Promise<MoneybirdSetup> {
  return request(base(), { method: 'GET' }, token)
}

export function updateMoneybirdSettings(token: string, input: UpdateMoneybirdSettingsInput): Promise<MoneybirdSettings> {
  return request(`${base()}/settings`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function sendToMoneybird(token: string, sourceType: MoneybirdSourceType, id: string): Promise<MoneybirdDocument> {
  return request(`${base()}/${SOURCE_PATH[sourceType]}/${id}/send`, { method: 'POST' }, token)
}

export function getMoneybirdDocument(token: string, sourceType: MoneybirdSourceType, id: string): Promise<MoneybirdDocumentState> {
  return request(`${base()}/${SOURCE_PATH[sourceType]}/${id}`, { method: 'GET' }, token)
}

// Deletes the estimate/invoice from Moneybird and unlinks it, so it can be sent again.
export async function deleteFromMoneybird(token: string, sourceType: MoneybirdSourceType, id: string): Promise<void> {
  await request(`${base()}/${SOURCE_PATH[sourceType]}/${id}`, { method: 'DELETE' }, token)
}
