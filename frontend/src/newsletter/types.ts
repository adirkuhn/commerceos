export type SubscriberStatus = 'pending' | 'confirmed' | 'unsubscribed'

export const SUBSCRIBER_STATUSES: readonly SubscriberStatus[] = ['pending', 'confirmed', 'unsubscribed']

export interface Subscriber {
  id: string
  email: string
  status: SubscriberStatus
  sales_channel_id?: string | null
  consent_at: string
  confirmed_at?: string | null
  unsubscribed_at?: string | null
  created_at: string
}

export interface SubscriberQuery {
  status?: SubscriberStatus
  search?: string
  page: number
  perPage: number
}
