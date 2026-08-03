export interface Notification {
  id: string
  type: string
  title: string
  message: string
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
