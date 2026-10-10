import type {
  MessageTemplate, CreateMessageTemplateInput, UpdateMessageTemplateInput,
  MessageTrigger, PreviewTemplateInput, TemplatePreview, MessageTemplateDefault,
  AutomationRule, CreateAutomationRuleInput, UpdateAutomationRuleInput,
  APIResponse,
} from './types'
import { CodedApiError } from '../api/codedApiError'

function apiBase(): string {
  return import.meta.env.VITE_API_BASE_URL
}

function base(): string {
  return `${apiBase()}/message-templates`
}

function rulesBase(): string {
  return `${apiBase()}/automation-rules`
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

export function resetMessageTemplate(token: string, id: string): Promise<MessageTemplate> {
  return request<MessageTemplate>(`${base()}/${id}/reset`, { method: 'POST' }, token)
}

export function listMessageTemplateDefaults(token: string): Promise<MessageTemplateDefault[]> {
  return request<MessageTemplateDefault[]>(`${apiBase()}/message-template-defaults`, { method: 'GET' }, token)
}

export function listAutomationRules(token: string): Promise<AutomationRule[]> {
  return request<AutomationRule[]>(`${rulesBase()}`, { method: 'GET' }, token)
}

export function createAutomationRule(token: string, input: CreateAutomationRuleInput): Promise<AutomationRule> {
  return request<AutomationRule>(`${rulesBase()}`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updateAutomationRule(token: string, id: string, input: UpdateAutomationRuleInput): Promise<AutomationRule> {
  return request<AutomationRule>(`${rulesBase()}/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function deleteAutomationRule(token: string, id: string): Promise<null> {
  return request<null>(`${rulesBase()}/${id}`, { method: 'DELETE' }, token)
}

export function listMessageTriggers(token: string): Promise<MessageTrigger[]> {
  return request<MessageTrigger[]>(`${apiBase()}/message-triggers`, { method: 'GET' }, token)
}

export function previewMessageTemplate(token: string, input: PreviewTemplateInput): Promise<TemplatePreview> {
  return request<TemplatePreview>(`${base()}/preview`, { method: 'POST', body: JSON.stringify(input) }, token)
}

// uploadMessageTemplateImage stores an image (logo, banner) for templates
// and returns its public URL.
export async function uploadMessageTemplateImage(token: string, file: File): Promise<string> {
  const form = new FormData()
  form.append('file', file)
  const res = await fetch(`${base()}/images`, { method: 'POST', headers: { Authorization: `Bearer ${token}` }, body: form })
  const body: APIResponse<{ url: string }> = await res.json()
  if (!res.ok || body.error || !body.data) throw CodedApiError.fromEnvelope(body.error, res.status)
  return body.data.url
}
