import { describe, expect, it } from 'vitest'
import spec from '../../../../backend/api/deadletters/openapi.yaml?raw'
import type { DeadLetter, DeadLetterPage, DeadLetterSummary } from '../types'

// Pins the hand-written types to the backend OpenAPI contract: a field
// added, renamed or dropped on one side fails here.
function requiredFields(yaml: string, schema: string): string[] {
  const block = yaml.replace(/\r\n/g, '\n').split(`\n    ${schema}:\n`)[1] ?? ''
  const m = /\n {6}required: \[([^\]]*)\]/.exec(`\n${block}`)
  return m ? m[1].split(',').map(s => s.trim()).sort() : []
}

const summary: Record<keyof DeadLetterSummary, true> = {
  stream: true, sequence: true, subject: true, consumer: true, consumer_known: true, event_id: true,
  tenant_id: true, error: true, num_delivered: true, dead_lettered_at: true, stored_at: true,
}
const detail: Record<keyof DeadLetter, true> = { ...summary, data: true }
const page: Record<keyof DeadLetterPage, true> = { items: true, next_cursor: true, total: true }

describe('dead letters contract', () => {
  it.each([
    ['DeadLetterSummary', summary],
    ['DeadLetter', detail],
    ['DeadLetterPage', page],
  ])('%s has exactly the fields the spec requires', (schema, sample) => {
    expect(Object.keys(sample).sort()).toEqual(requiredFields(spec, schema))
  })
})
