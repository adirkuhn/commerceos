export type MessageChannel = 'email' | 'sms' | 'whatsapp'

export type MessageCategory = 'purchase' | 'order_status' | 'stock_info' | 'tracking' | 'custom'

export interface MessageTemplate {
  id: string
  name: string
  category: MessageCategory
  channels: MessageChannel[]
  subject: string
  body: string
  enabled: boolean
  created_at: string
  updated_at: string
}

export interface CreateMessageTemplateInput {
  name: string
  category: MessageCategory
  channels: MessageChannel[]
  subject?: string
  body: string
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
