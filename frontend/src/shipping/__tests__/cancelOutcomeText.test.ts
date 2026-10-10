import { describe, it, expect } from 'vitest'
import i18next from 'i18next'
import { cancelOutcomeText } from '../cancelOutcomeText'

const t = (lng: string) => i18next.getFixedT(lng, 'shipping')

describe('cancelOutcomeText', () => {
  it('says MyParcel deleted it, in the reader’s language', () => {
    expect(cancelOutcomeText(t('en'), { deleted: true, carrierError: '' }, 'x')).toBe('Shipment deleted on MyParcel.')
    expect(cancelOutcomeText(t('nl'), { deleted: true, carrierError: '' }, 'x')).toBe('Zending verwijderd bij MyParcel.')
  })

  it('includes MyParcel’s own reason when it refused', () => {
    const text = cancelOutcomeText(t('pt-BR'), { deleted: false, carrierError: '422 label printed' }, 'x')
    expect(text).toContain('422 label printed')
    expect(text).toContain('MyParcel')
    expect(text).not.toContain('could not delete')
  })

  it('falls back to the stored English note without an outcome', () => {
    expect(cancelOutcomeText(t('nl'), null, 'Stored note')).toBe('Stored note')
  })
})
