// @vitest-environment jsdom
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { BulkAddWeightPortionsModal } from '../BulkAddWeightPortionsModal'
import type { Warehouse, Batch } from '../../inventory/types'

vi.mock('../../inventory/api', () => ({
  listWarehouses: vi.fn(),
  listBatches: vi.fn(),
}))

vi.mock('../api', () => ({
  bulkAddWeightPortions: vi.fn(),
}))

import * as inventoryApi from '../../inventory/api'
import * as pmsApi from '../api'

const warehouse: Warehouse = {
  id: 'warehouse-1', tenant_id: 't1', name: 'Main DC', address1: '', address2: '', city: '', province: '',
  zip: '', country: '', phone: '', active: true, created_at: '', updated_at: '',
}

const bulkBatch: Batch = {
  id: 'batch-lot-1', tenant_id: 't1', sku_id: 'sku-bulk-1', warehouse_id: 'warehouse-1',
  lot_number: 'LOT-9', expiry_date: '2026-11-01T00:00:00Z', received_at: '', created_at: '', updated_at: '',
  quantity: 30000, is_general_ledger: false,
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(inventoryApi.listWarehouses).mockResolvedValue([warehouse])
  vi.mocked(inventoryApi.listBatches).mockResolvedValue([bulkBatch])
})

function renderModal(overrides: Partial<Parameters<typeof BulkAddWeightPortionsModal>[0]> = {}) {
  const onCreated = vi.fn()
  const onClose = vi.fn()
  render(
    <BulkAddWeightPortionsModal
      token="test-token" bulkSkuId="sku-bulk-1" pricePerKiloMinorUnits={29990} currencyCode="EUR"
      onCreated={onCreated} onClose={onClose} {...overrides}
    />,
  )
  return { onCreated, onClose }
}

describe('BulkAddWeightPortionsModal', () => {
  it('fetches the bulk sku\'s own warehouses and batches on mount', async () => {
    renderModal()

    await waitFor(() => expect(inventoryApi.listWarehouses).toHaveBeenCalledWith('test-token'))
    await waitFor(() => expect(inventoryApi.listBatches).toHaveBeenCalledWith('test-token', 'sku-bulk-1'))
    expect(await screen.findByRole('option', { name: /LOT-9/ })).toBeInTheDocument()
  })

  it('shows a live price preview per entered weight, computed from price per kilo', async () => {
    renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })

    await userEvent.type(screen.getByLabelText('Weights (grams)'), '404, 458')

    // 29990 * 404 / 1000 = 12115.96 -> 12116 minor units -> EUR 121.16
    expect(await screen.findByText('EUR 121.16')).toBeInTheDocument()
    // 29990 * 458 / 1000 = 13735.42 -> 13735 minor units -> EUR 137.35
    expect(await screen.findByText('EUR 137.35')).toBeInTheDocument()
  })

  it('rejects a non-numeric weight and disables submit', async () => {
    renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })

    await userEvent.type(screen.getByLabelText('Weights (grams)'), '404, abc')

    expect(await screen.findByText(/isn't a positive whole number/)).toBeInTheDocument()
    expect(screen.getByRole('button', { name: /Create/ })).toBeDisabled()
  })

  it('submits the parsed weights, chosen warehouse/batch, and closes on success', async () => {
    vi.mocked(pmsApi.bulkAddWeightPortions).mockResolvedValue({
      created: [{ id: 'sku-1', sku_code: 'CARNE-SECA-404G', barcode: '2994721004047', price_minor_units: 12116, currency_code: 'EUR', weight_grams: 404 }],
    })
    const { onCreated, onClose } = renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })

    await userEvent.type(screen.getByLabelText('Weights (grams)'), '404')
    await userEvent.selectOptions(screen.getByLabelText('Bulk lot (optional)'), 'batch-lot-1')
    await userEvent.click(screen.getByRole('button', { name: /Create 1 item/ }))

    await waitFor(() => expect(pmsApi.bulkAddWeightPortions).toHaveBeenCalledWith('test-token', 'sku-bulk-1', {
      warehouse_id: 'warehouse-1', bulk_batch_id: 'batch-lot-1', weights_grams: [404],
      sku_code_prefix: undefined, reason: undefined,
    }))
    await waitFor(() => expect(onCreated).toHaveBeenCalledWith([expect.objectContaining({ sku_code: 'CARNE-SECA-404G' })]))
    expect(onClose).toHaveBeenCalled()
  })

  it('sends bulk_batch_id null when no lot is picked (general ledger)', async () => {
    vi.mocked(pmsApi.bulkAddWeightPortions).mockResolvedValue({ created: [] })
    renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })

    await userEvent.type(screen.getByLabelText('Weights (grams)'), '404')
    await userEvent.click(screen.getByRole('button', { name: /Create 1 item/ }))

    await waitFor(() => expect(pmsApi.bulkAddWeightPortions).toHaveBeenCalledWith('test-token', 'sku-bulk-1', expect.objectContaining({
      bulk_batch_id: null,
    })))
  })

  it('shows an error if the create call fails', async () => {
    vi.mocked(pmsApi.bulkAddWeightPortions).mockRejectedValue(new Error('source sku must be a weight pool with per-weight pricing and a scale code'))
    renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })

    await userEvent.type(screen.getByLabelText('Weights (grams)'), '404')
    await userEvent.click(screen.getByRole('button', { name: /Create 1 item/ }))

    expect(await screen.findByText(/must be a weight pool/)).toBeInTheDocument()
  })

  it('disables submit until at least one weight is entered', async () => {
    renderModal()
    await screen.findByRole('option', { name: /LOT-9/ })

    expect(screen.getByRole('button', { name: /Create/ })).toBeDisabled()
  })
})
