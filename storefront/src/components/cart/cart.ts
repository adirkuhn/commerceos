import type { ComponentDefinition } from '../types'

// The shopper's cart (data context "cart" -- per shopper, so these work on
// any page, e.g. a cart summary in the header).

const base = {
  container: false, allowedChildren: [], allowedParents: [], dataContext: 'cart' as const, requiredIn: [],
  editor: { removable: true, copyable: true, draggable: true },
}

export interface CartItemsProps { empty_message: string }
export const cartItems: ComponentDefinition<CartItemsProps> = {
  ...base, type: 'cart-items', name: 'Cart items', category: 'cart', icon: 'list',
  description: 'The products in the cart, with quantities the shopper can change.',
  props: { empty_message: { kind: 'text', label: 'Message when empty', maxLength: 200 } },
  defaults: { empty_message: '' },
  requiredIn: ['cart'],
  editor: { removable: false, copyable: false, draggable: true },
  styleGroups: ['spacing', 'visibility'],
}

export interface CartSummaryProps { show_coupon: boolean }
export const cartSummary: ComponentDefinition<CartSummaryProps> = {
  ...base, type: 'cart-summary', name: 'Cart totals', category: 'cart', icon: 'receipt',
  description: 'Subtotal, discount and a coupon field.',
  props: { show_coupon: { kind: 'toggle', label: 'Show coupon field' } },
  defaults: { show_coupon: true },
  styleGroups: ['spacing', 'colors', 'border', 'visibility'],
}

export interface CheckoutButtonProps { label: string }
export const checkoutButton: ComponentDefinition<CheckoutButtonProps> = {
  ...base, type: 'checkout-button', name: 'Checkout button', category: 'cart', icon: 'button',
  description: 'Takes the shopper to checkout. Disabled while the cart is empty.',
  props: { label: { kind: 'text', label: 'Label', maxLength: 60 } },
  defaults: { label: '' },
  styleGroups: ['spacing', 'layout', 'visibility'],
}
