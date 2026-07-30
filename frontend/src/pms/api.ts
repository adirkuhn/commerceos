import type {
  Category, Product, ProductDetail, Variant, SKU,
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

export function listProducts(token: string): Promise<Product[]> {
  return request<Product[]>(`${base()}/products`, { method: 'GET' }, token)
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
