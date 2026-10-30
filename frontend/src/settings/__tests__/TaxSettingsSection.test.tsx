// @vitest-environment jsdom
import { screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { TaxSettingsSection } from '../TaxSettingsSection'
import { renderAt } from '../../test/renderAt'
import type { TaxClass, TaxRate } from '../../tax/types'

vi.mock('../../auth/AuthContext', () => ({ useAuth: () => ({ token: 't' }) }))
vi.mock('../hooks/useTenantSettings', () => ({
  useTenantSettings: () => ({ settings: { prices_include_tax: true, default_tax_country: 'NL' }, save: vi.fn() }),
}))
let classes: TaxClass[] = []
let rates: TaxRate[] = []
const update = vi.fn()
vi.mock('../../tax/hooks/useTaxClasses', () => ({
  useTaxClasses: () => ({ classes, loading: false, error: null, create: vi.fn(), update, remove: vi.fn() }),
}))
vi.mock('../../tax/hooks/useTaxRates', () => ({
  useTaxRates: () => ({ rates, error: null, upsert: vi.fn(), remove: vi.fn() }),
}))

const cls = (id: string, name: string, isDefault = false) => ({ id, name, is_default: isDefault }) as TaxClass
const rate = (classId: string, country: string, bp: number) => ({ id: `${classId}-${country}`, tax_class_id: classId, country, rate_basis_points: bp }) as TaxRate

beforeEach(() => {
  vi.clearAllMocks()
  update.mockResolvedValue(undefined)
  classes = [cls('std', 'Standard'), cls('red', 'Reduced')]
  rates = [rate('std', 'NL', 2100), rate('red', 'NL', 900)]
})

const card = (name: string) => screen.getByText(name, { selector: 'span' }).closest('div.rounded-xl') as HTMLElement

describe('TaxSettingsSection default VAT class', () => {
  it('warns when no class is the default', () => {
    renderAt(<TaxSettingsSection />, '/settings/tax')
    expect(screen.getByRole('alert')).toHaveTextContent('No default VAT class')
  })

  it('makes a class the default with one click', async () => {
    renderAt(<TaxSettingsSection />, '/settings/tax')
    await userEvent.click(within(card('Reduced')).getByRole('button', { name: 'Make default' }))
    expect(update).toHaveBeenCalledWith('red', { name: 'Reduced', is_default: true })
  })

  it('the default class has no Make default button, and no warning when it has a rate', () => {
    classes = [cls('std', 'Standard'), cls('red', 'Reduced', true)]
    renderAt(<TaxSettingsSection />, '/settings/tax')
    expect(within(card('Reduced')).queryByRole('button', { name: 'Make default' })).not.toBeInTheDocument()
    expect(within(card('Standard')).getByRole('button', { name: 'Make default' })).toBeInTheDocument()
    expect(screen.queryByRole('alert')).not.toBeInTheDocument()
  })

  it('warns when the default class has no rate for the tax country', () => {
    classes = [cls('std', 'Standard', true)]
    rates = [rate('std', 'BE', 2100)]
    renderAt(<TaxSettingsSection />, '/settings/tax')
    expect(screen.getByRole('alert')).toHaveTextContent('has no rate for NL')
  })
})
