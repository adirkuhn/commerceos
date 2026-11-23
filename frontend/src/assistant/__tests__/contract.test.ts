import { describe, expect, it } from 'vitest'
import spec from '../../../../backend/api/assistant/openapi.yaml?raw'
import type { AssistantMessage, AssistantRequest, AssistantRequestDetail, AssistantRequestList, AssistantToolCall } from '../types'

// Pins the hand-written types to the backend OpenAPI contract: every field
// the spec declares exists here (and no extras), and required-ness matches.
// Enum values are pinned by src/contracts' enum contract test.
function schemaBlock(yaml: string, schema: string): string {
  const after = yaml.replace(/\r\n/g, '\n').split(`\n    ${schema}:\n`)[1] ?? ''
  return after.split(/\n {4}\w+:\n/)[0]
}

function requiredFields(yaml: string, schema: string): string[] {
  const m = /\n {6}required: \[([^\]]*)\]/.exec(`\n${schemaBlock(yaml, schema)}`)
  return m ? m[1].split(',').map(s => s.trim()).sort() : []
}

function allFields(yaml: string, schema: string): string[] {
  const props = schemaBlock(yaml, schema).split('\n      properties:\n')[1] ?? ''
  return [...props.matchAll(/^ {8}(\w+):/gm)].map(m => m[1]).sort()
}

const request: Record<keyof AssistantRequest, boolean> = {
  id: true, title: true, status: true, attempts: true, next_attempt_at: true, wait_reason: false, last_error: false,
  user_id: true, created_at: true, updated_at: true,
}
const toolCall: Record<keyof AssistantToolCall, boolean> = { id: true, name: true, arguments: true }
const message: Record<keyof AssistantMessage, boolean> = {
  seq: true, role: true, content: true, tool_calls: false, tool_name: false, question: true, notice: false, created_at: true,
}
const detail: Record<keyof AssistantRequestDetail, boolean> = { request: true, messages: true }
const list: Record<keyof AssistantRequestList, boolean> = { requests: true, next_cursor: false }

describe('assistant contract', () => {
  it.each([
    ['AssistantRequest', request],
    ['AssistantToolCall', toolCall],
    ['AssistantMessage', message],
    ['AssistantRequestDetail', detail],
    ['AssistantRequestList', list],
  ])('%s matches the spec', (schema, sample) => {
    expect(Object.keys(sample).sort()).toEqual(allFields(spec, schema))
    expect(Object.entries(sample).filter(([, req]) => req).map(([k]) => k).sort()).toEqual(requiredFields(spec, schema))
  })
})
