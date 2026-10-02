// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { SplitIntoPackagesModal } from '../SplitIntoPackagesModal'
import type { Warehouse, Batch } from '../../inventory/types'

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
  listBatches: vi.fn(),
}))

vi.mock('../api', () => ({
  packWeightPortions: vi.fn(),
}))

import * as inventoryApi from '../../inventory/api'
import * as pmsApi from '../api'

const warehouse: Warehouse = {
  id: 'warehouse-1', tenant_id: 't1', name: 'Main DC', address1: '', address2: '', city: '', province: '',
  zip: '', country: '', phone: '', active: true, created_at: '', updated_at: '',
}

function batch(overrides: Partial<Batch>): Batch {
  return {
    id: 'batch-lot-1', tenant_id: 't1', sku_id: 'sku-bulk-1', warehouse_id: 'warehouse-1',
    lot_number: 'LOT-9', expiry_date: '2030-10-10T00:00:00Z', received_at: '', created_at: '', updated_at: '',
    quantity: 5000, is_general_ledger: false, ...overrides,
  }
}

const lot = batch({})
const general = batch({ id: '', lot_number: '', expiry_date: null, quantity: 2000, is_general_ledger: true })
const emptyLot = batch({ id: 'batch-empty', lot_number: 'LOT-0', quantity: 0 })

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse])
  vi.mocked(inventoryApi.listBatches).mockResolvedValue([general, lot, emptyLot])
})

function renderModal(overrides: Partial<Parameters<typeof SplitIntoPackagesModal>[0]> = {}) {
  const onPacked = vi.fn()
  const onClose = vi.fn()
  render(
    <SplitIntoPackagesModal
      token="test-token" bulkSkuId="sku-bulk-1" pricePerKiloMinorUnits={29990} currencyCode="EUR"
      onPacked={onPacked} onClose={onClose} {...overrides}
    />,
  )
  return { onPacked, onClose }
}

async function addLine(quantity: string, grams: string) {
  const qty = screen.getByLabelText('Packages')
  await userEvent.clear(qty)
  await userEvent.type(qty, quantity)
  await userEvent.type(screen.getByLabelText('Weight (g)'), `${grams}{Enter}`)
}

describe('SplitIntoPackagesModal', () => {
  it('offers only lots that still have stock, preselecting the first lot', async () => {
    renderModal()

    const source = await screen.findByLabelText('From lot')
    await screen.findByRole('option', { name: /LOT-9/ })
    expect(within(source).getAllByRole('option')).toHaveLength(2)
    expect(screen.queryByRole('option', { name: /LOT-0/ })).not.toBeInTheDocument()
    expect(screen.getByRole('option', { name: /LOT-9/ })).toHaveTextContent('5.00 kg left')
    expect((source as HTMLSelectElement).value).toBe('warehouse-1|batch-lot-1')
  })

  it('preselects the lot it was opened from', async () => {
    renderModal({ initialScope: { warehouseId: 'warehouse-1', batchId: '' } })

    await screen.findByRole('option', { name: /LOT-9/ })
    expect((screen.getByLabelText('From lot') as HTMLSelectElement).value).toBe('warehouse-1|')
  })

  it('adds quantity x weight lines with Enter, folding a repeated weight into its line', async () => {
    renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })

    await addLine('10', '380')
    await addLine('1', '430')
    await addLine('2', '380')

    expect(screen.getByText('12 × 380g')).toBeInTheDocument()
    expect(screen.getByText('1 × 430g')).toBeInTheDocument()
    // 299.90/kg x 380g = 113.96 -> rounded down to EUR 113.89 each
    expect(screen.getByText(/113\.89/)).toBeInTheDocument()
    expect(screen.getByText(/2 weights · 13 packages · 4\.99 kg of 5\.00 kg/)).toBeInTheDocument()
  })

  it('removes a line', async () => {
    renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })
    await addLine('1', '430')

    await userEvent.click(screen.getByRole('button', { name: 'Remove 430g' }))

    expect(screen.queryByText('1 × 430g')).not.toBeInTheDocument()
  })

  it('blocks a list heavier than what is left in the lot', async () => {
    renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })

    await addLine('20', '380')

    expect(screen.getByText(/more than this lot has left/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Create packages' })).toBeDisabled()
  })

  it('submits the lines from the chosen lot, then shows what was created and restocked', async () => {
    vi.mocked(pmsApi.packWeightPortions).mockResolvedValue({ lines: [
      { sku_id: 's1', sku_code: 'CARNE-SECA-380G', barcode: 'x', price_minor_units: 11396, currency_code: 'EUR', weight_grams: 380, quantity: 10, created: false },
      { sku_id: 's2', sku_code: 'CARNE-SECA-430G', barcode: 'y', price_minor_units: 12896, currency_code: 'EUR', weight_grams: 430, quantity: 1, created: true },
    ] })
    const { onPacked, onClose } = renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })
    await addLine('10', '380')
    await addLine('1', '430')

    await userEvent.click(screen.getByRole('button', { name: 'Create packages' }))

    await waitFor(() => expect(pmsApi.packWeightPortions).toHaveBeenCalledWith('test-token', 'sku-bulk-1', {
      warehouse_id: 'warehouse-1', bulk_batch_id: 'batch-lot-1',
      lines: [{ weight_grams: 380, quantity: 10 }, { weight_grams: 430, quantity: 1 }],
      sku_code_prefix: undefined, reason: undefined,
    }))
    expect(await screen.findByText('Packages created')).toBeInTheDocument()
    expect(screen.getByText('CARNE-SECA-380G')).toBeInTheDocument()
    expect(screen.getByText(/\+10 · restocked/)).toBeInTheDocument()
    expect(screen.getByText(/\+1 · new/)).toBeInTheDocument()
    expect(onPacked).toHaveBeenCalled()
    expect(onClose).not.toHaveBeenCalled()
  })

  it('sends bulk_batch_id null for general-ledger stock', async () => {
    vi.mocked(pmsApi.packWeightPortions).mockResolvedValue({ lines: [] })
    renderModal({ initialScope: { warehouseId: 'warehouse-1', batchId: '' } })
    await screen.findByRole('option', { name: /LOT-9/ })
    await addLine('1', '380')

    await userEvent.click(screen.getByRole('button', { name: 'Create packages' }))

    await waitFor(() => expect(pmsApi.packWeightPortions).toHaveBeenCalledWith('test-token', 'sku-bulk-1', expect.objectContaining({
      warehouse_id: 'warehouse-1', bulk_batch_id: null,
    })))
  })

  it('shows the server error and keeps the lines', async () => {
    vi.mocked(pmsApi.packWeightPortions).mockRejectedValue(new Error('packWeightPortions: sku code already exists'))
    renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })
    await addLine('1', '380')

    await userEvent.click(screen.getByRole('button', { name: 'Create packages' }))

    expect(await screen.findByText(/sku code already exists/)).toBeInTheDocument()
    expect(screen.getByText('1 × 380g')).toBeInTheDocument()
  })

  it('shows an error when the lots fail to load', async () => {
    vi.mocked(inventoryApi.listBatches).mockRejectedValue(new Error('network down'))
    renderModal()

    expect(await screen.findByText(/network down/)).toBeInTheDocument()
  })

  it('says so when the pool has no stock to split', async () => {
    vi.mocked(inventoryApi.listBatches).mockResolvedValue([emptyLot])
    renderModal()

    expect(await screen.findByText(/No stock in this weight pool/)).toBeInTheDocument()
  })
})
