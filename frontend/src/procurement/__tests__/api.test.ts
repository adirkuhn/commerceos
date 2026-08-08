// @vitest-environment jsdom
import { describe, it, expect, vi, beforeEach } from 'vitest'
import * as api from '../api'
import type { PurchaseOrder, PurchaseOrderDetail, PurchaseOrderLineItem, DocumentParsingJob, APIResponse } from '../types'

const TOKEN = 'test-token'

function mockFetch<T>(data: T, status = 200) {
  const body: APIResponse<T> = { data, error: null }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }))
}

function mockFetchError(code: string, message: string, status = 422) {
  const body: APIResponse<null> = { data: null, error: { code, message } }
  vi.stubGlobal('fetch', vi.fn().mockResolvedValue({
    ok: false,
    status,
    json: () => Promise.resolve(body),
  }))
}

const samplePO: PurchaseOrder = {
  id: 'po-1',
  tenant_id: 'tenant-1',
  supplier_id: 'supplier-1',
  warehouse_id: 'warehouse-1',
  is_potential_duplicate: false,
  total_minor_units: 0,
  status: 'draft',
  currency_code: 'USD',
  expected_date: null,
  notes: '',
  created_at: '2026-08-05T00:00:00Z',
  updated_at: '2026-08-05T00:00:00Z',
}

const sampleLineItem: PurchaseOrderLineItem = {
  id: 'line-item-1',
  tenant_id: 'tenant-1',
  purchase_order_id: 'po-1',
  sku_id: null,
  raw_sku: '',
  raw_description: 'Widget A',
  position: 0,
  quantity: 10,
  unit_cost_minor_units: 500,
  line_type: 'product',
  line_type_confirmed: true,
  created_at: '2026-08-05T00:00:00Z',
  updated_at: '2026-08-05T00:00:00Z',
}

beforeEach(() => { vi.restoreAllMocks() })

describe('listPurchaseOrders', () => {
  it('returns array of purchase orders', async () => {
    mockFetch([samplePO])
    const result = await api.listPurchaseOrders(TOKEN)
    expect(result).toEqual([samplePO])
  })

  it('calls correct endpoint', async () => {
    mockFetch([])
    await api.listPurchaseOrders(TOKEN)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on server error', async () => {
    mockFetchError('INTERNAL', 'internal server error', 500)
    await expect(api.listPurchaseOrders(TOKEN)).rejects.toThrow('internal server error')
  })
})

describe('createPurchaseOrder', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch(samplePO, 201)
    const input = { supplier_id: 'supplier-1', warehouse_id: 'warehouse-1', currency_code: 'USD' }
    const result = await api.createPurchaseOrder(TOKEN, input)
    expect(result).toEqual(samplePO)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'supplier_id is required', 422)
    await expect(api.createPurchaseOrder(TOKEN, { supplier_id: '', warehouse_id: '', currency_code: '' })).rejects.toThrow('supplier_id is required')
  })
})

describe('getPurchaseOrder', () => {
  it('returns the purchase order detail', async () => {
    const detail: PurchaseOrderDetail = { purchase_order: samplePO, line_items: [sampleLineItem], documents: [] }
    mockFetch(detail)
    const result = await api.getPurchaseOrder(TOKEN, 'po-1')
    expect(result).toEqual(detail)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders/po-1',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'purchase order not found', 404)
    await expect(api.getPurchaseOrder(TOKEN, 'missing')).rejects.toThrow('purchase order not found')
  })
})

describe('updatePurchaseOrder', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch({ ...samplePO, status: 'sent' })
    const input = { supplier_id: 'supplier-1', warehouse_id: 'warehouse-1', status: 'sent' as const, currency_code: 'USD' }
    const result = await api.updatePurchaseOrder(TOKEN, 'po-1', input)
    expect(result.status).toBe('sent')
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders/po-1')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'purchase order not found', 404)
    await expect(api.updatePurchaseOrder(TOKEN, 'missing', { supplier_id: 's', warehouse_id: 'w', status: 'draft', currency_code: 'USD' })).rejects.toThrow('purchase order not found')
  })
})

describe('deletePurchaseOrder', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.deletePurchaseOrder(TOKEN, 'po-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders/po-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'purchase order not found', 404)
    await expect(api.deletePurchaseOrder(TOKEN, 'missing')).rejects.toThrow('purchase order not found')
  })
})

describe('addLineItem', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch(sampleLineItem, 201)
    const input = { raw_description: 'Widget A', quantity: 10, unit_cost_minor_units: 500 }
    const result = await api.addLineItem(TOKEN, 'po-1', input)
    expect(result).toEqual(sampleLineItem)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders/po-1/line-items')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on invalid quantity', async () => {
    mockFetchError('INVALID_QUANTITY', 'quantity must be positive', 422)
    await expect(api.addLineItem(TOKEN, 'po-1', { raw_description: 'X', quantity: 0, unit_cost_minor_units: 0 })).rejects.toThrow('quantity must be positive')
  })
})

describe('updateLineItem', () => {
  it('sends the input in body to the correct endpoint', async () => {
    mockFetch({ ...sampleLineItem, quantity: 25 })
    const input = { raw_description: 'Widget A', quantity: 25, unit_cost_minor_units: 600 }
    const result = await api.updateLineItem(TOKEN, 'line-item-1', input)
    expect(result.quantity).toBe(25)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-order-line-items/line-item-1')
    expect(opts.method).toBe('PATCH')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'line item not found', 404)
    await expect(api.updateLineItem(TOKEN, 'missing', { raw_description: 'X', quantity: 1, unit_cost_minor_units: 0 })).rejects.toThrow('line item not found')
  })
})

describe('removeLineItem', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.removeLineItem(TOKEN, 'line-item-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/procurement/purchase-order-line-items/line-item-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'line item not found', 404)
    await expect(api.removeLineItem(TOKEN, 'missing')).rejects.toThrow('line item not found')
  })
})

const sampleJob: DocumentParsingJob = {
  id: 'job-1',
  tenant_id: 'tenant-1',
  purchase_order_id: 'po-1',
  document_id: 'document-1',
  status: 'pending',
  error_message: '',
  created_at: '2026-08-05T00:00:00Z',
}

function makeFile(name: string, content: string): File {
  return new File([content], name, { type: 'text/csv' })
}

describe('attachDocumentToPurchaseOrder', () => {
  it('posts a multipart form with the file and returns the parsing job', async () => {
    mockFetch(sampleJob, 201)
    const file = makeFile('order.csv', 'sku,qty\nA,1')
    const result = await api.attachDocumentToPurchaseOrder(TOKEN, 'po-1', file)

    expect(result).toEqual(sampleJob)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders/po-1/documents')
    expect(opts.method).toBe('POST')
    expect(opts.body).toBeInstanceOf(FormData)
    expect((opts.body as FormData).get('file')).toBeInstanceOf(File)
    expect((opts.headers as Record<string, string>)['Content-Type']).toBeUndefined()
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'purchase order not found', 404)
    await expect(api.attachDocumentToPurchaseOrder(TOKEN, 'missing', makeFile('order.csv', 'x'))).rejects.toThrow('purchase order not found')
  })
})

describe('deletePurchaseOrderDocument', () => {
  it('calls the correct endpoint with DELETE', async () => {
    mockFetch(null)
    await api.deletePurchaseOrderDocument(TOKEN, 'document-1')
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/procurement/purchase-order-documents/document-1',
      expect.objectContaining({ method: 'DELETE' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'purchase order document not found', 404)
    await expect(api.deletePurchaseOrderDocument(TOKEN, 'missing')).rejects.toThrow('purchase order document not found')
  })
})

describe('createPurchaseOrderFromDocument', () => {
  it('posts a multipart form with the file and header fields', async () => {
    mockFetch(sampleJob, 201)
    const file = makeFile('order.csv', 'sku,qty\nA,1')
    const result = await api.createPurchaseOrderFromDocument(TOKEN, {
      supplier_id: 'supplier-1', warehouse_id: 'warehouse-1', currency_code: 'USD', file,
    })

    expect(result).toEqual(sampleJob)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders/documents')
    const fd = opts.body as FormData
    expect(fd.get('supplier_id')).toBe('supplier-1')
    expect(fd.get('warehouse_id')).toBe('warehouse-1')
    expect(fd.get('currency_code')).toBe('USD')
    expect(fd.get('file')).toBeInstanceOf(File)
  })

  it('throws on validation error', async () => {
    mockFetchError('VALIDATION_ERROR', 'document is missing required columns', 422)
    await expect(api.createPurchaseOrderFromDocument(TOKEN, {
      supplier_id: 'supplier-1', warehouse_id: 'warehouse-1', currency_code: 'USD', file: makeFile('bad.csv', 'nonsense'),
    })).rejects.toThrow('document is missing required columns')
  })

  it('omits supplier_id from the form when not provided', async () => {
    mockFetch(sampleJob, 201)
    const file = makeFile('order.csv', 'sku,qty\nA,1')
    await api.createPurchaseOrderFromDocument(TOKEN, { warehouse_id: 'warehouse-1', currency_code: 'USD', file })

    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    const fd = opts.body as FormData
    expect(fd.get('supplier_id')).toBeNull()
  })

  it('includes expected_date when provided', async () => {
    mockFetch(sampleJob, 201)
    const file = makeFile('order.csv', 'sku,qty\nA,1')
    await api.createPurchaseOrderFromDocument(TOKEN, {
      warehouse_id: 'warehouse-1', currency_code: 'USD', expected_date: '2026-08-05', file,
    })

    const [, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    const fd = opts.body as FormData
    expect(fd.get('expected_date')).toBe('2026-08-05')
  })
})

describe('linkLineItemToSku', () => {
  it('sends the sku_id to the correct endpoint', async () => {
    mockFetch(sampleLineItem)
    const result = await api.linkLineItemToSku(TOKEN, 'line-item-1', { sku_id: 'sku-1' })

    expect(result).toEqual(sampleLineItem)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-order-line-items/line-item-1/link')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual({ sku_id: 'sku-1' })
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'line item not found', 404)
    await expect(api.linkLineItemToSku(TOKEN, 'missing', { sku_id: 'sku-1' })).rejects.toThrow('line item not found')
  })
})

describe('createProductAndLinkLineItem', () => {
  it('sends the new-product fields to the correct endpoint', async () => {
    mockFetch(sampleLineItem)
    const input = { product_name: 'New Widget', sku_code: 'NEW-001', currency_code: 'USD', price_minor_units: 999 }
    const result = await api.createProductAndLinkLineItem(TOKEN, 'line-item-1', input)

    expect(result).toEqual(sampleLineItem)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-order-line-items/line-item-1/create-product-and-link')
    expect(JSON.parse(opts.body as string)).toEqual(input)
  })

  it('throws on duplicate sku code', async () => {
    mockFetchError('VALIDATION_ERROR', 'sku code already exists', 422)
    await expect(api.createProductAndLinkLineItem(TOKEN, 'line-item-1', {
      product_name: 'X', sku_code: 'DUP', currency_code: 'USD',
    })).rejects.toThrow('sku code already exists')
  })
})

describe('createSupplierAndAssignToPurchaseOrder', () => {
  it('sends the supplier name to the correct endpoint', async () => {
    mockFetch(samplePO)
    const result = await api.createSupplierAndAssignToPurchaseOrder(TOKEN, 'po-1', { name: 'New Supplier Co' })

    expect(result).toEqual(samplePO)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders/po-1/create-supplier-and-assign')
    expect(opts.method).toBe('POST')
    expect(JSON.parse(opts.body as string)).toEqual({ name: 'New Supplier Co' })
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'purchase order not found', 404)
    await expect(api.createSupplierAndAssignToPurchaseOrder(TOKEN, 'missing', { name: 'X' })).rejects.toThrow('purchase order not found')
  })
})

const sampleReceivingJob: import('../types').ReceivingParsingJob = {
  id: 'receiving-job-1',
  tenant_id: 'tenant-1',
  receiving_record_id: 'receiving-record-1',
  status: 'pending',
  error_message: '',
  created_at: '2026-08-05T00:00:00Z',
}

const sampleReceivingRecord: import('../types').ReceivingRecord = {
  id: 'receiving-record-1',
  tenant_id: 'tenant-1',
  purchase_order_id: 'po-1',
  invoice_file_url: 'https://example.com/invoice.csv',
  invoice_file_name: 'invoice.csv',
  content_type: 'text/csv',
  status: 'pending_review',
  is_potential_duplicate: false,
  created_at: '2026-08-05T00:00:00Z',
  updated_at: '2026-08-05T00:00:00Z',
}

describe('uploadInvoice', () => {
  it('posts a multipart form with the file and returns the parsing job', async () => {
    mockFetch(sampleReceivingJob, 201)
    const file = makeFile('invoice.csv', 'sku,qty\nA,1')
    const result = await api.uploadInvoice(TOKEN, 'po-1', file)

    expect(result).toEqual(sampleReceivingJob)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders/po-1/invoices')
    expect(opts.method).toBe('POST')
    expect(opts.body).toBeInstanceOf(FormData)
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'purchase order not found', 404)
    await expect(api.uploadInvoice(TOKEN, 'missing', makeFile('invoice.csv', 'x'))).rejects.toThrow('purchase order not found')
  })
})

describe('listReceivingRecords', () => {
  it('returns array of receiving records', async () => {
    mockFetch([sampleReceivingRecord])
    const result = await api.listReceivingRecords(TOKEN, 'po-1')

    expect(result).toEqual([sampleReceivingRecord])
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/procurement/purchase-orders/po-1/receiving-records',
      expect.objectContaining({ method: 'GET' }),
    )
  })
})

describe('getReceivingRecord', () => {
  it('returns record with discrepancies', async () => {
    const detail: import('../types').ReceivingRecordDetail = {
      receiving_record: sampleReceivingRecord,
      discrepancies: [{
        id: 'discrepancy-1', tenant_id: 'tenant-1', receiving_record_id: 'receiving-record-1',
        line_item_id: 'line-item-1', type: 'quantity_mismatch', expected_qty: 10, received_qty: 8,
        details: 'expected 10, invoice shows 8', created_at: '2026-08-05T00:00:00Z',
      }],
    }
    mockFetch(detail)
    const result = await api.getReceivingRecord(TOKEN, 'receiving-record-1')

    expect(result).toEqual(detail)
    expect(vi.mocked(fetch)).toHaveBeenCalledWith(
      'https://api.commerceos.dev.localhost/api/v1/procurement/receiving-records/receiving-record-1',
      expect.objectContaining({ method: 'GET' }),
    )
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'receiving record not found', 404)
    await expect(api.getReceivingRecord(TOKEN, 'missing')).rejects.toThrow('receiving record not found')
  })
})

describe('completeReceiving', () => {
  it('posts to the correct endpoint and returns the applied record', async () => {
    const applied = { ...sampleReceivingRecord, status: 'applied' as const }
    mockFetch(applied)
    const result = await api.completeReceiving(TOKEN, 'receiving-record-1')

    expect(result).toEqual(applied)
    const [url, opts] = vi.mocked(fetch).mock.calls[0] as [string, RequestInit]
    expect(url).toBe('https://api.commerceos.dev.localhost/api/v1/procurement/receiving-records/receiving-record-1/complete')
    expect(opts.method).toBe('POST')
  })

  it('throws on not found', async () => {
    mockFetchError('NOT_FOUND', 'receiving record not found', 404)
    await expect(api.completeReceiving(TOKEN, 'missing')).rejects.toThrow('receiving record not found')
  })
})
