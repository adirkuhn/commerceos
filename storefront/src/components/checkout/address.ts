import type { Address } from '../../api/storefront'

export const emptyAddress = (): Address => ({ first_name: '', last_name: '', address1: '', address2: '', city: '', postcode: '', country: 'NL', phone: '' })

/** Enough of an address to price delivery. */
export const deliverable = (a: Address) => Boolean(a.address1.trim() && a.city.trim() && a.postcode.trim() && a.country.trim())

/** Enough to place an order. */
export const complete = (a: Address) => Boolean(a.first_name.trim() && a.last_name.trim() && deliverable(a))

/** Only hosted payment pages on https are followed after checkout. */
export function safePaymentURL(url: string | null | undefined): string | undefined {
  if (!url) return undefined
  try {
    return new URL(url).protocol === 'https:' ? url : undefined
  } catch {
    return undefined
  }
}
