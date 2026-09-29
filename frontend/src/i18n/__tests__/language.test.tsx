// @vitest-environment jsdom
import { describe, it, expect, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { applyCompanyLanguage, initialLocale, LANGUAGE_COOKIE, setLanguage } from '../i18n'
import { LanguageSelect } from '../../components/layout/LanguageSelect'

afterEach(async () => {
  document.cookie = `${LANGUAGE_COOKIE}=; Path=/; Max-Age=0`
  await i18next.changeLanguage('en')
})

describe('admin language', () => {
  it('is remembered in a cookie and read back on the next visit', async () => {
    await setLanguage('pt-BR')
    expect(document.cookie).toContain(`${LANGUAGE_COOKIE}=pt-BR`)
    expect(document.documentElement.lang).toBe('pt-BR')
    expect(initialLocale()).toBe('pt-BR')
  })

  it('switches the whole admin from the picker', async () => {
    render(<LanguageSelect />)
    await userEvent.selectOptions(screen.getByRole('combobox'), 'nl')
    expect(i18next.t('layout:ui.signOut')).toBe('Uitloggen')
    expect(document.cookie).toContain(`${LANGUAGE_COOKIE}=nl`)
  })
})

describe('company language', () => {
  it('is the admin language for a user who picked none', async () => {
    await applyCompanyLanguage('pt-BR')
    expect(i18next.language).toBe('pt-BR')
    expect(document.cookie).not.toContain(`${LANGUAGE_COOKIE}=`)
  })

  it("never overrides the user's own pick", async () => {
    await setLanguage('nl')
    await applyCompanyLanguage('pt-BR')
    expect(i18next.language).toBe('nl')
  })

  it('unset company language changes nothing', async () => {
    await applyCompanyLanguage('')
    expect(i18next.language).toBe('en')
  })
})
