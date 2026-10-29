// @vitest-environment jsdom
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MoneybirdSettingsPage } from '../MoneybirdSettingsPage'
import type { MoneybirdSetup } from '../types'
import type { UseMoneybirdSetup } from '../hooks/useMoneybirdSetup'
import { CodedApiError } from '../../api/codedApiError'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 't' }) }))
vi.mock('../../tax/hooks/useTaxRates', () => ({ useTaxRates: () => ({ rates: [{ rate_basis_points: 2100 }, { rate_basis_points: 900 }, { rate_basis_points: 900 }] }) }))
vi.mock('../hooks/useMoneybirdSetup', () => ({ useMoneybirdSetup: vi.fn() }))
import { useMoneybirdSetup } from '../hooks/useMoneybirdSetup'

function setup(over: Partial<MoneybirdSetup['settings']> = {}): MoneybirdSetup {
  return {
    settings: {
      connection_id: 'conn-1', administration_id: 'adm-1', administration_name: 'Cantinho', default_ledger_account_id: 'led-1',
      auto_sync: false, tax_rates: [{ rate_basis_points: 2100, moneybird_tax_rate_id: 'tr-21' }],
      payment_methods: [{ method: 'bacs', moneybird_payment_method: 'bank_transfer' }], ...over,
    },
    administrations: [{ id: 'adm-1', name: 'Cantinho', currency: 'EUR' }, { id: 'adm-2', name: 'Other', currency: 'EUR' }],
    tax_rates: [{ id: 'tr-21', name: '21% btw', percentage: '21.0' }, { id: 'tr-9', name: '9% btw', percentage: '9.0' }],
    ledger_accounts: [{ id: 'led-1', name: 'Omzet' }, { id: 'led-2', name: 'Omzet export' }],
    payment_methods: ['cash', 'bank_transfer', 'pin'],
  }
}

let save: ReturnType<typeof vi.fn>

function mount(s: MoneybirdSetup = setup()) {
  save = vi.fn().mockResolvedValue(undefined)
  const hook: UseMoneybirdSetup = { setup: s, loading: false, error: null, save }
  vi.mocked(useMoneybirdSetup).mockReturnValue(hook)
  render(<MoneybirdSettingsPage onClose={vi.fn()} />)
}

const section = (name: string) => screen.getByRole('region', { name })

beforeEach(() => { vi.clearAllMocks() })

describe('MoneybirdSettingsPage', () => {
  it('saves the ledger account on its own', async () => {
    mount()
    await userEvent.selectOptions(screen.getByLabelText('Ledger account'), 'led-2')
    await userEvent.click(within(section('Administration')).getByRole('button', { name: 'Save' }))
    expect(save).toHaveBeenCalledWith({ default_ledger_account_id: 'led-2' })
    expect(await within(section('Administration')).findByRole('status')).toHaveTextContent('Saved')
  })

  it('switching administration warns and saves only the administration', async () => {
    mount()
    await userEvent.selectOptions(screen.getByRole('combobox', { name: 'Administration' }), 'adm-2')
    expect(screen.getByText(/clears the ledger account and VAT rates/)).toBeInTheDocument()
    expect(screen.queryByLabelText('Ledger account')).not.toBeInTheDocument()
    await userEvent.click(within(section('Administration')).getByRole('button', { name: 'Save' }))
    expect(save).toHaveBeenCalledWith({ administration_id: 'adm-2' })
  })

  it('maps each of the shop\'s VAT rates once and sends only mapped rates', async () => {
    mount()
    const vat = section('VAT rates')
    expect(within(vat).getByLabelText('VAT 21%')).toHaveValue('tr-21')
    expect(within(vat).getAllByLabelText('VAT 9%')).toHaveLength(1)

    await userEvent.selectOptions(within(vat).getByLabelText('VAT 9%'), 'tr-9')
    await userEvent.type(within(vat).getByLabelText('Add VAT rate (%)'), '0')
    await userEvent.click(within(vat).getByRole('button', { name: 'Add' }))
    expect(within(vat).getByLabelText('VAT 0%')).toHaveValue('')
    await userEvent.click(within(vat).getByRole('button', { name: 'Save' }))

    expect(save).toHaveBeenCalledWith({ tax_rates: [
      { rate_basis_points: 2100, moneybird_tax_rate_id: 'tr-21' },
      { rate_basis_points: 900, moneybird_tax_rate_id: 'tr-9' },
    ] })
  })

  it('VAT rates wait for an administration', () => {
    mount(setup({ administration_id: '', default_ledger_account_id: '', tax_rates: [] }))
    expect(within(section('VAT rates')).getByText('Choose an administration first.')).toBeInTheDocument()
  })

  it('adds and removes payment methods', async () => {
    mount()
    const payments = section('Payment methods')
    await userEvent.click(within(payments).getByRole('button', { name: 'Add payment method' }))
    await userEvent.type(within(payments).getAllByLabelText('Order payment method')[1], 'cash')
    await userEvent.selectOptions(within(payments).getAllByLabelText('Moneybird payment method')[1], 'cash')
    await userEvent.click(within(payments).getAllByRole('button', { name: 'Remove' })[0])
    await userEvent.click(within(payments).getByRole('button', { name: 'Save' }))

    expect(save).toHaveBeenCalledWith({ payment_methods: [{ method: 'cash', moneybird_payment_method: 'cash' }] })
  })

  it('auto sync saves when switched', async () => {
    mount()
    await userEvent.click(screen.getByLabelText('Send automatically'))
    expect(save).toHaveBeenCalledWith({ auto_sync: true })
  })

  it('auto sync flips back when the save fails', async () => {
    mount()
    save.mockRejectedValue(new CodedApiError('MONEYBIRD_REQUEST_FAILED', 'x', undefined))
    await userEvent.click(screen.getByLabelText('Send automatically'))
    expect(await within(section('Auto sync')).findByRole('alert')).toHaveTextContent("Moneybird didn't accept the request.")
    expect(screen.getByLabelText('Send automatically')).not.toBeChecked()
  })

  it('shows why a save failed', async () => {
    mount()
    save.mockRejectedValue(new CodedApiError('MONEYBIRD_UNKNOWN_ID', 'x', undefined))
    await userEvent.click(within(section('Administration')).getByRole('button', { name: 'Save' }))
    expect(await within(section('Administration')).findByRole('alert')).toHaveTextContent("Moneybird doesn't have this administration")
  })
})
