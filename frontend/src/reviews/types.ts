export interface Review {
  id: string
  sales_channel_id: string
  product_id: string
  customer_name: string
  customer_email: string
  rating: number
  title: string
  body: string
  verified: boolean
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
