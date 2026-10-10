import { describe, it, expect, afterEach } from 'vitest'
import i18next from 'i18next'
import { insightText } from '../insightText'

const t = (lng: string) => i18next.getFixedT(lng, 'reports')

afterEach(async () => { await i18next.changeLanguage('en') })

describe('insightText', () => {
  it('translates every rule with its numbers', () => {
    const en = t('en')
    expect(insightText(en, { code: 'revenue_declining', severity: 'warning', params: { percent: 12.5 }, message: '' }, 'en'))
      .toBe('Revenue is down 12.5% vs the previous period.')
    expect(insightText(en, { code: 'slow_moving_stock', severity: 'warning', params: { count: 1 }, message: '' }, 'en'))
      .toBe('1 product has stock on hand but zero sales this period.')
    expect(insightText(en, { code: 'restock_risk', severity: 'warning', params: { count: 3 }, message: '' }, 'en'))
      .toBe('3 products sold through 100% or more of their stock on hand this period — restock risk.')
  })

  it('uses the language’s number format', () => {
    const nl = t('nl')
    expect(insightText(nl, { code: 'incomplete_cost_data', severity: 'info', params: { percent: 40.5 }, message: '' }, 'nl'))
      .toContain('40,5%')
  })

  it('falls back to the message for an unknown rule or missing params', () => {
    const en = t('en')
    expect(insightText(en, { code: 'new_rule', severity: 'info', params: { count: 1 }, message: 'Stored' }, 'en')).toBe('Stored')
    expect(insightText(en, { code: 'slow_moving_stock', severity: 'info', message: 'Stored' }, 'en')).toBe('Stored')
  })
})
