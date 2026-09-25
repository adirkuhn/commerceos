import type { Product, ProductStatus } from './types'

// The product-level fields of the editor's General tab (everything except
// the flat SKU form and the variants toggle).
export interface ProductFields {
  name: string
  // Only ever sent to the server on an existing product's save (create
  // always derives slug from name server-side) -- this is what lets a
  // duplicated product's stuck "-copy" slug be fixed after the fact.
  slug: string
  description: string
  categoryId: string
  status: ProductStatus
  tags: string[]
}

export type SetProductField = <K extends keyof ProductFields>(key: K, value: ProductFields[K]) => void

export function productFieldsFromProduct(p: Product | undefined): ProductFields {
  return {
    name: p?.name ?? '',
    slug: p?.slug ?? '',
    description: p?.description ?? '',
    categoryId: p?.category_id ?? '',
    status: p?.status ?? 'draft',
    tags: p?.tags ?? [],
  }
}
