// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { BookLineSheet } from '../BookingLineSheet'
import type { BookingScan } from '../types'

vi.mock('../../inventory/api', () => ({
  listBatches: vi.fn(),
}))

import * as inventoryApi from '../../inventory/api'

const lotReceive: BookingScan = {
  id: 'r1', tenant_id: 't', booking_id: 'booking-1', sku_id: 'sku-1', lot_number: '31/10/2027', batch_id: null,
  action_type: 'receive', quantity: 47, write_off_quantity: 0, scanned_at: '2026-10-28T09:59:36Z', scanned_by: 'u',
} as BookingScan

function renderSheet(onBook = vi.fn(), scans: BookingScan[] = [lotReceive]) {
  render(
    <BookLineSheet
      token="tok" skuId="sku-1" description="Pão de queijo 1kg" remaining={0} scannedSoFar={47} expected={47}
      warehouseId="wh-1" pendingBatches={[{ lotNumber: '31/10/2027', expiryDate: null }]}
      scans={scans} onDeleteScan={vi.fn()} onCancel={vi.fn()} onBook={onBook}
    />,
  )
  return onBook
}

describe('BookLineSheet take-from batch default', () => {
  beforeEach(() => {
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([])
  })

  // The real booking (2026-10-28): switching to Damaged/lost reset the batch
  // to General ledger, the write-off was staged there, and Finish failed
  // because every received unit sat in the new lot.
  it('switching to Damaged/lost preselects the lot this booking received into', async () => {
    const onBook = renderSheet()

    await userEvent.click(screen.getByRole('button', { name: /Damaged\/lost/ }))
    expect(screen.getByRole('button', { name: /31\/10\/2027/, pressed: true })).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /General ledger/, pressed: false })).toBeInTheDocument()

    await userEvent.type(screen.getByRole('textbox', { name: /quantity/i }), '1')
    await userEvent.click(screen.getByRole('button', { name: 'Confirm' }))

    expect(onBook).toHaveBeenCalledWith(expect.objectContaining({ action_type: 'damaged', quantity: 1, lot_number: '31/10/2027', batch_id: undefined }))
  })

  it('switching to Remove preselects the same lot', async () => {
    renderSheet()

    await userEvent.click(screen.getByRole('button', { name: /^Remove/ }))

    expect(screen.getByRole('button', { name: /31\/10\/2027/, pressed: true })).toBeInTheDocument()
  })

  it('stays on General ledger when nothing was received into a lot or batch', async () => {
    renderSheet(vi.fn(), [])

    await userEvent.click(screen.getByRole('button', { name: /Damaged\/lost/ }))

    expect(screen.getByRole('button', { name: /General ledger/, pressed: true })).toBeInTheDocument()
  })

  it('going back to Receive clears the take-from choice', async () => {
    renderSheet()

    await userEvent.click(screen.getByRole('button', { name: /Damaged\/lost/ }))
    await userEvent.click(screen.getByRole('button', { name: /^Receive/ }))

    expect(screen.getByRole('button', { name: /General ledger/, pressed: true })).toBeInTheDocument()
  })
})
