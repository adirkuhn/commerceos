import type { Collection, CollectionInput } from './types'

function base(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/collections`
}

interface Envelope<T> {
  data: T | null
  total?: number
  error: { code?: string; message?: string } | null
}

async function request<T>(path: string, options: RequestInit, token: string): Promise<Envelope<T>> {
  const res = await fetch(`${base()}${path}`, {
    ...options,
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}`, ...options.headers },
  })
  const body: Envelope<T> = await res.json()
  if (!res.ok || body.error) throw new Error(body.error?.message ?? `HTTP ${res.status}`)
  return body
}

export async function listCollections(token: string, page = 1, perPage = 100): Promise<{ items: Collection[]; total: number }> {
  const body = await request<Collection[]>(`?page=${page}&per_page=${perPage}`, { method: 'GET' }, token)
  return { items: body.data ?? [], total: body.total ?? 0 }
}

export async function getCollection(token: string, id: string): Promise<Collection> {
  return (await request<Collection>(`/${id}`, { method: 'GET' }, token)).data!
}

export async function createCollection(token: string, input: CollectionInput): Promise<Collection> {
  return (await request<Collection>('', { method: 'POST', body: JSON.stringify(input) }, token)).data!
}

export async function updateCollection(token: string, id: string, input: CollectionInput): Promise<Collection> {
  return (await request<Collection>(`/${id}`, { method: 'PUT', body: JSON.stringify(input) }, token)).data!
}

export async function deleteCollection(token: string, id: string): Promise<void> {
  await request<null>(`/${id}`, { method: 'DELETE' }, token)
}

export async function setCollectionProducts(token: string, id: string, productIds: string[]): Promise<Collection> {
  return (await request<Collection>(`/${id}/products`, { method: 'PUT', body: JSON.stringify({ product_ids: productIds }) }, token)).data!
}
