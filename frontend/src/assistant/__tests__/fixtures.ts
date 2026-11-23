import type { AssistantMessage, AssistantRequest, AssistantRequestDetail } from '../types'

export function aRequest(over: Partial<AssistantRequest> = {}): AssistantRequest {
  return {
    id: 'r-1', title: 'Status of order CO-12?', status: 'queued', attempts: 0, next_attempt_at: '2026-11-23T12:00:30Z',
    wait_reason: null, last_error: null, user_id: 'u-1', created_at: '2026-11-23T12:00:00Z', updated_at: '2026-11-23T12:00:00Z',
    ...over,
  }
}

export function aMessage(over: Partial<AssistantMessage> = {}): AssistantMessage {
  return { seq: 1, role: 'user', content: 'Status of order CO-12?', question: false, created_at: '2026-11-23T12:00:00Z', ...over }
}

export function aDetail(request: Partial<AssistantRequest> = {}, messages: AssistantMessage[] = [aMessage()]): AssistantRequestDetail {
  return { request: aRequest(request), messages }
}
