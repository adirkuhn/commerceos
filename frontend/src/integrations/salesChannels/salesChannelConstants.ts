import type { EnrollmentMode, ChannelType } from '../types'

export const ENROLLMENT_LABELS: Record<EnrollmentMode, string> = {
  all: 'All products',
  category: 'By category',
  tag: 'By tag',
  manual: 'Manual only',
}

export function capitalize(s: string): string {
  return s.length === 0 ? s : s[0].toUpperCase() + s.slice(1)
}

export const TYPE_LABELS: Record<ChannelType, string> = {
  woocommerce: 'WooCommerce',
  bol: 'Bol.com',
  whatsapp: 'WhatsApp',
  internal: 'Internal store',
}

// A built-in storefront has nothing external to connect to (see backend
// domain/channels.ChannelType.RequiresConnection) -- its own form fields
// never render, and it's the one type able to configure a payment method
// directly (WhatsApp is the other, for its chat checkout).
export const INTERNAL_TYPE: ChannelType = 'internal'

// HEADLESS_API_ENDPOINTS is a reference snapshot of an internal channel's
// real commerce API -- backend/api/storefront/openapi.yaml is the actual
// source of truth (used for real request validation/codegen); this is
// just a legible summary for a developer wiring a headless frontend to
// this channel, not a substitute for that spec. Excludes /theme/* (the
// old Puck-editor-only routes -- irrelevant to a headless integration,
// see the "remove Freeform" commit). Keep in sync with openapi.yaml by
// hand if a storefront route is added/changed/removed.
export const HEADLESS_API_ENDPOINTS: { method: string; path: string; summary: string }[] = [
  { method: 'GET', path: '/products', summary: "List the calling channel's enrolled catalog" },
  { method: 'GET', path: '/products/{id}', summary: "Fetch one enrolled product's full variant/SKU/image detail" },
  { method: 'GET', path: '/products/{id}/recommendations', summary: 'Products historically bought alongside this one' },
  { method: 'GET', path: '/products/{id}/reviews', summary: "List a product's reviews, with a rating summary" },
  { method: 'POST', path: '/products/{id}/reviews', summary: 'Submit a review for a product' },
  { method: 'GET', path: '/categories', summary: "The tenant's whole category tree, flat with parent_id" },
  { method: 'POST', path: '/recommendations/cart', summary: "Cross-sell candidates for the shopper's whole cart" },
  { method: 'POST', path: '/carts', summary: "Start a new guest cart on the calling key's sales channel" },
  { method: 'GET', path: '/carts/{token}', summary: "Fetch a cart's current line items and subtotal" },
  { method: 'PUT', path: '/carts/{token}/items', summary: "Replace a cart's whole line set" },
  { method: 'PUT', path: '/carts/{token}/coupon', summary: 'Attach a coupon code to the cart, or remove it' },
  { method: 'POST', path: '/carts/{token}/shipping-options', summary: 'Priced delivery options for a shipping address' },
  { method: 'POST', path: '/carts/{token}/payment-intent', summary: "Start an embedded Stripe Elements card payment for this cart's total" },
  { method: 'POST', path: '/carts/{token}/checkout', summary: "Place an order for a cart's current contents" },
  { method: 'POST', path: '/wishlists', summary: 'Start a new, empty wishlist on the calling channel' },
  { method: 'GET', path: '/wishlists/{token}', summary: 'Fetch a wishlist by its token' },
  { method: 'PUT', path: '/wishlists/{token}/items', summary: 'Add product_id to the wishlist, or remove it' },
  { method: 'POST', path: '/auth/magic-link', summary: 'Email a one-time sign-in link to the given address' },
  { method: 'POST', path: '/auth/session', summary: 'Exchange a magic-link token for a signed-in session' },
  { method: 'GET', path: '/account/{token}', summary: "Fetch the signed-in shopper's profile" },
  { method: 'DELETE', path: '/account/{token}', summary: 'End the session' },
  { method: 'GET', path: '/account/{token}/orders', summary: "The signed-in shopper's order history" },
  { method: 'GET', path: '/orders/lookup', summary: 'Guest order-status lookup by order number + billing email' },
]

export function apiBaseURL(): string {
  return `${import.meta.env.VITE_API_BASE_URL}/storefront/v1`
}

export const CONNECTION_REQUIRED_TYPES: ChannelType[] = (Object.keys(TYPE_LABELS) as ChannelType[]).filter(t => t !== INTERNAL_TYPE)

export const PAYMENT_CAPABLE_TYPES: ChannelType[] = ['whatsapp', INTERNAL_TYPE]

// The same two types create a SalesOrder directly (Woo/Bol only ever
// import one that already has a warehouse from the source system) -- so
// they're also the only ones that need SalesChannel.default_warehouse_id
// configured before checkout can succeed. Same set as PAYMENT_CAPABLE_TYPES
// today because both concerns track "does this channel sell directly," but
// kept as its own name since that's incidental, not a rule.
export const WAREHOUSE_REQUIRED_TYPES: ChannelType[] = PAYMENT_CAPABLE_TYPES
