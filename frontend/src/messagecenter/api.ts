import type { MessageTemplate, CreateMessageTemplateInput, UpdateMessageTemplateInput, APIResponse } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/message-templates`
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

export function listMessageTemplates(token: string): Promise<MessageTemplate[]> {
  return request<MessageTemplate[]>(`${base()}`, { method: 'GET' }, token)
}

export function createMessageTemplate(token: string, input: CreateMessageTemplateInput): Promise<MessageTemplate> {
  return request<MessageTemplate>(`${base()}`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updateMessageTemplate(token: string, id: string, input: UpdateMessageTemplateInput): Promise<MessageTemplate> {
  return request<MessageTemplate>(`${base()}/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function deleteMessageTemplate(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/${id}`, { method: 'DELETE' }, token)
}
