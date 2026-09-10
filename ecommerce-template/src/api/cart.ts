import { cartRepository } from '@/repositories'
import type { CartItem } from '@/types'

export const getCart = (cartId: string) => cartRepository.get(cartId)
export const addCartItem = (cartId: string, item: Omit<CartItem, 'id'>) => cartRepository.addItem(cartId, item)
export const updateCartItemQuantity = (cartId: string, lineId: string, quantity: number) => cartRepository.updateQuantity(cartId, lineId, quantity)
export const removeCartItem = (cartId: string, lineId: string) => cartRepository.removeItem(cartId, lineId)
export const saveCartItemForLater = (cartId: string, lineId: string) => cartRepository.saveForLater(cartId, lineId)
export const moveCartItemToCart = (cartId: string, lineId: string) => cartRepository.moveToCart(cartId, lineId)
export const applyCartCoupon = (cartId: string, code: string) => cartRepository.applyCoupon(cartId, code)
export const removeCartCoupon = (cartId: string, code: string) => cartRepository.removeCoupon(cartId, code)
export const clearCart = (cartId: string) => cartRepository.clear(cartId)
