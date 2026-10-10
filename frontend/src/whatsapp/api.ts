import type { WhatsAppConversation, WhatsAppConversationMessage, APIResponse } from './types'
import { CodedApiError } from '../api/codedApiError'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/whatsapp`
}

async function request<T>(path: string, token: string, method: string = 'GET'): Promise<T> {
  const res = await fetch(path, {
    method,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new CodedApiError(body.error?.code ?? 'HTTP_ERROR', body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

export function listWhatsAppConversations(token: string): Promise<WhatsAppConversation[]> {
  return request<WhatsAppConversation[]>(`${base()}/conversations`, token)
}

export function listWhatsAppConversationMessages(token: string, phoneE164: string): Promise<WhatsAppConversationMessage[]> {
  return request<WhatsAppConversationMessage[]>(`${base()}/conversations/${encodeURIComponent(phoneE164)}/messages`, token)
}

export function deleteWhatsAppConversation(token: string, phoneE164: string): Promise<null> {
  return request<null>(`${base()}/conversations/${encodeURIComponent(phoneE164)}/messages`, token, 'DELETE')
}
