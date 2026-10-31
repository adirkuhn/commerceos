import i18next from 'i18next'
import type { EnrollmentMode, ChannelType } from '../types'
import { appEnv } from '../../config/appEnv'

export const ENROLLMENT_MODES: EnrollmentMode[] = ['all', 'category', 'tag', 'manual']

export function enrollmentLabel(mode: EnrollmentMode): string {
  switch (mode) {
    case 'all': return i18next.t('channels:enrollment.all')
    case 'category': return i18next.t('channels:enrollment.category')
    case 'tag': return i18next.t('channels:enrollment.tag')
    case 'manual': return i18next.t('channels:enrollment.manual')
  }
}

export function capitalize(s: string): string {
  return s.length === 0 ? s : s[0].toUpperCase() + s.slice(1)
}

export const CHANNEL_TYPES: ChannelType[] = ['woocommerce', 'bol', 'whatsapp', 'internal']

export function typeLabel(type: ChannelType): string {
  switch (type) {
    case 'woocommerce': return 'WooCommerce'
    case 'bol': return 'Bol.com'
    case 'whatsapp': return 'WhatsApp'
    case 'internal': return i18next.t('channels:type.internal')
  }
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
export function headlessApiEndpoints(): { method: string; path: string; summary: string }[] {
  const t = i18next.t
  return [
    { method: 'GET', path: '/products', summary: t('channels:api.products') },
    { method: 'GET', path: '/products/{id}', summary: t('channels:api.product') },
    { method: 'GET', path: '/products/{id}/recommendations', summary: t('channels:api.recommendations') },
    { method: 'GET', path: '/products/{id}/reviews', summary: t('channels:api.reviews') },
    { method: 'POST', path: '/products/{id}/reviews', summary: t('channels:api.submitReview') },
    { method: 'GET', path: '/categories', summary: t('channels:api.categories') },
    { method: 'POST', path: '/recommendations/cart', summary: t('channels:api.cartRecommendations') },
    { method: 'POST', path: '/carts', summary: t('channels:api.createCart') },
    { method: 'GET', path: '/carts/{token}', summary: t('channels:api.getCart') },
    { method: 'PUT', path: '/carts/{token}/items', summary: t('channels:api.cartItems') },
    { method: 'PUT', path: '/carts/{token}/coupon', summary: t('channels:api.cartCoupon') },
    { method: 'POST', path: '/carts/{token}/shipping-options', summary: t('channels:api.shippingOptions') },
    { method: 'POST', path: '/carts/{token}/payment-intent', summary: t('channels:api.paymentIntent') },
    { method: 'POST', path: '/carts/{token}/checkout', summary: t('channels:api.checkout') },
    { method: 'POST', path: '/wishlists', summary: t('channels:api.createWishlist') },
    { method: 'GET', path: '/wishlists/{token}', summary: t('channels:api.getWishlist') },
    { method: 'PUT', path: '/wishlists/{token}/items', summary: t('channels:api.wishlistItems') },
    { method: 'POST', path: '/auth/magic-link', summary: t('channels:api.magicLink') },
    { method: 'POST', path: '/auth/session', summary: t('channels:api.session') },
    { method: 'GET', path: '/account/{token}', summary: t('channels:api.account') },
    { method: 'DELETE', path: '/account/{token}', summary: t('channels:api.signOut') },
    { method: 'GET', path: '/account/{token}/orders', summary: t('channels:api.accountOrders') },
    { method: 'GET', path: '/orders/lookup', summary: t('channels:api.orderLookup') },
  ]
}

export function apiBaseURL(): string {
  return `${appEnv().VITE_API_BASE_URL}/storefront/v1`
}

export const CONNECTION_REQUIRED_TYPES: ChannelType[] = CHANNEL_TYPES.filter(t => t !== INTERNAL_TYPE)

export const PAYMENT_CAPABLE_TYPES: ChannelType[] = ['whatsapp', INTERNAL_TYPE]

// The same two types create a SalesOrder directly (Woo/Bol only ever
// import one that already has a warehouse from the source system) -- so
// they're also the only ones that need SalesChannel.default_warehouse_id
// configured before checkout can succeed. Same set as PAYMENT_CAPABLE_TYPES
// today because both concerns track "does this channel sell directly," but
// kept as its own name since that's incidental, not a rule.
export const WAREHOUSE_REQUIRED_TYPES: ChannelType[] = PAYMENT_CAPABLE_TYPES
