// Mirrors backend/api/deadletters/openapi.yaml -- pinned by
// __tests__/contract.test.ts and the enum contract test.

export type DeadLetterStream = 'events' | 'webhooks'

export const DEAD_LETTER_STREAMS: readonly DeadLetterStream[] = ['events', 'webhooks']

export interface DeadLetterSummary {
  stream: DeadLetterStream
  sequence: number
  // The event's original subject; a retry publishes here.
  subject: string
  // JetStream durable that gave up; '' when unknown (dead-lettered before
  // failure details were recorded) -- a retry then reaches every consumer.
  consumer: string
  consumer_known: boolean
  event_id: string
  tenant_id: string
  error: string
  num_delivered: number
  dead_lettered_at: string | null
  stored_at: string
}

export interface DeadLetter extends DeadLetterSummary {
  // The event exactly as published (usually JSON).
  data: string
}

export interface DeadLetterPage {
  items: DeadLetterSummary[]
  next_cursor: number | null
  // Every dead letter on the stream, ignoring filters.
  total: number
}

export interface DeadLetterFilter {
  stream: DeadLetterStream
  subject?: string
  consumer?: string
  tenantId?: string
  cursor?: number
  limit?: number
}

export interface APIResponse<T> {
  data: T
  error: { code: string; message: string; detail?: string } | null
}
