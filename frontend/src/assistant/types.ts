// Hand-written mirror of backend/api/assistant/openapi.yaml -- pinned to it
// by __tests__/contract.test.ts.

export type AssistantRequestStatus = 'queued' | 'processing' | 'needs_input' | 'answered' | 'cancelled'

export type AssistantWaitReason = 'model_busy' | 'no_provider' | 'provider_unsupported' | 'model_error'

export type AssistantMessageRole = 'user' | 'assistant' | 'tool'

export type AssistantNotice = 'too_many_steps'

export const ASSISTANT_STATUSES: readonly AssistantRequestStatus[] = ['queued', 'processing', 'needs_input', 'answered', 'cancelled']

export interface AssistantRequest {
  id: string
  title: string
  status: AssistantRequestStatus
  attempts: number
  next_attempt_at: string
  wait_reason?: AssistantWaitReason | null
  last_error?: string | null
  user_id: string
  created_at: string
  updated_at: string
}

export interface AssistantToolCall {
  id: string
  name: string
  arguments: string
}

export interface AssistantMessage {
  seq: number
  role: AssistantMessageRole
  content: string
  tool_calls?: AssistantToolCall[]
  tool_name?: string | null
  question: boolean
  notice?: AssistantNotice | null
  created_at: string
}

export interface AssistantRequestDetail {
  request: AssistantRequest
  messages: AssistantMessage[]
}

export interface AssistantRequestList {
  requests: AssistantRequest[]
  next_cursor?: string | null
}

export interface AssistantListQuery {
  status?: AssistantRequestStatus
  cursor?: string
  limit?: number
}
