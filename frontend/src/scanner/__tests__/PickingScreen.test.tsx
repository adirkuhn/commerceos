// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PickingScreen } from '../PickingScreen'
import type { SalesOrder } from '../../oms/types'
import type { PickingSessionDetail, CompletePickingResult } from '../types'

vi.mock('../api', () => ({
  getPickingSession: vi.fn(),
  scanPickItem: vi.fn(),
  substituteLineItem: vi.fn(),
  resolvePickScanCode: vi.fn(),
  deletePickScan: vi.fn(),
  pausePicking: vi.fn(),
  completePicking: vi.fn(),
}))

vi.mock('../../pms/searchCandidates', () => ({
  searchSKUCandidates: vi.fn(),
}))

vi.mock('../../pms/api', () => ({
  listProductMedia: vi.fn(),
}))

vi.mock('../../integrations/hooks/useConnections', () => ({
  useConnections: vi.fn(),
}))

vi.mock('../../shipping/api', () => ({
  createShipment: vi.fn(),
  getShippingSettings: vi.fn(),
  getShipmentLabel: vi.fn(),
  listShipmentsForOrder: vi.fn(),
}))

vi.mock('../errorSound', () => ({
  playErrorSound: vi.fn(),
}))

import * as api from '../api'
import * as pmsApi from '../../pms/api'
import * as shippingApi from '../../shipping/api'
import * as searchCandidatesApi from '../../pms/searchCandidates'
import { useConnections } from '../../integrations/hooks/useConnections'
import type { Connection } from '../../integrations/types'
import type { SKU } from '../../pms/types'

const TOKEN = 'test-token'

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1',
    number: '1001', order_key: '', status: 'processing', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    fulfilled: false,
    billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
    shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL', email: 'jane@example.com' },
    ...overrides,
  }
}

function detail(overrides: Partial<PickingSessionDetail> = {}): PickingSessionDetail {
  return {
    session: { id: 'ps-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '2026-08-22T10:00:00Z', completed_by: '' },
    line_items: [
      { id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000 },
    ],
    scans: [],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(useConnections).mockReturnValue({ connections: [], loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
})

describe('PickingScreen', () => {
  it('shows the billing address when it differs from shipping', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail())

    render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder({
      billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com', address_1: 'Kalverstraat 5', city: 'Amsterdam', postcode: '1012NX', country: 'NL' },
    })} onExit={vi.fn()} onAuthError={vi.fn()} />)

    expect(await screen.findByText(/Billing:/)).toBeInTheDocument()
    expect(screen.getByText(/Kalverstraat 5/)).toBeInTheDocument()
  })

  it('hides the billing address when it matches shipping', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail())

    render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder({
      billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' },
    })} onExit={vi.fn()} onAuthError={vi.fn()} />)

    await screen.findByText('Widget')
    expect(screen.queryByText(/Billing:/)).not.toBeInTheDocument()
  })

  it('shows the sold expiry date reported by WooCommerce when a line item has one', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail({
      line_items: [
        { id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000, sold_expiry_date: '2027-06-12' },
      ],
    }))

    render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder()} onExit={vi.fn()} onAuthError={vi.fn()} />)

    expect(await screen.findByText(/Sold exp\./)).toBeInTheDocument()
  })

  it('shows no expiry line when the line item has none (the normal case)', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail())

    render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder()} onExit={vi.fn()} onAuthError={vi.fn()} />)

    await screen.findByText('Widget')
    expect(screen.queryByText(/Sold exp\./)).not.toBeInTheDocument()
  })

  it('completing shows the divergence review before actually completing', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail())
    const result: CompletePickingResult = {
      completed: true,
      session: { id: 'ps-1', sales_order_id: 'so-1', status: 'completed', started_at: '2026-08-22T10:00:00Z', completed_by: '' },
      missing_items: [{ line_item_id: 'li-1', name: 'Widget', expected_qty: 2, picked_qty: 0 }],
    }
    vi.mocked(api.completePicking).mockResolvedValue(result)

    const user = userEvent.setup()
    render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder()} onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Finish' }))
    expect(await screen.findByText('Review before finishing')).toBeInTheDocument()
    expect(screen.getByText("1 line(s) aren't fully picked yet:")).toBeInTheDocument()
    expect(api.completePicking).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))
    expect(await screen.findByText('Order picked')).toBeInTheDocument()
    expect(screen.getByText("1 line(s) weren't fully picked:")).toBeInTheDocument()
  })

  // Regression test for a real bug: an order line for 4x Beans, 3 already
  // picked, and staff want to substitute only the last unit -- the confirm
  // button used to require covering the line's FULL original quantity (4)
  // regardless of what was already picked, making a partial substitute
  // impossible.
  it('substitute mode only requires covering the remaining (unpicked) quantity, not the full original quantity', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail({
      line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Beans', sku_code: 'BEANS-1', barcode: '111', quantity: 4, price_minor_units: 100, total_minor_units: 400 }],
      scans: [{ id: 'scan-0', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 3, scanned_at: '2026-08-22T10:00:30Z', scanned_by: 'user-1' }],
    }))
    const candidateSKU: SKU = {
      id: 'sku-2', tenant_id: 'tenant-1', variant_id: 'v-2', sku_code: 'BEANS-CAN', price_minor_units: 150, currency_code: 'USD',
      is_virtual: false, status: 'active', created_at: '2026-08-01T00:00:00Z', updated_at: '2026-08-01T00:00:00Z',
    }
    vi.mocked(searchCandidatesApi.searchSKUCandidates).mockResolvedValue([{ sku: candidateSKU, productId: 'prod-2', productName: 'Canned Beans' }])
    vi.mocked(api.substituteLineItem).mockResolvedValue({
      scans: [{ id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-2', action_type: 'substitute', quantity: 1, original_quantity_covered: 1, return_to_stock: false, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }],
    })

    const user = userEvent.setup()
    render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder()} onExit={vi.fn()} onAuthError={vi.fn()} />)
    await user.click(await screen.findByText('Beans'))
    await user.click(screen.getByRole('button', { name: /Substitute/i }))

    await user.type(screen.getByPlaceholderText('Search or scan barcode…'), 'canned')
    await user.click(screen.getByRole('button', { name: 'Find' }))
    await user.click(await screen.findByText('Canned Beans'))

    // Covering the OLD full-quantity requirement (4) must now be rejected.
    await user.clear(screen.getByLabelText('Covers (of original)'))
    await user.type(screen.getByLabelText('Covers (of original)'), '4')
    expect(screen.getByRole('button', { name: 'Confirm substitute' })).toBeDisabled()

    // Covering just what's left (1) must be accepted.
    await user.clear(screen.getByLabelText('Covers (of original)'))
    await user.type(screen.getByLabelText('Covers (of original)'), '1')
    expect(screen.getByRole('button', { name: 'Confirm substitute' })).toBeEnabled()

    await user.click(screen.getByRole('button', { name: 'Confirm substitute' }))

    await waitFor(() => expect(api.substituteLineItem).toHaveBeenCalledWith(TOKEN, 'ps-1', {
      line_item_id: 'li-1',
      lines: [{ sku_id: 'sku-2', substitute_quantity: 1, original_quantity_covered: 1, return_to_stock: false }],
    }))
  })

  it('shows a stock-shortfall dialog and resubmits with the chosen resolution', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail())
    vi.mocked(api.completePicking)
      .mockResolvedValueOnce({
        completed: false,
        stock_shortfalls: [{ scan_id: 'scan-1', sku_id: 'sku-1', name: 'Widget', needed: 5, short: 3 }],
      })
      .mockResolvedValueOnce({
        completed: true,
        session: { id: 'ps-1', sales_order_id: 'so-1', status: 'completed', started_at: '', completed_by: '' },
        missing_items: [],
      })

    const user = userEvent.setup()
    render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder()} onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Finish' }))
    await screen.findByText('Review before finishing')
    await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))

    expect(await screen.findByText('Not enough stock')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))

    expect(await screen.findByText('Order picked')).toBeInTheDocument()
    expect(api.completePicking).toHaveBeenLastCalledWith(TOKEN, 'ps-1', { 'scan-1': 'with_item' })
  })

  it('going back from the review screen returns to scanning without completing', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder()} onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Finish' }))
    await screen.findByText('Review before finishing')
    await user.click(screen.getByRole('button', { name: 'Back, scan more' }))

    expect(await screen.findByText('Picking')).toBeInTheDocument()
    expect(api.completePicking).not.toHaveBeenCalled()
  })

  it('shows the shipping method below the address on the review screen', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder({ shipping_method_title: 'PostNL PakjeGemak' })} onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Finish' }))

    expect(await screen.findByText('Review before finishing')).toBeInTheDocument()
    expect(screen.getByText('Damrak 1, 1012AB Amsterdam, NL')).toBeInTheDocument()
    expect(screen.getByText('Shipping: PostNL PakjeGemak')).toBeInTheDocument()
  })

  it('does not show "Create shipping label" when no MyParcel connection exists', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail())
    vi.mocked(api.completePicking).mockResolvedValue({
      completed: true,
      session: { id: 'ps-1', sales_order_id: 'so-1', status: 'completed', started_at: '', completed_by: '' },
      missing_items: [],
    })

    const user = userEvent.setup()
    render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder()} onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Finish' }))
    await screen.findByText('Review before finishing')
    await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))

    await screen.findByText('Order picked')
    expect(screen.queryByRole('button', { name: 'Create shipping label' })).not.toBeInTheDocument()
  })

  describe('with a MyParcel connection', () => {
    const myParcelConn: Connection = { id: 'conn-mp-1', provider: 'myparcel', store_url: '', status: 'connected', created_at: '2026-08-01T00:00:00Z' }

    beforeEach(() => {
      vi.mocked(useConnections).mockReturnValue({ connections: [myParcelConn], loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
      vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: null, carrier_overrides: [] })
      vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([])
      // openLabel.ts's blob->new-tab flow -- jsdom has no real implementation.
      URL.createObjectURL = vi.fn(() => 'blob:mock-url')
    })

    it('offers to reprint instead of creating a new label when a shipment already exists for the order', async () => {
      vi.mocked(api.getPickingSession).mockResolvedValue(detail())
      vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([{
        id: 'ship-existing', sales_order_id: 'so-1', connection_id: 'conn-mp-1', carrier: 1, barcode: '3SMYPA9',
        status: 1, status_main: 'registered', label_format: 'pdf',
        recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
        created_at: '2026-08-23T00:00:00Z',
      }])
      vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(new Blob(['%PDF-fake'], { type: 'application/pdf' }))
      const fakeTab = { close: vi.fn(), location: { href: '' } } as unknown as Window
      vi.spyOn(window, 'open').mockReturnValue(fakeTab)

      const user = userEvent.setup()
      render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder()} onExit={vi.fn()} onAuthError={vi.fn()} />)
      await screen.findByText('Widget')
      await user.click(screen.getByRole('button', { name: 'Finish' }))
      await screen.findByText('Review before finishing')

      await user.click(screen.getByRole('button', { name: 'Create shipping label' }))
      expect(await screen.findByText('Label already exists')).toBeInTheDocument()
      expect(screen.queryByText('Confirm shipping address')).not.toBeInTheDocument()
      expect(shippingApi.createShipment).not.toHaveBeenCalled()

      await user.click(screen.getByRole('button', { name: /Download & print again/ }))
      expect(await screen.findByText('Label opened in a new tab')).toBeInTheDocument()
      expect(fakeTab.location.href).toBe('blob:mock-url')
      expect(shippingApi.createShipment).not.toHaveBeenCalled()
    })

    it('lets staff create the shipping label from the review screen, before Confirm & Complete', async () => {
      vi.mocked(api.getPickingSession).mockResolvedValue(detail())
      vi.mocked(api.completePicking).mockResolvedValue({
        completed: true,
        session: { id: 'ps-1', sales_order_id: 'so-1', status: 'completed', started_at: '', completed_by: '' },
        missing_items: [],
      })
      vi.mocked(shippingApi.createShipment).mockResolvedValue({
        id: 'ship-1', sales_order_id: 'so-1', connection_id: 'conn-mp-1', carrier: 9, barcode: '3SMYPA1',
        status: 1, status_main: 'registered', label_format: 'pdf',
        recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
        created_at: '2026-08-23T00:00:00Z',
      })
      vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(new Blob(['%PDF-fake'], { type: 'application/pdf' }))
      const fakeTab = { close: vi.fn(), location: { href: '' } } as unknown as Window
      vi.spyOn(window, 'open').mockReturnValue(fakeTab)

      const user = userEvent.setup()
      render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder({ shipping_method_title: 'DHL Europe' })} onExit={vi.fn()} onAuthError={vi.fn()} />)
      await screen.findByText('Widget')
      await user.click(screen.getByRole('button', { name: 'Finish' }))
      await screen.findByText('Review before finishing')

      expect(api.completePicking).not.toHaveBeenCalled()
      await user.click(screen.getByRole('button', { name: 'Create shipping label' }))
      expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()
      expect(screen.getByLabelText('Carrier')).toHaveValue('9')

      await user.click(screen.getByRole('button', { name: 'Review label' }))
      await user.click(screen.getByRole('button', { name: 'Print label' }))

      await waitFor(() => expect(shippingApi.createShipment).toHaveBeenCalledWith(TOKEN, expect.objectContaining({
        connection_id: 'conn-mp-1', sales_order_id: 'so-1', carrier: 9,
      })))
      expect(api.completePicking).not.toHaveBeenCalled()

      // No overlay to close anymore -- the label opened directly in a new
      // browser tab.
      await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))
      expect(await screen.findByText('Order picked')).toBeInTheDocument()
    })

    it('creates a shipment via the address confirm -> label preview flow, then opens the print view', async () => {
      vi.mocked(api.getPickingSession).mockResolvedValue(detail())
      vi.mocked(api.completePicking).mockResolvedValue({
        completed: true,
        session: { id: 'ps-1', sales_order_id: 'so-1', status: 'completed', started_at: '', completed_by: '' },
        missing_items: [],
      })
      vi.mocked(shippingApi.createShipment).mockResolvedValue({
        id: 'ship-1', sales_order_id: 'so-1', connection_id: 'conn-mp-1', carrier: 1, barcode: '3SMYPA1',
        status: 1, status_main: 'registered', label_format: 'pdf',
        recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
        created_at: '2026-08-23T00:00:00Z',
      })
      vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(new Blob(['%PDF-fake'], { type: 'application/pdf' }))
      const fakeTab = { close: vi.fn(), location: { href: '' } } as unknown as Window
      vi.spyOn(window, 'open').mockReturnValue(fakeTab)

      const user = userEvent.setup()
      render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder()} onExit={vi.fn()} onAuthError={vi.fn()} />)
      await screen.findByText('Widget')
      await user.click(screen.getByRole('button', { name: 'Finish' }))
      await screen.findByText('Review before finishing')
      await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))
      await screen.findByText('Order picked')

      await user.click(screen.getByRole('button', { name: 'Create shipping label' }))
      expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()

      // Confirming the address only previews now -- nothing is created yet.
      await user.click(screen.getByRole('button', { name: 'Review label' }))
      expect(await screen.findByText('Ready to print')).toBeInTheDocument()
      expect(shippingApi.createShipment).not.toHaveBeenCalled()

      // Print label is what actually creates the shipment and opens the label in a new tab.
      await user.click(screen.getByRole('button', { name: 'Print label' }))

      await waitFor(() => expect(shippingApi.createShipment).toHaveBeenCalledWith(TOKEN, expect.objectContaining({
        connection_id: 'conn-mp-1', sales_order_id: 'so-1',
      })))
      expect(await screen.findByRole('button', { name: 'Print shipping label' })).toBeInTheDocument()
      expect(fakeTab.location.href).toBe('blob:mock-url')
    })

    it('auto-opens the print view when auto_print_enabled is on, and offers Retry print', async () => {
      vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({
        settings: {
          connection_id: 'conn-mp-1', default_carrier: 1, default_package_type: 1, default_label_format: 'pdf',
          default_signature_required: false, default_only_recipient: false, insure_by_default: false,
          myparcel_account_id: null, myparcel_shop_id: null, auto_print_enabled: true,
        },
        carrier_overrides: [],
      })
      vi.mocked(api.getPickingSession).mockResolvedValue(detail())
      vi.mocked(api.completePicking).mockResolvedValue({
        completed: true,
        session: { id: 'ps-1', sales_order_id: 'so-1', status: 'completed', started_at: '', completed_by: '' },
        missing_items: [],
      })
      vi.mocked(shippingApi.createShipment).mockResolvedValue({
        id: 'ship-1', sales_order_id: 'so-1', connection_id: 'conn-mp-1', carrier: 1, barcode: '3SMYPA1',
        status: 1, status_main: 'registered', label_format: 'pdf',
        recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
        created_at: '2026-08-23T00:00:00Z',
      })
      vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(new Blob(['%PDF-fake'], { type: 'application/pdf' }))
      const fakeTab = { close: vi.fn(), location: { href: '' } } as unknown as Window
      vi.spyOn(window, 'open').mockReturnValue(fakeTab)

      const user = userEvent.setup()
      render(<PickingScreen token={TOKEN} sessionId="ps-1" order={salesOrder()} onExit={vi.fn()} onAuthError={vi.fn()} />)
      await screen.findByText('Widget')
      await user.click(screen.getByRole('button', { name: 'Finish' }))
      await screen.findByText('Review before finishing')
      await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))
      await screen.findByText('Order picked')

      await user.click(screen.getByRole('button', { name: 'Create shipping label' }))
      await screen.findByText('Confirm shipping address')
      await user.click(screen.getByRole('button', { name: 'Review label' }))
      await screen.findByText('Ready to print')
      await user.click(screen.getByRole('button', { name: 'Print label' }))

      expect(await screen.findByText('Label opened in a new tab')).toBeInTheDocument()
      expect(await screen.findByRole('button', { name: 'Retry print' })).toBeInTheDocument()
    })
  })
})
