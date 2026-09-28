import { queryOptions, type FetchQueryOptions } from '@tanstack/react-query'
import type { ProductDetail, ProductList, StorefrontAPI } from '../api/storefront'
import type { ThemeNode } from '../core/document'
import type { ProductGridProps } from './commerce/productGrid'
import type { ProductCardProps } from './commerce/productCard'
import { recommendationsQuery, reviewsQuery } from '../runtime/page/queries'

// Catalog data a component shows. The same query options are prefetched
// during SSR (dataQueriesFor) and read by the renderer, so the page is
// rendered complete on the server and hydrates without refetching.

const STALE = 60_000

const EMPTY: ProductList = { items: [], total: 0 }

export function productGridQuery(api: StorefrontAPI, p: ProductGridProps) {
  const limit = Math.min(Math.max(Number(p.limit) || 8, 1), 48)
  const ids = (p.product_ids ?? []).slice(0, 48)
  const q =
    p.source === 'products' ? { productIds: ids, perPage: Math.max(ids.length, 1) }
    : p.source === 'category' ? { categoryId: p.category_id, perPage: limit }
    : p.source === 'collection' ? { collectionId: p.collection_id, perPage: limit }
    : { perPage: limit }
  const empty = (p.source === 'products' && ids.length === 0) || (p.source === 'category' && !p.category_id)
    || (p.source === 'collection' && !p.collection_id)
  return queryOptions({
    queryKey: ['products', q],
    queryFn: async () => {
      if (empty) return EMPTY
      const list = await api.listProducts(q)
      if (p.source !== 'products') return list
      // Keep the merchant's chosen order.
      const byId = new Map(list.items.map(i => [i.id, i]))
      return { ...list, items: ids.map(id => byId.get(id)).filter(i => i !== undefined) }
    },
    staleTime: STALE,
  })
}

export function productCardQuery(api: StorefrontAPI, p: ProductCardProps) {
  return queryOptions({
    queryKey: ['product-card', p.product_id],
    queryFn: async () => (p.product_id ? (await api.listProducts({ productIds: [p.product_id], perPage: 1 })).items[0] ?? null : null),
    staleTime: STALE,
  })
}

/** What a page is about, for components whose data depends on it. */
export interface LoaderContext {
  product?: ProductDetail
}

type AnyQuery = FetchQueryOptions<unknown, Error, unknown, readonly unknown[]>
type Loader = (api: StorefrontAPI, props: never, ctx: LoaderContext) => AnyQuery | undefined

const reviewLimit = (props: { limit?: number }) => Math.min(Math.max(Number(props.limit) || 10, 1), 50)

const loaders: Record<string, Loader> = {
  'product-grid': productGridQuery as unknown as Loader,
  'product-card': productCardQuery as unknown as Loader,
  'product-reviews': ((api, props: { limit?: number }, ctx) =>
    ctx.product ? reviewsQuery(api, ctx.product.id, reviewLimit(props)) : undefined) as Loader,
  'product-recommendations': ((api, _props, ctx) =>
    ctx.product ? recommendationsQuery(api, ctx.product.id) : undefined) as Loader,
}

export { reviewLimit }

/** Every catalog query a document's components need, for SSR prefetching. */
export function dataQueriesFor(root: ThemeNode, api: StorefrontAPI, ctx: LoaderContext = {}) {
  const out: AnyQuery[] = []
  const visit = (n: ThemeNode) => {
    const q = loaders[n.type]?.(api, (n.props ?? {}) as never, ctx)
    if (q) out.push(q)
    n.children?.forEach(visit)
  }
  visit(root)
  return out
}
