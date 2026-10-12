// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ExpiryAlertSection } from '../ExpiryAlertSection'
import type { TenantSettings } from '../types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ getTenantSettings: vi.fn(), updateTenantSettings: vi.fn() }))
import * as api from '../api'

const settings = { tenant_id: 't', default_currency: 'EUR', timezone: 'UTC', date_format: '', language: 'en', prices_include_tax: true,
  default_tax_country: 'NL', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0,
  picking_mode: 'scan', expiry_alert_enabled: false, expiry_alert_windows: [], updated_at: '' } as TenantSettings

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getTenantSettings).mockResolvedValue(settings)
})

describe('ExpiryAlertSection', () => {
  it('shows the saved config', async () => {
    vi.mocked(api.getTenantSettings).mockResolvedValue({ ...settings, expiry_alert_enabled: true, expiry_alert_windows: [90, 7] })
    render(<ExpiryAlertSection />)

    expect(await screen.findByRole('switch', { name: /close to expiry/i })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: '90 days' })).toBeChecked()
    expect(screen.getByRole('checkbox', { name: '30 days' })).not.toBeChecked()
    expect(screen.getByRole('checkbox', { name: '7 days' })).toBeChecked()
  })

  it('saves only the expiry report fields', async () => {
    vi.mocked(api.updateTenantSettings).mockResolvedValue({ ...settings, expiry_alert_enabled: true, expiry_alert_windows: [30, 15] })
    const user = userEvent.setup()
    render(<ExpiryAlertSection />)

    await user.click(await screen.findByRole('switch', { name: /close to expiry/i }))
    await user.click(screen.getByRole('checkbox', { name: '15 days' }))
    await user.click(screen.getByRole('checkbox', { name: '30 days' }))
    await user.click(screen.getByRole('button', { name: /save/i }))

    expect(api.updateTenantSettings).toHaveBeenCalledWith('tok', {
      expiry_alert_enabled: true, expiry_alert_windows: [30, 15],
    })
    expect(await screen.findByText('Saved')).toBeInTheDocument()
  })

  it('unchecking a window removes it', async () => {
    vi.mocked(api.getTenantSettings).mockResolvedValue({ ...settings, expiry_alert_enabled: true, expiry_alert_windows: [90, 7] })
    vi.mocked(api.updateTenantSettings).mockResolvedValue(settings)
    const user = userEvent.setup()
    render(<ExpiryAlertSection />)

    await user.click(await screen.findByRole('checkbox', { name: '90 days' }))
    await user.click(screen.getByRole('button', { name: /save/i }))

    expect(api.updateTenantSettings).toHaveBeenCalledWith('tok', expect.objectContaining({ expiry_alert_windows: [7] }))
  })

  it('blocks saving an enabled alert with no window', async () => {
    const user = userEvent.setup()
    render(<ExpiryAlertSection />)

    await user.click(await screen.findByRole('switch', { name: /close to expiry/i }))

    expect(screen.getByText(/pick at least one/i)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /save/i })).toBeDisabled()
  })

  it('shows a save error from the API', async () => {
    vi.mocked(api.updateTenantSettings).mockRejectedValue(new Error('expiry alert windows must be 90, 30, 15 or 7 days'))
    const user = userEvent.setup()
    render(<ExpiryAlertSection />)

    await user.click(await screen.findByRole('checkbox', { name: '7 days' }))
    await user.click(screen.getByRole('button', { name: /save/i }))

    expect(await screen.findByRole('alert')).toHaveTextContent('expiry alert windows must be 90, 30, 15 or 7 days')
  })

  it('a never-saved tenant has the alert off and no window picked', async () => {
    vi.mocked(api.getTenantSettings).mockResolvedValue({ ...settings, expiry_alert_enabled: undefined, expiry_alert_windows: undefined })
    render(<ExpiryAlertSection />)

    expect(await screen.findByRole('switch', { name: /close to expiry/i })).not.toBeChecked()
    expect(screen.getByRole('checkbox', { name: '90 days' })).not.toBeChecked()
  })
})
