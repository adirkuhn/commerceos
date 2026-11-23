import type {
  AssistantMessage, AssistantProposal, AssistantRequest, AssistantRequestDetail, ConversionProposalPreview, QuoteProposalPreview,
} from '../types'

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

export function aDetail(
  request: Partial<AssistantRequest> = {}, messages: AssistantMessage[] = [aMessage()], proposals: AssistantProposal[] = [],
): AssistantRequestDetail {
  return { request: aRequest(request), messages, proposals }
}

export function aQuotePreview(over: Partial<QuoteProposalPreview> = {}): QuoteProposalPreview {
  return {
    name: 'Ana Bakery', customer: { id: 'c-1', name: 'Ana Bakery', email: 'ana@bakery.test' }, discount_basis_points: 1000,
    currency_code: 'EUR', prices_include_tax: true, total_minor_units: 4500,
    lines: [{
      product_id: 'p-1', sku_id: 's-1', product_name: 'Farofa', variant_name: '1 kg', sku_code: 'FAR-1',
      quantity: 10, list_price_minor_units: 500, unit_price_minor_units: 450, line_total_minor_units: 4500,
    }],
    ...over,
  }
}

export function aConversionPreview(over: Partial<ConversionProposalPreview> = {}): ConversionProposalPreview {
  return {
    quote_id: 'q-1', quote_ref: 'Q-0012', name: 'Ana Bakery', customer: { id: 'c-1', name: 'Ana Bakery' }, line_count: 1,
    total_minor_units: 4500, currency_code: 'EUR', warehouse_id: 'w-1',
    warehouses: [{ id: 'w-1', name: 'Main warehouse' }, { id: 'w-2', name: 'Overflow' }],
    lines: [{
      product_id: 'p-1', sku_id: 's-1', product_name: 'Farofa', variant_name: '1 kg', sku_code: 'FAR-1',
      quantity: 10, unit_price_minor_units: 450, line_total_minor_units: 4500,
    }],
    shortfall: [{ product_id: 'p-1', sku_id: 's-1', product_name: 'Farofa', variant_name: '1 kg', sku_code: 'FAR-1', requested: 10, available: 4, short: 6 }],
    ...over,
  }
}

export function aProposal(over: Partial<AssistantProposal> = {}): AssistantProposal {
  return { id: 'p-1', kind: 'create_quote', status: 'pending', quote: aQuotePreview(), created_at: '2026-11-23T12:01:00Z', ...over }
}

export function aConversionProposal(over: Partial<AssistantProposal> = {}): AssistantProposal {
  return { id: 'p-2', kind: 'convert_quote', status: 'pending', conversion: aConversionPreview(), created_at: '2026-11-23T12:02:00Z', ...over }
}

// The tool message that made proposal `id`.
export function aProposedMessage(seq: number, id: string): AssistantMessage {
  return aMessage({ seq, role: 'tool', content: '', tool_name: 'propose_quote', notice: 'proposed', proposal_id: id })
}
