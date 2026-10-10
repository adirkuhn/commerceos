import type { TFunction } from 'i18next'

// shippingMethodName is how a shipping option reads to the shopper. The
// API's title is the carrier's own code (MyParcel price_comment, e.g.
// "standard", "morning") and doubles as the option's identity sent back at
// checkout, so it's translated only for display. Unknown codes show as-is,
// capitalized.
export function shippingMethodName(t: TFunction<'shop'>, title: string): string {
  switch (title) {
    case 'standard': return t('checkout.shippingStandard')
    case 'morning': return t('checkout.shippingMorning')
    case 'evening': return t('checkout.shippingEvening')
    case 'pickup': return t('checkout.shippingPickup')
    default: return title.charAt(0).toUpperCase() + title.slice(1)
  }
}
