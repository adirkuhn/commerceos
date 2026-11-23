import { describe, expect, it } from 'vitest'
import spec from '../../../../backend/api/assistant/openapi.yaml?raw'
import type {
  AssistantMessage, AssistantProposal, AssistantRequest, AssistantRequestDetail, AssistantRequestList, AssistantToolCall,
  ConfirmProposalRequest, ConversionProposalPreview, ProposalCustomer, ProposalLine, ProposalShortfall, ProposalWarehouse, QuoteProposalPreview,
} from '../types'

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
  seq: true, role: true, content: true, tool_calls: false, tool_name: false, question: true, notice: false, proposal_id: false, created_at: true,
}
const detail: Record<keyof AssistantRequestDetail, boolean> = { request: true, messages: true, proposals: true }
const proposal: Record<keyof AssistantProposal, boolean> = {
  id: true, kind: true, status: true, quote: false, conversion: false, last_error: false, result_id: false, result_ref: false,
  decided_by: false, decided_at: false, created_at: true,
}
const customer: Record<keyof ProposalCustomer, boolean> = { id: true, name: true, email: false }
const line: Record<keyof ProposalLine, boolean> = {
  product_id: true, sku_id: true, product_name: true, variant_name: true, sku_code: true, quantity: true,
  list_price_minor_units: false, unit_price_minor_units: true, line_total_minor_units: true,
}
const quotePreview: Record<keyof QuoteProposalPreview, boolean> = {
  name: true, customer: false, discount_basis_points: true, currency_code: true, prices_include_tax: true, lines: true, total_minor_units: true,
}
const warehouse: Record<keyof ProposalWarehouse, boolean> = { id: true, name: true }
const shortfall: Record<keyof ProposalShortfall, boolean> = {
  product_id: true, sku_id: true, product_name: true, variant_name: true, sku_code: true, requested: true, available: true, short: true,
}
const conversion: Record<keyof ConversionProposalPreview, boolean> = {
  quote_id: true, quote_ref: true, name: true, customer: false, line_count: true, total_minor_units: true, currency_code: true,
  warehouse_id: true, warehouses: true, shortfall: true, lines: true,
}
const confirmBody: Record<keyof ConfirmProposalRequest, boolean> = { warehouse_id: false }
const list: Record<keyof AssistantRequestList, boolean> = { requests: true, next_cursor: false }

describe('assistant contract', () => {
  it.each([
    ['AssistantRequest', request],
    ['AssistantToolCall', toolCall],
    ['AssistantMessage', message],
    ['AssistantRequestDetail', detail],
    ['AssistantRequestList', list],
    ['AssistantProposal', proposal],
    ['ProposalCustomer', customer],
    ['ProposalLine', line],
    ['QuoteProposalPreview', quotePreview],
    ['ProposalWarehouse', warehouse],
    ['ProposalShortfall', shortfall],
    ['ConversionProposalPreview', conversion],
    ['ConfirmProposalRequest', confirmBody],
  ])('%s matches the spec', (schema, sample) => {
    expect(Object.keys(sample).sort()).toEqual(allFields(spec, schema))
    expect(Object.entries(sample).filter(([, req]) => req).map(([k]) => k).sort()).toEqual(requiredFields(spec, schema))
  })
})
