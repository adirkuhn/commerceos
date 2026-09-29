// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { LocalizationSettingsSection } from '../LocalizationSettingsSection'
import { LANGUAGE_COOKIE } from '../../i18n/i18n'
import type { TenantSettings } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ getTenantSettings: vi.fn(), updateTenantSettings: vi.fn() }))
vi.mock('../../fx/hooks/useExchangeRates', () => ({
  useExchangeRates: () => ({ rates: [], loading: false, error: '', upsert: vi.fn(), remove: vi.fn() }),
}))
import * as api from '../api'

const settings = { tenant_id: 't', default_currency: 'EUR', timezone: 'Europe/Amsterdam', date_format: 'DD/MM/YYYY', language: 'nl',
  prices_include_tax: true, default_tax_country: 'NL', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0,
  default_target_markup_basis_points: 0, picking_mode: 'scan', updated_at: '' } as TenantSettings

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getTenantSettings).mockResolvedValue(settings)
})

afterEach(async () => {
  document.cookie = `${LANGUAGE_COOKIE}=; Path=/; Max-Age=0`
  await i18next.changeLanguage('en')
})

async function languageSelect() {
  return screen.findByRole('combobox', { name: 'Language' })
}

describe('LocalizationSettingsSection — language', () => {
  it('offers the languages CommerceOS speaks, each named in itself', async () => {
    render(<LocalizationSettingsSection />)
    const select = await languageSelect()

    expect(within(select).getAllByRole('option').map(o => [o.getAttribute('value'), o.textContent])).toEqual([
      ['en', 'English'], ['pt-BR', 'Português (Brasil)'], ['nl', 'Nederlands'],
    ])
    expect(select).toHaveValue('nl')
  })

  it('saves Portuguese as the company language, and the admin follows it for a user who picked none', async () => {
    vi.mocked(api.updateTenantSettings).mockResolvedValue({ ...settings, language: 'pt-BR' })
    const user = userEvent.setup()
    render(<LocalizationSettingsSection />)
    await screen.findByDisplayValue('Nederlands')

    await user.selectOptions(await languageSelect(), 'pt-BR')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(api.updateTenantSettings).toHaveBeenCalledWith('tok', expect.objectContaining({ language: 'pt-BR', default_currency: 'EUR' }))
    expect(await screen.findByText('Salvo.')).toBeInTheDocument()
    expect(i18next.language).toBe('pt-BR')
  })

  it('a user who picked their own language keeps it', async () => {
    document.cookie = `${LANGUAGE_COOKIE}=en; Path=/`
    vi.mocked(api.updateTenantSettings).mockResolvedValue({ ...settings, language: 'pt-BR' })
    const user = userEvent.setup()
    render(<LocalizationSettingsSection />)
    await screen.findByDisplayValue('Nederlands')

    await user.selectOptions(await languageSelect(), 'pt-BR')
    await user.click(screen.getByRole('button', { name: 'Save changes' }))

    expect(await screen.findByText('Saved.')).toBeInTheDocument()
    expect(i18next.language).toBe('en')
  })

  it('a never-saved tenant shows English', async () => {
    vi.mocked(api.getTenantSettings).mockResolvedValue({ ...settings, language: '' })
    render(<LocalizationSettingsSection />)
    await screen.findByDisplayValue('Europe/Amsterdam (UTC+2)')

    expect(await languageSelect()).toHaveValue('en')
  })
})
