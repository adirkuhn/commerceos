import type { ComponentDefinition } from '../types'

const base = {
  container: false, allowedChildren: [], allowedParents: [], dataContext: 'customer' as const, requiredIn: [],
  editor: { removable: true, copyable: true, draggable: true },
}

export interface LoginFormProps { title: string; button_label: string }
export const loginForm: ComponentDefinition<LoginFormProps> = {
  ...base, type: 'login-form', name: 'Sign-in form', category: 'customer', icon: 'login',
  description: 'Passwordless sign-in: shoppers get a sign-in link by email.',
  props: { title: { kind: 'text', label: 'Title', maxLength: 120 }, button_label: { kind: 'text', label: 'Button label', maxLength: 40 } },
  defaults: { title: 'Sign in', button_label: 'Email me a sign-in link' },
  requiredIn: ['login'], editor: { removable: false, copyable: false, draggable: true },
  styleGroups: ['spacing', 'visibility'],
}

export interface AccountOverviewProps { show_orders: boolean }
export const accountOverview: ComponentDefinition<AccountOverviewProps> = {
  ...base, type: 'account-overview', name: 'Account', category: 'customer', icon: 'user',
  description: 'The signed-in shopper\'s details and orders.',
  props: { show_orders: { kind: 'toggle', label: 'Show orders' } },
  defaults: { show_orders: true },
  requiredIn: ['account'], editor: { removable: false, copyable: false, draggable: true },
  styleGroups: ['spacing', 'visibility'],
}

export interface WishlistItemsProps { empty_message: string }
export const wishlistItems: ComponentDefinition<WishlistItemsProps> = {
  ...base, type: 'wishlist-items', name: 'Wishlist', category: 'customer', icon: 'heart',
  description: 'Products the shopper saved.',
  props: { empty_message: { kind: 'text', label: 'Message when empty', maxLength: 200 } },
  defaults: { empty_message: 'Nothing saved yet.' },
  requiredIn: ['wishlist'], editor: { removable: false, copyable: false, draggable: true },
  styleGroups: ['spacing', 'visibility'],
}

export interface WishlistButtonProps { add_label: string; remove_label: string }
export const wishlistButton: ComponentDefinition<WishlistButtonProps> = {
  ...base, dataContext: 'product', type: 'wishlist-button', name: 'Wishlist button', category: 'customer', icon: 'heart',
  description: 'Saves the product to the shopper\'s wishlist.',
  props: { add_label: { kind: 'text', label: 'Label', maxLength: 40 }, remove_label: { kind: 'text', label: 'Label when saved', maxLength: 40 } },
  defaults: { add_label: 'Save to wishlist', remove_label: 'Saved' },
  styleGroups: ['spacing', 'visibility'],
}
