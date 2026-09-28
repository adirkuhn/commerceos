import type { ProductDetail } from '../api/storefront'
import type { ListingPage } from '../runtime/page/ListingContext'

export interface PreviewData {
  product?: ProductDetail
  listing?: ListingPage
}

/** What the canvas previews a product/collection template with, observable
 *  so every canvas component re-renders when the merchant picks another. */
export class PreviewStore {
  private data: PreviewData = {}
  private readonly listeners = new Set<() => void>()

  snapshot = (): PreviewData => this.data

  subscribe = (fn: () => void): (() => void) => {
    this.listeners.add(fn)
    return () => this.listeners.delete(fn)
  }

  set(data: PreviewData) {
    this.data = data
    this.listeners.forEach(l => l())
  }
}
