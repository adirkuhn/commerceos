import i18next from 'i18next'
import { describe, it, expect, afterEach } from 'vitest'
import { formatLockAge, shortUserId } from '../lockDisplay'

const NOW = Date.parse('2026-09-29T12:00:00Z')

describe('packing texts in pt-BR', () => {
  afterEach(async () => { await i18next.changeLanguage('en') })

  it('formats lock ages and user fallbacks in Portuguese', async () => {
    await i18next.changeLanguage('pt-BR')
    expect(formatLockAge('2026-09-29T10:45:00Z', NOW)).toBe('1 h 15 min')
    expect(formatLockAge('2026-09-29T11:59:30Z', NOW)).toBe('<1 min')
    expect(shortUserId('abcdef123456')).toBe('Usuário abcdef')
  })
})
