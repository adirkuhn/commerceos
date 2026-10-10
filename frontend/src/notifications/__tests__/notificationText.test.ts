import { describe, it, expect, afterEach } from 'vitest'
import i18next from 'i18next'
import { notificationText } from '../notificationText'
import type { Notification } from '../types'

function n(overrides: Partial<Notification>): Notification {
  return { id: 'n', type: 'x', params: {}, title: 'Stored title', message: 'Stored message', read: false, created_at: '', ...overrides }
}

afterEach(async () => { await i18next.changeLanguage('en') })

describe('notificationText', () => {
  it('translates a known type with its params in the current language', async () => {
    await i18next.changeLanguage('nl')
    expect(notificationText(n({ type: 'inventory.stock.close_to_expiry', params: { count: 4, window_days: 30 } }))).toEqual({
      title: 'Voorraad bijna over datum', message: '4 producten verlopen binnen 30 dagen',
    })
  })

  it('uses the count == 1 form', async () => {
    await i18next.changeLanguage('pt-BR')
    expect(notificationText(n({ type: 'inventory.stock.close_to_expiry', params: { count: 1, window_days: 7 } })).message)
      .toBe('1 produto vence em até 7 dias')
  })

  it('appends optional parts only when their param is set', () => {
    const base = { products: 5, already_linked: 2, skipped: 1 }
    expect(notificationText(n({ type: 'import.completed', params: base })).message).toBe('Imported 5 products (2 already linked, 1 skipped)')
    expect(notificationText(n({ type: 'import.completed', params: { ...base, orders: 3 } })).message)
      .toBe('Imported 5 products (2 already linked, 1 skipped), 3 orders')
  })

  it('translates a type that needs no params', async () => {
    await i18next.changeLanguage('nl')
    expect(notificationText(n({ type: 'import.started' })).title).toBe('Import gestart')
  })

  it('keeps the stored text when params are missing (notification from before translations)', () => {
    expect(notificationText(n({ type: 'import.failed', params: {} }))).toEqual({ title: 'Stored title', message: 'Stored message' })
  })

  it('keeps the stored text for an unknown type', () => {
    expect(notificationText(n({ type: 'something.new', params: { a: 1 } }))).toEqual({ title: 'Stored title', message: 'Stored message' })
  })

  it('tolerates a response without params', () => {
    expect(notificationText(n({ type: 'something.new', params: undefined }))).toEqual({ title: 'Stored title', message: 'Stored message' })
  })
})
