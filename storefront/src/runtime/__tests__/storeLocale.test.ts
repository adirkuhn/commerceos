import { describe, it, expect } from 'vitest'
import { localeFromCookie, SHOP_LANGUAGE_COOKIE, splitLocale, storeLanguages, storePathFor, languageCookie } from '../storeLocale'

const multi = storeLanguages({ locale: 'pt-BR', locales: ['pt-BR', 'en', 'nl'] })

describe('storeLanguages', () => {
  it('is the default plus the others the store offers', () => {
    expect(multi).toEqual({ defaultLocale: 'pt-BR', offered: ['pt-BR', 'en', 'nl'] })
  })

  it('a store that only reports its language offers just that one; no store info reads as English', () => {
    expect(storeLanguages({ locale: 'nl' })).toEqual({ defaultLocale: 'nl', offered: ['nl'] })
    expect(storeLanguages(undefined)).toEqual({ defaultLocale: 'en', offered: ['en'] })
  })

  it('ignores languages the storefront cannot speak and always offers the default', () => {
    expect(storeLanguages({ locale: 'en', locales: ['fr', 'nl'] })).toEqual({ defaultLocale: 'en', offered: ['en', 'nl'] })
  })
})

describe('splitLocale', () => {
  it('reads an offered language from the first path segment', () => {
    expect(splitLocale('/en/products/red-shoes', multi)).toEqual({ locale: 'en', path: '/products/red-shoes', prefixed: true })
    expect(splitLocale('/nl', multi)).toEqual({ locale: 'nl', path: '/', prefixed: true })
    expect(splitLocale('/nl/', multi)).toEqual({ locale: 'nl', path: '/', prefixed: true })
  })

  it('a plain path is the default language', () => {
    expect(splitLocale('/products/red-shoes', multi)).toEqual({ locale: 'pt-BR', path: '/products/red-shoes', prefixed: false })
    expect(splitLocale('/', multi)).toEqual({ locale: 'pt-BR', path: '/', prefixed: false })
  })

  it('the default language, or one the store does not offer, is not a prefix (the page is not found)', () => {
    expect(splitLocale('/pt-BR/cart', multi).locale).toBe('pt-BR')
    expect(splitLocale('/pt-BR/cart', multi).path).toBe('/pt-BR/cart')
    expect(splitLocale('/nl/cart', storeLanguages({ locale: 'en' })).path).toBe('/nl/cart')
  })
})

describe('storePathFor', () => {
  it('links pages in the default language without a prefix, others with one', () => {
    expect(storePathFor('k 1', 'pt-BR', multi)('products/x')).toBe('/store/k%201/products/x')
    expect(storePathFor('k1', 'en', multi)('products/x')).toBe('/store/k1/en/products/x')
    expect(storePathFor('k1', 'en', multi)('')).toBe('/store/k1/en/')
  })
})

describe('language cookie', () => {
  it('remembers the pick for this store only', () => {
    expect(languageCookie('k1', 'nl')).toBe(`${SHOP_LANGUAGE_COOKIE}=nl; Path=/store/k1/; Max-Age=31536000; SameSite=Lax`)
  })

  it('reads back an offered language only', () => {
    expect(localeFromCookie(`a=1; ${SHOP_LANGUAGE_COOKIE}=en`, multi)).toBe('en')
    expect(localeFromCookie(`${SHOP_LANGUAGE_COOKIE}=de`, multi)).toBeUndefined()
    expect(localeFromCookie(undefined, multi)).toBeUndefined()
  })
})
