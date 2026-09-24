// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { ClearanceBatchSelect } from '../ClearanceBatchSelect'
import type { Batch } from '../../../inventory/types'

vi.mock('../../../inventory/api', () => ({
  listBatches: vi.fn(),
}))

import * as inventoryApi from '../../../inventory/api'

function batch(overrides: Partial<Batch> = {}): Batch {
  return {
    id: 'batch-1', tenant_id: 't', sku_id: 'sku-1', warehouse_id: 'wh-1', lot_number: 'L-42',
    expiry_date: '2099-10-01T00:00:00Z', received_at: '', created_at: '', updated_at: '',
    quantity: 7, is_general_ledger: false, ...overrides,
  }
}

beforeEach(() => vi.clearAllMocks())

describe('ClearanceBatchSelect', () => {
  it('lists only real batches that still have stock and have not expired', async () => {
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([
      batch({ id: 'general', is_general_ledger: true, lot_number: '', quantity: 50 }),
      batch(),
      batch({ id: 'empty', lot_number: 'L-EMPTY', quantity: 0 }),
      batch({ id: 'old', lot_number: 'L-OLD', expiry_date: '2000-01-01T00:00:00Z' }),
    ])

    render(<ClearanceBatchSelect token="tok" skuId="sku-1" value={null} onChange={vi.fn()} />)

    const select = await screen.findByLabelText('Clearance batch')
    expect(await screen.findByRole('option', { name: /L-42 .* 7 left/ })).toBeInTheDocument()
    expect(screen.getByRole('option', { name: 'Whole SKU (no batch)' })).toBeInTheDocument()
    expect(select.querySelectorAll('option')).toHaveLength(2)
    expect(inventoryApi.listBatches).toHaveBeenCalledWith('tok', 'sku-1')
  })

  it('reports the chosen batch, and null for the whole SKU', async () => {
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch()])
    const onChange = vi.fn()

    render(<ClearanceBatchSelect token="tok" skuId="sku-1" value={null} onChange={onChange} />)
    await screen.findByRole('option', { name: /L-42/ })
    await userEvent.selectOptions(screen.getByLabelText('Clearance batch'), 'batch-1')
    expect(onChange).toHaveBeenLastCalledWith('batch-1')

    await userEvent.selectOptions(screen.getByLabelText('Clearance batch'), '')
    expect(onChange).toHaveBeenLastCalledWith(null)
  })

  it('keeps showing a batch the line already has even once it is finished', async () => {
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([batch({ quantity: 0 })])

    render(<ClearanceBatchSelect token="tok" skuId="sku-1" value="batch-1" onChange={vi.fn()} />)

    expect(await screen.findByRole('option', { name: /L-42/ })).toBeInTheDocument()
    expect(screen.getByLabelText('Clearance batch')).toHaveValue('batch-1')
  })

  it('surfaces a load error', async () => {
    vi.mocked(inventoryApi.listBatches).mockRejectedValue(new Error('boom'))

    render(<ClearanceBatchSelect token="tok" skuId="sku-1" value={null} onChange={vi.fn()} />)

    expect(await screen.findByText('boom')).toBeInTheDocument()
  })
})
