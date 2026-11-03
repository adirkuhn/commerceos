import { describe, it, expect } from 'vitest'
import { deadLetterPath, parseDeadLetterRef, prettyData, waitingSince } from '../format'

describe('prettyData', () => {
  it('indents JSON', () => {
    expect(prettyData('{"event_id":"e-1","n":1}')).toBe('{\n  "event_id": "e-1",\n  "n": 1\n}')
  })
  it('keeps non-JSON data as is', () => {
    expect(prettyData('not json')).toBe('not json')
  })
})

describe('waitingSince', () => {
  it('prefers when it was dead-lettered', () => {
    expect(waitingSince({ dead_lettered_at: '2026-11-03T10:00:00Z', stored_at: '2026-11-03T10:00:01Z' })).toBe('2026-11-03T10:00:00Z')
  })
  it('falls back to when the stream stored it (old dead letters)', () => {
    expect(waitingSince({ dead_lettered_at: null, stored_at: '2026-11-01T08:00:00Z' })).toBe('2026-11-01T08:00:00Z')
  })
})

describe('dead letter URLs', () => {
  it('round-trips a stream and sequence', () => {
    expect(deadLetterPath('webhooks', 9)).toBe('/dead-letters/webhooks/9')
    expect(parseDeadLetterRef(['webhooks', '9'])).toEqual({ stream: 'webhooks', seq: 9 })
  })
  it.each([[[]], [['events']], [['orders', '1']], [['events', '0']], [['events', 'x']], [['events', '1', 'extra']]])(
    'rejects %j', segments => {
      expect(parseDeadLetterRef(segments)).toBeNull()
    })
})
