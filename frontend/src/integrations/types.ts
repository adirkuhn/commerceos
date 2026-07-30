export interface Connection {
  id: string
  provider: string
  store_url: string
  status: 'connected' | 'error'
  created_at: string
}

export interface CatalogEntry {
  provider: string
  name: string
}

export interface StartWooConnectResponse {
  authorize_url: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
