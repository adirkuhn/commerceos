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

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
