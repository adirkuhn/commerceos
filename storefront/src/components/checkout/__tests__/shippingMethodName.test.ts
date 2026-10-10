import { describe, it, expect } from 'vitest'
import { createI18n } from '../../../i18n/createI18n'
import { shippingMethodName } from '../shippingMethodName'

describe('shippingMethodName', () => {
  it('names the carrier delivery options in the shop language', () => {
    const nl = createI18n('nl')
    expect(shippingMethodName(nl.t, 'standard')).toBe('Standaard bezorging')
    expect(shippingMethodName(nl.t, 'morning')).toBe('Ochtendbezorging')
    expect(shippingMethodName(nl.t, 'evening')).toBe('Avondbezorging')
    expect(shippingMethodName(nl.t, 'pickup')).toBe('Afhalen bij een ophaalpunt')
  })

  it('shows an unknown option capitalized', () => {
    const pt = createI18n('pt-BR')
    expect(shippingMethodName(pt.t, 'sameday')).toBe('Sameday')
    expect(shippingMethodName(pt.t, '')).toBe('')
  })
})
