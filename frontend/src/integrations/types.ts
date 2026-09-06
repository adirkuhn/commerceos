export interface WooTaxClass {
  slug: string
  label: string
}

export interface WooCategory {
  external_id: string
  name: string
  slug: string
}

export interface ExternalCategoryLink {
  id: string
  connection_id: string
  provider: string
  external_id: string
  category_id: string
  created_at: string
  updated_at: string
}

export interface Connection {
  id: string
  provider: string
  store_url: string
  status: 'connected' | 'error'
  default_currency?: string | null
  default_warehouse_id?: string | null
  bol_economic_operator_id?: string | null
  bol_delivery_code?: string | null
  created_at: string
}

export interface CatalogEntry {
  provider: string
  name: string
}

export interface StartWooConnectResponse {
  authorize_url: string
}

export interface BolProductOverride {
  product_id: string
  connection_id: string
  ean_override?: string | null
  gpc_code?: string | null
  condition_category: 'NEW' | 'SECONDHAND' | 'REFURBISHED'
  fulfilment_method: 'FBR' | 'FBB' | 'SHIPPING_VIA_BOL' | 'BOL_DELIVERY_PROMISE'
  commission_override_basis_points?: number | null
  reference_override?: string | null
  custom_price_minor_units?: number | null
  // Bol-only product title override -- falls back to the product's real
  // name when unset. Never synced to any other channel.
  name_override?: string | null
}

// Product-content fields Bol's Product Content API needs to move a
// listing from "Onvoldoende" (insufficient) to "Compleet" -- field list
// extracted directly from Bol's own product-content page. Deliberately
// excludes category/GPC code, tracked separately on BolProductOverride.
export interface BolProductContent {
  product_id: string
  connection_id: string
  manufacturer_contact?: string | null
  manufacturer_address?: string | null
  manufacturer_name?: string | null
  shelf_life_closed?: string | null
  shelf_life_after_opening?: string | null
  ingredients?: string | null
  content_amount?: string | null
  content_unit?: string | null
  country_of_origin?: string | null
  substance?: string | null
  instruction_language?: string | null
  packaging_language?: string | null
  items_per_package?: number | null
  package_insert_url?: string | null
  is_gift?: boolean | null
  dermatologically_tested?: boolean | null
  hazard_statements?: string | null
  skin_type?: string | null
  hypoallergenic?: boolean | null
  contains_zinc_oxide?: boolean | null
  content_per_dispenser_amount?: string | null
  content_per_dispenser_unit?: string | null
  brand?: string | null
  fragrance_free?: boolean | null
  ph_neutral?: boolean | null
  cooling_effect?: boolean | null
  package_width_mm?: number | null
  package_height_mm?: number | null
  package_length_mm?: number | null
  package_weight_g?: number | null
  precaution_statements?: string | null
  precautions?: string | null
  warnings?: string | null
  video_url?: string | null
}

export interface BolCommission {
  ean: string
  fixed_amount: number
  percentage: number
  // True when this came from the merchant's own commission_override_basis_points
  // instead of a live call to Bol's Commissions API.
  overridden: boolean
}

export type ImportJobStatus = 'pending' | 'running' | 'completed' | 'failed'

// The import endpoint enqueues a background job and returns immediately
// instead of blocking on the full sync (which can take minutes for a real
// catalog) — see cmd/scheduler for the worker that actually runs it, and
// the notification center for how progress/completion is surfaced.
export interface ImportJob {
  id: string
  connection_id: string
  status: ImportJobStatus
  imported: number
  already_linked: number
  skipped_type: number
  errors: string[]
  error_message: string
  created_at: string
  started_at?: string | null
  finished_at?: string | null
}

export interface DeleteAllImportedProductsResult {
  deleted_sales_orders: number
  deleted_products: number
  deleted_categories: number
  errors: string[]
}

export interface FetchWooOrderResult {
  external_order_id: string
  sales_order_id?: string | null
  created: boolean
  updated: boolean
  unchanged: boolean
  errors: string[]
}

export type SyncEntityType =
  | 'products' | 'stock' | 'orders' | 'categories' | 'orders_reconciliation' | 'shipment_reconciliation'
  | 'bol_orders' | 'bol_products' | 'bol_stock'
export type SyncDirection = 'import' | 'export'

export interface SyncSchedule {
  entity_type: SyncEntityType
  direction: SyncDirection
  enabled: boolean
  interval_minutes: number
  default_warehouse_id?: string | null
  default_currency?: string | null
  last_run_at?: string | null
  last_run_status: '' | 'success' | 'error'
  last_run_error: string
}

export type ProductChannelListingSyncStatus = 'pending' | 'synced' | 'error'

export interface ProductChannelListing {
  id: string
  product_id: string
  connection_id: string
  provider: string
  external_id: string
  // variant_id is null for the product-level row (a flat single-SKU
  // product, or the parent of a Woo variable product) -- set to a
  // specific Variant's id for a per-variant row (separate-products mode,
  // or a variation nested under the parent in variations mode, see
  // is_variation).
  variant_id?: string | null
  is_variation?: boolean
  sync_status: ProductChannelListingSyncStatus
  last_synced_at?: string | null
  // Bol's own reason(s) the offer isn't for sale (checked after every
  // push) -- empty/absent means for sale (or not a Bol listing / never
  // checked). Distinct from sync_status: a push can be sync_status
  // "synced" (it genuinely succeeded) while Bol still won't list the
  // offer for sale.
  not_for_sale_note?: string
  created_at: string
  updated_at: string
}

export type SyncLogDirection = 'push' | 'import'
export type SyncLogStatus = 'success' | 'error'

export interface FieldChange {
  from: string
  to: string
}

export interface SyncLogEntry {
  id: string
  listing_id: string
  product_id: string
  connection_id: string
  provider: string
  external_id: string
  direction: SyncLogDirection
  status: SyncLogStatus
  error_message: string
  fields: Record<string, string>
  changes: Record<string, FieldChange>
  synced_at: string
  // Set once staff acknowledge this failure from the "needs attention"
  // view without retrying it.
  dismissed_at?: string | null
  // Only populated by listAllSyncLogs (the tenant-wide history list) — the
  // per-listing endpoint's caller already knows the product.
  product_name?: string
  store_url?: string
}

export type AIProvider = 'openai' | 'anthropic' | 'gemini' | 'ollama'

export interface AIConnection {
  id: string
  provider: AIProvider
  base_url?: string | null
  model?: string | null
  status: 'connected' | 'error'
  created_at: string
}

export type ChannelType = 'woocommerce' | 'bol' | 'internal' | 'whatsapp'
export type EnrollmentMode = 'all' | 'category' | 'tag' | 'manual'

export interface SalesChannel {
  id: string
  name: string
  type: ChannelType
  connection_id?: string | null
  enrollment_mode: EnrollmentMode
  enrollment_category_id?: string | null
  enrollment_tag?: string | null
  // Which connected payment integration (Stripe/Revolut) this channel
  // generates a payment link through -- null means disabled. Only
  // meaningful for type 'whatsapp' today.
  payment_connection_id?: string | null
  active: boolean
  created_at: string
  updated_at: string
}

export interface SalesChannelProduct {
  id: string
  name: string
  slug: string
  category_id?: string | null
  tags?: string[]
}

export interface SalesChannelProductEnrollment {
  product: SalesChannelProduct
  enrolled: boolean
}

export type WebhookFallbackStatus = 'pending_republish' | 'republished' | 'abandoned'

// A webhook ReceiveWebhookUseCase (backend) couldn't hand off to JetStream
// after 3 in-request publish retries -- saved here instead of lost. See
// backend/domain/webhooks.FallbackEvent.
export interface WebhookFallbackEvent {
  id: string
  provider: string
  event_id: string
  status: WebhookFallbackStatus
  received_at: string
}

export interface WebhookSecretResult {
  connection_id: string
  webhook_secret: string
}

// A tenant-configured fan-out rule -- "also POST every provider/topic
// webhook to target_url". Topic '' matches every event for provider. See
// backend/domain/webhooks.ForwardRule.
export interface WebhookForwardRule {
  id: string
  provider: string
  topic: string
  target_url: string
  active: boolean
  created_at: string
}

export type WebhookForwardDeliveryStatus = 'pending' | 'delivered' | 'failed'

// A webhook subscription actually registered AT the provider (WooCommerce
// or MyParcel) for this connection -- distinct from WebhookForwardRule,
// which is CommerceOS's own fan-out configuration on top of an event once
// received. See backend/domain/integrations.ProviderWebhook.
export interface ProviderWebhook {
  id: string
  provider: string
  topic: string
  delivery_url: string
  status: string
  // Only set by listAllProviderWebhooks (the tenant-wide aggregate read) --
  // empty on the single-connection listProviderWebhooks response, which
  // already has connectionId in scope via its own request.
  connection_id?: string
  store_label?: string
}

// One attempt to relay a received webhook to one ForwardRule's target_url
// -- the delivery log. See backend/domain/webhooks.ForwardDelivery.
export interface WebhookForwardDelivery {
  id: string
  rule_id: string
  provider: string
  topic: string
  target_url: string
  status: WebhookForwardDeliveryStatus
  attempts: number
  last_error: string
  next_attempt_at: string
  delivered_at: string | null
  created_at: string
}

// One inbound webhook this tenant received -- the cold-path landing record.
// See backend/domain/webhooks.WebhookEvent.
export interface WebhookEvent {
  id: string
  provider: string
  topic: string
  event_id: string
  payload: unknown
  received_at: string
  processed_at: string
  // "received" is the normal terminal state for most events (no per-topic
  // dispatch handler exists for their provider/topic) -- only
  // "processed"/"failed" mean a dispatch handler (myparcel's
  // shipment_status_change today) actually ran.
  status: 'received' | 'processed' | 'failed'
  dispatch_error: string
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
