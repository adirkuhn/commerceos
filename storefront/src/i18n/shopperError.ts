import type { i18n as I18n, ParseKeys } from 'i18next'
import { ApiError } from '../api/storefront'

// shopperError is what a shopper sees when a storefront call fails: the
// API error code's text in the shop's language (shop.json errors.<code>),
// or fallback, already translated by the caller with a literal key so the
// extractor keeps that key. The API's English message is never shown --
// it's for logs, not shoppers.
export function shopperError(err: unknown, i18n: I18n, fallback: string): string {
  if (err instanceof ApiError) {
    // Codes come from the API, so the key is only known at runtime
    // (kept by i18next-parser.config's keepRemoved for shop:errors).
    const key = `errors.${err.code}` as ParseKeys<'shop'>
    if (i18n.exists(key)) return i18n.t(key)
  }
  return fallback
}
