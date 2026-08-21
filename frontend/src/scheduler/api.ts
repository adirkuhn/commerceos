import type { Job, SyncScheduleRow, APIResponse } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/scheduler`
}

async function request<T>(path: string, token: string, options: RequestInit = {}): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...options.headers },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

export function listJobs(token: string): Promise<Job[]> {
  return request<Job[]>(`${base()}/jobs`, token)
}

export function runJobNow(token: string, jobKey: string): Promise<null> {
  return request<null>(`${base()}/jobs/${jobKey}/run-now`, token, { method: 'POST' })
}

export function updateJobConfig(token: string, jobKey: string, enabled: boolean, intervalMinutes: number): Promise<null> {
  return request<null>(`${base()}/jobs/${jobKey}`, token, {
    method: 'PATCH',
    body: JSON.stringify({ enabled, interval_minutes: intervalMinutes }),
  })
}

export function listSyncSchedules(token: string): Promise<SyncScheduleRow[]> {
  return request<SyncScheduleRow[]>(`${base()}/sync-schedules`, token)
}

export function runSyncScheduleNow(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/sync-schedules/${id}/run-now`, token, { method: 'POST' })
}
