// Hand-written mirror of backend/api/collections/openapi.yaml.

export type CollectionType = 'manual' | 'smart'
export type CollectionMatch = 'all' | 'any'
export type CollectionSort = 'manual' | 'name_asc' | 'price_asc' | 'price_desc' | 'newest'
export type CollectionRuleField = 'category' | 'tag' | 'price' | 'on_sale' | 'added_within_days'

export interface CollectionRule {
  field: CollectionRuleField
  value?: string
  include_subcategories?: boolean
  min_minor_units?: number | null
  max_minor_units?: number | null
  days?: number
}

/** A collection's shopper-facing texts in one language. */
export interface CollectionTexts {
  name?: string
  description?: string
  seo_title?: string
  seo_description?: string
}

export interface CollectionInput {
  name: string
  slug: string
  description?: string
  seo_title?: string
  seo_description?: string
  type: CollectionType
  match?: CollectionMatch
  rules?: CollectionRule[]
  sort: CollectionSort
  published?: boolean
  /** Texts per language other than the company one. */
  translations?: Record<string, CollectionTexts>
}

export interface Collection extends Required<Omit<CollectionInput, 'match' | 'rules' | 'published' | 'translations'>> {
  translations?: Record<string, CollectionTexts>
  id: string
  match: CollectionMatch
  rules: CollectionRule[]
  published: boolean
  image_asset_id?: string | null
  product_ids?: string[]
  created_at: string
  updated_at: string
}
