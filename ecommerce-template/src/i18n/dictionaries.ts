// Minimal dictionary-based i18n -- proves the pattern (spec section 13)
// without pulling in a full i18n library. Two locales: en-US and es-ES.
// A real deployment with many more strings would likely swap this for
// react-i18next or similar without changing how components call t().
export const LOCALES = ['en-US', 'es-ES'] as const
export type Locale = (typeof LOCALES)[number]

export const LOCALE_LABELS: Record<Locale, string> = {
  'en-US': 'English (US)',
  'es-ES': 'Español (España)',
}

export const DICTIONARIES: Record<Locale, Record<string, string>> = {
  'en-US': {
    'nav.shop': 'Shop', 'nav.search': 'Search', 'nav.wishlist': 'Wishlist', 'nav.account': 'Account', 'nav.cart': 'Cart',
    'cart.title': 'Your cart', 'cart.empty': 'Your cart is empty.', 'cart.subtotal': 'Subtotal', 'cart.shipping': 'Shipping',
    'cart.tax': 'Tax', 'cart.total': 'Total', 'cart.checkout': 'Checkout', 'cart.freeShipping': 'You qualify for free shipping!',
    'product.addToCart': 'Add to cart', 'product.buyNow': 'Buy now', 'product.outOfStock': 'Out of stock',
    'product.wishlist': 'Add to wishlist', 'product.compare': 'Compare',
    'checkout.customer': 'Customer', 'checkout.shipping': 'Shipping', 'checkout.payment': 'Payment', 'checkout.review': 'Review',
    'account.dashboard': 'Dashboard', 'account.orders': 'Orders', 'account.addresses': 'Addresses', 'account.profile': 'Profile',
  },
  'es-ES': {
    'nav.shop': 'Tienda', 'nav.search': 'Buscar', 'nav.wishlist': 'Favoritos', 'nav.account': 'Cuenta', 'nav.cart': 'Carrito',
    'cart.title': 'Tu carrito', 'cart.empty': 'Tu carrito está vacío.', 'cart.subtotal': 'Subtotal', 'cart.shipping': 'Envío',
    'cart.tax': 'Impuesto', 'cart.total': 'Total', 'cart.checkout': 'Pagar', 'cart.freeShipping': '¡Tienes envío gratis!',
    'product.addToCart': 'Añadir al carrito', 'product.buyNow': 'Comprar ahora', 'product.outOfStock': 'Agotado',
    'product.wishlist': 'Añadir a favoritos', 'product.compare': 'Comparar',
    'checkout.customer': 'Cliente', 'checkout.shipping': 'Envío', 'checkout.payment': 'Pago', 'checkout.review': 'Revisión',
    'account.dashboard': 'Panel', 'account.orders': 'Pedidos', 'account.addresses': 'Direcciones', 'account.profile': 'Perfil',
  },
}
