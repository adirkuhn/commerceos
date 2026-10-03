// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import type { SKU } from '../../pms/types'

const picked: { sku: Partial<SKU> } = { sku: {} }
vi.mock('../../components/common/ProductSkuPicker', () => ({
  ProductSkuPicker: ({ onSelect }: { onSelect: (sku: SKU, label: string) => void }) => (
    <button type="button" onClick={() => onSelect({ id: 'sku-pool', ...picked.sku } as SKU, 'Carne Seca — WOO-17522')}>pick fake sku</button>
  ),
}))

import { LineItemFormRow } from '../LineItemFormRow'

function renderRow(onSave = vi.fn().mockResolvedValue(undefined)) {
  render(<table><tbody><LineItemFormRow currencyCode="EUR" onSubmit={onSave} onCancel={vi.fn()} /></tbody></table>)
  return onSave
}

describe('LineItemFormRow: linking a weight pool', () => {
  it('books per kg: units per purchased item becomes 1000', async () => {
    picked.sku = { unit_type: 'kilo' }
    renderRow()

    await userEvent.click(screen.getByRole('button', { name: /link product/i }))
    await userEvent.click(screen.getByRole('button', { name: 'pick fake sku' }))

    expect(screen.getByLabelText(/units per purchased item/i)).toHaveValue(1000)
  })

  it('leaves an ordinary product at 1', async () => {
    picked.sku = { unit_type: 'unit' }
    renderRow()

    await userEvent.click(screen.getByRole('button', { name: /link product/i }))
    await userEvent.click(screen.getByRole('button', { name: 'pick fake sku' }))

    expect(screen.getByLabelText(/units per purchased item/i)).toHaveValue(1)
  })
})
