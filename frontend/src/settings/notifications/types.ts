// Mirrors backend/api/settings/openapi.yaml (NotificationPreferences*).
export type NotificationCategory = 'imports' | 'purchase_documents' | 'channel_offers' | 'stock' | 'other'

export interface NotificationCategoryPreference {
  category: NotificationCategory
  in_app: boolean
  email: boolean
}

export interface NotificationPreferencesInput {
  categories: NotificationCategoryPreference[]
  email_recipients: string[]
}

export interface NotificationPreferences extends NotificationPreferencesInput {
  updated_at: string | null
}
