export type PromotionType = 'percentage' | 'fixed_amount'
export type PromotionStatus = 'draft' | 'active' | 'ended'

export interface PromotionLine {
  id: string
  sku_id: string
  // The SKU's current code, for display -- empty string if the SKU has
  // since been deleted.
  sku_code: string
  discount_type: PromotionType
  // Basis points (1-10000) for percentage, minor currency units for fixed_amount.
  discount_value: number
}

export interface PromotionLineInput {
  sku_id: string
  discount_type: PromotionType
  discount_value: number
}

export interface Promotion {
  id: string
  name: string
  type: PromotionType
  starts_at: string
  ends_at?: string | null
  status: PromotionStatus
  created_at: string
  updated_at: string
}

export interface PromotionDetail extends Promotion {
  lines?: PromotionLine[]
}

export interface CreatePromotionResult {
  promotion: PromotionDetail
  // Non-blocking warnings, e.g. a targeted SKU already had a manual sale
  // price -- this promotion overrides it while active, but creation is
  // not blocked.
  warnings: string[]
}

export interface PromotionInput {
  name: string
  type: PromotionType
  starts_at: string
  ends_at?: string | null
  status?: PromotionStatus
  lines: PromotionLineInput[]
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
