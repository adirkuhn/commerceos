import { describe, it, expect } from 'vitest'
import { createI18n } from '../createI18n'
import { shopperError } from '../shopperError'
import { ApiError } from '../../api/storefront'

describe('shopperError', () => {
  it('shows a known API error code in the shop language', () => {
    const nl = createI18n('nl')
    expect(shopperError(new ApiError(422, 'INSUFFICIENT_STOCK', 'insufficient stock for requested quantity'), nl, 'fallback'))
      .toBe(nl.t('errors.INSUFFICIENT_STOCK'))
    expect(nl.t('errors.INSUFFICIENT_STOCK')).not.toMatch(/insufficient/i)
  })

  it('never shows the raw English message for an unknown code', () => {
    const pt = createI18n('pt-BR')
    expect(shopperError(new ApiError(500, 'INTERNAL', 'internal server error'), pt, 'fallback')).toBe('fallback')
    expect(shopperError(new Error('fetch failed'), pt, 'fallback')).toBe('fallback')
    expect(shopperError('boom', pt, 'fallback')).toBe('fallback')
  })

  it('keeps the generic texts the call sites fall back to', () => {
    for (const lang of ['en', 'pt-BR', 'nl'] as const) {
      const i18n = createI18n(lang)
      expect(i18n.getResource(lang, 'shop', 'checkout.failed')).toBeTruthy()
      expect(i18n.getResource(lang, 'shop', 'common.somethingWrong')).toBeTruthy()
    }
  })

  it('has a text for every shopper error code in every language', () => {
    const codes = ['CART_EMPTY', 'INSUFFICIENT_STOCK', 'PRODUCT_NOT_AVAILABLE', 'STORE_NOT_AVAILABLE', 'COUPON_NOT_REDEEMABLE',
      'COUPON_MIN_SUBTOTAL_NOT_MET', 'COUPON_NOT_FOUND', 'REVIEW_NAME_REQUIRED', 'REVIEW_EMAIL_REQUIRED', 'REVIEW_INVALID_RATING',
      'REVIEW_BODY_REQUIRED', 'SIGN_IN_LINK_INVALID', 'SIGN_IN_NOT_CONFIGURED', 'CARD_PAYMENT_NOT_SUPPORTED', 'CARD_PAYMENT_MISMATCH',
      'CART_NOT_FOUND', 'ORDER_NOT_FOUND', 'WISHLIST_NOT_FOUND', 'COLLECTION_NOT_FOUND', 'SESSION_NOT_FOUND', 'CART_ALREADY_CHECKED_OUT',
      'VALIDATION_ERROR']
    for (const lang of ['en', 'pt-BR', 'nl'] as const) {
      const i18n = createI18n(lang)
      for (const code of codes) {
        expect(i18n.getResource(lang, 'shop', `errors.${code}`), `${code} in ${lang}`).toBeTruthy()
      }
    }
  })
})
