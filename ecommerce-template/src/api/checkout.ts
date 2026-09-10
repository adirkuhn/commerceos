import { checkoutRepository } from '@/repositories'
import type { CheckoutInput } from '@/repositories'
import type { Cart } from '@/types'

export const placeOrder = (cart: Cart, input: CheckoutInput) => checkoutRepository.placeOrder(cart, input)
