import { useMutation, useQueryClient } from '@tanstack/react-query'
import { placeOrder } from '@/api/checkout'
import { clearCart } from '@/api/cart'
import type { Cart } from '@/types'
import type { CheckoutInput } from '@/repositories'
import { useCartStore } from '@/stores/cartStore'
import { analytics } from '@/services/analytics'
import { pluginRegistry } from '@/plugins'

export function usePlaceOrder() {
  const cartId = useCartStore(s => s.cartId)
  const queryClient = useQueryClient()

  return useMutation({
    mutationFn: async ({ cart, input }: { cart: Cart; input: CheckoutInput }) => {
      const order = await placeOrder(cart, input)
      await clearCart(cartId)
      return order
    },
    onSuccess: (order) => {
      queryClient.invalidateQueries({ queryKey: ['cart', cartId] })
      queryClient.invalidateQueries({ queryKey: ['orders'] })
      analytics.track('checkout_completed', { orderId: order.id, total: order.total })
      analytics.track('purchase', { orderId: order.id, total: order.total, currency: order.currency })
      pluginRegistry.emit('order_placed', order)
    },
  })
}
