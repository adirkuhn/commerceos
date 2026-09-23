// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SalesOrderPickerScreen } from '../SalesOrderPickerScreen'
import type { SalesOrder } from '../../oms/types'

vi.mock('../../oms/api', () => ({ listSalesOrders: vi.fn() }))
vi.mock('../api', () => ({ startOrResumePicking: vi.fn() }))
vi.mock('../../settings/api', () => ({ getTenantSettings: vi.fn() }))
import * as omsApi from '../../oms/api'
import * as api from '../api'
import * as settingsApi from '../../settings/api'

const order = { id: 'so-1', number: '1001', external_order_id: 'e', currency_code: 'EUR', total_minor_units: 100, billing: { first_name: 'Ann', last_name: 'Lee' } } as SalesOrder

function mode(picking_mode: 'scan' | 'list') {
  vi.mocked(settingsApi.getTenantSettings).mockResolvedValue({ picking_mode } as Awaited<ReturnType<typeof settingsApi.getTenantSettings>>)
}

function renderScreen() {
  const onPickingStarted = vi.fn()
  const onOrderChosenForList = vi.fn()
  render(<SalesOrderPickerScreen token="tok" onPickingStarted={onPickingStarted} onOrderChosenForList={onOrderChosenForList} onLogout={vi.fn()} onBack={vi.fn()} />)
  return { onPickingStarted, onOrderChosenForList }
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(omsApi.listSalesOrders).mockResolvedValue({ salesOrders: [order], total: 1 })
})

describe('SalesOrderPickerScreen', () => {
  it('scan mode starts a picking session', async () => {
    mode('scan')
    vi.mocked(api.startOrResumePicking).mockResolvedValue({ id: 'ps-1' } as Awaited<ReturnType<typeof api.startOrResumePicking>>)
    const user = userEvent.setup()
    const { onPickingStarted, onOrderChosenForList } = renderScreen()

    await user.click(await screen.findByRole('button', { name: /Ann Lee/, hidden: false }))

    expect(api.startOrResumePicking).toHaveBeenCalledWith('tok', 'so-1')
    expect(onPickingStarted).toHaveBeenCalledWith('ps-1', order)
    expect(onOrderChosenForList).not.toHaveBeenCalled()
  })

  it('list mode opens the order item list, no session', async () => {
    mode('list')
    const user = userEvent.setup()
    const { onOrderChosenForList } = renderScreen()
    const row = await screen.findByRole('button', { name: /Ann Lee/ })
    await vi.waitFor(() => expect(row).toBeEnabled())

    await user.click(row)

    expect(onOrderChosenForList).toHaveBeenCalledWith('so-1')
    expect(api.startOrResumePicking).not.toHaveBeenCalled()
  })

  it('falls back to scan mode and says so when the mode cannot load', async () => {
    vi.mocked(settingsApi.getTenantSettings).mockRejectedValue(new Error('boom'))
    renderScreen()

    expect(await screen.findByText(/Couldn't load the picking mode \(boom\)/)).toBeInTheDocument()
  })
})
