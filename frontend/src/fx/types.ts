export interface ExchangeRate {
  currency_code: string
  rate_basis_points: number
  updated_at: string
}

export interface UpsertExchangeRateInput {
  currency_code: string
  rate_basis_points: number
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
