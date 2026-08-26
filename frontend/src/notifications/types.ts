export interface Notification {
  id: string
  type: string
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
