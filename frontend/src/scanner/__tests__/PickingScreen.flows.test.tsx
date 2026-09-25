// @vitest-environment jsdom
// Characterization of PickingScreen's scan routing (resolve → pick, over-
// scan, wrong order, search fallback, weight barcodes), pausing, the line
// sheet and admin takeover -- the flows PickingScreen.test.tsx doesn't reach.
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { PickingScreen } from '../PickingScreen'
import type { SalesOrder } from '../../oms/types'
import type { PickingSessionDetail, PickScan } from '../types'
import type { SKU } from '../../pms/types'
import type { SKUCandidate } from '../../pms/searchCandidates'

vi.mock('../api', () => ({
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
  startOrResumePacking: vi.fn(),
}))
vi.mock('../../pms/searchCandidates', () => ({ searchSKUCandidates: vi.fn() }))
vi.mock('../../team/api', () => ({ listMembers: vi.fn().mockResolvedValue([]) }))
vi.mock('../../pms/api', () => ({ listProductMedia: vi.fn() }))
vi.mock('../errorSound', () => ({ playErrorSound: vi.fn() }))

import * as api from '../api'
import * as pmsApi from '../../pms/api'
import { searchSKUCandidates } from '../../pms/searchCandidates'
import { playErrorSound } from '../errorSound'

function tokenFor(role: string) {
  return `x.${btoa(JSON.stringify({ user_id: 'me', role }))}.y`
}
const TOKEN = tokenFor('customer_user')

const order = {
  id: 'so-1', number: '1001', external_order_id: 'ext-1', currency_code: 'EUR', total_minor_units: 5000,
  billing: { first_name: 'Jane', last_name: 'Doe' },
} as SalesOrder

const pickScan = (quantity: number): PickScan => ({ id: 'scan-1', session_id: 'ps-1', line_item_id: 'li-1', sku_id: 'sku-1', action_type: 'pick', quantity, scanned_at: '', scanned_by: '' })

function detail(overrides: Partial<PickingSessionDetail> = {}): PickingSessionDetail {
  return {
    session: { id: 'ps-1', sales_order_id: 'so-1', status: 'in_progress', started_at: '', completed_by: '' },
    line_items: [{ id: 'li-1', sku_id: 'sku-1', name: 'Widget', sku_code: 'WIDGET-1', barcode: '111', quantity: 2, price_minor_units: 1000, total_minor_units: 2000 }],
    scans: [],
    ...overrides,
  }
}

const candidate = (pricingMode: SKU['pricing_mode'] = 'fixed'): SKUCandidate => ({
  sku: { id: 'sku-1', sku_code: 'WIDGET-1', pricing_mode: pricingMode } as SKU, productId: 'p', productName: 'Widget product',
})

function renderScreen(props: { token?: string; onExit?: () => void } = {}) {
  render(<PickingScreen token={props.token ?? TOKEN} sessionId="ps-1" order={order} onExit={props.onExit ?? vi.fn()} onAuthError={vi.fn()} onContinueToPacking={vi.fn()} />)
}

async function scan(code: string) {
  const user = userEvent.setup()
  await screen.findByText('Widget')
  await user.type(screen.getByPlaceholderText('Scan or type barcode…'), code)
  await user.click(screen.getByRole('button', { name: 'Scan' }))
  return user
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(pmsApi.listProductMedia).mockResolvedValue([])
  vi.mocked(api.getPickingSession).mockResolvedValue(detail())
  vi.mocked(api.releasePickingSessionLock).mockResolvedValue(null)
  vi.mocked(api.scanPickItem).mockResolvedValue({ scan: pickScan(1) })
})

describe('PickingScreen scan routing', () => {
  it('a code for one of the order\'s lines picks one unit', async () => {
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget' })
    renderScreen()
    await scan('111')

    await waitFor(() => expect(api.scanPickItem).toHaveBeenCalledWith(TOKEN, 'ps-1', { sku_id: 'sku-1', action_type: 'pick', quantity: 1 }))
    expect(api.resolvePickScanCode).toHaveBeenCalledWith(TOKEN, 'ps-1', '111')
  })

  it('a weight barcode picks the package weight in kg', async () => {
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget', weight_grams: 1250 })
    renderScreen()
    await scan('2100000012503')

    await waitFor(() => expect(api.scanPickItem).toHaveBeenCalledWith(TOKEN, 'ps-1', { sku_id: 'sku-1', action_type: 'pick', quantity: 1.25 }))
  })

  it('over-scanning a fully picked line blocks behind an error', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail({ scans: [pickScan(2)] }))
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-1', product_name: 'Widget' })
    renderScreen()
    const user = await scan('111')

    expect(await screen.findByRole('dialog', { name: 'Scan error' })).toHaveTextContent('Already fully picked')
    expect(playErrorSound).toHaveBeenCalled()
    expect(api.scanPickItem).not.toHaveBeenCalled()

    await user.click(screen.getByRole('button', { name: 'OK' }))
    expect(screen.queryByRole('dialog', { name: 'Scan error' })).not.toBeInTheDocument()
  })

  it('an item the server rejects for this order blocks behind an error', async () => {
    vi.mocked(api.resolvePickScanCode).mockResolvedValue({ sku_id: 'sku-other', product_name: 'Other' })
    vi.mocked(api.scanPickItem).mockRejectedValue(new Error('not on order'))
    renderScreen()
    await scan('222')

    expect(await screen.findByRole('dialog', { name: 'Scan error' })).toHaveTextContent('Item not on this order')
    expect(api.scanPickItem).toHaveBeenCalledWith(TOKEN, 'ps-1', { sku_id: 'sku-other', action_type: 'pick', quantity: 1 })
    expect(screen.queryByText('not on order')).not.toBeInTheDocument()
  })

  it('an unrecognized code falls back to a search, and a match picks it', async () => {
    vi.mocked(api.resolvePickScanCode).mockRejectedValue(new Error('not found'))
    vi.mocked(searchSKUCandidates).mockResolvedValue([candidate()])
    renderScreen()
    const user = await scan('widg')

    expect(await screen.findByText('Multiple matches -- pick one:')).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: /Widget product/ }))

    await waitFor(() => expect(api.scanPickItem).toHaveBeenCalledWith(TOKEN, 'ps-1', { sku_id: 'sku-1', action_type: 'pick', quantity: 1 }))
    expect(screen.queryByText('Multiple matches -- pick one:')).not.toBeInTheDocument()
  })

  it('a by-weight search match is refused -- it needs its scale barcode', async () => {
    vi.mocked(api.resolvePickScanCode).mockRejectedValue(new Error('not found'))
    vi.mocked(searchSKUCandidates).mockResolvedValue([candidate('per_weight')])
    renderScreen()
    const user = await scan('widg')

    await user.click(await screen.findByRole('button', { name: /Widget product/ }))

    expect(await screen.findByRole('dialog', { name: 'Scan error' })).toHaveTextContent('Priced by weight')
    expect(api.scanPickItem).not.toHaveBeenCalled()
  })

  it('a code nothing matches is a blocking error', async () => {
    vi.mocked(api.resolvePickScanCode).mockRejectedValue(new Error('not found'))
    vi.mocked(searchSKUCandidates).mockResolvedValue([])
    renderScreen()
    await scan('zzz')

    expect(await screen.findByRole('dialog', { name: 'Scan error' })).toHaveTextContent('Barcode not recognized')
    expect(screen.getByPlaceholderText('Scan or type barcode…')).toBeDisabled()
  })
})

describe('PickingScreen session actions', () => {
  it('back pauses the session, then exits', async () => {
    vi.mocked(api.pausePicking).mockResolvedValue(detail().session)
    const onExit = vi.fn()
    const user = userEvent.setup()
    renderScreen({ onExit })
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Pause and go back' }))

    expect(api.pausePicking).toHaveBeenCalledWith(TOKEN, 'ps-1')
    await waitFor(() => expect(onExit).toHaveBeenCalled())
  })

  it('a failed pause stays and says why', async () => {
    vi.mocked(api.pausePicking).mockRejectedValue(new Error('Server down'))
    const onExit = vi.fn()
    const user = userEvent.setup()
    renderScreen({ onExit })
    await screen.findByText('Widget')

    await user.click(screen.getByRole('button', { name: 'Pause and go back' }))

    expect(await screen.findByText('Server down')).toBeInTheDocument()
    expect(onExit).not.toHaveBeenCalled()
  })

  it('tapping a line opens its sheet and picks the chosen quantity', async () => {
    const user = userEvent.setup()
    renderScreen()
    await user.click(await screen.findByRole('button', { name: /Widget/ }))

    await user.click(await screen.findByRole('button', { name: 'Confirm pick' }))

    await waitFor(() => expect(api.scanPickItem).toHaveBeenCalledWith(TOKEN, 'ps-1', { sku_id: 'sku-1', action_type: 'pick', quantity: 2, reason: undefined }))
  })

  it('lists items added during picking', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail({ scans: [
      { id: 'f1', session_id: 'ps-1', line_item_id: null, sku_id: 'sku-9', action_type: 'freebie', quantity: 1, reason: 'Sticker', scanned_at: '', scanned_by: '' },
      { id: 'n1', session_id: 'ps-1', line_item_id: null, sku_id: 'sku-8', action_type: 'new_item', quantity: 3, scanned_at: '', scanned_by: '' },
    ] }))
    renderScreen()

    expect(await screen.findByText('Added during picking:')).toBeInTheDocument()
    expect(screen.getByText('Sticker')).toBeInTheDocument()
    expect(screen.getByText('Freebie')).toBeInTheDocument()
    expect(screen.getAllByText('New item')).toHaveLength(2)
    expect(screen.getByText('×3')).toBeInTheDocument()
  })

  it('flags lines not yet linked to a product', async () => {
    vi.mocked(api.getPickingSession).mockResolvedValue(detail({
      line_items: [{ id: 'li-1', sku_id: null, name: 'Mystery', sku_code: '', barcode: '', quantity: 1, price_minor_units: 0, total_minor_units: 0 }],
    }))
    renderScreen()
    expect(await screen.findByText('Not linked to a product yet')).toBeInTheDocument()
  })

  it('an admin can take over another user\'s session', async () => {
    const admin = tokenFor('customer_owner')
    vi.mocked(api.getPickingSession).mockResolvedValue(detail({ session: { ...detail().session, locked_by: 'u-9', locked_at: new Date().toISOString() } }))
    vi.mocked(api.takeOverPickingSession).mockResolvedValue(detail().session)
    const user = userEvent.setup()
    renderScreen({ token: admin })

    expect(await screen.findByRole('status')).toHaveTextContent('is picking this order')
    await user.click(screen.getByRole('button', { name: 'Take over' }))

    expect(api.takeOverPickingSession).toHaveBeenCalledWith(admin, 'ps-1')
  })
})
