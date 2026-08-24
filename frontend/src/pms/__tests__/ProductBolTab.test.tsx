// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ProductBolTab } from '../ProductBolTab'
import type { Connection, BolProductOverride, BolCommission } from '../../integrations/types'
import type { SKU } from '../types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({
    token: 'test-token',
    user: { user_id: 'u1', tenant_id: 'tenant-1', role: 'customer_owner', exp: 0, iat: 0 },
    login: vi.fn(), logout: vi.fn(), isAuthenticated: true,
  }),
}))

vi.mock('../../integrations/api', () => ({
  getBolProductOverride: vi.fn(),
  getBolCommission: vi.fn(),
  upsertBolProductOverride: vi.fn(),
}))

import * as integrationsApi from '../../integrations/api'

const connection: Connection = { id: 'conn-1', provider: 'bol', store_url: '', status: 'connected', created_at: '' }
const sku: SKU = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'v1', sku_code: 'SKU-1', barcode: '0630509892495',
  price_minor_units: 1999, currency_code: 'EUR', is_virtual: false, tracks_expiry: false,
  cost_price_minor_units: 800, status: 'active',
} as unknown as SKU

const commission: BolCommission = { ean: '0630509892495', fixed_amount: 0.30, percentage: 12.5, overridden: false }

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.getBolProductOverride).mockResolvedValue(null)
  vi.mocked(integrationsApi.getBolCommission).mockResolvedValue(commission)
})

describe('ProductBolTab', () => {
  it('loads and displays live commission + sell/cost price, computes profit breakdown', async () => {
    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)

    await waitFor(() => expect(screen.getByDisplayValue('19.99')).toBeInTheDocument())
    expect(screen.getByDisplayValue('8.00')).toBeInTheDocument()
    expect(screen.getByLabelText(/Bol commission %/)).toHaveValue('12.50')
    expect(screen.getByText('Profit')).toBeInTheDocument()
  })

  it('prefills fields from an existing override', async () => {
    const override: BolProductOverride = {
      product_id: 'prod-1', connection_id: 'conn-1', ean_override: '9999999999999',
      gpc_code: '80006362', condition_category: 'REFURBISHED', fulfilment_method: 'FBR',
    }
    vi.mocked(integrationsApi.getBolProductOverride).mockResolvedValue(override)

    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)

    expect(await screen.findByDisplayValue('9999999999999')).toBeInTheDocument()
    expect(screen.getByDisplayValue('80006362')).toBeInTheDocument()
  })

  it('prefills custom price from an existing override and uses it as the sell price', async () => {
    const override: BolProductOverride = {
      product_id: 'prod-1', connection_id: 'conn-1', condition_category: 'NEW', fulfilment_method: 'FBR',
      custom_price_minor_units: 2499,
    }
    vi.mocked(integrationsApi.getBolProductOverride).mockResolvedValue(override)

    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)

    await waitFor(() => expect(screen.getAllByDisplayValue('24.99')).toHaveLength(2))
  })

  it('falls back to the default price when no custom price is set', async () => {
    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)

    await waitFor(() => expect(screen.getByDisplayValue('19.99')).toBeInTheDocument())
    expect(screen.getByLabelText(/Custom price/)).toHaveValue('')
  })

  it('saving sends the entered custom price in minor units', async () => {
    vi.mocked(integrationsApi.upsertBolProductOverride).mockResolvedValue({
      product_id: 'prod-1', connection_id: 'conn-1', condition_category: 'NEW', fulfilment_method: 'FBR',
    })
    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)
    await waitFor(() => expect(screen.getByDisplayValue('19.99')).toBeInTheDocument())

    await userEvent.type(screen.getByLabelText(/Custom price/), '24.99')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalledWith(
      'test-token', 'conn-1', 'prod-1',
      expect.objectContaining({ customPriceMinorUnits: 2499 }),
    ))
  })

  it('saving with an empty custom price sends null (falls back to default)', async () => {
    vi.mocked(integrationsApi.upsertBolProductOverride).mockResolvedValue({
      product_id: 'prod-1', connection_id: 'conn-1', condition_category: 'NEW', fulfilment_method: 'FBR',
    })
    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)
    await waitFor(() => expect(screen.getByDisplayValue('19.99')).toBeInTheDocument())

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalledWith(
      'test-token', 'conn-1', 'prod-1',
      expect.objectContaining({ customPriceMinorUnits: null }),
    ))
  })

  it('re-fetches commission after saving a new custom price', async () => {
    vi.mocked(integrationsApi.upsertBolProductOverride).mockResolvedValue({
      product_id: 'prod-1', connection_id: 'conn-1', condition_category: 'NEW', fulfilment_method: 'FBR',
      custom_price_minor_units: 2499,
    })
    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)
    await waitFor(() => expect(screen.getByDisplayValue('19.99')).toBeInTheDocument())
    expect(integrationsApi.getBolCommission).toHaveBeenCalledTimes(1)

    await userEvent.type(screen.getByLabelText(/Custom price/), '24.99')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.getBolCommission).toHaveBeenCalledTimes(2))
  })

  it('saving calls upsertBolProductOverride with the edited fields', async () => {
    vi.mocked(integrationsApi.upsertBolProductOverride).mockResolvedValue({
      product_id: 'prod-1', connection_id: 'conn-1', condition_category: 'NEW', fulfilment_method: 'FBR',
    })
    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)
    await waitFor(() => expect(screen.getByDisplayValue('19.99')).toBeInTheDocument())

    await userEvent.type(screen.getByLabelText('GPC Code'), '80006362')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalledWith(
      'test-token', 'conn-1', 'prod-1',
      expect.objectContaining({ gpcCode: '80006362', fulfilmentMethod: 'FBR' }),
    ))
  })

  it('surfaces a commission-fetch error instead of swallowing it', async () => {
    vi.mocked(integrationsApi.getBolCommission).mockRejectedValue(new Error('bol unreachable'))
    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)

    expect(await screen.findByText('bol unreachable')).toBeInTheDocument()
  })
})
