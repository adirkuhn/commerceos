// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next from 'i18next'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PickingModeSection } from '../PickingModeSection'
import type { TenantSettings } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ getTenantSettings: vi.fn(), updateTenantSettings: vi.fn() }))
import * as api from '../api'

const settings = { tenant_id: 't', default_currency: 'EUR', timezone: 'UTC', date_format: '', language: 'en', prices_include_tax: true,
  default_tax_country: 'NL', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0,
  picking_mode: 'scan', updated_at: '' } as TenantSettings

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getTenantSettings).mockResolvedValue(settings)
})

describe('PickingModeSection', () => {
  it('shows the current mode', async () => {
    render(<PickingModeSection />)

    expect(await screen.findByRole('radio', { name: /Scan each item/ })).toBeChecked()
    expect(screen.getByRole('radio', { name: /List/ })).not.toBeChecked()
  })

  it('switching to list saves the full settings snapshot with the new mode', async () => {
    vi.mocked(api.updateTenantSettings).mockResolvedValue({ ...settings, picking_mode: 'list' })
    const user = userEvent.setup()
    render(<PickingModeSection />)

    await user.click(await screen.findByRole('radio', { name: /List/ }))

    expect(api.updateTenantSettings).toHaveBeenCalledWith('tok', expect.objectContaining({ picking_mode: 'list', default_currency: 'EUR', prices_include_tax: true }))
    expect(await screen.findByText('Saved')).toBeInTheDocument()
    expect(screen.getByRole('radio', { name: /List/ })).toBeChecked()
  })

  it('shows a save error and keeps the previous mode', async () => {
    vi.mocked(api.updateTenantSettings).mockRejectedValue(new Error('picking mode must be scan or list'))
    const user = userEvent.setup()
    render(<PickingModeSection />)

    await user.click(await screen.findByRole('radio', { name: /List/ }))

    expect(await screen.findByRole('alert')).toHaveTextContent('picking mode must be scan or list')
    expect(screen.getByRole('radio', { name: /Scan each item/ })).toBeChecked()
  })

  it('a never-saved tenant defaults to scan', async () => {
    vi.mocked(api.getTenantSettings).mockResolvedValue({ ...settings, picking_mode: undefined })
    render(<PickingModeSection />)

    expect(await screen.findByRole('radio', { name: /Scan each item/ })).toBeChecked()
  })
})

describe('PickingModeSection — pt-BR', () => {
  it('speaks Portuguese when the admin language is pt-BR', async () => {
    await i18next.changeLanguage('pt-BR')
    try {
      render(<PickingModeSection />)
      expect(await screen.findByRole('radio', { name: /Escanear cada item/ })).toBeChecked()
      expect(screen.getByText('Separação')).toBeInTheDocument()
    } finally {
      await i18next.changeLanguage('en')
    }
  })
})
