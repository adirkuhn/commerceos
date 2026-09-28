import type { ComponentDefinition } from '../types'

// Header/footer building blocks.

const base = {
  container: false, allowedChildren: [], allowedParents: [], dataContext: '' as const, requiredIn: [],
  editor: { removable: true, copyable: true, draggable: true },
}

export interface SiteLogoProps { size: 'sm' | 'md' | 'lg' }
export const siteLogo: ComponentDefinition<SiteLogoProps> = {
  ...base, type: 'site-logo', name: 'Logo', category: 'navigation', icon: 'badge',
  description: 'Your logo from the theme settings (or the store name), linking home.',
  props: { size: { kind: 'select', label: 'Size', options: [{ value: 'sm', label: 'Small' }, { value: 'md', label: 'Medium' }, { value: 'lg', label: 'Large' }] } },
  defaults: { size: 'md' },
  styleGroups: ['spacing', 'visibility'],
}

export interface NavLink { label: string; link: string }
export interface NavigationMenuProps { items: NavLink[]; orientation: 'horizontal' | 'vertical' }
export const navigationMenu: ComponentDefinition<NavigationMenuProps> = {
  ...base, type: 'navigation-menu', name: 'Menu', category: 'navigation', icon: 'menu',
  description: 'Links to your pages, collections or anywhere else.',
  props: {
    items: { kind: 'links', label: 'Links', max: 12 },
    orientation: { kind: 'select', label: 'Direction', options: [{ value: 'horizontal', label: 'Side by side' }, { value: 'vertical', label: 'Stacked' }] },
  },
  defaults: { items: [{ label: 'Home', link: '/' }], orientation: 'horizontal' },
  styleGroups: ['typography', 'spacing', 'visibility'],
}

export interface CartLinkProps { label: string; show_count: boolean }
export const cartLink: ComponentDefinition<CartLinkProps> = {
  ...base, dataContext: 'cart', type: 'cart-link', name: 'Cart link', category: 'navigation', icon: 'cart',
  description: 'Link to the cart, with how many items are in it.',
  props: { label: { kind: 'text', label: 'Label', maxLength: 30 }, show_count: { kind: 'toggle', label: 'Show item count' } },
  defaults: { label: 'Cart', show_count: true },
  styleGroups: ['typography', 'spacing', 'visibility'],
}

export interface SearchBoxProps { placeholder: string }
export const searchBox: ComponentDefinition<SearchBoxProps> = {
  ...base, type: 'search-box', name: 'Search box', category: 'navigation', icon: 'search',
  description: 'Lets shoppers search your products.',
  props: { placeholder: { kind: 'text', label: 'Placeholder', maxLength: 60 } },
  defaults: { placeholder: 'Search products' },
  styleGroups: ['spacing', 'layout', 'visibility'],
}

export interface AccountLinkProps { login_label: string; account_label: string }
export const accountLink: ComponentDefinition<AccountLinkProps> = {
  ...base, dataContext: 'customer', type: 'account-link', name: 'Account link', category: 'navigation', icon: 'user',
  description: 'Sign in, or go to the account when signed in.',
  props: { login_label: { kind: 'text', label: 'Signed-out label', maxLength: 30 }, account_label: { kind: 'text', label: 'Signed-in label', maxLength: 30 } },
  defaults: { login_label: 'Sign in', account_label: 'Account' },
  styleGroups: ['typography', 'spacing', 'visibility'],
}
