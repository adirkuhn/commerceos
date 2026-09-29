// @vitest-environment jsdom
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { renderAt } from '../../test/renderAt'
import { ScanPicking } from '../ScanPicking'
import type { SalesOrder } from '../../oms/types'
import type { PickScan, PickingSessionDetail } from '../../scanner/types'

// Characterization tests for the parts of the scan-picking screen the page
// tests (src/pages/__tests__/Picking.test.tsx) don't exercise: the pick
// sheet's quantity, removing a staged scan, backing out of dialogs.

vi.mock('../../auth/AuthContext', () => ({
  useAuth: () => ({ token: 'test-token', user: { user_id: 'me', tenant_id: 't', role: 'customer_user', exp: 0, iat: 0 } }),
}))
vi.mock('../../team/api', () => ({ listMembers: vi.fn().mockResolvedValue([]) }))
vi.mock('../../oms/api', () => ({ listPickingQueue: vi.fn(), getSalesOrder: vi.fn() }))
vi.mock('../../pms/api', () => ({ listProductMedia: vi.fn().mockResolvedValue([]) }))
vi.mock('../../scanner/errorSound', () => ({ playErrorSound: vi.fn() }))
vi.mock('../../scanner/api', () => ({
  startOrResumePicking: vi.fn(),
  getPickingSession: vi.fn(),
  scanPickItem: vi.fn(),
  resolvePickScanCode: vi.fn(),
  deletePickScan: vi.fn(),
  pausePicking: vi.fn(),
  completePicking: vi.fn(),
  touchPickingSession: vi.fn(),
  resetPickingSession: vi.fn(),
  releasePickingSessionLock: vi.fn(),
  takeOverPickingSession: vi.fn(),
}))

import * as omsApi from '../../oms/api'
import * as pickingApi from '../../scanner/api'

const order: SalesOrder = {
  id: 'so-1', tenant_id: 'tenant-1', source: 'woocommerce', external_order_id: 'ext-1',
  number: '1001', order_key: '', status: 'processing', currency_code: 'USD',
  discount_total_minor_units: 0, shipping_total_minor_units: 0, tax_total_minor_units: 0,
  total_minor_units: 5000, occurred_at: '2026-08-10T12:00:00Z', created_at: '2026-08-10T12:00:00Z',
  fulfilled: false, billing: { first_name: 'Jane', last_name: 'Doe' },
}

const scan: PickScan = { id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity: 1, scanned_at: '2026-08-22T10:01:00Z', scanned_by: 'me' }

function detail(scans: PickScan[] = []): PickingSessionDetail {
  return {
    session: { id: 'ps-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '2026-08-22T10:00:00Z', completed_by: '' },
    line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 5, price_minor_units: 1000, total_minor_units: 5000 }],
    scans,
  }
}

async function openOrder() {
  const user = userEvent.setup()
  renderAt(<ScanPicking />, '/picking')
  await user.click(await screen.findByText('#1001'))
  await screen.findByText('Widget')
  return user
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.listPickingQueue).mockResolvedValue({ entries: [{ sales_order: order, lock: null, in_packing: false }], total: 1 })
  vi.mocked(omsApi.getSalesOrder).mockResolvedValue({ sales_order: order, line_items: [] })
  vi.mocked(pickingApi.startOrResumePicking).mockResolvedValue(detail().session)
  vi.mocked(pickingApi.releasePickingSessionLock).mockResolvedValue(null)
})

describe('ScanPicking', () => {
  it('marks the active queue entry in progress', async () => {
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    await openOrder()
    const entry = screen.getByRole('button', { pressed: true })
    expect(within(entry).getByText('In progress')).toBeInTheDocument()
  })

  it('tapping a line opens the pick sheet prefilled with the remaining quantity and records the chosen one', async () => {
    vi.mocked(pickingApi.getPickingSession)
      .mockResolvedValueOnce(detail([scan]))
      .mockResolvedValueOnce(detail([scan, { ...scan, id: 'scan-2', quantity: 3 }]))
    vi.mocked(pickingApi.scanPickItem).mockResolvedValue({ scan })
    const user = await openOrder()

    await user.click(screen.getByText('Widget'))
    const qty = screen.getByLabelText('Quantity')
    expect(qty).toHaveValue(4)
    expect(screen.getByPlaceholderText('Scan barcode or type and press Enter...')).toBeDisabled()

    fireEvent.change(qty, { target: { value: '3' } })
    await user.click(screen.getByRole('button', { name: 'Confirm pick' }))

    expect(pickingApi.scanPickItem).toHaveBeenCalledWith('test-token', 'ps-1', { sku_id: 'sku-1', action_type: 'pick', quantity: 3 })
    expect(await screen.findByText('4 / 5 items')).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Confirm pick' })).not.toBeInTheDocument()
  })

  it('closing the pick sheet records nothing', async () => {
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    const user = await openOrder()

    await user.click(screen.getByText('Widget'))
    await user.click(screen.getByRole('button', { name: 'Close' }))

    expect(screen.queryByRole('button', { name: 'Confirm pick' })).not.toBeInTheDocument()
    expect(pickingApi.scanPickItem).not.toHaveBeenCalled()
  })

  it('removes a staged scan and reloads the session', async () => {
    vi.mocked(pickingApi.getPickingSession)
      .mockResolvedValueOnce(detail([scan]))
      .mockResolvedValueOnce(detail())
    vi.mocked(pickingApi.deletePickScan).mockResolvedValue(null)
    const user = await openOrder()

    expect(screen.getByText('This session')).toBeInTheDocument()
    expect(screen.getByText('Picked · qty 1')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Remove scan' }))

    expect(pickingApi.deletePickScan).toHaveBeenCalledWith('test-token', 'ps-1', 'scan-1')
    await waitFor(() => expect(screen.queryByText('This session')).not.toBeInTheDocument())
    expect(screen.getByText('0 / 5 items')).toBeInTheDocument()
  })

  it('cancelling start over keeps the session untouched', async () => {
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    const user = await openOrder()

    await user.click(screen.getByRole('button', { name: 'Start over' }))
    await user.click(screen.getByRole('button', { name: 'Cancel' }))

    expect(screen.queryByText('Start over?')).not.toBeInTheDocument()
    expect(pickingApi.resetPickingSession).not.toHaveBeenCalled()
  })

  it('the review step lists missing lines and can go back to scanning', async () => {
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail([scan]))
    const user = await openOrder()

    await user.click(screen.getByRole('button', { name: /Complete/ }))
    expect(screen.getByText('Review before completing')).toBeInTheDocument()
    expect(screen.getByText("1 line isn't fully picked yet:")).toBeInTheDocument()
    expect(screen.getByText('expected 5 · picked 1')).toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Back to scanning' }))
    expect(screen.queryByText('Review before completing')).not.toBeInTheDocument()
    expect(screen.getByText('1 / 5 items')).toBeInTheDocument()
  })

  it('shows a failed pick inline and keeps the sheet open', async () => {
    vi.mocked(pickingApi.getPickingSession).mockResolvedValue(detail())
    vi.mocked(pickingApi.scanPickItem).mockRejectedValue(new Error('boom'))
    const user = await openOrder()

    await user.click(screen.getByText('Widget'))
    await user.click(screen.getByRole('button', { name: 'Confirm pick' }))

    expect(await screen.findByText('boom')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Confirm pick' })).toBeInTheDocument()
  })
})
