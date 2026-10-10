import type { Notification, APIResponse } from './types'
import { CodedApiError } from '../api/codedApiError'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/notifications`
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

export function listNotifications(token: string): Promise<Notification[]> {
  return request<Notification[]>(`${base()}`, { method: 'GET' }, token)
}

export function markNotificationRead(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/${id}/read`, { method: 'POST' }, token)
}

export function markAllNotificationsRead(token: string): Promise<null> {
  return request<null>(`${base()}/read-all`, { method: 'POST' }, token)
}
