import { describe, it, expect } from 'vitest'
import { createI18n } from '../createI18n'
import { resolveLocale } from '../locale'

describe('createI18n', () => {
  it('translates synchronously, interpolates and pluralises per language', () => {
    const pt = createI18n('pt-BR')
    expect(pt.t('cart.empty')).toBe('Seu carrinho está vazio.')
    expect(pt.t('listing.pageOf', { page: 2, total: 3 })).toBe('Página 2 de 3')
    expect(pt.t('nav.cartItems', { count: 1 })).toBe('1 item')
    expect(pt.t('nav.cartItems', { count: 4 })).toBe('4 itens')
    expect(createI18n('nl').t('product.addToCart')).toBe('In winkelwagen')
    expect(createI18n('en').t('product.addToCart')).toBe('Add to cart')
  })
})

describe('resolveLocale', () => {
  it('matches exact tags, then the language, else English', () => {
    expect(resolveLocale('pt-BR')).toBe('pt-BR')
    expect(resolveLocale('pt-PT')).toBe('pt-BR')
    expect(resolveLocale('nl-BE')).toBe('nl')
    expect(resolveLocale('de-DE')).toBe('en')
    expect(resolveLocale(undefined)).toBe('en')
  })
})

describe('orderStatusLabel', () => {
  it('shows known statuses in the store language and unknown ones as sent', async () => {
    const { orderStatusLabel } = await import('../orderStatus')
    const pt = createI18n('pt-BR')
    expect(orderStatusLabel(pt.t, 'shipped')).toBe('Enviado')
    expect(orderStatusLabel(pt.t, 'teleported')).toBe('teleported')
  })
})
