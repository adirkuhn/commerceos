// @vitest-environment jsdom
// Characterization of PackingScreen flows the order-change/lock/label tests
// don't reach: scanning, the blocking scan-error dialog, start over, the done
// screen and admin takeover.
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PackingScreen } from '../PackingScreen'
import type { SalesOrder } from '../../oms/types'
import type { PackingSessionDetail } from '../types'

vi.mock('../api', () => ({
  getPackingSession: vi.fn(),
  scanPackItem: vi.fn(),
  deletePackScan: vi.fn(),
  completePacking: vi.fn(),
  touchPackingSession: vi.fn(),
  resetPackingSession: vi.fn(),
  releasePackingSessionLock: vi.fn(),
  takeOverPackingSession: vi.fn(),
}))
vi.mock('../../team/api', () => ({ listMembers: vi.fn().mockResolvedValue([]) }))
vi.mock('../../pms/api', () => ({ listProductMedia: vi.fn() }))
vi.mock('../../integrations/hooks/useConnections', () => ({ useConnections: vi.fn() }))
vi.mock('../../shipping/api', () => ({
  createShipment: vi.fn(), getShippingSettings: vi.fn(), getShipmentLabel: vi.fn(), listShipmentsForOrder: vi.fn(),
}))
vi.mock('../errorSound', () => ({ playErrorSound: vi.fn() }))

import * as api from '../api'
import * as pmsApi from '../../pms/api'
import { useConnections } from '../../integrations/hooks/useConnections'
import { playErrorSound } from '../errorSound'

function tokenFor(role: string) {
  return `x.${btoa(JSON.stringify({ user_id: 'me', role }))}.y`
}
const TOKEN = tokenFor('customer_user')

const order = {
  id: 'so-1', number: '1001', external_order_id: 'ext-1', currency_code: 'EUR', total_minor_units: 5000,
  billing: { first_name: 'Jane', last_name: 'Doe' },
  shipping: { first_name: 'Jane', last_name: 'Doe', address_1: 'Damrak 1', city: 'Amsterdam', postcode: '1012AB', country: 'NL' },
} as SalesOrder

function detail(overrides: Partial<PackingSessionDetail> = {}): PackingSessionDetail {
  return {
    session: { id: 'pack-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '', completed_by: '' },
    line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000 }],
    scans: [],
    ...overrides,
  }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(useConnections).mockReturnValue({ connections: [], loading: false, error: null, refresh: vi.fn(), disconnect: vi.fn() })
  vi.mocked(api.getPackingSession).mockResolvedValue(detail())
  vi.mocked(api.releasePackingSessionLock).mockResolvedValue(null)
  vi.mocked(api.touchPackingSession).mockResolvedValue(null)
})

describe('PackingScreen scanning', () => {
  it('a scanned code is packed one unit at a time and the session reloads', async () => {
    vi.mocked(api.scanPackItem).mockResolvedValue({ scan: { id: 's1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pack', quantity: 1, scanned_at: '', scanned_by: '' } })
    const user = userEvent.setup()
    render(<PackingScreen token={TOKEN} sessionId="pack-1" order={order} onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Widget')

    await user.type(screen.getByPlaceholderText('Scan or type barcode…'), ' 111 ')
    await user.click(screen.getByRole('button', { name: 'Scan' }))

    await waitFor(() => expect(api.scanPackItem).toHaveBeenCalledWith(TOKEN, 'pack-1', { code: '111', quantity: 1 }))
    await waitFor(() => expect(api.getPackingSession).toHaveBeenCalledTimes(2))
    expect(screen.getByPlaceholderText('Scan or type barcode…')).toHaveValue('')
  })

  it('a failed scan blocks scanning behind an error dialog until acknowledged', async () => {
    vi.mocked(api.scanPackItem).mockRejectedValue(new Error('SKU not on this order'))
    const user = userEvent.setup()
    render(<PackingScreen token={TOKEN} sessionId="pack-1" order={order} onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Widget')

    await user.type(screen.getByPlaceholderText('Scan or type barcode…'), '999')
    await user.click(screen.getByRole('button', { name: 'Scan' }))

    const dialog = await screen.findByRole('dialog', { name: 'Scan error' })
    expect(dialog).toHaveTextContent('Item not on this order')
    expect(playErrorSound).toHaveBeenCalled()
    expect(screen.getByPlaceholderText('Scan or type barcode…')).toBeDisabled()
    expect(screen.getByText('SKU not on this order')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'OK' }))
    expect(screen.queryByRole('dialog', { name: 'Scan error' })).not.toBeInTheDocument()
    expect(screen.getByPlaceholderText('Scan or type barcode…')).not.toBeDisabled()
  })
})

describe('PackingScreen session actions', () => {
  it('start over asks first, then resets and reloads the session', async () => {
    vi.mocked(api.resetPackingSession).mockResolvedValue(detail().session)
    const user = userEvent.setup()
    render(<PackingScreen token={TOKEN} sessionId="pack-1" order={order} onExit={vi.fn()} onAuthError={vi.fn()} />)
    await screen.findByText('Widget')
    expect(screen.getByText('Damrak 1, 1012AB Amsterdam, NL', { exact: false })).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: /start over/i }))
    expect(await screen.findByText(/discards every item packed so far/)).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Cancel' }))
    expect(api.resetPackingSession).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: /start over/i }))
    const buttons = screen.getAllByRole('button', { name: 'Start over' })
    await user.click(buttons[buttons.length - 1])

    await waitFor(() => expect(api.resetPackingSession).toHaveBeenCalledWith(TOKEN, 'pack-1'))
    await waitFor(() => expect(screen.queryByText('Start over?')).not.toBeInTheDocument())
  })

  it('finishing shows the review, then the packed summary with missing lines', async () => {
    vi.mocked(api.completePacking).mockResolvedValue({
      completed: true, session: detail().session,
      missing_items: [{ line_item_id: 'li-1', name: 'Widget', expected_qty: 2, packed_qty: 0 }],
    })
    const onExit = vi.fn()
    const user = userEvent.setup()
    render(<PackingScreen token={TOKEN} sessionId="pack-1" order={order} onExit={onExit} onAuthError={vi.fn()} />)
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Finish' }))
    expect(await screen.findByText("1 line isn't fully packed yet:")).toBeInTheDocument()
    expect(screen.getByText('expected 2 · packed 0')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Back, scan more' }))
    expect(await screen.findByText('Packing')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Finish' }))
    await user.click(await screen.findByRole('button', { name: 'Confirm & complete' }))

    expect(await screen.findByText('Order packed')).toBeInTheDocument()
    expect(screen.getByText("1 line wasn't fully packed:")).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Done' }))
    expect(onExit).toHaveBeenCalled()
  })

  it('a fully packed order says so on the review and done screens', async () => {
    vi.mocked(api.getPackingSession).mockResolvedValue(detail({
      scans: [{ id: 's1', session_id: 'pack-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pack', quantity: 2, scanned_at: '', scanned_by: '' }],
    }))
    vi.mocked(api.completePacking).mockResolvedValue({ completed: true, session: detail().session, missing_items: [] })
    const user = userEvent.setup()
    render(<PackingScreen token={TOKEN} sessionId="pack-1" order={order} onExit={vi.fn()} onAuthError={vi.fn()} />)
    expect(await screen.findByText('2 / 2')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Finish' }))
    expect(await screen.findByText('Every line is fully packed.')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Confirm & complete' }))
    expect(await screen.findByText('Every line was fully packed.')).toBeInTheDocument()
  })

  it('an admin can take over another user\'s session', async () => {
    const admin = tokenFor('customer_admin')
    vi.mocked(api.getPackingSession).mockResolvedValue(detail({ session: { ...detail().session, locked_by: 'u-9', locked_at: new Date().toISOString() } }))
    vi.mocked(api.takeOverPackingSession).mockResolvedValue(detail().session)
    const user = userEvent.setup()
    render(<PackingScreen token={admin} sessionId="pack-1" order={order} onExit={vi.fn()} onAuthError={vi.fn()} />)

    await user.click(await screen.findByRole('button', { name: 'Take over' }))

    expect(api.takeOverPackingSession).toHaveBeenCalledWith(admin, 'pack-1')
  })

  it('an auth failure while loading hands off to onAuthError', async () => {
    const onAuthError = vi.fn()
    vi.mocked(api.getPackingSession).mockRejectedValue(Object.assign(new Error('unauthorized'), { status: 401 }))
    render(<PackingScreen token={TOKEN} sessionId="pack-1" order={order} onExit={vi.fn()} onAuthError={onAuthError} />)

    await waitFor(() => expect(onAuthError).toHaveBeenCalled())
  })
})
