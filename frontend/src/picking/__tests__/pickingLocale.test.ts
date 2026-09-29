import i18next from 'i18next'
import { describe, it, expect, afterEach } from 'vitest'
import { actionLabel } from '../pickingDisplay'

describe('picking texts in pt-BR', () => {
  afterEach(async () => { await i18next.changeLanguage('en') })

  it('names scan actions in Portuguese', async () => {
    await i18next.changeLanguage('pt-BR')
    expect(actionLabel('pick')).toBe('Separado')
    expect(actionLabel('substitute')).toBe('Substituído')
    expect(actionLabel('new_item')).toBe('Item novo')
  })
})
