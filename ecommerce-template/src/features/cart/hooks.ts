import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import {
  getCart, addCartItem, updateCartItemQuantity, removeCartItem, saveCartItemForLater, moveCartItemToCart,
  applyCartCoupon, removeCartCoupon,
} from '@/api/cart'
import type { CartItem } from '@/types'
import { useCartStore } from '@/stores/cartStore'
import { analytics } from '@/services/analytics'
import { toast } from '@/components/ui/toast'

function cartKey(cartId: string) {
  return ['cart', cartId] as const
}

export function useCart() {
  const cartId = useCartStore(s => s.cartId)
  const query = useQuery({ queryKey: cartKey(cartId), queryFn: () => getCart(cartId) })
  return { ...query, cartId }
}

export function useAddCartItem() {
  const cartId = useCartStore(s => s.cartId)
  const openDrawer = useCartStore(s => s.openDrawer)
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (item: Omit<CartItem, 'id'>) => addCartItem(cartId, item),
    onSuccess: (cart, item) => {
      queryClient.setQueryData(cartKey(cartId), cart)
      analytics.track('add_to_cart', { productId: item.productId, variantId: item.variantId, quantity: item.quantity })
      openDrawer()
    },
    onError: (error: Error) => toast({ title: 'Could not add to cart', description: error.message, variant: 'destructive' }),
  })
}

export function useUpdateCartItemQuantity() {
  const cartId = useCartStore(s => s.cartId)
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ lineId, quantity }: { lineId: string; quantity: number }) => updateCartItemQuantity(cartId, lineId, quantity),
    onSuccess: (cart) => queryClient.setQueryData(cartKey(cartId), cart),
  })
}

export function useRemoveCartItem() {
  const cartId = useCartStore(s => s.cartId)
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ lineId, productId }: { lineId: string; productId: string }) => removeCartItem(cartId, lineId).then(cart => ({ cart, productId })),
    onSuccess: ({ cart, productId }) => {
      queryClient.setQueryData(cartKey(cartId), cart)
      analytics.track('remove_from_cart', { productId })
    },
  })
}

export function useSaveForLater() {
  const cartId = useCartStore(s => s.cartId)
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (lineId: string) => saveCartItemForLater(cartId, lineId),
    onSuccess: (cart) => queryClient.setQueryData(cartKey(cartId), cart),
  })
}

export function useMoveToCart() {
  const cartId = useCartStore(s => s.cartId)
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (lineId: string) => moveCartItemToCart(cartId, lineId),
    onSuccess: (cart) => queryClient.setQueryData(cartKey(cartId), cart),
  })
}

export function useApplyCoupon() {
  const cartId = useCartStore(s => s.cartId)
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (code: string) => applyCartCoupon(cartId, code),
    onSuccess: (cart, code) => {
      queryClient.setQueryData(cartKey(cartId), cart)
      analytics.track('coupon_applied', { code })
      toast({ title: 'Coupon applied', variant: 'success' })
    },
    onError: (error: Error) => toast({ title: "Coupon didn't apply", description: error.message, variant: 'destructive' }),
  })
}

export function useRemoveCoupon() {
  const cartId = useCartStore(s => s.cartId)
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (code: string) => removeCartCoupon(cartId, code),
    onSuccess: (cart) => queryClient.setQueryData(cartKey(cartId), cart),
  })
}
