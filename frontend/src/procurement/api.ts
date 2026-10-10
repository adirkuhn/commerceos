import type {
  PurchaseOrder, PurchaseOrderDetail, PurchaseOrderLineItem, DocumentParsingJob,
  CreatePurchaseOrderInput, UpdatePurchaseOrderInput,
  AddLineItemInput, UpdateLineItemInput,
  CreatePurchaseOrderFromDocumentInput, LinkLineItemInput, CreateProductAndLinkLineItemInput,
  CreateSupplierAndAssignInput,
  ReceivingParsingJob, ReceivingRecord, ReceivingRecordDetail, ReceivingDiscrepancy,
  BookingDiscrepancyReport,
  APIResponse,
} from './types'
import { CodedApiError } from '../api/codedApiError'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/procurement`
}

async function request<T>(
  path: string,
  options: RequestInit,
  token: string,
): Promise<T> {
  const res = await fetch(path, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      Authorization: `Bearer ${token}`,
      ...options.headers,
    },
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as T
}

export function listPurchaseOrders(token: string): Promise<PurchaseOrder[]> {
  return request<PurchaseOrder[]>(`${base()}/purchase-orders`, { method: 'GET' }, token)
}

export function createPurchaseOrder(token: string, input: CreatePurchaseOrderInput): Promise<PurchaseOrder> {
  return request<PurchaseOrder>(`${base()}/purchase-orders`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function getPurchaseOrder(token: string, id: string): Promise<PurchaseOrderDetail> {
  return request<PurchaseOrderDetail>(`${base()}/purchase-orders/${id}`, { method: 'GET' }, token)
}

export function updatePurchaseOrder(token: string, id: string, input: UpdatePurchaseOrderInput): Promise<PurchaseOrder> {
  return request<PurchaseOrder>(`${base()}/purchase-orders/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function deletePurchaseOrder(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/purchase-orders/${id}`, { method: 'DELETE' }, token)
}

export function addLineItem(token: string, purchaseOrderId: string, input: AddLineItemInput): Promise<PurchaseOrderLineItem> {
  return request<PurchaseOrderLineItem>(`${base()}/purchase-orders/${purchaseOrderId}/line-items`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function updateLineItem(token: string, id: string, input: UpdateLineItemInput): Promise<PurchaseOrderLineItem> {
  return request<PurchaseOrderLineItem>(`${base()}/purchase-order-line-items/${id}`, { method: 'PATCH', body: JSON.stringify(input) }, token)
}

export function removeLineItem(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/purchase-order-line-items/${id}`, { method: 'DELETE' }, token)
}

async function requestMultipart<T>(path: string, formData: FormData, token: string): Promise<T> {
  const res = await fetch(path, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
    body: formData,
  })
  const body: APIResponse<T> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as T
}

export function attachDocumentToPurchaseOrder(token: string, purchaseOrderId: string, file: File): Promise<DocumentParsingJob> {
  const formData = new FormData()
  formData.append('file', file)
  return requestMultipart<DocumentParsingJob>(`${base()}/purchase-orders/${purchaseOrderId}/documents`, formData, token)
}

export function deletePurchaseOrderDocument(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/purchase-order-documents/${id}`, { method: 'DELETE' }, token)
}

export function getDocumentParsingJob(token: string, id: string): Promise<DocumentParsingJob> {
  return request<DocumentParsingJob>(`${base()}/document-parsing-jobs/${id}`, { method: 'GET' }, token)
}

export function retryDocumentParsingJob(token: string, documentId: string): Promise<DocumentParsingJob> {
  return request<DocumentParsingJob>(`${base()}/purchase-order-documents/${documentId}/retry`, { method: 'POST' }, token)
}

export function createPurchaseOrderFromDocument(token: string, input: CreatePurchaseOrderFromDocumentInput): Promise<DocumentParsingJob> {
  const formData = new FormData()
  formData.append('file', input.file)
  if (input.supplier_id) formData.append('supplier_id', input.supplier_id)
  if (input.warehouse_id) formData.append('warehouse_id', input.warehouse_id)
  formData.append('currency_code', input.currency_code)
  if (input.expected_date) formData.append('expected_date', input.expected_date)
  return requestMultipart<DocumentParsingJob>(`${base()}/purchase-orders/documents`, formData, token)
}

export function linkLineItemToSku(token: string, lineItemId: string, input: LinkLineItemInput): Promise<PurchaseOrderLineItem> {
  return request<PurchaseOrderLineItem>(`${base()}/purchase-order-line-items/${lineItemId}/link`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function createProductAndLinkLineItem(token: string, lineItemId: string, input: CreateProductAndLinkLineItemInput): Promise<PurchaseOrderLineItem> {
  return request<PurchaseOrderLineItem>(`${base()}/purchase-order-line-items/${lineItemId}/create-product-and-link`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function createSupplierAndAssignToPurchaseOrder(token: string, purchaseOrderId: string, input: CreateSupplierAndAssignInput): Promise<PurchaseOrder> {
  return request<PurchaseOrder>(`${base()}/purchase-orders/${purchaseOrderId}/create-supplier-and-assign`, { method: 'POST', body: JSON.stringify(input) }, token)
}

// ── Receiving ────────────────────────────────────────────────────────────────

export function uploadInvoice(token: string, purchaseOrderId: string, file: File): Promise<ReceivingParsingJob> {
  const formData = new FormData()
  formData.append('file', file)
  return requestMultipart<ReceivingParsingJob>(`${base()}/purchase-orders/${purchaseOrderId}/invoices`, formData, token)
}

export function getReceivingParsingJob(token: string, id: string): Promise<ReceivingParsingJob> {
  return request<ReceivingParsingJob>(`${base()}/receiving-parsing-jobs/${id}`, { method: 'GET' }, token)
}

export function listReceivingRecords(token: string, purchaseOrderId: string): Promise<ReceivingRecord[]> {
  return request<ReceivingRecord[]>(`${base()}/purchase-orders/${purchaseOrderId}/receiving-records`, { method: 'GET' }, token)
}

export function getReceivingRecord(token: string, id: string): Promise<ReceivingRecordDetail> {
  return request<ReceivingRecordDetail>(`${base()}/receiving-records/${id}`, { method: 'GET' }, token)
}

export function completeReceiving(token: string, id: string): Promise<ReceivingRecord> {
  return request<ReceivingRecord>(`${base()}/receiving-records/${id}/complete`, { method: 'POST' }, token)
}

export function acknowledgeReceivingDiscrepancy(token: string, discrepancyId: string): Promise<ReceivingDiscrepancy> {
  return request<ReceivingDiscrepancy>(`${base()}/receiving-discrepancies/${discrepancyId}/acknowledge`, { method: 'POST' }, token)
}

export function linkReceivingDiscrepancies(token: string, unexpectedDiscrepancyId: string, missingDiscrepancyId: string): Promise<void> {
  return request<void>(`${base()}/receiving-discrepancies/link`, {
    method: 'POST',
    body: JSON.stringify({ unexpected_discrepancy_id: unexpectedDiscrepancyId, missing_discrepancy_id: missingDiscrepancyId }),
  }, token)
}

export function flagDamagedLineItem(
  token: string, receivingRecordId: string, lineItemId: string, damagedQty: number, note: string,
): Promise<ReceivingDiscrepancy> {
  return request<ReceivingDiscrepancy>(`${base()}/receiving-records/${receivingRecordId}/line-items/${lineItemId}/damaged`, {
    method: 'POST',
    body: JSON.stringify({ damaged_qty: damagedQty, note }),
  }, token)
}

export function listBookingDiscrepancies(token: string): Promise<BookingDiscrepancyReport> {
  return request<BookingDiscrepancyReport>(`${base()}/bookings/discrepancies`, { method: 'GET' }, token)
}
