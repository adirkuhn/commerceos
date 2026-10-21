// @vitest-environment jsdom
// Characterization tests for Packing paths the main suite doesn't reach:
// removing a scan from the session log, and the "label already exists"
// branch of creating a label.
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { MemoryRouter } from 'react-router-dom'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import Packing from '../Packing'
import type { SalesOrder } from '../../oms/types'
import type { PackingSession, PackingSessionDetail } from '../../scanner/types'
import type { Shipment } from '../../shipping/types'

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: { user_id: 'me', tenant_id: 't', role: 'customer_user', exp: 0, iat: 0 }, login: vi.fn(), logout: vi.fn(), isAuthenticated: true }),
}))
vi.mock('../../oms/api', () => ({ listPackingQueue: vi.fn(), getSalesOrder: vi.fn(), getPickList: vi.fn() }))
vi.mock('../../team/api', () => ({ listMembers: vi.fn() }))
vi.mock('../../scanner/api', () => ({
  startOrResumePacking: vi.fn(), getPackingSession: vi.fn(), deletePackScan: vi.fn(),
  touchPackingSession: vi.fn(), releasePackingSessionLock: vi.fn(),
}))
vi.mock('../../pms/api', () => ({ listProductMedia: vi.fn() }))
vi.mock('../../shipping/api', () => ({
  listShippingConnections: vi.fn(),
  getShippingSettings: vi.fn(), listShipmentsForOrder: vi.fn(), lookupAddress: vi.fn(), getShipmentLabel: vi.fn(),
}))

import * as omsApi from '../../oms/api'
import * as teamApi from '../../team/api'
import * as packingApi from '../../scanner/api'
import * as pmsApi from '../../pms/api'
import * as shippingApi from '../../shipping/api'
import { shippingConnectionsOf } from '../../test/shippingConnections'

const order: SalesOrder = {
  id: 'so-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1', payment_status: 'paid',
  number: '1001', order_key: '', status: 'processing', currency_code: 'USD',
  discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
  total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
  fulfilled: true, packed: false, billing: { first_name: 'Jane', last_name: 'Doe', email: 'jane@example.com' },
}
const session: PackingSession = { id: 'pack-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '', completed_by: '' }
const scan = { id: 'scan-1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pack' as const, quantity: 1, scanned_at: '', scanned_by: '' }
function detail(scans: PackingSessionDetail['scans'] = []): PackingSessionDetail {
  return {
    session,
    line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000 }],
    scans,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(packingApi.releasePackingSessionLock).mockResolvedValue(null)
  vi.mocked(packingApi.touchPackingSession).mockResolvedValue(null)
  vi.mocked(omsApi.listPackingQueue).mockResolvedValue({ entries: [{ sales_order: order, lock: null, in_packing: false }], total: 1 })
  vi.mocked(teamApi.listMembers).mockResolvedValue([])
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([]))
  vi.mocked(packingApi.startOrResumePacking).mockResolvedValue(session)
})

describe('Packing (characterization)', () => {
  it('removing a scan from the session log deletes it and reloads', async () => {
    vi.mocked(packingApi.getPackingSession).mockResolvedValueOnce(detail([scan])).mockResolvedValueOnce(detail())
    vi.mocked(packingApi.deletePackScan).mockResolvedValue(null)

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    expect(await screen.findByText('1 / 2 items')).toBeInTheDocument()
    expect(screen.getByText('This session')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Remove scan' }))

    await waitFor(() => expect(packingApi.deletePackScan).toHaveBeenCalledWith('test-token', 'pack-1', 'scan-1'))
    expect(await screen.findByText('0 / 2 items')).toBeInTheDocument()
    expect(screen.queryByText('This session')).not.toBeInTheDocument()
  })

  it('a failed scan removal shows the error inline', async () => {
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail([scan]))
    vi.mocked(packingApi.deletePackScan).mockRejectedValue(new Error('cannot remove'))

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await user.click(await screen.findByRole('button', { name: 'Remove scan' }))

    expect(await screen.findByText('cannot remove')).toBeInTheDocument()
  })

  it('an order that already has a label asks before making another, and cancel closes it', async () => {
    vi.mocked(shippingApi.listShippingConnections).mockResolvedValue(shippingConnectionsOf([{ id: 'c1', provider: 'myparcel' }]))
    vi.mocked(shippingApi.getShippingSettings).mockResolvedValue({ settings: null } as unknown as Awaited<ReturnType<typeof shippingApi.getShippingSettings>>)
    vi.mocked(shippingApi.listShipmentsForOrder).mockResolvedValue([{ id: 'sh-1', barcode: '3SABC', cancelled_at: null } as Shipment])
    vi.mocked(packingApi.getPackingSession).mockResolvedValue(detail())

    const user = userEvent.setup()
    render(<MemoryRouter><Packing /></MemoryRouter>)
    await user.click(await screen.findByText('#1001'))
    await screen.findByText('Widget')
    await user.click(screen.getByRole('button', { name: 'Create label' }))

    expect(await screen.findByText('Label already exists')).toBeInTheDocument()
    expect(screen.queryByText('Confirm shipping address')).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    await waitFor(() => expect(screen.queryByText('Label already exists')).not.toBeInTheDocument())
  })
})
