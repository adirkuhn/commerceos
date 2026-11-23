import type { AssistantRequestStatus } from './types'

export const ASSISTANT_PATH = '/assistant'

// The detail page re-reads a request this often while the model may still
// be working on it.
export const POLL_INTERVAL_MS = 3000

export function assistantRequestPath(id: string): string {
  return `${ASSISTANT_PATH}/${encodeURIComponent(id)}`
}

// Waiting for or running the model -- the detail page polls.
export function isWorking(status: AssistantRequestStatus): boolean {
  return status === 'queued' || status === 'processing'
}

// Hasn't finished: the user can still cancel it.
export function isCancellable(status: AssistantRequestStatus): boolean {
  return isWorking(status) || status === 'needs_input' || status === 'awaiting_confirmation'
}

// The user answers a question, asks a follow-up, or asks for changes to
// what the assistant proposed.
export function canReply(status: AssistantRequestStatus): boolean {
  return status === 'needs_input' || status === 'answered' || status === 'awaiting_confirmation'
}

// Where a confirmed proposal's result opens.
export function quotePath(id: string): string {
  return `/quotes/${encodeURIComponent(id)}`
}

export function salesOrderPath(id: string): string {
  return `/sales-orders/${encodeURIComponent(id)}`
}

// Matches the backend's RequireRole on /assistant.
export const ASSISTANT_ROLES: readonly string[] = ['customer_owner', 'customer_admin', 'super_admin']

export function canUseAssistant(role: string | undefined): boolean {
  return role !== undefined && ASSISTANT_ROLES.includes(role)
}
