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
  resolvePickScanCode: vi.fn(),
  deletePickScan: vi.fn(),
  pausePicking: vi.fn(),
  completePicking: vi.fn(),
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
}))

vi.mock('../errorSound', () => ({
  playErrorSound: vi.fn(),
}))

import * as api from '../api'
import * as pmsApi from '../../pms/api'
import * as shippingApi from '../../shipping/api'
import { useConnections } from '../../integrations/hooks/useConnections'
import type { Connection } from '../../integrations/types'

const TOKEN = 'test-token'

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1',
    number: '1001', order_key: '', status: 'processing', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    fulfilled: false,
    billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
    shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' },
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
  it('completing shows the divergence review before actually completing', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail())
    const result: CompletePickingResult = {
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

  it('does not show "Create shipping label" when no MyParcel connection exists', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail())
    vi.mocked(api.completePicking).mockResolvedValue({
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
    })

    it('creates a shipment via the address confirm -> label preview flow, then opens the print view', async () => {
      vi.mocked(api.getPickingSession).mockResolvedValue(detail())
      vi.mocked(api.completePicking).mockResolvedValue({
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
      vi.stubGlobal('print', vi.fn())

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

      // Print label is what actually creates the shipment and opens the print view.
      await user.click(screen.getByRole('button', { name: 'Print label' }))

      await waitFor(() => expect(shippingApi.createShipment).toHaveBeenCalledWith(TOKEN, expect.objectContaining({
        connection_id: 'conn-mp-1', sales_order_id: 'so-1',
      })))
      expect(await screen.findByTitle('Close')).toBeInTheDocument()
      await user.click(screen.getByTitle('Close'))
      expect(await screen.findByRole('button', { name: 'Print shipping label' })).toBeInTheDocument()
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
      vi.stubGlobal('print', vi.fn())

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

      expect(await screen.findByTitle('Close')).toBeInTheDocument()
      await user.click(screen.getByTitle('Close'))
      expect(await screen.findByRole('button', { name: 'Retry print' })).toBeInTheDocument()
    })
  })
})
