export type PromotionType = 'percentage' | 'fixed_amount' | 'fixed_price'
export type PromotionStatus = 'draft' | 'active' | 'ended'

export interface PromotionLine {
  id: string
  sku_id: string
  // The SKU's current code, for display -- empty string if the SKU has
  // since been deleted.
  sku_code: string
  discount_type: PromotionType
  // Basis points (1-10000) for percentage, minor currency units for fixed_amount (amount off) and fixed_price (the sale price).
  discount_value: number
  // Set on a clearance line: the batch being sold off. The line ends on its
  // own (SKU back to its pre-promotion price) once the batch is sold out,
  // discarded, removed or expired.
  batch_id?: string | null
  // When a clearance line's batch finished; absent while it's still running.
  finished_at?: string | null
}

export interface PromotionLineInput {
  sku_id: string
  batch_id?: string | null
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
