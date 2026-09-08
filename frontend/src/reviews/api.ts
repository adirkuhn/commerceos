import type { Review, APIResponse } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/reviews`
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

export function listReviews(token: string): Promise<Review[]> {
  return request<Review[]>(`${base()}`, { method: 'GET' }, token)
}

export function deleteReview(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/${id}`, { method: 'DELETE' }, token)
}
