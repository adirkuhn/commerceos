import type {
  ProductTexts,
  Category, Product, ProductDetail, Variant, SKU, Media, ResolveBarcodeResult,
  CreateCategoryPayload, UpdateCategoryPayload,
  CreateProductPayload, UpdateProductPayload,
  CreateVariantPayload, UpdateVariantPayload,
  CreateSKUPayload, UpdateSKUPayload,
  SKUMarginInsight, SKUCostPriceHistoryEntry, SKUPriceHistoryEntry, MarginReportRow, SKUSupplierCosts,
  ProductHistoryEntry,
  SKUCurrencyPrice, UpsertSKUCurrencyPriceInput,
  BarcodeAlias, CreateBarcodeAliasPayload,
  ScaleCodeAlias, CreateScaleCodeAliasPayload,
  PackWeightPortionsInput, PackWeightPortionsResult, LinkExistingWeightPackagesResult,
  ProductMergeRequest, ProductMergePreview, ProductMergeResult,
  APIResponse,
} from './types'
import { CodedApiError } from '../api/codedApiError'
import { appEnv } from '../config/appEnv'

function base(): string {
  return `${appEnv().VITE_API_BASE_URL}/pms`
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

// ── Categories ────────────────────────────────────────────────────────────────

export function listCategories(token: string): Promise<Category[]> {
  return request<Category[]>(`${base()}/categories`, { method: 'GET' }, token)
}

export function createCategory(token: string, payload: CreateCategoryPayload): Promise<Category> {
  return request<Category>(`${base()}/categories`, { method: 'POST', body: JSON.stringify(payload) }, token)
}

export function updateCategory(token: string, id: string, payload: UpdateCategoryPayload): Promise<Category> {
  return request<Category>(`${base()}/categories/${id}`, { method: 'PATCH', body: JSON.stringify(payload) }, token)
}

export function deleteCategory(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/categories/${id}`, { method: 'DELETE' }, token)
}

// ── Products ──────────────────────────────────────────────────────────────────

// low/out: only products low on / out of stock -- the server filters,
// counts and paginates (see openapi listProducts).
export type ProductStockFilter = 'low' | 'out'

export interface ListProductsParams {
  page?: number
  perPage?: number
  search?: string
  stock?: ProductStockFilter
}

export interface ListProductsResult {
  products: Product[]
  total: number
}

export async function listProducts(token: string, params: ListProductsParams = {}): Promise<ListProductsResult> {
  const qs = new URLSearchParams()
  if (params.page) qs.set('page', String(params.page))
  if (params.perPage) qs.set('per_page', String(params.perPage))
  if (params.search) qs.set('search', params.search)
  if (params.stock) qs.set('stock', params.stock)
  const query = qs.toString()

  const res = await fetch(`${base()}/products${query ? `?${query}` : ''}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<Product[]> & { total?: number } = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return { products: body.data ?? [], total: body.total ?? 0 }
}

export function createProduct(token: string, payload: CreateProductPayload): Promise<Product> {
  return request<Product>(`${base()}/products`, { method: 'POST', body: JSON.stringify(payload) }, token)
}

export function getProduct(token: string, id: string): Promise<ProductDetail> {
  return request<ProductDetail>(`${base()}/products/${id}`, { method: 'GET' }, token)
}

export function updateProduct(token: string, id: string, payload: UpdateProductPayload): Promise<Product> {
  return request<Product>(`${base()}/products/${id}`, { method: 'PATCH', body: JSON.stringify(payload) }, token)
}

// Replaces the product's name/description in the store's other languages.
export function setProductTranslations(token: string, id: string, translations: Record<string, ProductTexts>): Promise<Product> {
  return request<Product>(`${base()}/products/${id}/translations`, { method: 'PUT', body: JSON.stringify({ translations }) }, token)
}

// Sets the product's own low stock line, in stock units (grams for a
// weighed product); null = back to the shop's setting.
export function setProductLowStockThreshold(token: string, id: string, threshold: number | null): Promise<Product> {
  return request<Product>(`${base()}/products/${id}/low-stock-threshold`, { method: 'PUT', body: JSON.stringify({ low_stock_threshold: threshold }) }, token)
}

export function deleteProduct(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/products/${id}`, { method: 'DELETE' }, token)
}

// Clones name/category/tags plus every variant and SKU into a new product,
// always draft and never pushed to any channel.
export function duplicateProduct(token: string, id: string): Promise<ProductDetail> {
  return request<ProductDetail>(`${base()}/products/${id}/duplicate`, { method: 'POST' }, token)
}

// ── Variants ──────────────────────────────────────────────────────────────────

export function createVariant(token: string, productId: string, payload: CreateVariantPayload): Promise<Variant> {
  return request<Variant>(`${base()}/products/${productId}/variants`, { method: 'POST', body: JSON.stringify(payload) }, token)
}

export function updateVariant(token: string, id: string, payload: UpdateVariantPayload): Promise<Variant> {
  return request<Variant>(`${base()}/variants/${id}`, { method: 'PATCH', body: JSON.stringify(payload) }, token)
}

export function deleteVariant(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/variants/${id}`, { method: 'DELETE' }, token)
}

// ── SKUs ──────────────────────────────────────────────────────────────────────

export function createSKU(token: string, variantId: string, payload: CreateSKUPayload): Promise<SKU> {
  return request<SKU>(`${base()}/variants/${variantId}/skus`, { method: 'POST', body: JSON.stringify(payload) }, token)
}

export function updateSKU(token: string, id: string, payload: UpdateSKUPayload): Promise<SKU> {
  return request<SKU>(`${base()}/skus/${id}`, { method: 'PATCH', body: JSON.stringify(payload) }, token)
}

export function deleteSKU(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/skus/${id}`, { method: 'DELETE' }, token)
}

export function getSKUMarginInsight(token: string, id: string): Promise<SKUMarginInsight> {
  return request<SKUMarginInsight>(`${base()}/skus/${id}/margin-insight`, { method: 'GET' }, token)
}

export function listSKUCurrencyPrices(token: string, id: string): Promise<SKUCurrencyPrice[]> {
  return request<SKUCurrencyPrice[]>(`${base()}/skus/${id}/currency-prices`, { method: 'GET' }, token)
}

export function upsertSKUCurrencyPrice(token: string, id: string, payload: UpsertSKUCurrencyPriceInput): Promise<SKUCurrencyPrice> {
  return request<SKUCurrencyPrice>(`${base()}/skus/${id}/currency-prices`, { method: 'PUT', body: JSON.stringify(payload) }, token)
}

export function deleteSKUCurrencyPrice(token: string, id: string, currencyCode: string): Promise<null> {
  return request<null>(`${base()}/skus/${id}/currency-prices/${currencyCode}`, { method: 'DELETE' }, token)
}

export function listBarcodeAliases(token: string, id: string): Promise<BarcodeAlias[]> {
  return request<BarcodeAlias[]>(`${base()}/skus/${id}/barcode-aliases`, { method: 'GET' }, token)
}

export function createBarcodeAlias(token: string, id: string, payload: CreateBarcodeAliasPayload): Promise<BarcodeAlias> {
  return request<BarcodeAlias>(`${base()}/skus/${id}/barcode-aliases`, { method: 'POST', body: JSON.stringify(payload) }, token)
}

export function deleteBarcodeAlias(token: string, aliasId: string): Promise<null> {
  return request<null>(`${base()}/barcode-aliases/${aliasId}`, { method: 'DELETE' }, token)
}

export function listScaleCodeAliases(token: string, id: string): Promise<ScaleCodeAlias[]> {
  return request<ScaleCodeAlias[]>(`${base()}/skus/${id}/scale-code-aliases`, { method: 'GET' }, token)
}

export function createScaleCodeAlias(token: string, id: string, payload: CreateScaleCodeAliasPayload): Promise<ScaleCodeAlias> {
  return request<ScaleCodeAlias>(`${base()}/skus/${id}/scale-code-aliases`, { method: 'POST', body: JSON.stringify(payload) }, token)
}

export function deleteScaleCodeAlias(token: string, aliasId: string): Promise<null> {
  return request<null>(`${base()}/scale-code-aliases/${aliasId}`, { method: 'DELETE' }, token)
}

export function listSKUCostPriceHistory(token: string, id: string, limit?: number): Promise<SKUCostPriceHistoryEntry[]> {
  const qs = limit ? `?limit=${limit}` : ''
  return request<SKUCostPriceHistoryEntry[]>(`${base()}/skus/${id}/cost-price-history${qs}`, { method: 'GET' }, token)
}

export function listSKUSupplierCosts(
  token: string, id: string, opts: { supplierId?: string; cursor?: number; limit?: number } = {},
): Promise<SKUSupplierCosts> {
  const params = new URLSearchParams()
  if (opts.supplierId) params.set('supplier_id', opts.supplierId)
  if (opts.cursor) params.set('cursor', String(opts.cursor))
  if (opts.limit) params.set('limit', String(opts.limit))
  const qs = params.toString() ? `?${params}` : ''
  return request<SKUSupplierCosts>(`${base()}/skus/${id}/supplier-costs${qs}`, { method: 'GET' }, token)
}

export function listSKUPriceHistory(token: string, id: string, limit?: number): Promise<SKUPriceHistoryEntry[]> {
  const qs = limit ? `?limit=${limit}` : ''
  return request<SKUPriceHistoryEntry[]>(`${base()}/skus/${id}/price-history${qs}`, { method: 'GET' }, token)
}

export function listProductHistory(token: string, id: string): Promise<ProductHistoryEntry[]> {
  return request<ProductHistoryEntry[]>(`${base()}/products/${id}/history`, { method: 'GET' }, token)
}

export function listVariantHistory(token: string, id: string): Promise<ProductHistoryEntry[]> {
  return request<ProductHistoryEntry[]>(`${base()}/variants/${id}/history`, { method: 'GET' }, token)
}

export function setManualCostPrice(token: string, id: string, costPriceMinorUnits: number): Promise<SKU> {
  return request<SKU>(`${base()}/skus/${id}/cost-price`, { method: 'PUT', body: JSON.stringify({ cost_price_minor_units: costPriceMinorUnits }) }, token)
}

export function clearCostPriceOverride(token: string, id: string): Promise<SKU> {
  return request<SKU>(`${base()}/skus/${id}/cost-price`, { method: 'DELETE' }, token)
}

export interface ListMarginReportParams {
  page?: number
  perPage?: number
  belowTargetOnly?: boolean
}

export interface ListMarginReportResult {
  rows: MarginReportRow[]
  total: number
}

export async function listMarginReport(token: string, params: ListMarginReportParams = {}): Promise<ListMarginReportResult> {
  const qs = new URLSearchParams()
  if (params.page) qs.set('page', String(params.page))
  if (params.perPage) qs.set('per_page', String(params.perPage))
  if (params.belowTargetOnly) qs.set('below_target_only', 'true')
  const res = await fetch(`${base()}/margin-report?${qs.toString()}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<MarginReportRow[]> & { meta?: { total?: number } } = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return { rows: body.data ?? [], total: body.meta?.total ?? 0 }
}

// resolveBarcode is used by every scanner mini-app that turns a scan into a
// SKU (inventory adjust/move/remove, price lookup, pick order) -- tried
// server-side first as a unit barcode, then as a case/box alias.
export function resolveBarcode(token: string, code: string): Promise<ResolveBarcodeResult> {
  return request<ResolveBarcodeResult>(`${base()}/skus/barcode-lookup/${encodeURIComponent(code)}`, { method: 'GET' }, token)
}

// ── Media ─────────────────────────────────────────────────────────────────────

export function listProductMedia(token: string, productId: string): Promise<Media[]> {
  return request<Media[]>(`${base()}/products/${productId}/media`, { method: 'GET' }, token)
}

// Deliberately doesn't go through request(): that helper always sets
// Content-Type: application/json, which would break multipart/form-data —
// the browser must set that header itself (it includes the boundary).
export async function uploadProductMedia(token: string, productId: string, file: File, altText?: string): Promise<Media> {
  const form = new FormData()
  form.append('file', file)
  if (altText) form.append('alt_text', altText)

  const res = await fetch(`${base()}/products/${productId}/media`, {
    method: 'POST',
    headers: { Authorization: `Bearer ${token}` },
    body: form,
  })
  const body: APIResponse<Media> = await res.json()
  if (!res.ok || body.error) {
    throw CodedApiError.fromEnvelope(body.error, res.status)
  }
  return body.data as Media
}

export function updateMedia(token: string, id: string, altText: string): Promise<Media> {
  return request<Media>(`${base()}/media/${id}`, { method: 'PATCH', body: JSON.stringify({ alt_text: altText }) }, token)
}

export function deleteMedia(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/media/${id}`, { method: 'DELETE' }, token)
}

// ── Weight packages ──────────────────────────────────────────────────────────

export function linkExistingWeightPackages(token: string, bulkSkuId: string, dryRun: boolean): Promise<LinkExistingWeightPackagesResult> {
  return request<LinkExistingWeightPackagesResult>(`${base()}/skus/${bulkSkuId}/weight-packages/link-existing`, { method: 'POST', body: JSON.stringify({ dry_run: dryRun }) }, token)
}

export function packWeightPortions(token: string, bulkSkuId: string, input: PackWeightPortionsInput): Promise<PackWeightPortionsResult> {
  return request<PackWeightPortionsResult>(`${base()}/skus/${bulkSkuId}/weight-packages`, { method: 'POST', body: JSON.stringify(input) }, token)
}

// ── Product merge ────────────────────────────────────────────────────────────

export function previewProductMerge(token: string, duplicateId: string, input: ProductMergeRequest): Promise<ProductMergePreview> {
  return request<ProductMergePreview>(`${base()}/products/${duplicateId}/merge-preview`, { method: 'POST', body: JSON.stringify(input) }, token)
}

export function mergeProduct(token: string, duplicateId: string, input: ProductMergeRequest): Promise<ProductMergeResult> {
  return request<ProductMergeResult>(`${base()}/products/${duplicateId}/merge`, { method: 'POST', body: JSON.stringify(input) }, token)
}
