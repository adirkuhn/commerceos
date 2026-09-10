import { useWishlistStore } from '@/stores/wishlistStore'
import { useProductsByIds } from '@/features/products/hooks'

// Wishlist product ids live in Zustand (client state); this hook joins
// them against the product repository to get full Product objects for
// the wishlist page -- same server/client split as cart.
export function useWishlistProducts() {
  const productIds = useWishlistStore(s => s.productIds)
  return useProductsByIds(productIds)
}
