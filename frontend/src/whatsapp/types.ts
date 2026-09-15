export type WhatsAppMessageDirection = 'inbound' | 'outbound'

// One row of the conversations list -- the "WhatsApp Web"-style admin
// view. See backend/domain/whatsapp.ConversationSummary.
export interface WhatsAppConversation {
  phone_e164: string
  customer_id?: string | null
  customer_name?: string
  last_direction: WhatsAppMessageDirection
  last_body: string
  last_message_at: string
}

// One message in a conversation's full thread. See
// backend/domain/whatsapp.Message.
export interface WhatsAppConversationMessage {
  id: string
  phone_e164: string
  customer_id?: string | null
  direction: WhatsAppMessageDirection
  body: string
  created_at: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
