import type {
  Category, Product, ProductDetail, Variant, SKU, Media, ResolveBarcodeResult,
  CreateCategoryPayload, UpdateCategoryPayload,
  CreateProductPayload, UpdateProductPayload,
  CreateVariantPayload, UpdateVariantPayload,
  CreateSKUPayload, UpdateSKUPayload,
  APIResponse,
} from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/pms`
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
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
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

export interface ListProductsParams {
  page?: number
  perPage?: number
  search?: string
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
  const query = qs.toString()

  const res = await fetch(`${base()}/products${query ? `?${query}` : ''}`, {
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
  })
  const body: APIResponse<Product[]> & { total?: number } = await res.json()
  if (!res.ok || body.error) {
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
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

export function deleteProduct(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/products/${id}`, { method: 'DELETE' }, token)
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
    throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  }
  return body.data as Media
}

export function updateMedia(token: string, id: string, altText: string): Promise<Media> {
  return request<Media>(`${base()}/media/${id}`, { method: 'PATCH', body: JSON.stringify({ alt_text: altText }) }, token)
}

export function deleteMedia(token: string, id: string): Promise<null> {
  return request<null>(`${base()}/media/${id}`, { method: 'DELETE' }, token)
}
