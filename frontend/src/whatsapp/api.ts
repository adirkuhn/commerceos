import type { WhatsAppConversation, WhatsAppConversationMessage, APIResponse } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/whatsapp`
}

async function request<T>(path: string, token: string): Promise<T> {
  const res = await fetch(path, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as T
}

export function listWhatsAppConversations(token: string): Promise<WhatsAppConversation[]> {
  return request<WhatsAppConversation[]>(`${base()}/conversations`, token)
}

export function listWhatsAppConversationMessages(token: string, phoneE164: string): Promise<WhatsAppConversationMessage[]> {
  return request<WhatsAppConversationMessage[]>(`${base()}/conversations/${encodeURIComponent(phoneE164)}/messages`, token)
}
