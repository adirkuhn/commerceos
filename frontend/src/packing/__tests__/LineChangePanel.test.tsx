// @vitest-environment jsdom
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, it, expect, vi } from 'vitest'
import { LineChangePanel } from '../LineChangePanel'
import type { PickingSessionLineItem } from '../../scanner/types'

const line: PickingSessionLineItem = { id: 'li-1', sku_id: 'sku-1', name: 'Beans', sku_code: 'B-1', barcode: '111', quantity: 3, price_minor_units: 100, total_minor_units: 300 }

function renderPanel(coveredSoFar = 0) {
  const props = { onPack: vi.fn(), onSubstitute: vi.fn(), onRefund: vi.fn(), onClose: vi.fn() }
  render(<LineChangePanel lineItem={line} coveredSoFar={coveredSoFar} busy={false} {...props} />)
  return props
}

describe('LineChangePanel', () => {
  it('packs the remaining quantity by default', async () => {
    const user = userEvent.setup()
    const { onPack } = renderPanel(1)

    expect(screen.getByLabelText('Quantity')).toHaveValue(2)
    await user.click(screen.getByRole('button', { name: 'Confirm pack' }))

    expect(onPack).toHaveBeenCalledWith(2)
  })

  it('substitute needs rows covering exactly what is left', async () => {
    const user = userEvent.setup()
    const { onSubstitute } = renderPanel(1)
    await user.click(screen.getByRole('button', { name: /Substitute/ }))
    const confirm = screen.getByRole('button', { name: 'Confirm substitute' })

    await user.type(screen.getByPlaceholderText('Scan or type replacement SKU/barcode'), 'SUB-1')
    await user.type(screen.getByLabelText('Covers (of original)'), '1')
    expect(confirm).toBeDisabled()
    expect(screen.getByText('Covers 1 of 2')).toBeInTheDocument()

    await user.clear(screen.getByLabelText('Covers (of original)'))
    await user.type(screen.getByLabelText('Covers (of original)'), '2')
    await user.click(screen.getByLabelText('Return original quantity to stock'))
    await user.click(confirm)

    expect(onSubstitute).toHaveBeenCalledWith([{ code: 'SUB-1', substituteQuantity: 1, coveredQuantity: 2, returnToStock: true }])
  })

  it('substitute can split across several replacement products', async () => {
    const user = userEvent.setup()
    const { onSubstitute } = renderPanel(0)
    await user.click(screen.getByRole('button', { name: /Substitute/ }))
    await user.click(screen.getByRole('button', { name: '+ Add another substitute' }))

    const codes = screen.getAllByPlaceholderText('Scan or type replacement SKU/barcode')
    const covers = screen.getAllByLabelText('Covers (of original)')
    await user.type(codes[0], 'A')
    await user.type(covers[0], '1')
    await user.type(codes[1], 'B')
    await user.type(covers[1], '2')
    await user.click(screen.getByRole('button', { name: 'Confirm substitute' }))

    expect(onSubstitute).toHaveBeenCalledWith([
      { code: 'A', substituteQuantity: 1, coveredQuantity: 1, returnToStock: false },
      { code: 'B', substituteQuantity: 1, coveredQuantity: 2, returnToStock: false },
    ])
  })

  it('refund removes part of the line with a reason', async () => {
    const user = userEvent.setup()
    const { onRefund } = renderPanel(0)
    await user.click(screen.getByRole('button', { name: /Refund/ }))

    await user.clear(screen.getByLabelText('Quantity to refund'))
    await user.type(screen.getByLabelText('Quantity to refund'), '2')
    await user.type(screen.getByLabelText('Reason (optional)'), 'damaged')
    await user.click(screen.getByRole('button', { name: 'Confirm refund' }))

    expect(onRefund).toHaveBeenCalledWith(2, 'damaged')
  })

  it('refund cannot exceed the line quantity', async () => {
    const user = userEvent.setup()
    renderPanel(0)
    await user.click(screen.getByRole('button', { name: /Refund/ }))

    await user.clear(screen.getByLabelText('Quantity to refund'))
    await user.type(screen.getByLabelText('Quantity to refund'), '9')

    expect(screen.getByRole('button', { name: 'Confirm refund' })).toBeDisabled()
  })

  it('mode buttons expose which one is active', async () => {
    const user = userEvent.setup()
    renderPanel(0)
    expect(screen.getByRole('button', { name: 'Pack' })).toHaveAttribute('aria-pressed', 'true')

    await user.click(screen.getByRole('button', { name: /Refund/ }))

    expect(screen.getByRole('button', { name: /Refund/ })).toHaveAttribute('aria-pressed', 'true')
    expect(screen.getByRole('button', { name: 'Pack' })).toHaveAttribute('aria-pressed', 'false')
  })

  it('closes', async () => {
    const user = userEvent.setup()
    const { onClose } = renderPanel(0)

    await user.click(screen.getByRole('button', { name: 'Close' }))

    expect(onClose).toHaveBeenCalled()
  })
})
