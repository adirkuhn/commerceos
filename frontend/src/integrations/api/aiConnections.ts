import type { AIConnection, AIProvider } from '../types'
import { base, request } from './client'

// The tenant's AI connections: an ordered list every AI feature routes
// through -- the first enabled one is tried, any failure moves to the next.

export function listAIConnections(token: string): Promise<AIConnection[]> {
  return request<AIConnection[]>(`${base()}/ai-connections`, { method: 'GET' }, token)
}

export interface AIConnectionPayload {
  provider: AIProvider
  apiKey?: string
  baseUrl?: string
  model?: string
  fallbackModels?: string
  label?: string
  enabled?: boolean
}

export function createAIConnection(token: string, p: AIConnectionPayload): Promise<AIConnection> {
  return request<AIConnection>(
    `${base()}/ai-connections`,
    {
      method: 'POST',
      body: JSON.stringify({
        provider: p.provider,
        api_key: p.apiKey || undefined,
        base_url: p.baseUrl || undefined,
        model: p.model || undefined,
        fallback_models: p.fallbackModels || undefined,
        label: p.label || undefined,
        enabled: p.enabled,
      }),
    },
    token,
  )
}

// Only the fields present are sent. An empty apiKey is never sent: the
// stored key is kept (it never comes back to the browser).
export type AIConnectionPatch = Partial<AIConnectionPayload>

export function updateAIConnection(token: string, id: string, p: AIConnectionPatch): Promise<AIConnection> {
  const body: Record<string, unknown> = {}
  if (p.provider !== undefined) body.provider = p.provider
  if (p.apiKey) body.api_key = p.apiKey
  if (p.baseUrl !== undefined) body.base_url = p.baseUrl
  if (p.model !== undefined) body.model = p.model
  if (p.fallbackModels !== undefined) body.fallback_models = p.fallbackModels
  if (p.label !== undefined) body.label = p.label
  if (p.enabled !== undefined) body.enabled = p.enabled
  return request<AIConnection>(
    `${base()}/ai-connections/${encodeURIComponent(id)}`,
    { method: 'PATCH', body: JSON.stringify(body) },
    token,
  )
}

export function deleteAIConnection(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/ai-connections/${encodeURIComponent(id)}`, { method: 'DELETE' }, token)
}

export function reorderAIConnections(token: string, ids: string[]): Promise<AIConnection[]> {
  return request<AIConnection[]>(
    `${base()}/ai-connections/order`,
    { method: 'PUT', body: JSON.stringify({ ids }) },
    token,
  )
}
