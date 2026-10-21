// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PackingScreen } from '../PackingScreen'
import type { SalesOrder } from '../../oms/types'
import type { PackingSessionDetail } from '../types'
import type { SKU } from '../../pms/types'

vi.mock('../api', () => ({
  getPackingSession: vi.fn(),
  scanPackItem: vi.fn(),
  substitutePackLine: vi.fn(),
  refundPackLine: vi.fn(),
  deletePackScan: vi.fn(),
  completePacking: vi.fn(),
  touchPackingSession: vi.fn(),
  resetPackingSession: vi.fn(),
  releasePackingSessionLock: vi.fn(),
  takeOverPackingSession: vi.fn(),
}))
vi.mock('../../team/api', () => ({ listMembers: vi.fn().mockResolvedValue([]) }))
vi.mock('../../pms/searchCandidates', () => ({ searchSKUCandidates: vi.fn() }))
vi.mock('../../pms/api', () => ({ listProductMedia: vi.fn() }))
vi.mock('../../shipping/api', () => ({
  listShippingConnections: vi.fn(),
  createShipment: vi.fn(), getShippingSettings: vi.fn(), getShipmentLabel: vi.fn(), listShipmentsForOrder: vi.fn(),
}))
vi.mock('../errorSound', () => ({ playErrorSound: vi.fn() }))

import * as api from '../api'
import * as pmsApi from '../../pms/api'
import * as searchCandidatesApi from '../../pms/searchCandidates'
import * as shippingApi from '../../shipping/api'
import { shippingConnectionsOf } from '../../test/shippingConnections'

// A scanner token for user "me".
const TOKEN = `x.${btoa(JSON.stringify({ user_id: 'me', role: 'customer_user' }))}.y`

const order = {
  id: 'so-1', number: '1001', external_order_id: 'ext-1', currency_code: 'EUR', total_minor_units: 5000,
  billing: { first_name: 'Jane', last_name: 'Doe' },
} as SalesOrder

function detail(overrides: Partial<PackingSessionDetail> = {}): PackingSessionDetail {
  return {
    session: { id: 'pack-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '', completed_by: '' },
    line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000 }],
    scans: [],
    ...overrides,
  }
}

const candidate = { sku: { id: 'sku-sub', sku_code: 'SUB-1' } as SKU, productId: 'p', productName: 'Other widget' }

function renderScreen() {
  render(<PackingScreen token={TOKEN} sessionId="pack-1" order={order} onExit={vi.fn()} onAuthError={vi.fn()} />)
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([]))
  vi.mocked(api.getPackingSession).mockResolvedValue(detail())
  vi.mocked(searchCandidatesApi.searchSKUCandidates).mockResolvedValue([candidate])
  vi.mocked(api.releasePackingSessionLock).mockResolvedValue(null)
  vi.mocked(api.touchPackingSession).mockResolvedValue(null)
})

describe('PackingScreen order changes', () => {
  it('a line can be substituted from its sheet', async () => {
    vi.mocked(api.substitutePackLine).mockResolvedValue({ scans: [] })
    const user = userEvent.setup()
    renderScreen()

    await user.click(await screen.findByRole('button', { name: /Widget/ }))
    await user.click(screen.getByRole('button', { name: /Replace/ }))
    await user.type(screen.getByPlaceholderText('Search or scan barcode…'), 'SUB{Enter}')
    await user.click(await screen.findByText('Other widget'))
    await user.type(screen.getByLabelText('Covers (of original)'), '2')
    await user.click(screen.getByRole('button', { name: 'Confirm replace' }))

    await waitFor(() => expect(api.substitutePackLine).toHaveBeenCalledWith(TOKEN, 'pack-1', {
      line_item_id: 'li-1', lines: [{ sku_id: 'sku-sub', substitute_quantity: 1, original_quantity_covered: 2, return_to_stock: false }],
    }))
  })

  it('a line can be refunded from its sheet', async () => {
    vi.mocked(api.refundPackLine).mockResolvedValue({ scan: { id: 'r', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'refund', quantity: 2, scanned_at: '', scanned_by: '' } })
    const user = userEvent.setup()
    renderScreen()

    await user.click(await screen.findByRole('button', { name: /Widget/ }))
    await user.click(screen.getByRole('button', { name: /Refund/ }))
    await user.click(screen.getByRole('button', { name: 'Confirm refund' }))

    await waitFor(() => expect(api.refundPackLine).toHaveBeenCalledWith(TOKEN, 'pack-1', { line_item_id: 'li-1', quantity: 2, reason: undefined }))
  })

  it('an extra item can be added as a freebie', async () => {
    vi.mocked(api.scanPackItem).mockResolvedValue({ scan: { id: 'x', session_id: 'pack-1', line_item_id: null, sku_id: 'sku-sub', action_type: 'freebie', quantity: 1, scanned_at: '', scanned_by: '' } })
    const user = userEvent.setup()
    renderScreen()

    await user.click(await screen.findByRole('button', { name: /Add freebie or new item/ }))
    await user.type(screen.getByPlaceholderText('Search or scan barcode…'), 'SUB{Enter}')
    await user.click(await screen.findByText('Other widget'))
    await user.click(screen.getByRole('button', { name: 'Add to order' }))

    await waitFor(() => expect(api.scanPackItem).toHaveBeenCalledWith(TOKEN, 'pack-1', {
      sku_id: 'sku-sub', action_type: 'freebie', quantity: 1, price_minor_units: undefined, reason: 'Other widget',
    }))
  })

  it('lists staged extra items with a way to remove them', async () => {
    vi.mocked(api.getPackingSession).mockResolvedValue(detail({ scans: [
      { id: 'x1', session_id: 'pack-1', line_item_id: null, sku_id: 'sku-9', action_type: 'new_item', quantity: 1, reason: 'Jam jar', scanned_at: '', scanned_by: '' },
    ] }))
    vi.mocked(api.deletePackScan).mockResolvedValue(null)
    const user = userEvent.setup()
    renderScreen()

    const extras = await screen.findByRole('list', { name: 'Extra items' })
    expect(within(extras).getByText('New item · Jam jar · qty 1')).toBeInTheDocument()
    await user.click(within(extras).getByRole('button', { name: 'Remove Jam jar' }))

    expect(api.deletePackScan).toHaveBeenCalledWith(TOKEN, 'pack-1', 'x1')
  })

  it('finishing with a stock shortfall asks per item and finishes with the decision', async () => {
    vi.mocked(api.completePacking)
      .mockResolvedValueOnce({ completed: false, stock_shortfalls: [{ scan_id: 'x1', sku_id: 'sku-9', name: 'Jam jar', needed: 2, short: 1 }] })
      .mockResolvedValueOnce({ completed: true, session: detail().session, missing_items: [] })
    const user = userEvent.setup()
    renderScreen()

    await user.click(await screen.findByRole('button', { name: /Finish/ }))
    await user.click(await screen.findByRole('button', { name: /Confirm/ }))
    expect(await screen.findByText('Not enough stock')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Skip it' }))
    const confirms = screen.getAllByRole('button', { name: 'Confirm & complete' })
    await user.click(confirms[confirms.length - 1])

    await waitFor(() => expect(api.completePacking).toHaveBeenLastCalledWith(TOKEN, 'pack-1', { x1: 'without_item' }))
  })
})

describe('PackingScreen locks and labels', () => {
  it('shows the sold expiry date WooCommerce reported for a line', async () => {
    vi.mocked(api.getPackingSession).mockResolvedValue(detail({
      line_items: [{ ...detail().line_items[0], sold_expiry_date: '2027-03-09' }],
    }))
    renderScreen()

    expect(await screen.findByText('Sold exp. 09/03/2027')).toBeInTheDocument()
  })

  it('shows no sold expiry for a line without one', async () => {
    renderScreen()

    await screen.findByText('Widget')
    expect(screen.queryByText(/Sold exp\./)).not.toBeInTheDocument()
  })

  it('another user\'s session is view only', async () => {
    vi.mocked(api.getPackingSession).mockResolvedValue(detail({ session: { ...detail().session, locked_by: 'u-9', locked_at: new Date().toISOString() } }))
    renderScreen()

    expect(await screen.findByRole('status')).toHaveTextContent('is packing this order')
    expect(screen.getByPlaceholderText('Scan or type barcode…')).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Finish' })).toBeDisabled()
  })

  it('leaving a session you hold releases it', async () => {
    vi.mocked(api.getPackingSession).mockResolvedValue(detail({ session: { ...detail().session, locked_by: 'me', locked_at: new Date().toISOString() } }))
    const { unmount } = render(<PackingScreen token={TOKEN} sessionId="pack-1" order={order} onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Widget')

    unmount()

    expect(api.releasePackingSessionLock).toHaveBeenCalledWith(TOKEN, 'pack-1')
  })

  it('a label can be created before finishing', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([{ id: 'c1', provider: 'myparcel' }]))
    vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: null } as unknown as Awaited<ReturnType<typeof shippingApi.getShippingSettings>>)
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([])
    const user = userEvent.setup()
    renderScreen()

    await user.click(await screen.findByRole('button', { name: 'Create label' }))

    expect(api.touchPackingSession).toHaveBeenCalledWith(TOKEN, 'pack-1')
    expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()
  })
  it('warns and blocks the label while the connection misses a sender address', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([{ id: 'c1', provider: 'postnl', missing: ['sender_address'] }]))
    vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: null } as unknown as Awaited<ReturnType<typeof shippingApi.getShippingSettings>>)
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([])
    const user = userEvent.setup()
    render(<MemoryRouter><PackingScreen token={TOKEN} sessionId="pack-1" order={order} onExit={vi.fn()} onAuthError={vi.fn()} /></MemoryRouter>)

    await user.click(await screen.findByRole('button', { name: 'Create label' }))

    expect(await screen.findByText('PostNL needs a sender address before it can create labels.')).toBeInTheDocument()
    expect(screen.getByRole('link', { name: 'Set up the shipping addresses' })).toHaveAttribute('href', '/integrations#shipping-addresses')
    expect(screen.getByRole('button', { name: 'Review label' })).toBeDisabled()
  })
  it('the review screen offers the label before finishing', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([{ id: 'c1', provider: 'myparcel' }]))
    vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: null } as unknown as Awaited<ReturnType<typeof shippingApi.getShippingSettings>>)
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([])
    const user = userEvent.setup()
    renderScreen()

    await user.click(await screen.findByRole('button', { name: 'Finish' }))
    await screen.findByText('Review before finishing')
    await user.click(screen.getByRole('button', { name: 'Create shipping label' }))

    expect(await screen.findByText('Confirm shipping address')).toBeInTheDocument()
    expect(api.completePacking).not.toHaveBeenCalled()
  })
})
