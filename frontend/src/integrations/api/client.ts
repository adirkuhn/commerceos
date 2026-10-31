import type { APIResponse } from '../types'
import { CodedApiError } from '../../api/codedApiError'
import { appEnv } from '../../config/appEnv'

export function base(): string {
  return `${appEnv().VITE_API_BASE_URL}/integrations`
}

export async function request<T>(
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
