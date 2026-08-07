export type PurchaseOrderStatus = 'draft' | 'sent' | 'partially_received' | 'received' | 'cancelled' | 'needs_review'

export interface PurchaseOrder {
  id: string
  tenant_id: string
  // Nil when uploaded without a supplier picked up front -- the PO starts
  // needs_review until one is linked or created via the review UI's
  // one-click "create supplier" action.
  supplier_id?: string | null
  // Nil until stock is actually received against this PO -- assigned at
  // the first booking/receiving action, never required at order time.
  warehouse_id?: string | null
  status: PurchaseOrderStatus
  currency_code: string
  expected_date?: string | null
  notes: string
  // AI/deterministic extraction suggestions from the uploaded document,
  // set only while supplier_id is nil -- '' means no suggestion.
  suggested_supplier_name?: string
  suggested_currency_code?: string
  // Set when a document attached to this PO has the same content as a
  // document attached to a *different* PO -- tenant-wide, not scoped to
  // supplier.
  is_potential_duplicate: boolean
  duplicate_of_purchase_order_id?: string | null
  created_at: string
  updated_at: string
}

// LineItemType: 'product' (or '', treated the same) is an actual item that
// needs a SKU match. Anything else is a charge/adjustment row that must
// never be forced through product matching.
export type LineItemType = '' | 'product' | 'shipping' | 'discount' | 'credit_note' | 'fee' | 'other_charge'

export interface PurchaseOrderLineItem {
  id: string
  tenant_id: string
  purchase_order_id: string
  sku_id?: string | null
  raw_sku: string
  raw_description: string
  quantity: number
  unit_cost_minor_units: number
  line_type: LineItemType
  // False when line_type came from the AI's classification and hasn't been
  // reviewed by a human yet.
  line_type_confirmed: boolean
  created_at: string
  updated_at: string
}

export interface PurchaseOrderDocument {
  id: string
  purchase_order_id: string
  // Already a directly fetchable URL (see backend pms.FileStorage.Save) --
  // no separate download endpoint needed, just link straight to it.
  file_url: string
  file_name: string
  content_type: string
  // Set when another document with the same content already exists on
  // this PO or another PO from the same supplier -- not a block, just a
  // heads-up.
  is_potential_duplicate: boolean
  created_at: string
}

export interface PurchaseOrderDetail {
  purchase_order: PurchaseOrder
  line_items: PurchaseOrderLineItem[]
  documents: PurchaseOrderDocument[]
}

export interface CreatePurchaseOrderInput {
  supplier_id: string
  warehouse_id?: string | null
  currency_code: string
  expected_date?: string | null
  notes?: string
}

export interface UpdatePurchaseOrderInput {
  supplier_id: string
  warehouse_id?: string | null
  status: PurchaseOrderStatus
  currency_code: string
  expected_date?: string | null
  notes?: string
}

export interface AddLineItemInput {
  sku_id?: string | null
  raw_sku?: string
  raw_description: string
  quantity: number
  unit_cost_minor_units: number
  line_type?: LineItemType
}

export type UpdateLineItemInput = AddLineItemInput

export type DocumentParsingJobStatus = 'pending' | 'running' | 'completed' | 'failed'

export interface DocumentParsingJob {
  id: string
  tenant_id: string
  purchase_order_id: string
  document_id: string
  status: DocumentParsingJobStatus
  error_message: string
  created_at: string
  started_at?: string | null
  finished_at?: string | null
}

export interface CreatePurchaseOrderFromDocumentInput {
  // Omit to upload without picking a supplier -- the resulting PO starts
  // needs_review until one is linked or created via the review UI.
  supplier_id?: string
  warehouse_id?: string
  currency_code: string
  expected_date?: string
  file: File
}

export interface LinkLineItemInput {
  sku_id: string
}

export interface CreateSupplierAndAssignInput {
  name: string
}

export interface CreateProductAndLinkLineItemInput {
  product_name: string
  sku_code: string
  barcode?: string
  price_minor_units?: number
  currency_code: string
}

export type ReceivingParsingJobStatus = 'pending' | 'running' | 'completed' | 'failed'

export interface ReceivingParsingJob {
  id: string
  tenant_id: string
  receiving_record_id: string
  status: ReceivingParsingJobStatus
  error_message: string
  created_at: string
  started_at?: string | null
  finished_at?: string | null
}

export type ReceivingRecordStatus = 'pending_review' | 'applied'

export interface ReceivingRecord {
  id: string
  tenant_id: string
  purchase_order_id: string
  invoice_file_url: string
  invoice_file_name: string
  content_type: string
  status: ReceivingRecordStatus
  is_potential_duplicate: boolean
  created_at: string
  updated_at: string
}

export type DiscrepancyType = 'quantity_mismatch' | 'missing_item' | 'unexpected_item' | 'expiry_mismatch'

export interface ReceivingDiscrepancy {
  id: string
  tenant_id: string
  receiving_record_id: string
  // Nil for unexpected_item, since there's no PO line item to point at.
  line_item_id?: string | null
  type: DiscrepancyType
  expected_qty: number
  received_qty: number
  details: string
  created_at: string
}

export interface ReceivingRecordDetail {
  receiving_record: ReceivingRecord
  discrepancies: ReceivingDiscrepancy[]
}

export interface APIError {
  code: string
  message: string
}

export interface APIResponse<T> {
  data: T | null
  error: APIError | null
}
