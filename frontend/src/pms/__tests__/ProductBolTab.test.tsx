// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ProductBolTab } from '../ProductBolTab'
import type { Connection, BolProductOverride, BolPricingPreview } from '../../integrations/types'
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
  previewBolPricing: vi.fn(),
  upsertBolProductOverride: vi.fn(),
  getBolProductContent: vi.fn(),
  upsertBolProductContent: vi.fn(),
  pushProductToBol: vi.fn(),
  getProductChannelListing: vi.fn(),
}))

import * as integrationsApi from '../../integrations/api'

const connection: Connection = { id: 'conn-1', provider: 'bol', store_url: '', status: 'connected', created_at: '' }
const sku: SKU = {
  id: 'sku-1', tenant_id: 't1', variant_id: 'v1', sku_code: 'SKU-1', barcode: '0630509892495',
  price_minor_units: 1999, currency_code: 'EUR', is_virtual: false, tracks_expiry: false,
  cost_price_minor_units: 800, status: 'active',
} as unknown as SKU

function ov(fields: Partial<BolProductOverride> = {}): BolProductOverride {
  return {
    product_id: 'prod-1', connection_id: 'conn-1', condition_category: 'NEW', fulfilment_method: 'FBR',
    bundle_pricing_enabled: false, bundle_tiers: [], ...fields,
  }
}

const preview: BolPricingPreview = {
  ean: '0630509892495', regular_price_minor_units: 1999, cost_price_minor_units: 800, product_vat_basis_points: 900,
  shipping_minor_units: 495, levels: [{
    quantity: 1, unit_price_minor_units: 3149, auto: true, fee_fixed_minor_units: 30, fee_percent_basis_points: 1500,
    product_vat_minor_units: 260, bol_fee_incl_vat_minor_units: 608, shipping_share_minor_units: 495,
    net_minor_units: 1836, regular_net_minor_units: 1834, below_regular: false, profit_minor_units: 1036,
    total_price_minor_units: 3149, total_net_minor_units: 1836, total_profit_minor_units: 1036,
  }],
}

async function renderLoaded(props: Partial<Parameters<typeof ProductBolTab>[0]> = {}) {
  render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} {...props} />)
  await screen.findByLabelText('Product title override')
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(integrationsApi.getBolProductOverride).mockResolvedValue(null)
  vi.mocked(integrationsApi.previewBolPricing).mockResolvedValue(preview)
  vi.mocked(integrationsApi.upsertBolProductOverride).mockResolvedValue(ov())
  vi.mocked(integrationsApi.getBolProductContent).mockResolvedValue(null)
  vi.mocked(integrationsApi.pushProductToBol).mockResolvedValue([])
  vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue(null)
})

describe('ProductBolTab', () => {
  it('shows the Bol prices the backend calculated', async () => {
    await renderLoaded()

    expect((await screen.findAllByText('31.49')).length).toBeGreaterThan(0)
    expect(screen.getByLabelText('Bol price')).toHaveAttribute('placeholder', '31.49')
    expect(integrationsApi.previewBolPricing).toHaveBeenCalledWith('test-token', 'conn-1', 'prod-1', 'sku-1', {
      custom_price_minor_units: null, commission_override_basis_points: null, bundle_pricing_enabled: false, bundle_tiers: [],
    })
  })

  it('prefills fields from an existing override', async () => {
    vi.mocked(integrationsApi.getBolProductOverride).mockResolvedValue(ov({
      ean_override: '9999999999994', gpc_code: '80006362', condition_category: 'REFURBISHED',
    }))

    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)

    expect(await screen.findByDisplayValue('9999999999994')).toBeInTheDocument()
    expect(screen.getByDisplayValue('80006362')).toBeInTheDocument()
  })

  // Bol-only title override: reaches PushProductToBolUseCase's Title
  // (backend), never the product's real name -- see push_product_to_bol.go.
  describe('title override', () => {
    it('shows the real product name as a placeholder when no override is set', async () => {
      await renderLoaded({ productName: 'Real Product Name' })

      expect(screen.getByPlaceholderText('Real Product Name')).toBeInTheDocument()
      expect(screen.getByLabelText('Product title override')).toHaveValue('')
    })

    it('prefills the field from an existing name override', async () => {
      vi.mocked(integrationsApi.getBolProductOverride).mockResolvedValue(ov({ name_override: 'Custom Bol Title' }))

      render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)

      expect(await screen.findByDisplayValue('Custom Bol Title')).toBeInTheDocument()
    })

    it('saving sends the entered title override', async () => {
      await renderLoaded()

      await userEvent.type(screen.getByLabelText('Product title override'), 'Custom Bol Title')
      await userEvent.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalledWith(
        'test-token', 'conn-1', 'prod-1',
        expect.objectContaining({ nameOverride: 'Custom Bol Title' }),
      ))
    })

    it('saving with an empty title override sends null (falls back to the real name)', async () => {
      await renderLoaded()

      await userEvent.click(screen.getByRole('button', { name: 'Save' }))

      await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalledWith(
        'test-token', 'conn-1', 'prod-1',
        expect.objectContaining({ nameOverride: null }),
      ))
    })
  })

  it('prefills custom price, fee and bulk levels from an existing override', async () => {
    vi.mocked(integrationsApi.getBolProductOverride).mockResolvedValue(ov({
      custom_price_minor_units: 2499, commission_override_basis_points: 1200,
      bundle_pricing_enabled: true, bundle_tiers: [{ quantity: 3, unit_price_minor_units: 2199 }],
    }))

    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)

    await waitFor(() => expect(screen.getByLabelText('Bol price')).toHaveValue('24.99'))
    expect(screen.getByLabelText('Bol fee %')).toHaveValue('12.00')
    expect(screen.getByLabelText('Bulk discount')).toBeChecked()
    expect(screen.getByLabelText('From units 1')).toHaveValue('3')
    expect(screen.getByLabelText('Price each 1')).toHaveValue('21.99')
  })

  it('saving sends the custom price in minor units', async () => {
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('Bol price'), '24.99')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalledWith(
      'test-token', 'conn-1', 'prod-1',
      expect.objectContaining({ customPriceMinorUnits: 2499 }),
    ))
  })

  it('saving with an empty custom price sends null (automatic price)', async () => {
    await renderLoaded()

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalledWith(
      'test-token', 'conn-1', 'prod-1',
      expect.objectContaining({ customPriceMinorUnits: null, commissionOverrideBasisPoints: null }),
    ))
  })

  it('saving sends the bulk levels', async () => {
    await renderLoaded()

    await userEvent.click(screen.getByLabelText('Bulk discount'))
    await userEvent.type(screen.getByLabelText('Price each 1'), '18')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalledWith(
      'test-token', 'conn-1', 'prod-1',
      expect.objectContaining({
        bundlePricingEnabled: true,
        bundleTiers: [
          { quantity: 2, unit_price_minor_units: 1800 }, { quantity: 4, unit_price_minor_units: null },
          { quantity: 6, unit_price_minor_units: null },
        ],
      }),
    ))
  })

  it('saving other fields keeps the saved bulk levels', async () => {
    vi.mocked(integrationsApi.getBolProductOverride).mockResolvedValue(ov({
      bundle_pricing_enabled: true, bundle_tiers: [{ quantity: 3, unit_price_minor_units: null }],
    }))
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('GPC code'), '80006362')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalledWith(
      'test-token', 'conn-1', 'prod-1',
      expect.objectContaining({ bundlePricingEnabled: true, bundleTiers: [{ quantity: 3, unit_price_minor_units: null }] }),
    ))
  })

  it('saving calls upsertBolProductOverride with the edited fields', async () => {
    await renderLoaded()

    await userEvent.type(screen.getByLabelText('GPC code'), '80006362')
    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalledWith(
      'test-token', 'conn-1', 'prod-1',
      expect.objectContaining({ gpcCode: '80006362', fulfilmentMethod: 'FBR' }),
    ))
  })

  // Saving used to also fire a synchronous POST push-product-bol (same call
  // as the "Push to Bol" button) piggybacked on the save. That call ran
  // Bol's live API inline in the request and 500'd in prod, making a
  // successful save look like a failure. Save now only persists + relies on
  // the pms.product.changed outbox event -> async sync job to push to Bol
  // (see bol_product_content.go / sync_product_to_channels.go).
  it('saving the override does not push the product to Bol', async () => {
    await renderLoaded()

    await userEvent.click(screen.getByRole('button', { name: 'Save' }))

    await waitFor(() => expect(integrationsApi.upsertBolProductOverride).toHaveBeenCalled())
    expect(integrationsApi.pushProductToBol).not.toHaveBeenCalled()
  })

  it('surfaces a pricing error instead of swallowing it', async () => {
    vi.mocked(integrationsApi.previewBolPricing).mockRejectedValue(new Error('bol unreachable'))
    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)

    expect(await screen.findByText('bol unreachable')).toBeInTheDocument()
  })

  describe('product content', () => {
    it('is collapsed by default and prefills saved fields once expanded', async () => {
      vi.mocked(integrationsApi.getBolProductContent).mockResolvedValue({
        product_id: 'prod-1', connection_id: 'conn-1', brand: 'Hipoglós', ingredients: 'Petrolatum, Water',
      })
      render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)
      await screen.findByLabelText('Product title override')

      expect(screen.queryByLabelText('Brand')).not.toBeInTheDocument()
      await userEvent.click(screen.getByRole('button', { name: /Product content/ }))

      expect(await screen.findByLabelText('Brand')).toHaveValue('Hipoglós')
      expect(screen.getByLabelText('Ingredients')).toHaveValue('Petrolatum, Water')
    })

    it('saving sends the edited fields to upsertBolProductContent', async () => {
      vi.mocked(integrationsApi.upsertBolProductContent).mockResolvedValue({
        product_id: 'prod-1', connection_id: 'conn-1', brand: 'Hipoglós',
      })
      render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)
      await screen.findByLabelText('Product title override')
      await userEvent.click(screen.getByRole('button', { name: /Product content/ }))

      await userEvent.type(await screen.findByLabelText('Brand'), 'Hipoglós')
      await userEvent.click(screen.getByLabelText('Hypoallergenic'))
      await userEvent.click(screen.getByRole('button', { name: 'Save product content' }))

      await waitFor(() => expect(integrationsApi.upsertBolProductContent).toHaveBeenCalledWith(
        'test-token', 'conn-1', 'prod-1',
        expect.objectContaining({ brand: 'Hipoglós', hypoallergenic: true }),
      ))
      expect(integrationsApi.pushProductToBol).not.toHaveBeenCalled()
    })

    // Regression test: these fields used to be free-text inputs, but Bol
    // only accepts a value from a fixed list for them -- any typed value
    // that isn't an exact match gets silently dropped, same failure mode
    // as the wrong-attribute-id bug. They're now selects populated from
    // Bol's real data model (bolContentLovs.ts) so a saved value is
    // guaranteed to be one Bol recognizes.
    it('LOV-constrained fields (skin type, country of origin, etc) are selects, not free text', async () => {
      vi.mocked(integrationsApi.upsertBolProductContent).mockResolvedValue({
        product_id: 'prod-1', connection_id: 'conn-1', skin_type: 'Droge huid', country_of_origin: 'Nederland',
      })
      render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)
      await screen.findByLabelText('Product title override')
      await userEvent.click(screen.getByRole('button', { name: /Product content/ }))

      const skinType = await screen.findByLabelText('Skin type')
      expect(skinType.tagName).toBe('SELECT')
      await userEvent.selectOptions(skinType, 'Droge huid')
      await userEvent.selectOptions(screen.getByLabelText('Country of origin'), 'Nederland')
      await userEvent.click(screen.getByRole('button', { name: 'Save product content' }))

      await waitFor(() => expect(integrationsApi.upsertBolProductContent).toHaveBeenCalledWith(
        'test-token', 'conn-1', 'prod-1',
        expect.objectContaining({ skin_type: 'Droge huid', country_of_origin: 'Nederland' }),
      ))
    })
  })

  // Regression test: a push can succeed (sync_status "synced", no error
  // anywhere) while Bol still won't list the offer for sale -- that only
  // ever surfaced through Bol's own seller portal until this fix wired up
  // GET .../not-for-sale-reasons server-side and stored the result on the
  // listing. Without this banner the merchant had no way to find out from
  // CommerceOS at all.
  it('shows a banner when the listing has a not_for_sale_note', async () => {
    vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue({
      id: 'listing-1', product_id: 'prod-1', connection_id: 'conn-1', provider: 'bol',
      external_id: 'offer-1', sync_status: 'synced',
      not_for_sale_note: 'NL: essential product information is missing (missing: Manufacturer Name)',
      created_at: '', updated_at: '',
    })
    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)

    expect(await screen.findByText('Not for sale on Bol')).toBeInTheDocument()
    expect(screen.getByText(/Manufacturer Name/)).toBeInTheDocument()
  })

  it('shows no banner when the listing has no not_for_sale_note', async () => {
    vi.mocked(integrationsApi.getProductChannelListing).mockResolvedValue({
      id: 'listing-1', product_id: 'prod-1', connection_id: 'conn-1', provider: 'bol',
      external_id: 'offer-1', sync_status: 'synced', created_at: '', updated_at: '',
    })
    render(<ProductBolTab productId="prod-1" connection={connection} sku={sku} />)
    await screen.findByLabelText('Product title override')

    expect(screen.queryByText('Not for sale on Bol')).not.toBeInTheDocument()
  })
})
