// Hand-written mirror of backend/api/assistant/openapi.yaml -- pinned to it
// by __tests__/contract.test.ts.

export type AssistantRequestStatus = 'queued' | 'processing' | 'needs_input' | 'answered' | 'awaiting_confirmation' | 'cancelled'

export type AssistantWaitReason = 'model_busy' | 'no_provider' | 'provider_unsupported' | 'model_error'

export type AssistantMessageRole = 'user' | 'assistant' | 'tool'

export type AssistantNotice = 'too_many_steps' | 'proposed' | 'proposal_confirmed' | 'proposal_rejected'

export type AssistantProposalKind = 'create_quote' | 'convert_quote'

export type AssistantProposalStatus = 'pending' | 'executing' | 'confirmed' | 'rejected' | 'superseded'

export const ASSISTANT_STATUSES: readonly AssistantRequestStatus[] = ['queued', 'processing', 'needs_input', 'awaiting_confirmation', 'answered', 'cancelled']

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
  proposal_id?: string | null
  created_at: string
}

export interface AssistantRequestDetail {
  request: AssistantRequest
  messages: AssistantMessage[]
  proposals: AssistantProposal[]
}

export interface ProposalCustomer {
  id: string
  name: string
  email?: string
}

export interface ProposalLine {
  product_id: string
  sku_id: string
  product_name: string
  variant_name: string
  sku_code: string
  quantity: number
  list_price_minor_units?: number
  unit_price_minor_units: number
  line_total_minor_units: number
}

export interface QuoteProposalPreview {
  name: string
  customer?: ProposalCustomer | null
  discount_basis_points: number
  currency_code: string
  prices_include_tax: boolean
  lines: ProposalLine[]
  total_minor_units: number
}

export interface ProposalWarehouse {
  id: string
  name: string
}

export interface ProposalShortfall {
  product_id: string
  sku_id: string
  product_name: string
  variant_name: string
  sku_code: string
  requested: number
  available: number
  short: number
}

export interface ConversionProposalPreview {
  quote_id: string
  quote_ref: string
  name: string
  customer?: ProposalCustomer | null
  line_count: number
  total_minor_units: number
  currency_code: string
  warehouse_id: string
  warehouses: ProposalWarehouse[]
  shortfall: ProposalShortfall[]
  lines: ProposalLine[]
}

export interface AssistantProposal {
  id: string
  kind: AssistantProposalKind
  status: AssistantProposalStatus
  quote?: QuoteProposalPreview
  conversion?: ConversionProposalPreview
  last_error?: string | null
  result_id?: string | null
  result_ref?: string | null
  decided_by?: string | null
  decided_at?: string | null
  created_at: string
}

export interface ConfirmProposalRequest {
  warehouse_id?: string
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
