export interface Notification {
  id: string
  type: string
  // Values the type's translated text is filled with (see
  // notificationText); title/message are the English fallback.
  params?: Record<string, unknown>
  title: string
  message: string
  // Frontend-relative deep link (e.g. "/products/<id>") -- empty/absent
  // means no link, clicking only marks it read.
  link?: string
  read: boolean
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
