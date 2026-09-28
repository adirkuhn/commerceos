import type { ComponentDefinition } from '../types'

export interface CheckoutFormProps { button_label: string }
export const checkoutForm: ComponentDefinition<CheckoutFormProps> = {
  type: 'checkout-form', name: 'Checkout form', category: 'cart', icon: 'card',
  description: 'Contact, address, delivery choice and payment. Every checkout page needs one.',
  props: { button_label: { kind: 'text', label: 'Button label', maxLength: 40 } },
  defaults: { button_label: 'Place order' },
  container: false, allowedChildren: [], allowedParents: [], dataContext: 'cart', requiredIn: ['checkout'],
  editor: { removable: false, copyable: false, draggable: true },
  styleGroups: ['spacing', 'visibility'],
}

export interface OrderStatusProps { title: string }
export const orderStatus: ComponentDefinition<OrderStatusProps> = {
  type: 'order-status', name: 'Order status', category: 'cart', icon: 'receipt',
  description: 'The order just placed: number, total and status.',
  props: { title: { kind: 'text', label: 'Title', maxLength: 120 } },
  defaults: { title: 'Thank you for your order' },
  container: false, allowedChildren: [], allowedParents: [], dataContext: 'order', requiredIn: ['order-confirmation'],
  editor: { removable: false, copyable: false, draggable: true },
  styleGroups: ['spacing', 'visibility'],
}
