// @vitest-environment jsdom
import { render, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { useState } from 'react'
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { MergeProductDialog } from '../MergeProductDialog'
import type { Product, ProductMergePreview, ProductMergeResult } from '../types'

vi.mock('../api', () => ({ listProducts: vi.fn(), previewProductMerge: vi.fn(), mergeProduct: vi.fn() }))
import * as api from '../api'

const product = (id: string, name: string): Product => ({
  id, tenant_id: 't1', name, slug: id, description: '', status: 'active', has_variants: false, tags: [], created_at: '', updated_at: '',
})

const preview: ProductMergePreview = {
  source: { id: 'dup', name: 'Guarana (new)', skus: [{ id: 'd1', sku_code: 'GUA-NEW', barcode: '871', on_hand: 4, cost_price_minor_units: 200 }] },
  target: { id: 'orig', name: 'Guarana Antarctica', skus: [{ id: 'o1', sku_code: 'GUA-350', barcode: '789', on_hand: 6, cost_price_minor_units: 100 }] },
  sku_mapping: [{ source_sku_id: 'd1', target_sku_id: 'o1' }],
  targets: [{ sku_id: 'o1', on_hand_after: 10, cost_before_minor_units: 100, cost_after_minor_units: 140 }],
}

function Harness({ onMerged = vi.fn() }: { onMerged?: (r: ProductMergeResult) => void }) {
  const [targetId, setTargetId] = useState<string | null>(null)
  return (
    <MergeProductDialog
      token="t" duplicate={{ id: 'dup', name: 'Guarana (new)' }} currency="EUR"
      targetId={targetId} onChooseTarget={setTargetId} onMerged={onMerged} onClose={vi.fn()}
    />
  )
}

async function chooseTarget() {
  await userEvent.type(screen.getByLabelText('The existing product to keep'), 'Guar')
  await userEvent.click(await screen.findByRole('button', { name: 'Guarana Antarctica' }))
}

beforeEach(() => {
  vi.clearAllMocks()
  vi.mocked(api.listProducts).mockResolvedValue({ products: [product('dup', 'Guarana (new)'), product('orig', 'Guarana Antarctica')], total: 2 })
  vi.mocked(api.previewProductMerge).mockResolvedValue(preview)
})

describe('MergeProductDialog', () => {
  it('searches the existing product, never offering the duplicate itself', async () => {
    render(<Harness />)
    await userEvent.type(screen.getByLabelText('The existing product to keep'), 'Guar')

    expect(await screen.findByRole('button', { name: 'Guarana Antarctica' })).toBeInTheDocument()
    expect(screen.queryByRole('button', { name: 'Guarana (new)' })).not.toBeInTheDocument()
  })

  it('shows the SKU matching and stock and cost after the merge', async () => {
    render(<Harness />)
    await chooseTarget()

    await waitFor(() => expect(api.previewProductMerge).toHaveBeenCalledWith('t', 'dup', { target_product_id: 'orig' }))
    expect(await screen.findByLabelText('Merge GUA-NEW into')).toHaveValue('o1')
    const after = screen.getByLabelText('After the merge')
    expect(within(after).getByText('10')).toBeInTheDocument()
    expect(within(after).getByText(/1,40|1\.40/)).toBeInTheDocument()
  })

  it('merges only after typing the existing product name and a final confirmation', async () => {
    const onMerged = vi.fn()
    vi.mocked(api.mergeProduct).mockResolvedValue({ target_product_id: 'orig', skus: [] })
    render(<Harness onMerged={onMerged} />)
    await chooseTarget()
    const mergeButton = await screen.findByRole('button', { name: 'Merge permanently' })
    await screen.findByLabelText('After the merge')

    expect(mergeButton).toBeDisabled()
    await userEvent.type(screen.getByLabelText(/to confirm/), 'Guarana Antarctica')
    expect(mergeButton).toBeEnabled()

    await userEvent.click(mergeButton)
    const confirmDialog = await screen.findByRole('alertdialog')
    expect(within(confirmDialog).getByText(/will be deleted/)).toBeInTheDocument()
    await userEvent.click(within(confirmDialog).getByRole('button', { name: 'Merge now' }))

    await waitFor(() => expect(onMerged).toHaveBeenCalledWith({ target_product_id: 'orig', skus: [] }))
    expect(api.mergeProduct).toHaveBeenCalledWith('t', 'dup', { target_product_id: 'orig', sku_mapping: [{ source_sku_id: 'd1', target_sku_id: 'o1' }] })
  })

  it('shows why a merge failed', async () => {
    vi.mocked(api.mergeProduct).mockRejectedValue(new Error('only an owner or admin can merge products'))
    render(<Harness />)
    await chooseTarget()
    await screen.findByLabelText('After the merge')
    await userEvent.type(screen.getByLabelText(/to confirm/), 'Guarana Antarctica')
    await userEvent.click(screen.getByRole('button', { name: 'Merge permanently' }))
    await userEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: 'Merge now' }))

    expect(await screen.findByText('only an owner or admin can merge products')).toBeInTheDocument()
  })

  it('asks to match every SKU when products have several', async () => {
    vi.mocked(api.previewProductMerge).mockResolvedValue({
      ...preview,
      source: { ...preview.source, skus: [...preview.source.skus, { id: 'd2', sku_code: 'GUA-6', barcode: '', on_hand: 0 }] },
      sku_mapping: [{ source_sku_id: 'd1', target_sku_id: 'o1' }], targets: [],
    })
    render(<Harness />)
    await chooseTarget()

    expect(await screen.findByText('Match every SKU of this product to continue.')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Merge permanently' })).toBeDisabled()
  })
})
