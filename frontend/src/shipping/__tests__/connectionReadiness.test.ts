import { describe, it, expect } from 'vitest'
import i18next from 'i18next'
import { connectionReady, missingSetupTexts } from '../connectionReadiness'
import { MISSING_SETUP } from '../types'

const t = (lng: string) => i18next.getFixedT(lng, 'shipping')

describe('connectionReady', () => {
  it('is ready with nothing missing (or no connection to check)', () => {
    expect(connectionReady({ missing: [] })).toBe(true)
    expect(connectionReady(null)).toBe(true)
  })

  it('is not ready while something is missing', () => {
    expect(connectionReady({ missing: [MISSING_SETUP.senderAddress] })).toBe(false)
  })
})

describe('missingSetupTexts', () => {
  it('names the provider for a missing sender address, in every language', () => {
    const conn = { provider: 'postnl', missing: [MISSING_SETUP.senderAddress] }
    expect(missingSetupTexts(t('en'), conn)).toEqual(['PostNL needs a sender address before it can create labels.'])
    expect(missingSetupTexts(t('nl'), conn)[0]).toContain('PostNL')
    expect(missingSetupTexts(t('nl'), conn)[0]).toContain('afzenderadres')
    expect(missingSetupTexts(t('pt-BR'), conn)[0]).toContain('remetente')
  })

  it('is empty when nothing is missing', () => {
    expect(missingSetupTexts(t('en'), { provider: 'myparcel', missing: [] })).toEqual([])
  })
})
