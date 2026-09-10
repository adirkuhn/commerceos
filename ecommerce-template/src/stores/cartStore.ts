import { create } from 'zustand'
import { persist } from 'zustand/middleware'

// Cart *contents* are server state (see hooks/useCart.ts, TanStack Query
// against the mock cart repository, keyed by cartId). This store only
// holds client-only cart UI state: which cart this browser owns, and
// whether the drawer is open -- exactly the split spec section 19 asks
// for.
interface CartUIState {
  cartId: string
  isDrawerOpen: boolean
  openDrawer: () => void
  closeDrawer: () => void
  toggleDrawer: () => void
}

function newCartId(): string {
  return typeof crypto !== 'undefined' && crypto.randomUUID ? crypto.randomUUID() : `cart-${Date.now()}-${Math.random().toString(16).slice(2)}`
}

export const useCartStore = create<CartUIState>()(
  persist(
    (set) => ({
      cartId: newCartId(),
      isDrawerOpen: false,
      openDrawer: () => set({ isDrawerOpen: true }),
      closeDrawer: () => set({ isDrawerOpen: false }),
      toggleDrawer: () => set(s => ({ isDrawerOpen: !s.isDrawerOpen })),
    }),
    // isDrawerOpen deliberately not persisted (see partialize) -- a
    // reload should never reopen the drawer, but cartId must survive it
    // so the cart itself does. persist's JSON serialization already drops
    // function-valued keys (openDrawer etc), so the live actions from
    // create() are never overwritten on rehydrate.
    { name: 'aurora-cart-ui', partialize: (state): Pick<CartUIState, 'cartId'> => ({ cartId: state.cartId }) },
  ),
)
