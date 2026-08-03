// Cross-component invalidation for the product count/list. useProducts
// instances are independent per component (no shared cache) — the Sidebar
// badge, the Products page, and anything that mutates products outside
// useProducts itself (bulk WooCommerce import/delete) all hold their own
// copy. Without this, an action in one place (e.g. "Delete imported
// products" on the Integrations page) never reaches the Sidebar badge's
// already-mounted useProducts instance.
type Listener = () => void

const listeners = new Set<Listener>()

export function notifyProductsChanged(): void {
  listeners.forEach(l => l())
}

export function subscribeProductsChanged(listener: Listener): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}
