import { CodedApiError } from '../api/codedApiError'

export { CodedApiError }

// JSON request for settings endpoints that keeps the API's error code
// (the shared request helpers only keep the message), so forms can place
// an error next to the field it belongs to.

interface Envelope<T> {
  data: T | null
  error: { code: string; message: string } | null
}

export async function codedRequest<T>(method: 'GET' | 'PUT', url: string, token: string, body?: unknown): Promise<T> {
  const res = await fetch(url, {
    method,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
    ...(body === undefined ? {} : { body: JSON.stringify(body) }),
  })
  const envelope: Envelope<T> = await res.json()
  if (!res.ok || envelope.error) {
    throw new CodedApiError(envelope.error?.code ?? 'HTTP_ERROR', envelope.error?.message ?? `HTTP ${res.status}`)
  }
  return envelope.data as T
}
