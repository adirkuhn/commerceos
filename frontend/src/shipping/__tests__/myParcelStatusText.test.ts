import { describe, it, expect } from 'vitest'
import i18next from 'i18next'
import { myParcelStatusText, MYPARCEL_STATUS_CODES } from '../myParcelStatusText'

const t = (lng: string) => i18next.getFixedT(lng, 'shipping')

describe('myParcelStatusText', () => {
  it('names every MyParcel status in every language', () => {
    for (const lng of ['en', 'pt-BR', 'nl']) {
      for (const code of MYPARCEL_STATUS_CODES) {
        const text = myParcelStatusText(t(lng), code, 'fallback')
        expect(text, `${code} ${lng}`).not.toBe('fallback')
        expect(text, `${code} ${lng}`).not.toMatch(/myparcelStatus/)
      }
    }
    expect(myParcelStatusText(t('en'), 8, 'x')).toBe('Delivered - ready for pickup')
    expect(myParcelStatusText(t('nl'), 8, 'x')).toBe('Bezorgd - klaar om op te halen')
  })

  it('shows an unknown code with its number', () => {
    expect(myParcelStatusText(t('pt-BR'), 99, 'Unknown status (99)')).toBe('Status desconhecido (99)')
  })
})
