import type { DeadLetterStream, DeadLetterSummary } from './types'
import { DEAD_LETTER_STREAMS } from './types'

// The event data, indented when it is JSON, unchanged otherwise.
export function prettyData(data: string): string {
  try {
    return JSON.stringify(JSON.parse(data), null, 2)
  } catch {
    return data
  }
}

// When a dead letter started waiting: when it was dead-lettered, or, for
// one from before failure details were recorded, when the stream stored it.
export function waitingSince(d: Pick<DeadLetterSummary, 'dead_lettered_at' | 'stored_at'>): string {
  return d.dead_lettered_at ?? d.stored_at
}

export const DEAD_LETTERS_PATH = '/dead-letters'

export function deadLetterPath(stream: DeadLetterStream, seq: number): string {
  return `${DEAD_LETTERS_PATH}/${stream}/${seq}`
}

// Reads "/dead-letters/<stream>/<seq>" sub-path segments; null for the list
// or anything that doesn't name a real dead letter.
export function parseDeadLetterRef(segments: string[]): { stream: DeadLetterStream; seq: number } | null {
  if (segments.length !== 2) return null
  const [stream, rawSeq] = segments
  const seq = Number(rawSeq)
  if (!(DEAD_LETTER_STREAMS as readonly string[]).includes(stream) || !Number.isInteger(seq) || seq < 1) return null
  return { stream: stream as DeadLetterStream, seq }
}
