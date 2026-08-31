// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Picking from '../Picking'
import type { SalesOrder } from '../../oms/types'
import type { PickingSession, PickingSessionDetail, ScanPickItemResult, CompletePickingResult, ResolvePickScanCodeResult } from '../../scanner/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: null, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))

vi.mock('../../oms/api', () => ({
  listSalesOrders: vi.fn(),
}))

vi.mock('../../scanner/api', () => ({
  startOrResumePicking: vi.fn(),
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
  listShipmentsForOrder: vi.fn(),
  lookupAddress: vi.fn(),
}))

vi.mock('../../scanner/errorSound', () => ({
  playErrorSound: vi.fn(),
}))

import * as omsApi from '../../oms/api'
import * as pickingApi from '../../scanner/api'
import * as pmsApi from '../../pms/api'
import * as shippingApi from '../../shipping/api'
import { useConnections } from '../../integrations/hooks/useConnections'
import type { Connection } from '../../integrations/types'
import { playErrorSound } from '../../scanner/errorSound'

function salesOrder(overrides: Partial<SalesOrder> = {}): SalesOrder {
  return {
    id: 'so-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1',
    number: '1001', order_key: '', status: 'processing', currency_code: 'USD',
    discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
    total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
    fulfilled: false,
    billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
    ...overrides,
  }
}

function session(overrides: Partial<PickingSession> = {}): PickingSession {
  return { id: 'ps-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '2026-08-22T10:00:00Z', completed_by: '', ...overrides }
}

function detail(overrides: Partial<PickingSessionDetail> = {}): PickingSessionDetail {
  return {
    session: session(),
    line_items: [
      { id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000 },
    ],
    scans: [],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [salesOrder()], total: 1 })
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(useConnections).mockReturnValue({ connections: [], loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
  vi.mocked(shippingApi.lookupAddress).mockResolvedValue(null)
})

describe('Picking page', () => {
  it('lists unfulfilled orders and filters by search', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({ id: 'so-1', number: '1001' }), salesOrder({ id: 'so-2', number: '2002', billing: { first_name: 'Bob', last_name: 'Smith' } })],
      total: 2,
    })
    render(<Picking />)

    expect(await screen.findByText('#1001')).toBeInTheDocument()
    expect(screen.getByText('#2002')).toBeInTheDocument()

    const user = userEvent.setup()
    await user.type(screen.getByPlaceholderText('Search orders...'), 'Bob')

    expect(screen.queryByText('#1001')).not.toBeInTheDocument()
    expect(screen.getByText('#2002')).toBeInTheDocument()
  })

  it('selecting an order starts/resumes a session and shows its line items', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))

    expect(await screen.findByText('Widget')).toBeInTheDocument()
    expect(pickingApi.startOrResumePicking).toHaveBeenCalledWith('test-token', 'so-1')
    expect(screen.getByText('0 / 2 items')).toBeInTheDocument()
  })

  it('scanning a code that matches a line item marks one unit picked immediately, with no sheet', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession)
      .mockResolvedValueOnce(detail())
      .mockResolvedValueOnce(detail({ scans: [{ id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }] }))
    const resolved: ResolvePickScanCodeResult = { sku_id: 'sku-1', product_name: 'Widget' }
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue(resolved)
    const scanResult: ScanPickItemResult = { scan: { id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' } }
    vi.mocked(pickingApi.scanPickItem).mockResolvedValue(scanResult)

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.type(screen.getByPlaceholderText('Scan barcode or type and press Enter...'), '111{Enter}')

    await waitFor(() => expect(pickingApi.scanPickItem).toHaveBeenCalledWith('test-token', 'ps-1', {
      sku_id: 'sku-1', line_item_id: 'li-1', action_type: 'pick', quantity: 1,
    }))
    expect(screen.queryByText('Confirm pick')).not.toBeInTheDocument()
    expect(await screen.findByText('1 / 2 items')).toBeInTheDocument()
  })

  it('scanning a code already fully picked shows a blocking error and stages nothing', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({
      scans: [{ id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 2, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' }],
    }))
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget' })

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    const input = screen.getByPlaceholderText('Scan barcode or type and press Enter...')
    await user.type(input, '111{Enter}')

    expect(await screen.findByText('Already fully picked')).toBeInTheDocument()
    expect(pickingApi.scanPickItem).not.toHaveBeenCalled()
    expect(playErrorSound).toHaveBeenCalled()
    expect(input).toBeDisabled()

    await user.click(screen.getByRole('button', { name: 'OK' }))
    expect(screen.queryByText('Already fully picked')).not.toBeInTheDocument()
    expect(input).toBeEnabled()
  })

  it('scanning a code not on the order shows a blocking error instead of opening Add item', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-other', product_name: 'Gizmo' })

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.type(screen.getByPlaceholderText('Scan barcode or type and press Enter...'), '999{Enter}')

    expect(await screen.findByText('Item not on this order')).toBeInTheDocument()
    expect(pickingApi.scanPickItem).not.toHaveBeenCalled()
    expect(screen.queryByText('Add item not on order')).not.toBeInTheDocument()
    expect(playErrorSound).toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'OK' }))
    expect(screen.queryByText('Item not on this order')).not.toBeInTheDocument()
  })

  it('shows the shipping address in the header once an order is selected', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({ shipping: { address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' } })],
      total: 1,
    })
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    expect(screen.getByText('Damrak 1, 1012AB Amsterdam, NL')).toBeInTheDocument()
  })

  it('substitute mode records a substitute scan for the original line item, using the replacement SKU', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-2', product_name: 'Widget Replacement' })
    vi.mocked(pickingApi.scanPickItem).mockResolvedValue({ scan: { id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-2', action_type: 'substitute', quantity: 2, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'user-1' } })

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    // Row click (not a scan) is now the only way into the Pick/Substitute
    // panel -- a scan of the line's own barcode auto-marks it picked instead.
    await user.click(screen.getByText('Widget'))
    await screen.findByText('Confirm pick')
    await user.click(screen.getByRole('button', { name: 'Substitute' }))
    await user.type(screen.getByPlaceholderText('Scan or type replacement SKU/barcode'), 'sub-code{Enter}')

    await waitFor(() => expect(pickingApi.scanPickItem).toHaveBeenCalledWith('test-token', 'ps-1', {
      sku_id: 'sku-2', line_item_id: 'li-1', action_type: 'substitute',
    }))
  })

  it('pausing returns to the order list', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.pausePicking).mockResolvedValue(session({ status: 'paused' }))

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: /Pause/ }))

    await waitFor(() => expect(pickingApi.pausePicking).toHaveBeenCalledWith('test-token', 'ps-1'))
    expect(await screen.findByText('Select an order to start picking')).toBeInTheDocument()
  })

  it('completing shows missing items and marks the order fulfilled', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    const result: CompletePickingResult = {
      session: session({ status: 'completed' }),
      missing_items: [{ line_item_id: 'li-1', name: 'Widget', expected_qty: 2, picked_qty: 0 }],
    }
    vi.mocked(pickingApi.completePicking).mockResolvedValue(result)

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Complete' }))

    // Complete no longer fires the API directly -- it opens a review step
    // showing the divergences first.
    expect(await screen.findByText('Review before completing')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))

    expect(await screen.findByText('#1001 marked fulfilled')).toBeInTheDocument()
    const missing = screen.getByText('Not fully picked').closest('div')!
    expect(within(missing).getByText('Widget: 0/2')).toBeInTheDocument()
  })

  it('shows the shipping method below the address on the review screen', async () => {
    vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
      salesOrders: [salesOrder({
        shipping: { address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' },
        shipping_method_title: 'PostNL PakjeGemak',
      })],
      total: 1,
    })
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Complete' }))

    expect(await screen.findByText('Review before completing')).toBeInTheDocument()
    expect(screen.getByText('Shipping: PostNL PakjeGemak')).toBeInTheDocument()
  })

  it('does not show "Create shipping label" when no MyParcel connection exists', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.completePicking).mockResolvedValue({ session: session({ status: 'completed' }), missing_items: [] })

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Complete' }))
    await screen.findByText('Review before completing')
    await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))

    await screen.findByText('#1001 marked fulfilled')
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
      vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
      vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
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
      render(<Picking />)
      await user.click(await screen.findByText('#1001'))
      await screen.findByText('Widget')
      await user.click(screen.getByRole('button', { name: 'Complete' }))
      await screen.findByText('Review before completing')

      await user.click(screen.getByRole('button', { name: 'Create shipping label' }))
      expect(await screen.findByText('Label already exists')).toBeInTheDocument()
      expect(screen.queryByText('Confirm shipping address')).not.toBeInTheDocument()
      expect(shippingApi.createShipment).not.toHaveBeenCalled()

      await user.click(screen.getByRole('button', { name: /Download & print again/ }))
      expect(await screen.findByText('Label opened in a new tab')).toBeInTheDocument()
      expect(shippingApi.getShipmentLabel).toHaveBeenCalledWith('test-token', 'ship-existing', 'A4')
      expect(shippingApi.createShipment).not.toHaveBeenCalled()
    })

    it('lets staff create the shipping label from the review screen, before Confirm & Complete', async () => {
      vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
        salesOrders: [salesOrder({
          shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' },
          shipping_method_title: 'DHL Europe',
        })],
        total: 1,
      })
      vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
      vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
      vi.mocked(pickingApi.completePicking).mockResolvedValue({ session: session({ status: 'completed' }), missing_items: [] })
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
      render(<Picking />)
      await user.click(await screen.findByText('#1001'))
      await screen.findByText('Widget')
      await user.click(screen.getByRole('button', { name: 'Complete' }))
      await screen.findByText('Review before completing')

      // Label creation is reachable here, before Confirm & Complete has run.
      expect(pickingApi.completePicking).not.toHaveBeenCalled()
      await user.click(screen.getByRole('button', { name: 'Create shipping label' }))
      expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()

      // The order's shipping method ("DHL Europe") best-effort pre-selects
      // DHL in the carrier dropdown -- still just a default, staff can pick
      // any other carrier from the same select.
      expect(screen.getByLabelText('Carrier')).toHaveValue('9')

      await user.click(screen.getByRole('button', { name: 'Review label' }))
      await user.click(screen.getByRole('button', { name: 'Print label' }))

      await waitFor(() => expect(shippingApi.createShipment).toHaveBeenCalledWith('test-token', expect.objectContaining({
        connection_id: 'conn-mp-1', sales_order_id: 'so-1', carrier: 9,
      })))
      expect(pickingApi.completePicking).not.toHaveBeenCalled()

      // Confirm & Complete still works independently of the label having
      // been created (no in-app overlay to close anymore -- the label
      // opened directly in a new browser tab).
      await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))
      expect(await screen.findByText('#1001 marked fulfilled')).toBeInTheDocument()
    })

    it('creates a shipment from the confirmed address and shows Print shipping label afterward', async () => {
      vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
        salesOrders: [salesOrder({ shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' } })],
        total: 1,
      })
      vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
      vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
      vi.mocked(pickingApi.completePicking).mockResolvedValue({ session: session({ status: 'completed' }), missing_items: [] })
      vi.mocked(shippingApi.createShipment).mockResolvedValue({
        id: 'ship-1', sales_order_id: 'so-1', connection_id: 'conn-mp-1', carrier: 1, barcode: '3SMYPA1',
        status: 1, status_main: 'registered', label_format: 'pdf',
        recipient: { person: 'Jane Doe', street: 'Damrak', city: 'Amsterdam', cc: 'NL' },
        created_at: '2026-08-23T00:00:00Z',
      })
      // Print label always opens the label in a new tab now (the click
      // itself is the print intent) -- give it something to fetch so it
      // doesn't hang.
      vi.mocked(shippingApi.getShipmentLabel).mockResolvedValue(new Blob(['%PDF-fake'], { type: 'application/pdf' }))
      const fakeTab = { close: vi.fn(), location: { href: '' } } as unknown as Window
      vi.spyOn(window, 'open').mockReturnValue(fakeTab)

      const user = userEvent.setup()
      render(<Picking />)
      await user.click(await screen.findByText('#1001'))
      await screen.findByText('Widget')
      await user.click(screen.getByRole('button', { name: 'Complete' }))
      await screen.findByText('Review before completing')
      await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))
      await screen.findByText('#1001 marked fulfilled')

      await user.click(screen.getByRole('button', { name: 'Create shipping label' }))
      expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()

      // Confirming the address only previews now -- nothing is created yet.
      await user.click(screen.getByRole('button', { name: 'Review label' }))
      expect(await screen.findByText('Ready to print')).toBeInTheDocument()
      expect(shippingApi.createShipment).not.toHaveBeenCalled()

      // Print label is what actually creates the shipment.
      await user.click(screen.getByRole('button', { name: 'Print label' }))

      await waitFor(() => expect(shippingApi.createShipment).toHaveBeenCalledWith('test-token', expect.objectContaining({
        connection_id: 'conn-mp-1', sales_order_id: 'so-1',
      })))
      // The label opened in a new tab on its own -- the underlying "Print
      // shipping label" re-access button (auto_print_enabled is off) is
      // already visible, no overlay to close.
      expect(await screen.findByRole('button', { name: 'Print shipping label' })).toBeInTheDocument()
      expect(fakeTab.location.href).toBe('blob:mock-url')
    })

    // Regression: activeOrder switching (clicking a different order in the
    // sidebar) never remounts anything -- pendingRecipient/labelStep etc
    // used to survive the switch untouched, so a previous order's already-
    // edited recipient (set once staff reached the preview step) kept
    // winning over the new order's own address the next time "Create
    // shipping label" was clicked.
    it('does not carry a previous order\'s recipient into a different order\'s label flow', async () => {
      vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
        salesOrders: [
          salesOrder({ id: 'so-1', number: '1001', shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' } }),
          salesOrder({ id: 'so-2', number: '2002', billing: { first_name: 'Bob', last_name: 'Smith' }, shipping: { first_name: 'Bob', last_name: 'Smith', address_1: 'Kalverstraat 2', city: 'Amsterdam', postcode: '1012NX', country: 'NL' } }),
        ],
        total: 2,
      })
      vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
      vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
      vi.mocked(pickingApi.completePicking).mockResolvedValue({ session: session({ status: 'completed' }), missing_items: [] })

      const user = userEvent.setup()
      render(<Picking />)

      // Complete order #1001, start the label flow, and advance to the
      // preview step (this is what sets pendingRecipient) for so-1, then
      // back out without printing.
      await user.click(await screen.findByText('#1001'))
      await screen.findByText('Widget')
      await user.click(screen.getByRole('button', { name: 'Complete' }))
      await screen.findByText('Review before completing')
      await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))
      await screen.findByText('#1001 marked fulfilled')
      await user.click(screen.getByRole('button', { name: 'Create shipping label' }))
      expect(await screen.findByLabelText('Street')).toHaveValue('Damrak')
      await user.click(screen.getByRole('button', { name: 'Review label' }))
      expect(await screen.findByText('Ready to print')).toBeInTheDocument()
      await user.click(screen.getByLabelText('Close'))

      // Switch to a different order and go through the same flow.
      await user.click(screen.getByText('#2002'))
      await screen.findByText('Widget')
      await user.click(screen.getByRole('button', { name: 'Complete' }))
      await screen.findByText('Review before completing')
      await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))
      await screen.findByText('#2002 marked fulfilled')
      await user.click(screen.getByRole('button', { name: 'Create shipping label' }))

      expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()
      expect(screen.getByLabelText('Name')).toHaveValue('Bob Smith')
      expect(screen.getByLabelText('Street')).toHaveValue('Kalverstraat')
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
      vi.mocked(omsApi.listSalesOrders).mockResolvedValue({
        salesOrders: [salesOrder({ shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' } })],
        total: 1,
      })
      vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
      vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
      vi.mocked(pickingApi.completePicking).mockResolvedValue({ session: session({ status: 'completed' }), missing_items: [] })
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
      render(<Picking />)
      await user.click(await screen.findByText('#1001'))
      await screen.findByText('Widget')
      await user.click(screen.getByRole('button', { name: 'Complete' }))
      await screen.findByText('Review before completing')
      await user.click(screen.getByRole('button', { name: 'Confirm & Complete' }))
      await screen.findByText('#1001 marked fulfilled')

      await user.click(screen.getByRole('button', { name: 'Create shipping label' }))
      expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: 'Review label' }))
      expect(await screen.findByText('Ready to print')).toBeInTheDocument()
      await user.click(screen.getByRole('button', { name: 'Print label' }))

      expect(await screen.findByText('Label opened in a new tab')).toBeInTheDocument()
      expect(await screen.findByRole('button', { name: 'Retry print' })).toBeInTheDocument()
    })
  })
})

describe('Picking page product thumbnail', () => {
  it('shows a product thumbnail for a line item linked to a SKU with media', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({
      line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000, product_id: 'product-1' }],
    }))
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([
      { id: 'media-1', tenant_id: 't1', product_id: 'product-1', url: 'https://example.com/thumb.jpg', type: 'image', position: 0, alt_text: 'Widget photo', created_at: '' },
    ])

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    const img = await screen.findByAltText('Widget photo')
    expect(img).toHaveAttribute('src', 'https://example.com/thumb.jpg')
    expect(pmsApi.listProductMedia).toHaveBeenCalledWith('test-token', 'product-1')
  })

  it('falls back to a placeholder icon when the line has no product_id', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({
      line_items: [{ id: 'li-1', sku_id: null, name: 'Unmatched item', sku_code: '', barcode: '', quantity: 1, price_minor_units: 0, total_minor_units: 0, product_id: null }],
    }))

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))

    await screen.findByText('Unmatched item')
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
    expect(pmsApi.listProductMedia).not.toHaveBeenCalled()
  })

  it('falls back to a placeholder icon when the product has no media', async () => {
    vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(session())
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail({
      line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000, product_id: 'product-1' }],
    }))
    vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])

    const user = userEvent.setup()
    render(<Picking />)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')

    await waitFor(() => expect(pmsApi.listProductMedia).toHaveBeenCalledWith('test-token', 'product-1'))
    expect(screen.queryByRole('img')).not.toBeInTheDocument()
  })
})
