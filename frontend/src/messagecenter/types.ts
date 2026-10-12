export type MessageChannel = 'email' | 'sms' | 'whatsapp'

export type MessageCategory = 'purchase' | 'order_status' | 'stock_info' | 'tracking' | 'custom'

export interface MessageTemplate {
  id: string
  name: string
  category: MessageCategory
  channels: MessageChannel[]
  subject: string
  // Email body: HTML with {{token}} placeholders.
  body: string
  // WhatsApp/SMS text; empty means body is used.
  text_body?: string
  // The visual email editor's project data body was built from.
  design?: Record<string, unknown> | null
  // The built-in default this template came from ('' when none).
  default_key?: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateMessageTemplateInput {
  name: string
  category: MessageCategory
  channels: MessageChannel[]
  subject?: string
  body?: string
  text_body?: string
  design?: Record<string, unknown> | null
  // The default this template starts from; enables "reset to default".
  default_key?: string
}

export interface UpdateMessageTemplateInput extends CreateMessageTemplateInput {
  enabled: boolean
}

export type TriggerSubject =
  | 'pms.product.deleted'
  | 'storefront.order.placed'
  | 'pms.category.changed'
  | 'pms.product.changed'
  | 'payments.refund.manual_review_required'
  | 'payments.refund.issued'
  | 'integrations.connection.connected'
  | 'oms.sales_order.status_changed'
  | 'inventory.stock.changed'
  | 'shipping.shipment.fulfillment_signal'
  | 'inventory.stock.close_to_expiry_reported'
  | 'oms.sales_order.created'

export type Operator = 'eq' | 'neq' | 'in' | 'gt' | 'lt'

export interface Condition {
  field: string
  operator: Operator
  value: unknown
}

export interface AutomationRule {
  id: string
  name: string
  trigger_subject: TriggerSubject
  conditions: Condition[]
  template_id: string
  channel_override?: MessageChannel
  // Always emailed on top of whoever the trigger goes to.
  extra_recipients: string[]
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateAutomationRuleInput {
  name: string
  trigger_subject: TriggerSubject
  conditions?: Condition[]
  template_id: string
  channel_override?: MessageChannel
  extra_recipients?: string[]
}

export interface UpdateAutomationRuleInput extends CreateAutomationRuleInput {
  enabled: boolean
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}

export interface TemplateVariable {
  name: string
  type: 'text' | 'list'
  example: unknown
  // For a list: what each repetition offers.
  fields: TemplateVariable[]
}

// A domain event a rule can fire on, with what its templates can use
// (backend GET /message-triggers).
export interface MessageTrigger {
  subject: TriggerSubject
  audience: 'customer' | 'staff'
  has_order: boolean
  variables: TemplateVariable[]
  sample_data: Record<string, unknown>
}

export interface PreviewTemplateInput {
  trigger_subject: TriggerSubject
  subject?: string
  body?: string
  text_body?: string
}

export interface TemplatePreview {
  subject: string
  html: string
  text: string
}

// A built-in default template (GET /message-template-defaults), in the
// store's language.
export interface MessageTemplateDefault {
  key: string
  trigger_subject: TriggerSubject
  name: string
  // One line on what the message is and when it is sent.
  description: string
  category: MessageCategory
  channels: MessageChannel[]
  subject: string
  body: string
  text_body: string
}
