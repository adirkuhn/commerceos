// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { LowStockThresholdSection } from '../LowStockThresholdSection'
import type { TenantSettings } from '../types'
import { CodedApiError } from '../../api/codedApiError'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 'tok' }) }))
vi.mock('../api', () => ({ getTenantSettings: vi.fn(), updateTenantSettings: vi.fn() }))
import * as api from '../api'

const settings = { tenant_id: 't', default_currency: 'EUR', timezone: 'UTC', date_format: '', language: 'en', prices_include_tax: true,
  default_tax_country: 'NL', inventory_sync_rules_enabled: false, batch_expiry_threshold_days: 0, default_target_markup_basis_points: 0,
  picking_mode: 'scan', expiry_alert_enabled: false, expiry_alert_windows: [], low_stock_threshold: 10, updated_at: '' } as TenantSettings

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.getTenantSettings).mockResolvedValue(settings)
})

describe('LowStockThresholdSection', () => {
  it('shows the saved threshold', async () => {
    vi.mocked(api.getTenantSettings).mockResolvedValue({ ...settings, low_stock_threshold: 15 })
    render(<LowStockThresholdSection />)

    expect(await screen.findByLabelText('Low stock at or below')).toHaveValue(15)
  })

  it('saves only the low stock threshold', async () => {
    vi.mocked(api.updateTenantSettings).mockResolvedValue({ ...settings, low_stock_threshold: 25 })
    const user = userEvent.setup()
    render(<LowStockThresholdSection />)

    const input = await screen.findByLabelText('Low stock at or below')
    await user.clear(input)
    await user.type(input, '25')
    await user.click(screen.getByRole('button', { name: /save/i }))

    expect(api.updateTenantSettings).toHaveBeenCalledWith('tok', { low_stock_threshold: 25 })
    expect(await screen.findByText('Saved.')).toBeInTheDocument()
  })

  it('accepts 0 (never flag low stock)', async () => {
    vi.mocked(api.updateTenantSettings).mockResolvedValue({ ...settings, low_stock_threshold: 0 })
    const user = userEvent.setup()
    render(<LowStockThresholdSection />)

    const input = await screen.findByLabelText('Low stock at or below')
    await user.clear(input)
    await user.type(input, '0')
    await user.click(screen.getByRole('button', { name: /save/i }))

    expect(api.updateTenantSettings).toHaveBeenCalledWith('tok', { low_stock_threshold: 0 })
  })

  it('shows a save error from the API', async () => {
    vi.mocked(api.updateTenantSettings).mockRejectedValue(new CodedApiError('SETTINGS_INVALID_LOW_STOCK_THRESHOLD', 'bad'))
    const user = userEvent.setup()
    render(<LowStockThresholdSection />)

    await screen.findByLabelText('Low stock at or below')
    await user.click(screen.getByRole('button', { name: /save/i }))

    expect(await screen.findByText('The low stock threshold must be 0 or more.')).toBeInTheDocument()
  })
})
